// Package fallback provides circuit breaker pattern implementation.
package fallback

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed allows requests to pass through.
	StateClosed CircuitState = iota
	// StateOpen blocks all requests.
	StateOpen
	// StateHalfOpen allows limited requests to test recovery.
	StateHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	name string

	// Configuration
	maxFailures      int
	resetTimeout     time.Duration
	halfOpenRequests int

	// State
	mu                sync.RWMutex
	state             CircuitState
	failures          int
	lastFailureTime   time.Time
	halfOpenSuccesses int
	halfOpenFailures  int

	// Metrics
	totalRequests  int64
	totalSuccesses int64
	totalFailures  int64
	circuitOpens   int64
	lastOpenTime   time.Time

	// Callbacks
	onStateChange func(from, to CircuitState)
}

// CircuitOption configures a circuit breaker.
type CircuitOption func(*CircuitBreaker)

// WithMaxFailures sets the failure threshold.
func WithMaxFailures(n int) CircuitOption {
	return func(cb *CircuitBreaker) {
		cb.maxFailures = n
	}
}

// WithResetTimeout sets the timeout before attempting recovery.
func WithResetTimeout(d time.Duration) CircuitOption {
	return func(cb *CircuitBreaker) {
		cb.resetTimeout = d
	}
}

// WithHalfOpenRequests sets the number of test requests in half-open state.
func WithHalfOpenRequests(n int) CircuitOption {
	return func(cb *CircuitBreaker) {
		cb.halfOpenRequests = n
	}
}

// WithStateChangeCallback sets a callback for state transitions.
func WithStateChangeCallback(fn func(from, to CircuitState)) CircuitOption {
	return func(cb *CircuitBreaker) {
		cb.onStateChange = fn
	}
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, opts ...CircuitOption) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             name,
		maxFailures:      5,
		resetTimeout:     30 * time.Second,
		halfOpenRequests: 3,
		state:            StateClosed,
	}

	for _, opt := range opts {
		opt(cb)
	}

	return cb
}

// Execute runs the given function through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, store pocket.Store, fn pocket.ExecFunc, input any) (any, error) {
	// Check if we can execute
	if err := cb.canExecute(); err != nil {
		return nil, err
	}

	// Execute the function
	result, err := fn(ctx, input)

	// Record the result
	cb.recordResult(err == nil)

	return result, err
}

// canExecute checks if the circuit allows execution.
func (cb *CircuitBreaker) canExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.transitionTo(StateHalfOpen)
			return nil
		}
		return fmt.Errorf("circuit breaker %s is open", cb.name)

	case StateHalfOpen:
		// Check if we've hit the limit for half-open requests
		totalHalfOpen := cb.halfOpenSuccesses + cb.halfOpenFailures
		if totalHalfOpen >= cb.halfOpenRequests {
			// Determine next state based on results
			if cb.halfOpenFailures > 0 {
				// Any failure in half-open goes back to open
				cb.transitionTo(StateOpen)
				return fmt.Errorf("circuit breaker %s is open", cb.name)
			} else {
				// All successes, close the circuit
				cb.transitionTo(StateClosed)
				return nil
			}
		}
		return nil

	default:
		return fmt.Errorf("circuit breaker %s in unknown state", cb.name)
	}
}

// recordResult updates the circuit breaker state based on execution result.
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.totalSuccesses++
		cb.onSuccess()
	} else {
		cb.totalFailures++
		cb.onFailure()
	}
}

// onSuccess handles successful execution.
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		// Reset failure count on success
		cb.failures = 0

	case StateHalfOpen:
		cb.halfOpenSuccesses++
		// Check if we should close the circuit
		if cb.halfOpenSuccesses >= cb.halfOpenRequests {
			cb.transitionTo(StateClosed)
		}
	}
}

// onFailure handles failed execution.
func (cb *CircuitBreaker) onFailure() {
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		cb.halfOpenFailures++
		// Any failure in half-open immediately opens the circuit
		cb.transitionTo(StateOpen)
	}
}

// transitionTo changes the circuit breaker state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	// Reset counters for new state
	switch newState {
	case StateClosed:
		cb.failures = 0
		cb.halfOpenSuccesses = 0
		cb.halfOpenFailures = 0

	case StateOpen:
		cb.circuitOpens++
		cb.lastOpenTime = time.Now()
		cb.lastFailureTime = time.Now()

	case StateHalfOpen:
		cb.halfOpenSuccesses = 0
		cb.halfOpenFailures = 0
	}

	// Call state change callback if set
	if cb.onStateChange != nil {
		// Call in goroutine to avoid holding lock
		go cb.onStateChange(oldState, newState)
	}
}

// GetState returns the current circuit state.
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns circuit breaker metrics.
func (cb *CircuitBreaker) GetMetrics() CircuitMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitMetrics{
		Name:            cb.name,
		State:           cb.state.String(),
		TotalRequests:   cb.totalRequests,
		TotalSuccesses:  cb.totalSuccesses,
		TotalFailures:   cb.totalFailures,
		CircuitOpens:    cb.circuitOpens,
		LastOpenTime:    cb.lastOpenTime,
		CurrentFailures: cb.failures,
	}
}

// CircuitMetrics contains circuit breaker statistics.
type CircuitMetrics struct {
	Name            string
	State           string
	TotalRequests   int64
	TotalSuccesses  int64
	TotalFailures   int64
	CircuitOpens    int64
	LastOpenTime    time.Time
	CurrentFailures int
}

// CircuitBreakerPolicy wraps an exec function with circuit breaker protection.
type CircuitBreakerPolicy struct {
	name     string
	breaker  *CircuitBreaker
	primary  pocket.ExecFunc
	fallback Handler
}

// NewCircuitBreakerPolicy creates a policy with circuit breaker protection.
func NewCircuitBreakerPolicy(name string, primary pocket.ExecFunc, fallback Handler, opts ...CircuitOption) *CircuitBreakerPolicy {
	return &CircuitBreakerPolicy{
		name:     name,
		breaker:  NewCircuitBreaker(name, opts...),
		primary:  primary,
		fallback: fallback,
	}
}

// Name returns the policy name.
func (p *CircuitBreakerPolicy) Name() string {
	return p.name
}

// Execute runs with circuit breaker protection.
func (p *CircuitBreakerPolicy) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	// Try primary through circuit breaker
	result, err := p.breaker.Execute(ctx, store, p.primary, input)
	if err == nil {
		return result, nil
	}

	// Store circuit breaker error
	_ = store.Set(ctx, fmt.Sprintf("circuit:%s:error", p.name), err)
	_ = store.Set(ctx, fmt.Sprintf("circuit:%s:state", p.name), p.breaker.GetState().String())

	// Use fallback if available
	if p.fallback != nil {
		return p.fallback(ctx, store, input, err)
	}

	return nil, err
}

// ToNode converts a circuit breaker policy to a pocket Node.
func ToCircuitBreakerNode(name string, primary pocket.ExecFunc, fallback Handler, opts ...CircuitOption) pocket.Node {
	policy := NewCircuitBreakerPolicy(name, primary, fallback, opts...)

	return pocket.NewNode[any, any](name, pocket.Steps{
		Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Pass input and store to exec step
			return map[string]interface{}{
				"input": input,
				"store": store,
			}, nil
		},
		Exec: func(ctx context.Context, prepData any) (any, error) {
			// Extract store and input
			data := prepData.(map[string]interface{})
			store := data["store"].(pocket.Store)
			input := data["input"]

			return policy.Execute(ctx, store, input)
		},
	})
}

// CircuitBreakerGroup manages multiple circuit breakers.
type CircuitBreakerGroup struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerGroup creates a new circuit breaker group.
func NewCircuitBreakerGroup() *CircuitBreakerGroup {
	return &CircuitBreakerGroup{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Get returns a circuit breaker by name, creating it if necessary.
func (g *CircuitBreakerGroup) Get(name string, opts ...CircuitOption) *CircuitBreaker {
	g.mu.RLock()
	cb, exists := g.breakers[name]
	g.mu.RUnlock()

	if exists {
		return cb
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := g.breakers[name]; exists {
		return cb
	}

	// Create new circuit breaker
	cb = NewCircuitBreaker(name, opts...)
	g.breakers[name] = cb

	return cb
}

// GetAllMetrics returns metrics for all circuit breakers in the group.
func (g *CircuitBreakerGroup) GetAllMetrics() []CircuitMetrics {
	g.mu.RLock()
	defer g.mu.RUnlock()

	metrics := make([]CircuitMetrics, 0, len(g.breakers))
	for _, cb := range g.breakers {
		metrics = append(metrics, cb.GetMetrics())
	}

	return metrics
}

// Reset resets all circuit breakers in the group.
func (g *CircuitBreakerGroup) Reset() {
	g.mu.RLock()
	breakers := make([]*CircuitBreaker, 0, len(g.breakers))
	for _, cb := range g.breakers {
		breakers = append(breakers, cb)
	}
	g.mu.RUnlock()

	for _, cb := range breakers {
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failures = 0
		cb.mu.Unlock()
	}
}
