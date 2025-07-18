package pocket_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agentstation/pocket"
)

func TestRunConcurrent(t *testing.T) {
	store := pocket.NewStore()
	counter := int32(0)

	// Create nodes that increment a counter
	nodes := make([]*pocket.Node, 5)
	for i := range nodes {
		i := i
		nodes[i] = pocket.NewNode(fmt.Sprintf("node%d", i),
			pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
				atomic.AddInt32(&counter, 1)
				time.Sleep(10 * time.Millisecond) // Simulate work
				return fmt.Sprintf("result%d", i), nil
			}),
		)
	}

	start := time.Now()
	results, err := pocket.RunConcurrent(context.Background(), nodes, store)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("RunConcurrent() error = %v", err)
	}

	// Check all nodes executed
	if int(counter) != len(nodes) {
		t.Errorf("counter = %d, want %d", counter, len(nodes))
	}

	// Check results
	if len(results) != len(nodes) {
		t.Errorf("len(results) = %d, want %d", len(results), len(nodes))
	}

	// Check concurrent execution (should be faster than sequential)
	expectedSequential := time.Duration(len(nodes)) * 10 * time.Millisecond
	if duration >= expectedSequential {
		t.Errorf("duration = %v, want < %v (sequential time)", duration, expectedSequential)
	}
}

func TestPipeline(t *testing.T) {
	store := pocket.NewStore()

	// Create pipeline stages
	double := pocket.NewNode("double",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return input.(int) * 2, nil
		}),
	)

	addTen := pocket.NewNode("addTen",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return input.(int) + 10, nil
		}),
	)

	toString := pocket.NewNode("toString",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Result: %d", input.(int)), nil
		}),
	)

	nodes := []*pocket.Node{double, addTen, toString}

	tests := []struct {
		name  string
		input any
		want  any
	}{
		{
			name:  "positive number",
			input: 5,
			want:  "Result: 20", // computed as: (5 * 2) + 10 = 20
		},
		{
			name:  "zero",
			input: 0,
			want:  "Result: 10", // computed as: (0 * 2) + 10 = 10
		},
		{
			name:  "negative number",
			input: -5,
			want:  "Result: 0", // computed as: (-5 * 2) + 10 = 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pocket.Pipeline(context.Background(), nodes, store, tt.input)
			if err != nil {
				t.Fatalf("Pipeline() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Pipeline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFanOut(t *testing.T) {
	store := pocket.NewStore()

	// Create a processor that squares numbers
	square := pocket.NewNode("square",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			n := input.(int)
			return n * n, nil
		}),
	)

	items := []int{1, 2, 3, 4, 5}

	results, err := pocket.FanOut(context.Background(), square, store, items)
	if err != nil {
		t.Fatalf("FanOut() error = %v", err)
	}

	// Check results
	expected := []any{1, 4, 9, 16, 25}
	if len(results) != len(expected) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(expected))
	}

	for i, got := range results {
		if got != expected[i] {
			t.Errorf("results[%d] = %v, want %v", i, got, expected[i])
		}
	}
}

func TestFanIn(t *testing.T) {
	store := pocket.NewStore()

	// Create source nodes that produce values
	source1 := pocket.NewNode("source1",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return 10, nil
		}),
	)

	source2 := pocket.NewNode("source2",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return 20, nil
		}),
	)

	source3 := pocket.NewNode("source3",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return 30, nil
		}),
	)

	// Create fan-in that sums the results
	fanIn := pocket.NewFanIn(func(results []any) (any, error) {
		sum := 0
		for _, r := range results {
			sum += r.(int)
		}
		return sum, nil
	}, source1, source2, source3)

	result, err := fanIn.Run(context.Background(), store)
	if err != nil {
		t.Fatalf("FanIn.Run() error = %v", err)
	}

	expected := 60 // 10 + 20 + 30
	if result != expected {
		t.Errorf("FanIn.Run() = %v, want %v", result, expected)
	}
}

func TestBuilderFluent(t *testing.T) {
	store := pocket.NewStore()

	// Create a complex flow using builder
	flow, err := pocket.NewBuilder(store).
		Add(pocket.NewNode("input",
			pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
				// Validate input
				n := input.(int)
				if n < 0 {
					return nil, fmt.Errorf("negative input not allowed")
				}
				return n, nil
			}),
		)).
		Add(pocket.NewNode("process",
			pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
				// Process the number
				return input.(int) * 10, nil
			}),
		)).
		Add(pocket.NewNode("format",
			pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
				// Format output
				return fmt.Sprintf("Processed: %d", input.(int)), nil
			}),
		)).
		Connect("input", "default", "process").
		Connect("process", "default", "format").
		Start("input").
		Build()

	if err != nil {
		t.Fatalf("Builder.Build() error = %v", err)
	}

	// Test successful flow
	result, err := flow.Run(context.Background(), 5)
	if err != nil {
		t.Fatalf("Flow.Run() error = %v", err)
	}

	expected := "Processed: 50"
	if result != expected {
		t.Errorf("Flow.Run() = %v, want %v", result, expected)
	}

	// Test error case
	_, err = flow.Run(context.Background(), -5)
	if err == nil {
		t.Error("Flow.Run() with negative input error = nil, want error")
	}
}

func BenchmarkPipeline(b *testing.B) {
	store := pocket.NewStore()

	// Create simple pipeline
	nodes := make([]*pocket.Node, 3)
	for i := range nodes {
		nodes[i] = pocket.NewNode(fmt.Sprintf("node%d", i),
			pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
				return input.(int) + 1, nil
			}),
		)
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := pocket.Pipeline(ctx, nodes, store, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunConcurrent(b *testing.B) {
	store := pocket.NewStore()

	// Create nodes
	nodes := make([]*pocket.Node, 10)
	for i := range nodes {
		nodes[i] = pocket.NewNode(fmt.Sprintf("node%d", i),
			pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
				// Simulate some work
				sum := 0
				for j := 0; j < 100; j++ {
					sum += j
				}
				return sum, nil
			}),
		)
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := pocket.RunConcurrent(ctx, nodes, store)
		if err != nil {
			b.Fatal(err)
		}
	}
}
