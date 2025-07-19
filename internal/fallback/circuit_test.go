package fallback

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agentstation/pocket"
)

func TestCircuitBreaker(t *testing.T) {
	t.Run("opens after max failures", func(t *testing.T) {
		cb := NewCircuitBreaker("test", WithMaxFailures(2))
		store := pocket.NewStore()
		ctx := context.Background()

		failingFunc := func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("service error")
		}

		// First two failures should work
		_, err1 := cb.Execute(ctx, store, failingFunc, "input1")
		if err1 == nil {
			t.Error("expected error on first call")
		}

		_, err2 := cb.Execute(ctx, store, failingFunc, "input2")
		if err2 == nil {
			t.Error("expected error on second call")
		}

		// Third call should fail with circuit open
		_, err3 := cb.Execute(ctx, store, failingFunc, "input3")
		if err3 == nil || err3.Error() != "circuit breaker test is open" {
			t.Errorf("expected circuit open error, got: %v", err3)
		}

		// Verify state
		if cb.GetState() != StateOpen {
			t.Errorf("expected circuit to be open, got: %v", cb.GetState())
		}
	})

	t.Run("closes after successful half-open test", func(t *testing.T) {
		cb := NewCircuitBreaker("test",
			WithMaxFailures(1),
			WithResetTimeout(50*time.Millisecond),
			WithHalfOpenRequests(2),
		)
		store := pocket.NewStore()
		ctx := context.Background()

		// Fail once to open circuit
		failingFunc := func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("service error")
		}
		cb.Execute(ctx, store, failingFunc, "input1")

		// Wait for reset timeout
		time.Sleep(100 * time.Millisecond)

		// Successful function
		successFunc := func(ctx context.Context, input any) (any, error) {
			return "success", nil
		}

		// Should transition to half-open and allow request
		result1, err1 := cb.Execute(ctx, store, successFunc, "input2")
		if err1 != nil {
			t.Errorf("expected success in half-open state, got: %v", err1)
		}
		if result1 != "success" {
			t.Errorf("expected success result, got: %v", result1)
		}

		// Second successful request should close circuit
		result2, err2 := cb.Execute(ctx, store, successFunc, "input3")
		if err2 != nil {
			t.Errorf("expected success, got: %v", err2)
		}
		if result2 != "success" {
			t.Errorf("expected success result, got: %v", result2)
		}

		// Verify circuit is closed
		if cb.GetState() != StateClosed {
			t.Errorf("expected circuit to be closed, got: %v", cb.GetState())
		}
	})

	t.Run("metrics tracking", func(t *testing.T) {
		cb := NewCircuitBreaker("test", WithMaxFailures(2))
		store := pocket.NewStore()
		ctx := context.Background()

		successFunc := func(ctx context.Context, input any) (any, error) {
			return "success", nil
		}
		failingFunc := func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("fail")
		}

		// Mix of success and failure
		cb.Execute(ctx, store, successFunc, "1")
		cb.Execute(ctx, store, failingFunc, "2")
		cb.Execute(ctx, store, successFunc, "3")

		metrics := cb.GetMetrics()
		if metrics.TotalRequests != 3 {
			t.Errorf("expected 3 total requests, got: %d", metrics.TotalRequests)
		}
		if metrics.TotalSuccesses != 2 {
			t.Errorf("expected 2 successes, got: %d", metrics.TotalSuccesses)
		}
		if metrics.TotalFailures != 1 {
			t.Errorf("expected 1 failure, got: %d", metrics.TotalFailures)
		}
	})
}

func TestCircuitBreakerPolicy(t *testing.T) {
	t.Run("uses fallback on circuit open", func(t *testing.T) {
		primaryCalled := false
		fallbackCalled := false

		primary := func(ctx context.Context, input any) (any, error) {
			primaryCalled = true
			return nil, errors.New("primary failed")
		}

		fallback := func(ctx context.Context, store pocket.StoreWriter, input any, err error) (any, error) {
			fallbackCalled = true
			return "fallback result", nil
		}

		policy := NewCircuitBreakerPolicy("test", primary, fallback,
			WithMaxFailures(0), // Open immediately
		)

		store := pocket.NewStore()
		ctx := context.Background()

		// First call should use primary and fail
		result1, err1 := policy.Execute(ctx, store, "input1")
		if err1 != nil {
			t.Errorf("expected fallback to handle error, got: %v", err1)
		}
		if result1 != "fallback result" {
			t.Errorf("expected fallback result, got: %v", result1)
		}

		// Reset flags
		primaryCalled = false
		fallbackCalled = false

		// Second call should skip primary (circuit open) and use fallback
		result2, err2 := policy.Execute(ctx, store, "input2")
		if err2 != nil {
			t.Errorf("expected fallback to handle error, got: %v", err2)
		}
		if result2 != "fallback result" {
			t.Errorf("expected fallback result, got: %v", result2)
		}
		if primaryCalled {
			t.Error("primary should not be called when circuit is open")
		}
		if !fallbackCalled {
			t.Error("fallback should be called when circuit is open")
		}
	})
}

func TestCircuitBreakerGroup(t *testing.T) {
	t.Run("manages multiple circuit breakers", func(t *testing.T) {
		group := NewCircuitBreakerGroup()

		// Get creates new breaker
		cb1 := group.Get("service1", WithMaxFailures(5))
		if cb1 == nil {
			t.Error("expected circuit breaker to be created")
		}

		// Get returns existing breaker
		cb2 := group.Get("service1")
		if cb1 != cb2 {
			t.Error("expected same circuit breaker instance")
		}

		// Different service gets different breaker
		cb3 := group.Get("service2", WithMaxFailures(3))
		if cb3 == cb1 {
			t.Error("expected different circuit breaker for different service")
		}

		// Test metrics
		store := pocket.NewStore()
		ctx := context.Background()
		
		successFunc := func(ctx context.Context, input any) (any, error) {
			return "ok", nil
		}
		
		cb1.Execute(ctx, store, successFunc, "test")
		cb3.Execute(ctx, store, successFunc, "test")

		allMetrics := group.GetAllMetrics()
		if len(allMetrics) != 2 {
			t.Errorf("expected 2 breakers in metrics, got: %d", len(allMetrics))
		}
	})

	t.Run("reset all breakers", func(t *testing.T) {
		group := NewCircuitBreakerGroup()
		store := pocket.NewStore()
		ctx := context.Background()

		// Create and open a breaker
		cb := group.Get("test", WithMaxFailures(0))
		failFunc := func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("fail")
		}
		cb.Execute(ctx, store, failFunc, "test")

		if cb.GetState() != StateOpen {
			t.Error("expected circuit to be open")
		}

		// Reset all
		group.Reset()

		if cb.GetState() != StateClosed {
			t.Error("expected circuit to be closed after reset")
		}
	})
}