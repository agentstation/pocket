package flow_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/internal/flow"
)

func TestFlowAsNode(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create a simple flow that doubles a number
	doubler := pocket.NewNode[any, any]("double",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return n * 2, nil
		}),
	)

	doublerFlow := pocket.NewFlow(doubler, store)

	// Create another flow that adds 10
	adder := pocket.NewNode[any, any]("add10",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return n + 10, nil
		}),
	)

	adderFlow := pocket.NewFlow(adder, store)

	// Compose the flows: double then add 10
	// Using the Flow.AsNode method
	doubleNode := doublerFlow.AsNode("double-flow")
	addNode := adderFlow.AsNode("add-flow")

	// Connect them
	doubleNode.Connect("default", addNode)

	// Create composite flow
	compositeFlow := pocket.NewFlow(doubleNode, store)

	// Test: 5 * 2 + 10 = 20
	result, err := compositeFlow.Run(ctx, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

func TestNestedFlowBuilder(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create flows for different operations
	multiplyFlow := pocket.NewFlow(
		pocket.NewNode[any, any]("multiply",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				n := input.(int)
				return n * 3, nil
			}),
		),
		store,
	)

	subtractFlow := pocket.NewFlow(
		pocket.NewNode[any, any]("subtract",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				n := input.(int)
				return n - 5, nil
			}),
		),
		store,
	)

	// Build nested flow
	nestedFlow, err := flow.NewNestedFlowBuilder("math", store).
		AddFlow("multiply", multiplyFlow).
		AddFlow("subtract", subtractFlow).
		Connect("multiply", "default", "subtract").
		Build()

	if err != nil {
		t.Fatalf("failed to build nested flow: %v", err)
	}

	// Test: 10 * 3 - 5 = 25
	result, err := nestedFlow.Run(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 25 {
		t.Errorf("expected 25, got %v", result)
	}
}

func TestComposeFlows(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create a chain of flows
	flows := make([]*pocket.Flow, 3)
	
	// Flow 1: Add 1
	flows[0] = pocket.NewFlow(
		pocket.NewNode[any, any]("add1",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(int) + 1, nil
			}),
		),
		store,
	)

	// Flow 2: Multiply by 2
	flows[1] = pocket.NewFlow(
		pocket.NewNode[any, any]("mul2",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(int) * 2, nil
			}),
		),
		store,
	)

	// Flow 3: Add 5
	flows[2] = pocket.NewFlow(
		pocket.NewNode[any, any]("add5",
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				return input.(int) + 5, nil
			}),
		),
		store,
	)

	// Compose them
	composed, err := flow.ComposeFlows("math-chain", store, flows...)
	if err != nil {
		t.Fatalf("failed to compose flows: %v", err)
	}

	// Test: (10 + 1) * 2 + 5 = 27
	result, err := composed.Run(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 27 {
		t.Errorf("expected 27, got %v", result)
	}
}

func TestFlowWithStore(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Set initial values in store
	store.Set(ctx, "multiplier", 5)
	store.Set(ctx, "input", 10)

	// Create a flow that reads from store
	calculatorFlow := pocket.NewFlow(
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
	calcNode := flow.AsNodeWithStore(calculatorFlow, "calc", "input", "result")

	// Create wrapper flow
	wrapperFlow := pocket.NewFlow(calcNode, store)

	// Run - should read from "input" key and write to "result" key
	_, err := wrapperFlow.Run(ctx, nil)
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

func TestParallelFlows(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create flows that return different values
	flows := make([]*pocket.Flow, 3)
	
	for i := range flows {
		i := i // capture loop variable
		flows[i] = pocket.NewFlow(
			pocket.NewNode[any, any](fmt.Sprintf("flow%d", i),
				pocket.WithExec(func(ctx context.Context, input any) (any, error) {
					return fmt.Sprintf("result-%d", i), nil
				}),
			),
			store,
		)
	}

	// Run in parallel
	results, err := flow.ParallelFlows(ctx, store, flows...)
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