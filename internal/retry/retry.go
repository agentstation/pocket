package retry

import (
	"context"
	"fmt"
	"time"
)

// Policy defines retry behavior.
type Policy struct {
	// MaxAttempts is the maximum number of attempts (0 = no retry).
	MaxAttempts int
	// InitialDelay is the initial delay between retries.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// Multiplier is the factor by which the delay increases.
	Multiplier float64
	// Jitter adds randomness to delays to avoid thundering herd.
	Jitter bool
}

// DefaultPolicy returns a sensible default retry policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// Do executes fn with the retry policy.
func (p Policy) Do(ctx context.Context, fn func() error) error {
	if p.MaxAttempts <= 0 {
		return fn()
	}

	var lastErr error
	delay := p.InitialDelay

	for attempt := 0; attempt <= p.MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Calculate next delay
			delay = time.Duration(float64(delay) * p.Multiplier)
			if delay > p.MaxDelay {
				delay = p.MaxDelay
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check if error is retryable
		if !IsRetryable(lastErr) {
			return lastErr
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", p.MaxAttempts+1, lastErr)
}

// IsRetryable determines if an error should trigger a retry.
// Override this for custom retry logic.
var IsRetryable = func(err error) bool {
	// By default, all errors are retryable
	// In practice, you'd check for specific error types
	return true
}

// Exponential creates an exponential backoff retry policy.
func Exponential(maxAttempts int) Policy {
	return Policy{
		MaxAttempts:  maxAttempts,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// Linear creates a linear retry policy with fixed delays.
func Linear(maxAttempts int, delay time.Duration) Policy {
	return Policy{
		MaxAttempts:  maxAttempts,
		InitialDelay: delay,
		MaxDelay:     delay,
		Multiplier:   1.0,
		Jitter:       false,
	}
}
