// Package node provides advanced node composition utilities.
package node

import (
	"context"
	"fmt"
	"time"

	"github.com/agentstation/pocket"
)

// Middleware modifies node behavior.
type Middleware func(*pocket.Node) *pocket.Node

// Chain combines multiple middlewares.
func Chain(middlewares ...Middleware) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		for i := len(middlewares) - 1; i >= 0; i-- {
			node = middlewares[i](node)
		}
		return node
	}
}

// Logging adds structured logging to a node.
func Logging(logger pocket.Logger) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		originalPrep := node.Prep
		originalExec := node.Exec
		originalPost := node.Post

		node.Prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			logger.Debug(ctx, "node prep starting", "node", node.Name, "input_type", fmt.Sprintf("%T", input))
			start := time.Now()

			result, err := originalPrep(ctx, store, input)

			logger.Debug(ctx, "node prep completed",
				"node", node.Name,
				"duration", time.Since(start),
				"error", err)

			return result, err
		}

		node.Exec = func(ctx context.Context, input any) (any, error) {
			logger.Info(ctx, "node exec starting", "node", node.Name)
			start := time.Now()

			result, err := originalExec(ctx, input)

			if err != nil {
				logger.Error(ctx, "node exec failed",
					"node", node.Name,
					"duration", time.Since(start),
					"error", err)
			} else {
				logger.Info(ctx, "node exec completed",
					"node", node.Name,
					"duration", time.Since(start),
					"result_type", fmt.Sprintf("%T", result))
			}

			return result, err
		}

		node.Post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			logger.Debug(ctx, "node post starting", "node", node.Name)

			output, next, err := originalPost(ctx, store, input, prep, exec)

			logger.Debug(ctx, "node post completed",
				"node", node.Name,
				"next", next,
				"error", err)

			return output, next, err
		}

		return node
	}
}

// Timing adds execution timing to a node.
func Timing() Middleware {
	return func(node *pocket.Node) *pocket.Node {
		originalPrep := node.Prep
		originalExec := node.Exec
		originalPost := node.Post

		// Track timing in prep step
		node.Prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Get existing timing data
			key := fmt.Sprintf("node:%s:total_duration", node.Name)
			countKey := fmt.Sprintf("node:%s:execution_count", node.Name)

			total, _ := store.Get(ctx, key)
			count, _ := store.Get(ctx, countKey)

			totalDuration := time.Duration(0)
			execCount := int64(0)

			if d, ok := total.(time.Duration); ok {
				totalDuration = d
			}
			if c, ok := count.(int64); ok {
				execCount = c
			}

			result, err := originalPrep(ctx, store, input)

			// Pass timing data through
			if err == nil {
				return map[string]interface{}{
					"prepResult": result,
					"timingData": map[string]interface{}{
						"totalDuration": totalDuration,
						"execCount":     execCount,
						"execStart":     time.Now(),
					},
				}, nil
			}
			return result, err
		}

		// Time the exec step
		node.Exec = func(ctx context.Context, input any) (any, error) {
			// Extract timing data if available
			actualInput := input
			execStart := time.Now()

			if data, ok := input.(map[string]interface{}); ok {
				if prepResult, ok := data["prepResult"]; ok {
					actualInput = prepResult
				}
				if timingData, ok := data["timingData"].(map[string]interface{}); ok {
					if start, ok := timingData["execStart"].(time.Time); ok {
						execStart = start
					}
				}
			}

			result, err := originalExec(ctx, actualInput)
			duration := time.Since(execStart)

			// Return result with timing
			return map[string]interface{}{
				"execResult":   result,
				"execDuration": duration,
				"execError":    err,
			}, err
		}

		// Record timing in post step
		node.Post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			// Extract exec result and timing
			execResult := exec
			var execDuration time.Duration

			if data, ok := exec.(map[string]interface{}); ok {
				if result, ok := data["execResult"]; ok {
					execResult = result
				}
				if duration, ok := data["execDuration"].(time.Duration); ok {
					execDuration = duration
				}
			}

			// Extract timing data from prep
			var totalDuration time.Duration
			var execCount int64

			if data, ok := prep.(map[string]interface{}); ok {
				if timingData, ok := data["timingData"].(map[string]interface{}); ok {
					if d, ok := timingData["totalDuration"].(time.Duration); ok {
						totalDuration = d
					}
					if c, ok := timingData["execCount"].(int64); ok {
						execCount = c
					}
				}
			}

			// Update timing metrics
			totalDuration += execDuration
			execCount++

			_ = store.Set(ctx, fmt.Sprintf("node:%s:last_duration", node.Name), execDuration)
			_ = store.Set(ctx, fmt.Sprintf("node:%s:total_duration", node.Name), totalDuration)
			_ = store.Set(ctx, fmt.Sprintf("node:%s:execution_count", node.Name), execCount)
			_ = store.Set(ctx, fmt.Sprintf("node:%s:avg_duration", node.Name), totalDuration/time.Duration(execCount))

			// Call original post with correct data
			actualPrep := prep
			if data, ok := prep.(map[string]interface{}); ok {
				if prepResult, ok := data["prepResult"]; ok {
					actualPrep = prepResult
				}
			}

			return originalPost(ctx, store, input, actualPrep, execResult)
		}

		return node
	}
}

// Metrics adds comprehensive metrics collection.
func Metrics(collector MetricsCollector) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		originalPrep := node.Prep
		originalExec := node.Exec
		originalPost := node.Post

		node.Prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			collector.RecordPhaseStart(node.Name, "prep")
			result, err := originalPrep(ctx, store, input)
			collector.RecordPhaseEnd(node.Name, "prep", err)
			return result, err
		}

		node.Exec = func(ctx context.Context, input any) (any, error) {
			collector.RecordPhaseStart(node.Name, "exec")
			result, err := originalExec(ctx, input)
			collector.RecordPhaseEnd(node.Name, "exec", err)
			return result, err
		}

		node.Post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			collector.RecordPhaseStart(node.Name, "post")
			output, next, err := originalPost(ctx, store, input, prep, exec)
			collector.RecordPhaseEnd(node.Name, "post", err)
			collector.RecordRouting(node.Name, next)
			return output, next, err
		}

		return node
	}
}

// MetricsCollector collects node metrics.
type MetricsCollector interface {
	RecordPhaseStart(nodeName, phase string)
	RecordPhaseEnd(nodeName, phase string, err error)
	RecordRouting(nodeName, next string)
}

// Retry adds retry logic with backoff.
func Retry(maxAttempts int, backoff time.Duration) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		// Since we can't modify node configuration after creation,
		// we need to wrap the exec function with retry logic
		originalExec := node.Exec

		node.Exec = func(ctx context.Context, input any) (any, error) {
			var lastErr error
			for attempt := 0; attempt < maxAttempts; attempt++ {
				if attempt > 0 {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(backoff * time.Duration(attempt)):
						// Exponential backoff
					}
				}

				result, err := originalExec(ctx, input)
				if err == nil {
					return result, nil
				}
				lastErr = err
			}
			return nil, fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
		}
		return node
	}
}

// Timeout adds timeout to node execution.
func Timeout(duration time.Duration) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		originalExec := node.Exec

		node.Exec = func(ctx context.Context, input any) (any, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, duration)
			defer cancel()

			done := make(chan struct{})
			var result any
			var err error

			go func() {
				result, err = originalExec(timeoutCtx, input)
				close(done)
			}()

			select {
			case <-done:
				return result, err
			case <-timeoutCtx.Done():
				return nil, fmt.Errorf("node %s timed out after %v", node.Name, duration)
			}
		}
		return node
	}
}

// RateLimit adds rate limiting to a node.
func RateLimit(rps, burst int) Middleware {
	// Simple token bucket implementation
	tokens := make(chan struct{}, burst)

	// Fill bucket
	for i := 0; i < burst; i++ {
		tokens <- struct{}{}
	}

	// Refill tokens
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(rps))
		defer ticker.Stop()

		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
				// Bucket full
			}
		}
	}()

	return func(node *pocket.Node) *pocket.Node {
		originalExec := node.Exec

		node.Exec = func(ctx context.Context, input any) (any, error) {
			select {
			case <-tokens:
				// Got token, proceed
				result, err := originalExec(ctx, input)

				// Return token on completion
				select {
				case tokens <- struct{}{}:
				default:
				}

				return result, err
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		return node
	}
}

// CircuitBreaker adds circuit breaker pattern.
func CircuitBreaker(threshold int, timeout time.Duration) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		failures := 0
		lastFailure := time.Time{}
		state := "closed" // closed, open, half-open

		originalExec := node.Exec

		node.Exec = func(ctx context.Context, input any) (any, error) {
			// Check circuit state
			if state == "open" {
				if time.Since(lastFailure) > timeout {
					state = "half-open"
				} else {
					return nil, fmt.Errorf("circuit breaker is open for node %s", node.Name)
				}
			}

			result, err := originalExec(ctx, input)

			if err != nil {
				failures++
				lastFailure = time.Now()

				if failures >= threshold {
					state = "open"
				}

				return nil, err
			}

			// Success - reset if in half-open
			if state == "half-open" {
				state = "closed"
				failures = 0
			}

			return result, nil
		}

		return node
	}
}

// Validation adds input/output validation.
func Validation(validateInput, validateOutput func(any) error) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		if validateInput != nil {
			originalPrep := node.Prep
			node.Prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				if err := validateInput(input); err != nil {
					return nil, fmt.Errorf("input validation failed: %w", err)
				}
				return originalPrep(ctx, store, input)
			}
		}

		if validateOutput != nil {
			originalPost := node.Post
			node.Post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				output, next, err := originalPost(ctx, store, input, prep, exec)
				if err != nil {
					return output, next, err
				}

				if err := validateOutput(output); err != nil {
					return nil, "", fmt.Errorf("output validation failed: %w", err)
				}

				return output, next, nil
			}
		}

		return node
	}
}

// Transform adds input/output transformation.
func Transform(transformInput, transformOutput func(any) any) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		if transformInput != nil {
			originalPrep := node.Prep
			node.Prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				transformed := transformInput(input)
				return originalPrep(ctx, store, transformed)
			}
		}

		if transformOutput != nil {
			originalPost := node.Post
			node.Post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				output, next, err := originalPost(ctx, store, input, prep, exec)
				if err != nil {
					return output, next, err
				}

				transformed := transformOutput(output)
				return transformed, next, nil
			}
		}

		return node
	}
}

// ErrorHandler adds custom error handling.
func ErrorHandler(handler func(error) error) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		// Wrap prep
		originalPrep := node.Prep
		node.Prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			result, err := originalPrep(ctx, store, input)
			if err != nil {
				if handledErr := handler(err); handledErr != nil {
					return nil, handledErr
				}
				// Error was handled, return result with no error
				return result, nil
			}
			return result, nil
		}

		// Wrap exec
		originalExec := node.Exec
		node.Exec = func(ctx context.Context, input any) (any, error) {
			result, err := originalExec(ctx, input)
			if err != nil {
				if handledErr := handler(err); handledErr != nil {
					return nil, handledErr
				}
				// Error was handled, return result with no error
				return result, nil
			}
			return result, nil
		}

		// Wrap post
		originalPost := node.Post
		node.Post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			output, next, err := originalPost(ctx, store, input, prep, exec)
			if err != nil {
				if handledErr := handler(err); handledErr != nil {
					return nil, "", handledErr
				}
				// Error was handled, return result with no error
				return output, next, nil
			}
			return output, next, nil
		}

		return node
	}
}

// Apply applies middleware to a node.
func Apply(node *pocket.Node, middlewares ...Middleware) *pocket.Node {
	for _, mw := range middlewares {
		node = mw(node)
	}
	return node
}
