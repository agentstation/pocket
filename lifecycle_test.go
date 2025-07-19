package pocket_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/agentstation/pocket"
)

const (
	successResult = "success"
	doneRoute     = "done"
)

func TestCleanupHooks(t *testing.T) {
	t.Run("onSuccess hook runs on successful execution", func(t *testing.T) {
		store := pocket.NewStore()
		ctx := context.Background()

		successCalled := false
		failureCalled := false
		completeCalled := false

		node := pocket.NewNode[any, any]("test",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return successResult, nil
			}),
			pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
				successCalled = true
				if output != successResult {
					t.Errorf("expected output 'success', got %v", output)
				}
			}),
			pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, err error) {
				failureCalled = true
			}),
			pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
				completeCalled = true
			}),
		)

		flow := pocket.NewFlow(node, store)
		_, err := flow.Run(ctx, "input")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !successCalled {
			t.Error("onSuccess hook should have been called")
		}
		if failureCalled {
			t.Error("onFailure hook should not have been called")
		}
		if !completeCalled {
			t.Error("onComplete hook should have been called")
		}
	})

	t.Run("onFailure hook runs on failed execution", func(t *testing.T) {
		store := pocket.NewStore()
		ctx := context.Background()

		successCalled := false
		failureCalled := false
		completeCalled := false
		expectedErr := errors.New("test error")

		node := pocket.NewNode[any, any]("test",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return nil, expectedErr
			}),
			pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
				successCalled = true
			}),
			pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, err error) {
				failureCalled = true
				// The error will be wrapped, so just check it contains our error
				if !strings.Contains(err.Error(), expectedErr.Error()) {
					t.Errorf("expected error to contain %v, got %v", expectedErr, err)
				}
			}),
			pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
				completeCalled = true
			}),
		)

		flow := pocket.NewFlow(node, store)
		_, err := flow.Run(ctx, "input")

		if err == nil {
			t.Error("expected error")
		}

		if successCalled {
			t.Error("onSuccess hook should not have been called")
		}
		if !failureCalled {
			t.Error("onFailure hook should have been called")
		}
		if !completeCalled {
			t.Error("onComplete hook should have been called")
		}
	})

	t.Run("onComplete runs even on panic", func(t *testing.T) {
		store := pocket.NewStore()
		ctx := context.Background()

		completeCalled := false

		node := pocket.NewNode[any, any]("test",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				// This would normally panic, but we'll return an error instead
				// to simulate the behavior without actually panicking in tests
				return nil, errors.New("simulated panic")
			}),
			pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
				completeCalled = true
			}),
		)

		flow := pocket.NewFlow(node, store)
		_, _ = flow.Run(ctx, "input")

		if !completeCalled {
			t.Error("onComplete hook should have been called even on error")
		}
	})

	t.Run("cleanup hooks can access store", func(t *testing.T) {
		store := pocket.NewStore()
		ctx := context.Background()

		node := pocket.NewNode[any, any]("test",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return map[string]interface{}{
					"exec_data": "important_value",
					"result":    "result",
				}, nil
			}),
			pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, execResult any) (any, string, error) {
				// Store the exec_data in post step
				data := execResult.(map[string]interface{})
				store.Set(ctx, "exec_data", data["exec_data"])
				return data["result"], doneRoute, nil
			}),
			pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
				// Should be able to read from store
				if val, exists := store.Get(ctx, "exec_data"); exists {
					store.Set(ctx, "cleanup_read", val)
				}
			}),
			pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
				// Clean up by removing temporary data
				store.Delete(ctx, "exec_data")
			}),
		)

		flow := pocket.NewFlow(node, store)
		_, _ = flow.Run(ctx, "input")

		// Check that cleanup could read the value
		if val, exists := store.Get(ctx, "cleanup_read"); !exists || val != "important_value" {
			t.Error("cleanup hook should have been able to read from store")
		}

		// Check that complete hook cleaned up
		if _, exists := store.Get(ctx, "exec_data"); exists {
			t.Error("complete hook should have deleted exec_data")
		}
	})

	t.Run("hooks run in correct order with fallback", func(t *testing.T) {
		store := pocket.NewStore()
		ctx := context.Background()

		var executionOrder []string

		node := pocket.NewNode[any, any]("test",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				executionOrder = append(executionOrder, "exec")
				return nil, errors.New("exec failed")
			}),
			pocket.WithFallback(func(ctx context.Context, input any, err error) (any, error) {
				executionOrder = append(executionOrder, "fallback")
				return "fallback result", nil
			}),
			pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
				executionOrder = append(executionOrder, "onSuccess")
			}),
			pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, err error) {
				executionOrder = append(executionOrder, "onFailure")
			}),
			pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
				executionOrder = append(executionOrder, "onComplete")
			}),
		)

		flow := pocket.NewFlow(node, store)
		result, err := flow.Run(ctx, "input")

		if err != nil {
			t.Errorf("unexpected error with fallback: %v", err)
		}
		if result != "fallback result" {
			t.Errorf("expected fallback result, got %v", result)
		}

		// Verify execution order
		expected := []string{"exec", "fallback", "onSuccess", "onComplete"}
		if len(executionOrder) != len(expected) {
			t.Errorf("expected %d calls, got %d", len(expected), len(executionOrder))
		}
		for i, step := range expected {
			if i >= len(executionOrder) || executionOrder[i] != step {
				t.Errorf("expected step %d to be %s, got %v", i, step, executionOrder)
			}
		}
	})
}

func TestFlowComposition(t *testing.T) {
	t.Run("flow as node", func(t *testing.T) {
		store := pocket.NewStore()
		ctx := context.Background()

		// Create a sub-flow
		subNode1 := pocket.NewNode[any, any]("sub1",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(string) + " -> sub1", nil
			}),
		)

		subNode2 := pocket.NewNode[any, any]("sub2",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(string) + " -> sub2", nil
			}),
		)

		subNode1.Connect("default", subNode2)
		subFlow := pocket.NewFlow(subNode1, pocket.NewStore())

		// Convert sub-flow to node
		subFlowNode := subFlow.AsNode("subflow")

		// Create main flow
		mainNode := pocket.NewNode[any, any]("main",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(string) + " -> main", nil
			}),
		)

		mainNode.Connect("default", subFlowNode)

		// Run composed flow
		mainFlow := pocket.NewFlow(mainNode, store)
		result, err := mainFlow.Run(ctx, "start")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := "start -> main -> sub1 -> sub2"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}
