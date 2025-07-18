package pocket_test

import (
	"context"
	"testing"

	"github.com/agentstation/pocket"
)

// Simple processor for benchmarking
type benchProcessor struct {
	work func()
}

func (p *benchProcessor) Process(ctx context.Context, input any) (any, error) {
	if p.work != nil {
		p.work()
	}
	return input, nil
}

// Benchmark node creation
func BenchmarkNewNode(b *testing.B) {
	proc := &benchProcessor{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pocket.NewNode("bench", proc)
	}
}

// Benchmark single node execution
func BenchmarkSingleNodeExecution(b *testing.B) {
	proc := &benchProcessor{}
	node := pocket.NewNode("bench", proc)
	store := pocket.NewStore()
	flow := pocket.NewFlow(node, store)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = flow.Run(ctx, "test")
	}
}

// Benchmark pipeline execution with multiple nodes
func BenchmarkPipelineMultiNode(b *testing.B) {
	nodes := make([]*pocket.Node, 5)
	for i := range nodes {
		nodes[i] = pocket.NewNode("bench", &benchProcessor{})
	}
	store := pocket.NewStore()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pocket.Pipeline(ctx, nodes, store, "test")
	}
}

// Benchmark concurrent execution with many nodes
func BenchmarkRunConcurrentManyNodes(b *testing.B) {
	nodes := make([]*pocket.Node, 10)
	for i := range nodes {
		nodes[i] = pocket.NewNode("bench", &benchProcessor{})
	}
	store := pocket.NewStore()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pocket.RunConcurrent(ctx, nodes, store)
	}
}

// Benchmark store operations
func BenchmarkStoreOperations(b *testing.B) {
	store := pocket.NewStore()
	
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.Set("key", "value")
		}
	})
	
	b.Run("Get", func(b *testing.B) {
		store.Set("key", "value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Get("key")
		}
	})
	
	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.Set("key", "value")
			store.Delete("key")
		}
	})
}

// Benchmark flow with routing
func BenchmarkFlowWithRouting(b *testing.B) {
	// Create nodes
	start := pocket.NewNode("start", &benchProcessor{})
	nodeA := pocket.NewNode("a", &benchProcessor{})
	nodeB := pocket.NewNode("b", &benchProcessor{})
	
	// Setup routing
	start.Router = pocket.RouterFunc(func(ctx context.Context, result any) (string, error) {
		if i := result.(int); i%2 == 0 {
			return "even", nil
		}
		return "odd", nil
	})
	
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

// Benchmark builder pattern
func BenchmarkBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builder := pocket.NewBuilder(pocket.NewStore())
		builder.
			Add(pocket.NewNode("a", &benchProcessor{})).
			Add(pocket.NewNode("b", &benchProcessor{})).
			Add(pocket.NewNode("c", &benchProcessor{})).
			Connect("a", "default", "b").
			Connect("b", "default", "c").
			Start("a")
		_, _ = builder.Build()
	}
}

// Benchmark memory allocations for node creation
func BenchmarkNodeCreationAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = pocket.NewNode("bench", pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}))
	}
}

// Benchmark typed store operations
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

// Benchmark scoped store
func BenchmarkScopedStore(b *testing.B) {
	store := pocket.NewStore()
	scoped := pocket.NewScopedStore(store, "test")
	
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			scoped.Set("key", "value")
		}
	})
	
	b.Run("Get", func(b *testing.B) {
		scoped.Set("key", "value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = scoped.Get("key")
		}
	})
}