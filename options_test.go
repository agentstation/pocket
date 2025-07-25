package pocket_test

import (
	"context"
	"testing"
	"time"

	"github.com/agentstation/pocket"
)

const (
	defaultRoute = "default"
)

func TestFunctionalOptions(t *testing.T) {
	t.Run("simple node with just exec", testSimpleNodeWithExec)
	t.Run("node with all lifecycle functions", testNodeWithAllLifecycle)
	t.Run("node with error handler", testNodeWithErrorHandler)
}

func testSimpleNodeWithExec(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	node := pocket.NewNode[any, any]("simple",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				return "processed: " + input.(string), nil
			},
		},
	)

	graph := pocket.NewGraph(node, store)
	result, err := graph.Run(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "processed: test" {
		t.Errorf("expected 'processed: test', got %v", result)
	}
}

func testNodeWithAllLifecycle(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()
	var prepCalled, execCalled, postCalled bool

	node := pocket.NewNode[any, any]("full",
		pocket.Steps{
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				prepCalled = true
				return input.(string) + "-prep", nil
			},
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				execCalled = true
				return prepResult.(string) + "-exec", nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				postCalled = true
				return exec.(string) + "-post", defaultRoute, nil
			},
		},
	)

	graph := pocket.NewGraph(node, store)
	result, err := graph.Run(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !prepCalled || !execCalled || !postCalled {
		t.Error("not all lifecycle functions were called")
	}

	if result != "test-prep-exec-post" {
		t.Errorf("expected 'test-prep-exec-post', got %v", result)
	}
}

func testNodeWithErrorHandler(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("node with options", func(t *testing.T) {
		retryCount := 0
		node := pocket.NewNode[any, any]("retry-test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					retryCount++
					if retryCount < 3 {
						return nil, pocket.ErrInvalidInput
					}
					return "success after retries", nil
				},
			},
			pocket.WithRetry(3, 10*time.Millisecond),
		)

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if retryCount != 3 {
			t.Errorf("expected 3 retry attempts, got %d", retryCount)
		}

		if result != "success after retries" {
			t.Errorf("expected 'success after retries', got %v", result)
		}
	})

	t.Run("node with cleanup hooks", func(t *testing.T) {
		var successCalled, completeCalled bool

		node := pocket.NewNode[any, any]("hooks",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return successResult, nil
				},
			},
			pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
				successCalled = true
				if output != successResult {
					t.Errorf("expected output '%s' in success hook, got %v", successResult, output)
				}
			}),
			pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
				completeCalled = true
			}),
		)

		graph := pocket.NewGraph(node, store)
		_, err := graph.Run(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !successCalled {
			t.Error("onSuccess hook not called")
		}
		if !completeCalled {
			t.Error("onComplete hook not called")
		}
	})
}

func TestGlobalDefaults(t *testing.T) {
	// Save current defaults to restore later
	defer pocket.ResetDefaults()

	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("global default prep", func(t *testing.T) {
		var defaultPrepCalled bool

		// Set global default prep
		pocket.SetDefaultPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			defaultPrepCalled = true
			return input, nil
		})

		// Create node without prep - should use global default
		node := pocket.NewNode[any, any]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return "executed", nil
				},
			},
		)

		graph := pocket.NewGraph(node, store)
		_, err := graph.Run(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !defaultPrepCalled {
			t.Error("global default prep not called")
		}

		// We can't check store.Get since prep step can't write anymore
		// Just verify the function was called
	})

	t.Run("node prep overrides global default", func(t *testing.T) {
		var globalCalled, nodeCalled bool

		// Set global default
		pocket.SetDefaultPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			globalCalled = true
			return input, nil
		})

		// Create node with its own prep
		node := pocket.NewNode[any, any]("override",
			pocket.Steps{
				Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
					nodeCalled = true
					return input, nil
				},
				Exec: func(ctx context.Context, input any) (any, error) {
					return "executed", nil
				},
			},
		)

		graph := pocket.NewGraph(node, store)
		_, err := graph.Run(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if globalCalled {
			t.Error("global prep should not be called when node has its own")
		}
		if !nodeCalled {
			t.Error("node prep should be called")
		}
	})

	t.Run("set multiple defaults", func(t *testing.T) {
		var prepCalled, postCalled bool

		// Set multiple defaults at once
		pocket.SetDefaults(
			pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				prepCalled = true
				return input.(string) + "-prep", nil
			}),
			pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				postCalled = true
				return exec.(string) + "-post", defaultRoute, nil
			}),
			pocket.WithTimeout(5*time.Second),
		)

		// Create node with just exec
		node := pocket.NewNode[any, any]("minimal",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return input.(string) + "-exec", nil
				},
			},
		)

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !prepCalled || !postCalled {
			t.Error("default lifecycle functions not called")
		}

		if result != "test-prep-exec-post" {
			t.Errorf("expected 'test-prep-exec-post', got %v", result)
		}
	})
}
