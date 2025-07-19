package pocket_test

import (
	"context"
	"testing"

	"github.com/agentstation/pocket"
)

// Using constants from lifecycle_test.go

// Benchmark node creation.
func BenchmarkNewNode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pocket.NewNode[any, any]("bench",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input, nil
			}),
		)
	}
}

// Benchmark single node execution.
func BenchmarkSingleNodeExecution(b *testing.B) {
	node := pocket.NewNode[any, any]("bench",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}),
	)
	store := pocket.NewStore()
	flow := pocket.NewFlow(node, store)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = flow.Run(ctx, "test")
	}
}

// Benchmark lifecycle steps.
func BenchmarkLifecycleSteps(b *testing.B) {
	node := pocket.NewNode[any, any]("lifecycle",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			return input, nil
		}),
		pocket.WithExec(func(ctx context.Context, prep any) (any, error) {
			return prep, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			return exec, doneRoute, nil
		}),
	)

	store := pocket.NewStore()
	flow := pocket.NewFlow(node, store)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = flow.Run(ctx, "test")
	}
}

// Benchmark pipeline execution with multiple nodes.
func BenchmarkPipelineMultiNode(b *testing.B) {
	nodes := make([]*pocket.Node, 5)
	for i := range nodes {
		nodes[i] = pocket.NewNode[any, any]("bench",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input, nil
			}),
		)
	}
	store := pocket.NewStore()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pocket.Pipeline(ctx, nodes, store, "test")
	}
}

// Benchmark concurrent execution with many nodes.
func BenchmarkRunConcurrentManyNodes(b *testing.B) {
	nodes := make([]*pocket.Node, 10)
	for i := range nodes {
		nodes[i] = pocket.NewNode[any, any]("bench",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input, nil
			}),
		)
	}
	store := pocket.NewStore()
	ctx := context.Background()
	inputs := make([]any, len(nodes))
	for i := range inputs {
		inputs[i] = "test"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pocket.RunConcurrent(ctx, nodes, store, inputs)
	}
}

// Benchmark store operations.
func BenchmarkStoreOperations(b *testing.B) {
	store := pocket.NewStore()
	ctx := context.Background()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.Set(ctx, "key", "value")
		}
	})

	b.Run("Get", func(b *testing.B) {
		store.Set(ctx, "key", "value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Get(ctx, "key")
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.Set(ctx, "key", "value")
			store.Delete(ctx, "key")
		}
	})
}

// Benchmark flow with routing.
func BenchmarkFlowWithRouting(b *testing.B) {
	// Create nodes
	start := pocket.NewNode[any, any]("start",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			if i := result.(int); i%2 == 0 {
				return result, "even", nil
			}
			return result, "odd", nil
		}),
	)

	nodeA := pocket.NewNode[any, any]("a",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}),
	)

	nodeB := pocket.NewNode[any, any]("b",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}),
	)

	start.Connect("even", nodeA)
	start.Connect("odd", nodeB)

	store := pocket.NewStore()
	flow := pocket.NewFlow(start, store)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = flow.Run(ctx, i)
	}
}

// Benchmark builder pattern.
func BenchmarkBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builder := pocket.NewBuilder(pocket.NewStore())

		nodeA := pocket.NewNode[any, any]("a",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input, nil
			}),
		)

		nodeB := pocket.NewNode[any, any]("b",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input, nil
			}),
		)

		nodeC := pocket.NewNode[any, any]("c",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input, nil
			}),
		)

		builder.
			Add(nodeA).
			Add(nodeB).
			Add(nodeC).
			Connect("a", "default", "b").
			Connect("b", "default", "c").
			Start("a")
		_, _ = builder.Build()
	}
}

// Benchmark memory allocations for node creation.
func BenchmarkNodeCreationAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = pocket.NewNode[any, any]("bench",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input, nil
			}),
		)
	}
}

// Benchmark typed store operations.
func BenchmarkTypedStore(b *testing.B) {
	store := pocket.NewStore()
	typedStore := pocket.NewTypedStore[string](store)
	ctx := context.Background()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = typedStore.Set(ctx, "key", "value")
		}
	})

	b.Run("Get", func(b *testing.B) {
		_ = typedStore.Set(ctx, "key", "value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = typedStore.Get(ctx, "key")
		}
	})
}

// Benchmark scoped store.
func BenchmarkScopedStore(b *testing.B) {
	store := pocket.NewStore()
	scoped := store.Scope("test")
	ctx := context.Background()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = scoped.Set(ctx, "key", "value")
		}
	})

	b.Run("Get", func(b *testing.B) {
		_ = scoped.Set(ctx, "key", "value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = scoped.Get(ctx, "key")
		}
	})
}
