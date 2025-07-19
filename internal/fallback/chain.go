package fallback

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Chain represents a sophisticated fallback chain with advanced features.
type Chain struct {
	name     string
	links    []Link
	strategy Strategy
	metrics  *Metrics
	mu       sync.RWMutex
}

// Link represents a single link in the fallback chain.
type Link struct {
	Name      string
	Handler   pocket.ExecFunc
	Weight    float64
	Condition func(ctx context.Context, store pocket.Store, input any) bool
	Transform func(input any) any
}

// Strategy defines how the chain executes.
type Strategy interface {
	Execute(ctx context.Context, chain *Chain, store pocket.Store, input any) (any, error)
}

// Metrics tracks chain execution metrics.
type Metrics struct {
	mu              sync.RWMutex
	totalExecutions int64
	linkExecutions  map[string]int64
	linkSuccesses   map[string]int64
	linkFailures    map[string]int64
	linkLatencies   map[string][]time.Duration
}

// NewChain creates a new fallback chain.
func NewChain(name string) *Chain {
	return &Chain{
		name:     name,
		links:    []Link{},
		strategy: &SequentialStrategy{},
		metrics: &Metrics{
			linkExecutions: make(map[string]int64),
			linkSuccesses:  make(map[string]int64),
			linkFailures:   make(map[string]int64),
			linkLatencies:  make(map[string][]time.Duration),
		},
	}
}

// AddLink adds a link to the chain.
func (c *Chain) AddLink(link Link) *Chain {
	c.mu.Lock()
	defer c.mu.Unlock()

	if link.Weight == 0 {
		link.Weight = 1.0
	}
	c.links = append(c.links, link)
	return c
}

// WithStrategy sets the execution strategy.
func (c *Chain) WithStrategy(strategy Strategy) *Chain {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.strategy = strategy
	return c
}

// Execute runs the fallback chain.
func (c *Chain) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	c.metrics.mu.Lock()
	c.metrics.totalExecutions++
	c.metrics.mu.Unlock()

	return c.strategy.Execute(ctx, c, store, input)
}

// GetMetrics returns chain execution metrics.
func (c *Chain) GetMetrics() MetricsSnapshot {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	snapshot := MetricsSnapshot{
		TotalExecutions: c.metrics.totalExecutions,
		LinkStats:       make(map[string]LinkStats),
	}

	for name, execs := range c.metrics.linkExecutions {
		stats := LinkStats{
			Executions: execs,
			Successes:  c.metrics.linkSuccesses[name],
			Failures:   c.metrics.linkFailures[name],
		}

		if latencies := c.metrics.linkLatencies[name]; len(latencies) > 0 {
			var total time.Duration
			for _, d := range latencies {
				total += d
			}
			stats.AvgLatency = total / time.Duration(len(latencies))
		}

		snapshot.LinkStats[name] = stats
	}

	return snapshot
}

// MetricsSnapshot represents a point-in-time view of metrics.
type MetricsSnapshot struct {
	TotalExecutions int64
	LinkStats       map[string]LinkStats
}

// LinkStats contains statistics for a single link.
type LinkStats struct {
	Executions int64
	Successes  int64
	Failures   int64
	AvgLatency time.Duration
}

// SequentialStrategy executes links in order until one succeeds.
type SequentialStrategy struct{}

func (s *SequentialStrategy) Execute(ctx context.Context, chain *Chain, store pocket.Store, input any) (any, error) {
	chain.mu.RLock()
	links := chain.links
	chain.mu.RUnlock()

	var lastErr error

	for i, link := range links {
		// Check condition if defined
		if link.Condition != nil && !link.Condition(ctx, store, input) {
			continue
		}

		// Transform input if needed
		linkInput := input
		if link.Transform != nil {
			linkInput = link.Transform(input)
		}

		// Record start time
		start := time.Now()

		// Update metrics
		chain.metrics.mu.Lock()
		chain.metrics.linkExecutions[link.Name]++
		chain.metrics.mu.Unlock()

		// Execute link
		result, err := link.Handler(ctx, linkInput)

		// Record latency
		latency := time.Since(start)
		chain.metrics.mu.Lock()
		chain.metrics.linkLatencies[link.Name] = append(chain.metrics.linkLatencies[link.Name], latency)
		if err == nil {
			chain.metrics.linkSuccesses[link.Name]++
		} else {
			chain.metrics.linkFailures[link.Name]++
		}
		chain.metrics.mu.Unlock()

		if err == nil {
			// Store which link succeeded
			_ = store.Set(ctx, fmt.Sprintf("chain:%s:succeeded_at", chain.name), link.Name)
			return result, nil
		}

		lastErr = err
		// Store the error for debugging
		_ = store.Set(ctx, fmt.Sprintf("chain:%s:link_%d_error", chain.name, i), err)
	}

	return nil, fmt.Errorf("all %d links failed, last error: %w", len(links), lastErr)
}

// ParallelStrategy executes links concurrently and returns the first success.
type ParallelStrategy struct {
	timeout time.Duration
}

// NewParallelStrategy creates a parallel execution strategy.
func NewParallelStrategy(timeout time.Duration) *ParallelStrategy {
	return &ParallelStrategy{timeout: timeout}
}

func (s *ParallelStrategy) Execute(ctx context.Context, chain *Chain, store pocket.Store, input any) (any, error) {
	chain.mu.RLock()
	links := chain.links
	chain.mu.RUnlock()

	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	type result struct {
		value any
		err   error
		link  string
	}

	resultCh := make(chan result, len(links))

	// Launch all links concurrently
	for _, link := range links {
		go func(l Link) {
			// Check condition
			if l.Condition != nil && !l.Condition(ctx, store, input) {
				resultCh <- result{err: fmt.Errorf("condition not met"), link: l.Name}
				return
			}

			linkInput := input
			if l.Transform != nil {
				linkInput = l.Transform(input)
			}

			start := time.Now()
			chain.metrics.mu.Lock()
			chain.metrics.linkExecutions[l.Name]++
			chain.metrics.mu.Unlock()

			value, err := l.Handler(ctx, linkInput)

			latency := time.Since(start)
			chain.metrics.mu.Lock()
			chain.metrics.linkLatencies[l.Name] = append(chain.metrics.linkLatencies[l.Name], latency)
			if err == nil {
				chain.metrics.linkSuccesses[l.Name]++
			} else {
				chain.metrics.linkFailures[l.Name]++
			}
			chain.metrics.mu.Unlock()

			resultCh <- result{value: value, err: err, link: l.Name}
		}(link)
	}

	// Collect results
	var errors []error
	for i := 0; i < len(links); i++ {
		select {
		case r := <-resultCh:
			if r.err == nil {
				_ = store.Set(ctx, fmt.Sprintf("chain:%s:succeeded_at", chain.name), r.link)
				return r.value, nil
			}
			errors = append(errors, fmt.Errorf("%s: %w", r.link, r.err))
		case <-ctx.Done():
			return nil, fmt.Errorf("parallel execution timed out: %w", ctx.Err())
		}
	}

	return nil, fmt.Errorf("all parallel executions failed: %v", errors)
}

// WeightedRandomStrategy randomly selects links based on weights.
type WeightedRandomStrategy struct {
	maxAttempts int
	random      func() float64
}

// NewWeightedRandomStrategy creates a weighted random strategy.
func NewWeightedRandomStrategy(maxAttempts int) *WeightedRandomStrategy {
	return &WeightedRandomStrategy{
		maxAttempts: maxAttempts,
		random:      func() float64 { return float64(time.Now().UnixNano()%100) / 100.0 },
	}
}

func (s *WeightedRandomStrategy) Execute(ctx context.Context, chain *Chain, store pocket.Store, input any) (any, error) {
	chain.mu.RLock()
	links := chain.links
	chain.mu.RUnlock()

	// Calculate total weight
	var totalWeight float64
	eligibleLinks := make([]Link, 0, len(links))

	for _, link := range links {
		if link.Condition == nil || link.Condition(ctx, store, input) {
			totalWeight += link.Weight
			eligibleLinks = append(eligibleLinks, link)
		}
	}

	if len(eligibleLinks) == 0 {
		return nil, fmt.Errorf("no eligible links found")
	}

	// Track attempted links
	attempted := make(map[string]bool)
	var lastErr error

	for attempt := 0; attempt < s.maxAttempts && len(attempted) < len(eligibleLinks); attempt++ {
		// Select a link based on weight
		r := s.random() * totalWeight
		var cumWeight float64

		for _, link := range eligibleLinks {
			if attempted[link.Name] {
				continue
			}

			cumWeight += link.Weight
			if r <= cumWeight {
				attempted[link.Name] = true

				linkInput := input
				if link.Transform != nil {
					linkInput = link.Transform(input)
				}

				result, err := link.Handler(ctx, linkInput)
				if err == nil {
					_ = store.Set(ctx, fmt.Sprintf("chain:%s:succeeded_at", chain.name), link.Name)
					return result, nil
				}

				lastErr = err
				break
			}
		}
	}

	return nil, fmt.Errorf("weighted random strategy failed after %d attempts: %w", s.maxAttempts, lastErr)
}

// AdaptiveChain automatically adjusts link weights based on success rates.
type AdaptiveChain struct {
	*Chain
	learningRate float64
}

// NewAdaptiveChain creates a chain that learns from execution history.
func NewAdaptiveChain(name string, learningRate float64) *AdaptiveChain {
	return &AdaptiveChain{
		Chain:        NewChain(name),
		learningRate: learningRate,
	}
}

// Execute runs the chain and updates weights based on results.
func (a *AdaptiveChain) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	result, err := a.Chain.Execute(ctx, store, input)

	// Update weights based on success/failure
	a.updateWeights()

	return result, err
}

func (a *AdaptiveChain) updateWeights() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.links {
		link := &a.links[i]

		successes := float64(a.metrics.linkSuccesses[link.Name])
		failures := float64(a.metrics.linkFailures[link.Name])
		total := successes + failures

		if total > 0 {
			successRate := successes / total
			// Adjust weight based on success rate
			link.Weight = link.Weight*(1-a.learningRate) + successRate*a.learningRate

			// Ensure weight stays positive
			if link.Weight < 0.1 {
				link.Weight = 0.1
			}
		}
	}
}
