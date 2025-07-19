package graph_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/internal/graph"
)

func TestGraphAsNode(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create a simple graph that doubles a number
	doubler := pocket.NewNode[any, any]("double",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return n * 2, nil
		}),
	)

	doublerGraph := pocket.NewGraph(doubler, store)

	// Create another graph that adds 10
	adder := pocket.NewNode[any, any]("add10",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return n + 10, nil
		}),
	)

	adderGraph := pocket.NewGraph(adder, store)

	// Compose the graphs: double then add 10
	// Using the Graph.AsNode method
	doubleNode := doublerGraph.AsNode("double-graph")
	addNode := adderGraph.AsNode("add-graph")

	// Connect them
	doubleNode.Connect("default", addNode)

	// Create composite graph
	compositeGraph := pocket.NewGraph(doubleNode, store)

	// Execute test case: 5 * 2 + 10 = 20
	result, err := compositeGraph.Run(ctx, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

func TestNestedGraphBuilder(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create graphs for different operations
	multiplyGraph := pocket.NewGraph(
		pocket.NewNode[any, any]("multiply",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				n := input.(int)
				return n * 3, nil
			}),
		),
		store,
	)

	subtractGraph := pocket.NewGraph(
		pocket.NewNode[any, any]("subtract",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				n := input.(int)
				return n - 5, nil
			}),
		),
		store,
	)

	// Build nested graph
	nestedGraph, err := graph.NewNestedGraphBuilder("math", store).
		AddGraph("multiply", multiplyGraph).
		AddGraph("subtract", subtractGraph).
		Connect("multiply", "default", "subtract").
		Build()

	if err != nil {
		t.Fatalf("failed to build nested graph: %v", err)
	}

	// Execute test case: 10 * 3 - 5 = 25
	result, err := nestedGraph.Run(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 25 {
		t.Errorf("expected 25, got %v", result)
	}
}

func TestComposeGraphs(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create a chain of graphs
	graphs := make([]*pocket.Graph, 3)

	// Graph 1: Add 1
	graphs[0] = pocket.NewGraph(
		pocket.NewNode[any, any]("add1",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(int) + 1, nil
			}),
		),
		store,
	)

	// Graph 2: Multiply by 2
	graphs[1] = pocket.NewGraph(
		pocket.NewNode[any, any]("mul2",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(int) * 2, nil
			}),
		),
		store,
	)

	// Graph 3: Add 5
	graphs[2] = pocket.NewGraph(
		pocket.NewNode[any, any]("add5",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(int) + 5, nil
			}),
		),
		store,
	)

	// Compose them
	composed, err := graph.ComposeGraphs("math-chain", store, graphs...)
	if err != nil {
		t.Fatalf("failed to compose graphs: %v", err)
	}

	// Execute test case: (10 + 1) * 2 + 5 = 27
	result, err := composed.Run(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 27 {
		t.Errorf("expected 27, got %v", result)
	}
}

func TestGraphWithStore(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Set initial values in store
	store.Set(ctx, "multiplier", 5)
	store.Set(ctx, "input", 10)

	// Create a graph that reads from store
	calculatorGraph := pocket.NewGraph(
		pocket.NewNode[any, any]("calculator",
			pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				mult, _ := store.Get(ctx, "multiplier")
				return mult, nil
			}),
			pocket.WithExec(func(ctx context.Context, mult any) (any, error) {
				input, _ := store.Get(ctx, "input")
				return input.(int) * mult.(int), nil
			}),
		),
		store,
	)

	// Use AsNodeWithStore
	calcNode := graph.AsNodeWithStore(calculatorGraph, "calc", "input", "result")

	// Create wrapper graph
	wrapperGraph := pocket.NewGraph(calcNode, store)

	// Run - should read from "input" key and write to "result" key
	_, err := wrapperGraph.Run(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check result was written to store
	result, exists := store.Get(ctx, "result")
	if !exists {
		t.Fatal("result not found in store")
	}

	if result != 50 { // 10 * 5
		t.Errorf("expected 50, got %v", result)
	}
}

func TestParallelGraphs(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create graphs that return different values
	graphs := make([]*pocket.Graph, 3)

	for i := range graphs {
		i := i // capture loop variable
		graphs[i] = pocket.NewGraph(
			pocket.NewNode[any, any](fmt.Sprintf("graph%d", i),
				pocket.WithExec(func(ctx context.Context, input any) (any, error) {
					return fmt.Sprintf("result-%d", i), nil
				}),
			),
			store,
		)
	}

	// Run in parallel
	results, err := graph.ParallelGraphs(ctx, store, graphs...)
	if err != nil {
		t.Fatalf("parallel execution failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Check results
	for i, result := range results {
		expected := fmt.Sprintf("result-%d", i)
		if result != expected {
			t.Errorf("expected %q, got %v", expected, result)
		}
	}
}
