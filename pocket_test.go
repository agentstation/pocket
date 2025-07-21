package pocket_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/agentstation/pocket"
)

const (
	testResult = "result"
)

func TestNodeLifecycle(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    any
		wantErr bool
	}{
		{
			name:  "string transformation",
			input: "hello",
			want:  "HELLO",
		},
		{
			name:  "number doubling",
			input: 5,
			want:  10,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node pocket.Node

			switch tt.name {
			case "string transformation":
				node = pocket.NewNode[any, any]("transform",
					pocket.Steps{
						Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
							s, ok := input.(string)
							if !ok {
								return nil, fmt.Errorf("expected string")
							}
							return s, nil
						},
						Exec: func(ctx context.Context, s any) (any, error) {
							return strings.ToUpper(s.(string)), nil
						},
						Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
							return result, doneRoute, nil
						},
					},
				)

			case "number doubling":
				node = pocket.NewNode[any, any]("double",
					pocket.Steps{
						Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
							n, ok := input.(int)
							if !ok {
								return nil, fmt.Errorf("expected int")
							}
							return n, nil
						},
						Exec: func(ctx context.Context, n any) (any, error) {
							return n.(int) * 2, nil
						},
						Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
							return result, doneRoute, nil
						},
					},
				)

			case "nil input":
				node = pocket.NewNode[any, any]("nilcheck",
					pocket.Steps{
						Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
							if input == nil {
								return nil, errors.New("nil input")
							}
							return input, nil
						},
					},
				)
			}

			store := pocket.NewStore()
			graph := pocket.NewGraph(node, store)

			got, err := graph.Run(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeConnections(t *testing.T) {
	// Create test nodes using lifecycle
	start := pocket.NewNode[any, any]("start",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				return "processed", nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
				return result, "next", nil
			},
		},
	)

	middle := pocket.NewNode[any, any]("middle",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				return input.(string) + "-middle", nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
				return result, "default", nil
			},
		},
	)

	end := pocket.NewNode[any, any]("end",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				return input.(string) + "-end", nil
			},
		},
	)

	// Test connections
	start.Connect("next", middle)
	middle.Connect("default", end)

	store := pocket.NewStore()
	graph := pocket.NewGraph(start, store)

	result, err := graph.Run(context.Background(), "input")
	if err != nil {
		t.Fatalf("Graph execution failed: %v", err)
	}

	expected := "processed-middle-end"
	if result != expected {
		t.Errorf("Graph result = %v, want %v", result, expected)
	}
}

func TestGraphExecution(t *testing.T) {
	tests := []struct {
		name       string
		setupGraph func() (*pocket.Graph, pocket.Store)
		input      any
		want       any
		wantErr    error
	}{
		{
			name: "simple graph",
			setupGraph: func() (*pocket.Graph, pocket.Store) {
				node := pocket.NewNode[any, any]("simple",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return testResult, nil
						},
					},
				)
				store := pocket.NewStore()
				return pocket.NewGraph(node, store), store
			},
			input: "test",
			want:  testResult,
		},
		{
			name: "graph with routing",
			setupGraph: func() (*pocket.Graph, pocket.Store) {
				router := pocket.NewNode[any, any]("router",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return input, nil
						},
						Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
							if result.(int) > 10 {
								return result, "big", nil
							}
							return result, "small", nil
						},
					},
				)

				big := pocket.NewNode[any, any]("big",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return "big number", nil
						},
					},
				)

				small := pocket.NewNode[any, any]("small",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return "small number", nil
						},
					},
				)

				router.Connect("big", big)
				router.Connect("small", small)

				store := pocket.NewStore()
				return pocket.NewGraph(router, store), store
			},
			input: 15,
			want:  "big number",
		},
		{
			name: "graph with error",
			setupGraph: func() (*pocket.Graph, pocket.Store) {
				node := pocket.NewNode[any, any]("error",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return nil, errors.New("process error")
						},
					},
				)
				store := pocket.NewStore()
				return pocket.NewGraph(node, store), store
			},
			input:   "test",
			wantErr: errors.New("node error: exec failed: process error"),
		},
		{
			name: "no start node",
			setupGraph: func() (*pocket.Graph, pocket.Store) {
				store := pocket.NewStore()
				return pocket.NewGraph(nil, store), store
			},
			input:   "test",
			wantErr: pocket.ErrNoStartNode,
		},
		{
			name: "prep step error",
			setupGraph: func() (*pocket.Graph, pocket.Store) {
				node := pocket.NewNode[any, any]("prep-error",
					pocket.Steps{
						Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
							return nil, errors.New("prep failed")
						},
					},
				)
				store := pocket.NewStore()
				return pocket.NewGraph(node, store), store
			},
			input:   "test",
			wantErr: errors.New("node prep-error: prep failed: prep failed"),
		},
		{
			name: "post step error",
			setupGraph: func() (*pocket.Graph, pocket.Store) {
				node := pocket.NewNode[any, any]("post-error",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return "result", nil
						},
						Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
							return nil, "", errors.New("post failed")
						},
					},
				)
				store := pocket.NewStore()
				return pocket.NewGraph(node, store), store
			},
			input:   "test",
			wantErr: errors.New("node post-error: post failed: post failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, _ := tt.setupGraph()
			got, err := graph.Run(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Run() error = nil, wantErr %v", tt.wantErr)
				} else if tt.wantErr == pocket.ErrNoStartNode && err != pocket.ErrNoStartNode {
					t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Run() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	tests := []struct {
		name  string
		op    func()
		check func(t *testing.T)
	}{
		{
			name: "set and get",
			op: func() {
				_ = store.Set(ctx, "key1", "value1")
				_ = store.Set(ctx, "key2", 42)
			},
			check: func(t *testing.T) {
				val1, ok := store.Get(ctx, "key1")
				if !ok || val1 != "value1" {
					t.Errorf("Get(key1) = %v, %v; want value1, true", val1, ok)
				}

				val2, ok := store.Get(ctx, "key2")
				if !ok || val2 != 42 {
					t.Errorf("Get(key2) = %v, %v; want 42, true", val2, ok)
				}
			},
		},
		{
			name: "get missing key",
			op:   func() {},
			check: func(t *testing.T) {
				_, ok := store.Get(ctx, "missing")
				if ok {
					t.Error("Get(missing) returned true, want false")
				}
			},
		},
		{
			name: "delete",
			op: func() {
				_ = store.Set(ctx, "temp", "data")
				_ = store.Delete(ctx, "temp")
			},
			check: func(t *testing.T) {
				_, ok := store.Get(ctx, "temp")
				if ok {
					t.Error("Get(temp) after Delete returned true, want false")
				}
			},
		},
		{
			name: "scoped store",
			op: func() {
				userStore := store.Scope("user")
				userStore.Set(ctx, "123", "Alice")
			},
			check: func(t *testing.T) {
				// Should be accessible via scoped store
				userStore := store.Scope("user")
				val, ok := userStore.Get(ctx, "123")
				if !ok || val != "Alice" {
					t.Errorf("Scoped Get(123) = %v, %v; want Alice, true", val, ok)
				}

				// Should be accessible via main store with prefix
				val2, ok := store.Get(ctx, "user:123")
				if !ok || val2 != "Alice" {
					t.Errorf("Get(user:123) = %v, %v; want Alice, true", val2, ok)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.op()
			tt.check(t)
		})
	}
}

func TestWithRetry(t *testing.T) {
	attempts := 0
	ctx := context.Background()

	node := pocket.NewNode[any, any]("retry",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				attempts++
				if attempts < 3 {
					return nil, errors.New("temporary error")
				}
				return successResult, nil
			},
		},
		pocket.WithRetry(3, 10*time.Millisecond),
	)

	store := pocket.NewStore()
	graph := pocket.NewGraph(node, store)

	start := time.Now()
	result, err := graph.Run(ctx, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if result != successResult {
		t.Errorf("Expected 'success', got %v", result)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Check that retry delays were applied (2 retries * 10ms)
	if duration < 20*time.Millisecond {
		t.Errorf("Expected duration >= 20ms, got %v", duration)
	}
}

func TestWithTimeout(t *testing.T) {
	node := pocket.NewNode[any, any]("slow",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				select {
				case <-time.After(100 * time.Millisecond):
					return "completed", nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		},
		pocket.WithTimeout(10*time.Millisecond),
	)

	store := pocket.NewStore()
	graph := pocket.NewGraph(node, store)

	_, err := graph.Run(context.Background(), nil)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// The error will be wrapped, so check if it contains deadline exceeded
	if !strings.Contains(err.Error(), "context deadline exceeded") &&
		!strings.Contains(err.Error(), "failed after 1 attempts") {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestBuilder(t *testing.T) {
	store := pocket.NewStore()

	node1 := pocket.NewNode[any, any]("node1",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				return "from-node1", nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
				return result, "default", nil
			},
		},
	)

	node2 := pocket.NewNode[any, any]("node2",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				return input.(string) + "-node2", nil
			},
		},
	)

	tests := []struct {
		name    string
		build   func() (*pocket.Graph, error)
		wantErr bool
	}{
		{
			name: "successful build",
			build: func() (*pocket.Graph, error) {
				return pocket.NewBuilder(store).
					Add(node1).
					Add(node2).
					Connect("node1", "default", "node2").
					Start("node1").
					Build()
			},
			wantErr: false,
		},
		{
			name: "no start node",
			build: func() (*pocket.Graph, error) {
				return pocket.NewBuilder(store).
					Build()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := tt.build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && graph == nil {
				t.Error("Build() returned nil graph without error")
			}
		})
	}
}

func TestTypedNode(t *testing.T) {
	type User struct {
		Name string
	}

	type Greeting struct {
		Message string
	}

	// Create a typed node with lifecycle
	node := pocket.NewNode[User, Greeting]("greet",
		pocket.Steps{
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				user, ok := input.(User)
				if !ok {
					return nil, fmt.Errorf("expected User type")
				}
				if user.Name == "" {
					return nil, fmt.Errorf("name required")
				}
				return user, nil
			},
			Exec: func(ctx context.Context, prepData any) (any, error) {
				user := prepData.(User)
				return Greeting{Message: fmt.Sprintf("Hello, %s!", user.Name)}, nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
				return result, doneRoute, nil
			},
		},
	)

	// Verify types are set
	if node.InputType() == nil {
		t.Error("NewNode did not set InputType")
	}
	if node.OutputType() == nil {
		t.Error("NewNode did not set OutputType")
	}

	// Execute the node
	store := pocket.NewStore()
	graph := pocket.NewGraph(node, store)

	result, err := graph.Run(context.Background(), User{Name: "Alice"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	greeting, ok := result.(Greeting)
	if !ok {
		t.Fatalf("Result is not Greeting type, got %T", result)
	}

	if greeting.Message != "Hello, Alice!" {
		t.Errorf("Greeting message = %v, want %v", greeting.Message, "Hello, Alice!")
	}
}

func TestValidateGraph(t *testing.T) {
	type Input struct{ Value int }
	type Output struct{ Result string }
	type Different struct{ Data float64 }

	tests := []struct {
		name       string
		setupGraph func() pocket.Node
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid graph with matching types",
			setupGraph: func() pocket.Node {
				// Create nodes with matching input/output types
				node1 := pocket.NewNode[Input, Output]("node1",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return Output{Result: "processed"}, nil
						},
					},
				)

				node2 := pocket.NewNode[Output, Different]("node2",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return Different{Data: 3.14}, nil
						},
					},
				)

				node1.Connect("default", node2)
				return node1
			},
			wantErr: false,
		},
		{
			name: "invalid graph with type mismatch",
			setupGraph: func() pocket.Node {
				// Create nodes with mismatched types
				node1 := pocket.NewNode[Input, Output]("node1",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return Output{Result: "processed"}, nil
						},
					},
				)

				// node2 expects Different but node1 outputs Output
				node2 := pocket.NewNode[Different, Input]("node2",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return Input{Value: 42}, nil
						},
					},
				)

				node1.Connect("default", node2)
				return node1
			},
			wantErr: true,
			errMsg:  "type mismatch",
		},
		{
			name: "graph with untyped nodes (should pass)",
			setupGraph: func() pocket.Node {
				// Mix of typed and untyped nodes
				typedNode := pocket.NewNode[Input, Output]("typed",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return Output{Result: "processed"}, nil
						},
					},
				)

				// Untyped node - no validation performed
				untypedNode := pocket.NewNode[any, any]("untyped",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return testResult, nil
						},
					},
				)

				typedNode.Connect("default", untypedNode)
				return typedNode
			},
			wantErr: false,
		},
		{
			name: "cyclic graph validation",
			setupGraph: func() pocket.Node {
				node1 := pocket.NewNode[Input, Output]("node1",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return Output{Result: "processed"}, nil
						},
						Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
							return result, defaultRoute, nil
						},
					},
				)

				node2 := pocket.NewNode[Output, Input]("node2",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return Input{Value: 42}, nil
						},
						Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
							return result, defaultRoute, nil
						},
					},
				)

				// Create a cycle
				node1.Connect("default", node2)
				node2.Connect("default", node1)

				return node1
			},
			wantErr: false, // Cycles are allowed, validation handles visited nodes
		},
		{
			name: "interface type compatibility",
			setupGraph: func() pocket.Node {
				// With any,any nodes, type checking is not enforced
				// This test validates that nodes can be connected regardless of type
				node1 := pocket.NewNode[any, any]("producer",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return struct{ Value string }{Value: "test"}, nil
						},
					},
				)

				node2 := pocket.NewNode[any, any]("consumer",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return testResult, nil
						},
					},
				)

				node1.Connect("default", node2)
				return node1
			},
			wantErr: false, // With any,any nodes, all types are compatible
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startNode := tt.setupGraph()
			err := pocket.ValidateGraph(startNode)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGraph() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateGraph() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestLifecycleSteps(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	// Track execution order
	var executionOrder []string

	node := pocket.NewNode[any, any]("lifecycle",
		pocket.Steps{
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				executionOrder = append(executionOrder, "prep")
				return input.(string) + "-prepped", nil
			},
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				executionOrder = append(executionOrder, "exec")
				return prepResult.(string) + "-executed", nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prepResult, execResult any) (any, string, error) {
				executionOrder = append(executionOrder, "post")
				// Verify all values are available
				if input.(string) != "input" {
					t.Errorf("Post got wrong input: %v", input)
				}
				if prepResult.(string) != "input-prepped" {
					t.Errorf("Post got wrong prepResult: %v", prepResult)
				}
				if execResult.(string) != "input-prepped-executed" {
					t.Errorf("Post got wrong execResult: %v", execResult)
				}
				return execResult.(string) + "-posted", doneRoute, nil
			},
		},
	)

	graph := pocket.NewGraph(node, store)
	result, err := graph.Run(ctx, "input")

	if err != nil {
		t.Fatalf("Graph failed: %v", err)
	}

	// Check execution order
	expectedOrder := []string{"prep", "exec", "post"}
	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("Wrong execution order length: %v", executionOrder)
	}
	for i, step := range expectedOrder {
		if executionOrder[i] != step {
			t.Errorf("Step %d: got %s, want %s", i, executionOrder[i], step)
		}
	}

	// Check final result
	if result != "input-prepped-executed-posted" {
		t.Errorf("Wrong final result: %v", result)
	}
}

func TestRetryPerStep(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	prepAttempts := 0
	execAttempts := 0

	node := pocket.NewNode[any, any]("retry-steps",
		pocket.Steps{
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				prepAttempts++
				if prepAttempts < 2 {
					return nil, errors.New("prep retry")
				}
				return "prep-success", nil
			},
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				execAttempts++
				if execAttempts < 2 {
					return nil, errors.New("exec retry")
				}
				return "exec-success", nil
			},
		},
		pocket.WithRetry(3, 10*time.Millisecond),
	)

	graph := pocket.NewGraph(node, store)
	result, err := graph.Run(ctx, "input")

	if err != nil {
		t.Fatalf("Graph failed: %v", err)
	}

	if prepAttempts != 2 {
		t.Errorf("Prep attempts = %d, want 2", prepAttempts)
	}
	if execAttempts != 2 {
		t.Errorf("Exec attempts = %d, want 2", execAttempts)
	}
	if result != "exec-success" {
		t.Errorf("Result = %v, want exec-success", result)
	}
}

func TestErrorHandler(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	var capturedError error

	node := pocket.NewNode[any, any]("error-handler",
		pocket.Steps{
			Exec: func(ctx context.Context, input any) (any, error) {
				return nil, errors.New("test error")
			},
		},
		pocket.WithErrorHandler(func(err error) {
			capturedError = err
		}),
	)

	graph := pocket.NewGraph(node, store)
	_, err := graph.Run(ctx, "input")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if capturedError == nil {
		t.Fatal("Error handler not called")
	}

	if !strings.Contains(capturedError.Error(), "exec failed") {
		t.Errorf("Wrong error captured: %v", capturedError)
	}
}
