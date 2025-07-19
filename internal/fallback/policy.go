// Package fallback provides fallback mechanisms for pocket workflows.
package fallback

import (
	"context"
	"fmt"
	"time"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/internal/retry"
)

// Policy defines a fallback strategy.
type Policy interface {
	// Execute runs the primary function with fallback support.
	Execute(ctx context.Context, store pocket.Store, input any) (any, error)
	// Name returns the policy name.
	Name() string
}

// Handler is a function that can serve as a fallback.
type Handler func(ctx context.Context, store pocket.StoreWriter, input any, err error) (any, error)

// SimplePolicy provides basic fallback functionality.
type SimplePolicy struct {
	name     string
	primary  pocket.ExecFunc
	fallback Handler
}

// NewSimplePolicy creates a basic fallback policy.
func NewSimplePolicy(name string, primary pocket.ExecFunc, fallback Handler) *SimplePolicy {
	return &SimplePolicy{
		name:     name,
		primary:  primary,
		fallback: fallback,
	}
}

// Name returns the policy name.
func (p *SimplePolicy) Name() string {
	return p.name
}

// Execute runs with fallback support.
func (p *SimplePolicy) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	result, err := p.primary(ctx, input)
	if err == nil {
		return result, nil
	}

	// Log the primary failure
	_ = store.Set(ctx, fmt.Sprintf("fallback:%s:primary_error", p.name), err)

	// Execute fallback
	return p.fallback(ctx, store, input, err)
}

// ChainPolicy executes a chain of handlers until one succeeds.
type ChainPolicy struct {
	name     string
	handlers []pocket.ExecFunc
	options  chainOptions
}

type chainOptions struct {
	stopOnFirstSuccess bool
	collectErrors      bool
	timeout            time.Duration
}

// ChainOption configures a chain policy.
type ChainOption func(*chainOptions)

// StopOnFirstSuccess stops the chain when a handler succeeds.
func StopOnFirstSuccess() ChainOption {
	return func(o *chainOptions) {
		o.stopOnFirstSuccess = true
	}
}

// CollectErrors stores all errors encountered.
func CollectErrors() ChainOption {
	return func(o *chainOptions) {
		o.collectErrors = true
	}
}

// WithTimeout sets a timeout for the entire chain.
func WithTimeout(d time.Duration) ChainOption {
	return func(o *chainOptions) {
		o.timeout = d
	}
}

// NewChainPolicy creates a chain fallback policy.
func NewChainPolicy(name string, handlers []pocket.ExecFunc, opts ...ChainOption) *ChainPolicy {
	policy := &ChainPolicy{
		name:     name,
		handlers: handlers,
		options: chainOptions{
			stopOnFirstSuccess: true,
		},
	}

	for _, opt := range opts {
		opt(&policy.options)
	}

	return policy
}

// Name returns the policy name.
func (p *ChainPolicy) Name() string {
	return p.name
}

// Execute runs the chain of handlers.
func (p *ChainPolicy) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	if p.options.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.options.timeout)
		defer cancel()
	}

	var errors []error
	var lastResult any

	for i, handler := range p.handlers {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("chain timed out at handler %d: %w", i, ctx.Err())
		default:
		}

		result, err := handler(ctx, input)
		
		if err == nil {
			if p.options.stopOnFirstSuccess {
				return result, nil
			}
			lastResult = result
		} else if p.options.collectErrors {
			errors = append(errors, fmt.Errorf("handler %d: %w", i, err))
		}

		// Store intermediate results
		_ = store.Set(ctx, fmt.Sprintf("fallback:%s:handler_%d_result", p.name, i), result)
		_ = store.Set(ctx, fmt.Sprintf("fallback:%s:handler_%d_error", p.name, i), err)
	}

	if !p.options.stopOnFirstSuccess && lastResult != nil {
		return lastResult, nil
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("all handlers failed: %v", errors)
	}

	return nil, fmt.Errorf("all %d handlers failed", len(p.handlers))
}

// RetryWithFallbackPolicy combines retry and fallback strategies.
type RetryWithFallbackPolicy struct {
	name        string
	primary     pocket.ExecFunc
	fallback    Handler
	retryPolicy retry.Policy
}

// NewRetryWithFallbackPolicy creates a policy that retries before falling back.
func NewRetryWithFallbackPolicy(name string, primary pocket.ExecFunc, fallback Handler, retryPolicy retry.Policy) *RetryWithFallbackPolicy {
	return &RetryWithFallbackPolicy{
		name:        name,
		primary:     primary,
		fallback:    fallback,
		retryPolicy: retryPolicy,
	}
}

// Name returns the policy name.
func (p *RetryWithFallbackPolicy) Name() string {
	return p.name
}

// Execute runs with retry and fallback support.
func (p *RetryWithFallbackPolicy) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	// Try with retry first
	var primaryErr error
	err := p.retryPolicy.Do(ctx, func() error {
		result, err := p.primary(ctx, input)
		if err != nil {
			primaryErr = err
			return err
		}
		// Store successful result
		store.Set(ctx, fmt.Sprintf("fallback:%s:result", p.name), result)
		return nil
	})

	if err == nil {
		// Retrieve the result
		result, _ := store.Get(ctx, fmt.Sprintf("fallback:%s:result", p.name))
		return result, nil
	}

	// All retries failed, use fallback
	store.Set(ctx, fmt.Sprintf("fallback:%s:retry_failed", p.name), primaryErr)
	return p.fallback(ctx, store, input, primaryErr)
}

// CachedFallbackPolicy uses cached results as fallback.
type CachedFallbackPolicy struct {
	name       string
	primary    pocket.ExecFunc
	cacheKey   func(input any) string
	ttl        time.Duration
	staleOK    bool
}

// NewCachedFallbackPolicy creates a policy that falls back to cached results.
func NewCachedFallbackPolicy(name string, primary pocket.ExecFunc, cacheKey func(any) string) *CachedFallbackPolicy {
	return &CachedFallbackPolicy{
		name:     name,
		primary:  primary,
		cacheKey: cacheKey,
		ttl:      5 * time.Minute,
		staleOK:  true,
	}
}

// WithTTL sets the cache TTL.
func (p *CachedFallbackPolicy) WithTTL(ttl time.Duration) *CachedFallbackPolicy {
	p.ttl = ttl
	return p
}

// WithStaleOK allows serving stale cache on failure.
func (p *CachedFallbackPolicy) WithStaleOK(ok bool) *CachedFallbackPolicy {
	p.staleOK = ok
	return p
}

// Name returns the policy name.
func (p *CachedFallbackPolicy) Name() string {
	return p.name
}

// Execute runs with cache fallback support.
func (p *CachedFallbackPolicy) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	cacheKey := fmt.Sprintf("cache:%s:%s", p.name, p.cacheKey(input))
	timestampKey := cacheKey + ":timestamp"

	// Try primary function
	result, err := p.primary(ctx, input)
	if err == nil {
		// Update cache
		store.Set(ctx, cacheKey, result)
		store.Set(ctx, timestampKey, time.Now())
		return result, nil
	}

	// Check cache
	cached, exists := store.Get(ctx, cacheKey)
	if !exists {
		return nil, fmt.Errorf("primary failed and no cache available: %w", err)
	}

	// Check if cache is still valid
	if timestamp, ok := store.Get(ctx, timestampKey); ok {
		if ts, ok := timestamp.(time.Time); ok {
			age := time.Since(ts)
			if age <= p.ttl {
				// Cache is fresh
				return cached, nil
			}
			if !p.staleOK {
				return nil, fmt.Errorf("primary failed and cache is stale (%v old): %w", age, err)
			}
		}
	}

	// Return stale cache with warning
	store.Set(ctx, fmt.Sprintf("fallback:%s:stale_cache_used", p.name), true)
	return cached, nil
}

// PolicyBuilder helps construct complex fallback policies.
type PolicyBuilder struct {
	name     string
	policies []Policy
}

// compositePolicy chains multiple policies together.
type compositePolicy struct {
	name     string
	policies []Policy
}

// Name returns the policy name.
func (p *compositePolicy) Name() string {
	return p.name
}

// Execute runs each policy in sequence until one succeeds.
func (p *compositePolicy) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	var lastErr error
	for i, policy := range p.policies {
		result, err := policy.Execute(ctx, store, input)
		if err == nil {
			return result, nil
		}
		lastErr = err
		// Store the error for debugging
		store.Set(ctx, fmt.Sprintf("composite:%s:policy_%d_error", p.name, i), err)
	}
	return nil, fmt.Errorf("all %d policies failed, last error: %w", len(p.policies), lastErr)
}

// NewPolicyBuilder creates a new policy builder.
func NewPolicyBuilder(name string) *PolicyBuilder {
	return &PolicyBuilder{
		name:     name,
		policies: []Policy{},
	}
}

// Add adds a policy to the builder.
func (b *PolicyBuilder) Add(policy Policy) *PolicyBuilder {
	b.policies = append(b.policies, policy)
	return b
}

// Build creates a composite policy.
func (b *PolicyBuilder) Build() Policy {
	if len(b.policies) == 1 {
		return b.policies[0]
	}

	// Create a composite policy that chains the policies
	// Note: This is a workaround for the design issue where policies need store access
	// but ExecFunc doesn't provide it. The policies should be redesigned.
	return &compositePolicy{
		name:     b.name,
		policies: b.policies,
	}
}

// ToNode converts a fallback policy to a pocket Node.
func ToNode(policy Policy) *pocket.Node {
	return pocket.NewNode[any, any](policy.Name(),
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Pass input and store to exec phase
			return map[string]interface{}{
				"input": input,
				"store": store,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Extract store and input
			data := prepData.(map[string]interface{})
			store := data["store"].(pocket.Store)
			input := data["input"]
			
			return policy.Execute(ctx, store, input)
		}),
	)
}