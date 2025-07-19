package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Retry adds retry logic with exponential backoff.
func Retry(maxAttempts int, backoff time.Duration) Middleware {
	return func(node pocket.Node) pocket.Node {
		return &middlewareNode{
			inner: node,
			name:  node.Name(),
			exec: func(ctx context.Context, input any) (any, error) {
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

					result, err := node.Exec(ctx, input)
					if err == nil {
						return result, nil
					}
					lastErr = err
				}
				return nil, fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
			},
		}
	}
}

// Timeout adds timeout to node execution.
func Timeout(duration time.Duration) Middleware {
	return func(node pocket.Node) pocket.Node {
		return &middlewareNode{
			inner: node,
			name:  node.Name(),
			exec: func(ctx context.Context, input any) (any, error) {
				timeoutCtx, cancel := context.WithTimeout(ctx, duration)
				defer cancel()

				done := make(chan struct{})
				var result any
				var err error

				go func() {
					result, err = node.Exec(timeoutCtx, input)
					close(done)
				}()

				select {
				case <-done:
					return result, err
				case <-timeoutCtx.Done():
					return nil, fmt.Errorf("node %s timed out after %v", node.Name(), duration)
				}
			},
		}
	}
}

// RateLimit adds rate limiting to a node using a token bucket.
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

	return func(node pocket.Node) pocket.Node {
		return &middlewareNode{
			inner: node,
			name:  node.Name(),
			exec: func(ctx context.Context, input any) (any, error) {
				select {
				case <-tokens:
					// Got token, proceed
					result, err := node.Exec(ctx, input)

					// Return token on completion
					select {
					case tokens <- struct{}{}:
					default:
					}

					return result, err
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		}
	}
}

// CircuitBreaker adds circuit breaker pattern.
func CircuitBreaker(threshold int, timeout time.Duration) Middleware {
	return func(node pocket.Node) pocket.Node {
		var mu sync.Mutex
		failures := 0
		lastFailure := time.Time{}
		state := "closed" // closed, open, half-open

		return &middlewareNode{
			inner: node,
			name:  node.Name(),
			exec: func(ctx context.Context, input any) (any, error) {
				mu.Lock()
				// Check circuit state
				if state == "open" {
					if time.Since(lastFailure) > timeout {
						state = "half-open"
					} else {
						mu.Unlock()
						return nil, fmt.Errorf("circuit breaker is open for node %s", node.Name())
					}
				}
				mu.Unlock()

				result, err := node.Exec(ctx, input)

				mu.Lock()
				defer mu.Unlock()

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
			},
		}
	}
}

// Validation adds input/output validation.
func Validation(validateInput, validateOutput func(any) error) Middleware {
	return func(node pocket.Node) pocket.Node {
		wrapper := &middlewareNode{
			inner: node,
			name:  node.Name(),
		}

		if validateInput != nil {
			wrapper.prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				if err := validateInput(input); err != nil {
					return nil, fmt.Errorf("input validation failed: %w", err)
				}
				return node.Prep(ctx, store, input)
			}
		}

		if validateOutput != nil {
			wrapper.post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				output, next, err := node.Post(ctx, store, input, prep, exec)
				if err != nil {
					return output, next, err
				}

				if err := validateOutput(output); err != nil {
					return nil, "", fmt.Errorf("output validation failed: %w", err)
				}

				return output, next, nil
			}
		}

		return wrapper
	}
}

// Transform adds input/output transformation.
func Transform(transformInput, transformOutput func(any) any) Middleware {
	return func(node pocket.Node) pocket.Node {
		wrapper := &middlewareNode{
			inner: node,
			name:  node.Name(),
		}

		if transformInput != nil {
			wrapper.prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				transformed := transformInput(input)
				return node.Prep(ctx, store, transformed)
			}
		}

		if transformOutput != nil {
			wrapper.post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				output, next, err := node.Post(ctx, store, input, prep, exec)
				if err != nil {
					return output, next, err
				}

				transformed := transformOutput(output)
				return transformed, next, nil
			}
		}

		return wrapper
	}
}

// ErrorHandler adds custom error handling.
func ErrorHandler(handler func(error) error) Middleware {
	return func(node pocket.Node) pocket.Node {
		return &middlewareNode{
			inner: node,
			name:  node.Name(),
			prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				result, err := node.Prep(ctx, store, input)
				if err != nil {
					if handledErr := handler(err); handledErr != nil {
						return nil, handledErr
					}
					// Error was handled, return result with no error
					return result, nil
				}
				return result, nil
			},
			exec: func(ctx context.Context, input any) (any, error) {
				result, err := node.Exec(ctx, input)
				if err != nil {
					if handledErr := handler(err); handledErr != nil {
						return nil, handledErr
					}
					// Error was handled, return result with no error
					return result, nil
				}
				return result, nil
			},
			post: func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				output, next, err := node.Post(ctx, store, input, prep, exec)
				if err != nil {
					if handledErr := handler(err); handledErr != nil {
						return nil, "", handledErr
					}
					// Error was handled, return result with no error
					return output, next, nil
				}
				return output, next, nil
			},
		}
	}
}
