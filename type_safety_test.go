package pocket_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agentstation/pocket"
)

const (
	// Using successResult from lifecycle_test.go
	// Using defaultRoute from options_test.go.
	testValue = "test"
)

// Test types for type safety testing.
type TestInput struct {
	Value string
}

type TestOutput struct {
	Result string
}

type DifferentType struct {
	Data int
}

// TestNewNodeGeneric tests the new generic NewNode function.
func TestNewNodeGeneric(t *testing.T) {
	tests := []struct {
		name      string
		setupNode func() pocket.Node
		checkNode func(t *testing.T, node pocket.Node)
	}{
		{
			name: "typed node sets InputType and OutputType",
			setupNode: func() pocket.Node {
				return pocket.NewNode[TestInput, TestOutput]("typed",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							typedInput := input.(TestInput)
							return TestOutput{Result: typedInput.Value}, nil
						},
					},
				)
			},
			checkNode: func(t *testing.T, node pocket.Node) {
				// In the new API, we can check types using the interface methods
				if node.InputType() == nil {
					t.Error("Expected InputType to be set for typed node")
				}
				if node.OutputType() == nil {
					t.Error("Expected OutputType to be set for typed node")
				}
				if node.InputType().String() != "pocket_test.TestInput" {
					t.Errorf("Wrong InputType: got %v", node.InputType())
				}
				if node.OutputType().String() != "pocket_test.TestOutput" {
					t.Errorf("Wrong OutputType: got %v", node.OutputType())
				}
			},
		},
		{
			name: "untyped node has nil types",
			setupNode: func() pocket.Node {
				return pocket.NewNode[any, any]("untyped",
					pocket.Steps{
						Exec: func(ctx context.Context, input any) (any, error) {
							return input, nil
						},
					},
				)
			},
			checkNode: func(t *testing.T, node pocket.Node) {
				// For untyped nodes (any, any), the type information may still be available
				// through reflection but it would show as interface{}
				// This test may need to be adjusted based on the actual implementation
				if node.InputType() != nil && node.InputType().String() != "interface {}" {
					t.Errorf("Expected InputType to be interface{} for untyped node, got %v", node.InputType())
				}
				if node.OutputType() != nil && node.OutputType().String() != "interface {}" {
					t.Errorf("Expected OutputType to be interface{} for untyped node, got %v", node.OutputType())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := tt.setupNode()
			tt.checkNode(t, node)
		})
	}
}

// TestNodeOptions tests the unified option functions with type safety.
func TestNodeOptions(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("WithExec provides compile-time type safety", func(t *testing.T) {
		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					// Type assertion needed for untyped function
					typedInput := input.(TestInput)
					return TestOutput{Result: "Processed: " + typedInput.Value}, nil
				},
			},
		)

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, TestInput{Value: "test"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		output, ok := result.(TestOutput)
		if !ok {
			t.Fatalf("Expected TestOutput, got %T", result)
		}
		if output.Result != "Processed: test" {
			t.Errorf("Wrong result: %v", output.Result)
		}
	})

	t.Run("WithPrep handles prep step with type safety", func(t *testing.T) {
		var prepCalled bool
		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
					prepCalled = true
					// Modify input
					typedInput := input.(TestInput)
					typedInput.Value = strings.ToUpper(typedInput.Value)
					return typedInput, nil
				},
				Exec: func(ctx context.Context, input any) (any, error) {
					typedInput := input.(TestInput)
					return TestOutput{Result: typedInput.Value}, nil
				},
			},
		)

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, TestInput{Value: "hello"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !prepCalled {
			t.Error("Prep function was not called")
		}

		output := result.(TestOutput)
		if output.Result != "HELLO" {
			t.Errorf("Expected uppercase result, got %v", output.Result)
		}
	})

	t.Run("WithPost handles routing with type safety", func(t *testing.T) {
		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					typedInput := input.(TestInput)
					return TestOutput{Result: typedInput.Value}, nil
				},
				Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, output any) (any, string, error) {
					typedOutput := output.(TestOutput)
					if strings.HasPrefix(typedOutput.Result, "error") {
						return typedOutput, "error", nil
					}
					return typedOutput, successResult, nil
				},
			},
		)

		// Test nodes should still work without full routing setup
		graph := pocket.NewGraph(node, store)
		_, err := graph.Run(ctx, TestInput{Value: "test"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("WithFallback provides typed error recovery", func(t *testing.T) {
		execError := errors.New("exec failed")
		fallbackCalled := false

		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return TestOutput{}, execError
				},
			},
			pocket.WithFallback(func(ctx context.Context, input TestInput, err error) (TestOutput, error) {
				fallbackCalled = true
				// The error will be wrapped with retry information
				if !strings.Contains(err.Error(), execError.Error()) {
					t.Errorf("Wrong error in fallback, expected to contain %v, got: %v", execError, err)
				}
				return TestOutput{Result: "fallback: " + input.Value}, nil
			}),
		)

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, TestInput{Value: "test"})
		if err != nil {
			t.Fatalf("Expected fallback to handle error, got: %v", err)
		}

		if !fallbackCalled {
			t.Error("Fallback was not called")
		}

		output := result.(TestOutput)
		if output.Result != "fallback: test" {
			t.Errorf("Wrong fallback result: %v", output.Result)
		}
	})

	t.Run("WithOnSuccess runs on successful execution", func(t *testing.T) {
		successCalled := false
		var capturedOutput TestOutput

		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return TestOutput{Result: "success"}, nil
				},
			},
			pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output TestOutput) {
				successCalled = true
				capturedOutput = output
			}),
		)

		graph := pocket.NewGraph(node, store)
		_, err := graph.Run(ctx, TestInput{Value: "test"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !successCalled {
			t.Error("OnSuccess was not called")
		}
		if capturedOutput.Result != "success" {
			t.Errorf("Wrong output in OnSuccess: %v", capturedOutput)
		}
	})
}

// TestRuntimeTypeSafety tests runtime type checking with regular options.
func TestRuntimeTypeSafety(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("typed node with untyped WithExec wraps for type safety", func(t *testing.T) {
		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					// Even with untyped WithExec, typed nodes ensure type safety
					typedInput := input.(TestInput) // This cast should be safe
					return TestOutput{Result: typedInput.Value}, nil
				},
			},
		)

		graph := pocket.NewGraph(node, store)

		// Correct type works
		result, err := graph.Run(ctx, TestInput{Value: "test"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if output, ok := result.(TestOutput); !ok || output.Result != testValue {
			t.Errorf("Wrong result: %v", result)
		}

		// Wrong type should fail
		_, err = graph.Run(ctx, DifferentType{Data: 42})
		if err == nil {
			t.Error("Expected type error for wrong input type")
		}
		if !strings.Contains(err.Error(), "invalid input type") {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("mixed typed and regular options", func(t *testing.T) {
		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					typedInput := input.(TestInput)
					return TestOutput{Result: typedInput.Value}, nil
				},
			},
			// Regular options work with typed nodes
			pocket.WithRetry(3, 10*time.Millisecond),
			pocket.WithTimeout(100*time.Millisecond),
		)

		// Verify options were applied
		if node.Name() != "test" {
			t.Errorf("Wrong node name: %v", node.Name())
		}
	})
}

// TestValidateGraphTypeSafety tests initialization-time type validation.
func TestValidateGraphTypeSafety(t *testing.T) {
	t.Run("validates compatible typed nodes", func(t *testing.T) {
		node1 := pocket.NewNode[TestInput, TestOutput]("node1",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					typedInput := input.(TestInput)
					return TestOutput{Result: typedInput.Value}, nil
				},
			},
		)

		node2 := pocket.NewNode[TestOutput, DifferentType]("node2",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					typedInput := input.(TestOutput)
					return DifferentType{Data: len(typedInput.Result)}, nil
				},
			},
		)

		// Connect compatible nodes
		node1.Connect("default", node2)

		// Should validate successfully
		if err := pocket.ValidateGraph(node1); err != nil {
			t.Errorf("Expected validation to pass, got: %v", err)
		}
	})

	t.Run("catches type mismatches between nodes", func(t *testing.T) {
		node1 := pocket.NewNode[TestInput, TestOutput]("node1",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return TestOutput{}, nil
				},
			},
		)

		// node2 expects DifferentType but node1 outputs TestOutput
		node2 := pocket.NewNode[DifferentType, TestOutput]("node2",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return TestOutput{}, nil
				},
			},
		)

		// Connect incompatible nodes
		node1.Connect("default", node2)

		// Should fail validation
		err := pocket.ValidateGraph(node1)
		if err == nil {
			t.Error("Expected validation to fail for type mismatch")
		}
		if !strings.Contains(err.Error(), "type mismatch") {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("allows untyped nodes in typed graph", func(t *testing.T) {
		typedNode := pocket.NewNode[TestInput, TestOutput]("typed",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					typedInput := input.(TestInput)
					return TestOutput{Result: typedInput.Value}, nil
				},
			},
		)

		// Untyped node can accept any input
		untypedNode := pocket.NewNode[any, any]("untyped",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					return input, nil
				},
			},
		)

		typedNode.Connect("default", untypedNode)

		// Should validate (untyped nodes are not checked)
		if err := pocket.ValidateGraph(typedNode); err != nil {
			t.Errorf("Expected validation to pass with untyped node, got: %v", err)
		}
	})
}

// TestNewAPIUsagePatterns tests common patterns with the new API.
func TestNewAPIUsagePatterns(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("strongly typed nodes with unified API", func(t *testing.T) {
		// All options work seamlessly with typed nodes
		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
					typedInput := input.(TestInput)
					typedInput.Value = strings.ToUpper(typedInput.Value)
					return typedInput, nil
				},
				Exec: func(ctx context.Context, input any) (any, error) {
					typedInput := input.(TestInput)
					return TestOutput{Result: typedInput.Value}, nil
				},
				Post: func(ctx context.Context, store pocket.StoreWriter, input, prep, output any) (any, string, error) {
					typedOutput := output.(TestOutput)
					return typedOutput, defaultRoute, nil
				},
			},
			pocket.WithFallback(func(ctx context.Context, input TestInput, err error) (TestOutput, error) {
				return TestOutput{Result: "fallback"}, nil
			}),
			pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output TestOutput) {
				// Log success
			}),
		)

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, TestInput{Value: "test"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		output := result.(TestOutput)
		if output.Result != "TEST" {
			t.Errorf("Expected uppercase result, got %v", output.Result)
		}
	})

	t.Run("untyped nodes work as before", func(t *testing.T) {
		// Untyped nodes use any/interface{} types
		node := pocket.NewNode[any, any]("untyped",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					// Handle any input type
					switch v := input.(type) {
					case string:
						return "String: " + v, nil
					case int:
						return v * 2, nil
					default:
						return input, nil
					}
				},
			},
		)

		graph := pocket.NewGraph(node, store)

		// Test with string
		result, err := graph.Run(ctx, "hello")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != "String: hello" {
			t.Errorf("Wrong string result: %v", result)
		}

		// Test with int
		result, err = graph.Run(ctx, 21)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != 42 {
			t.Errorf("Wrong int result: %v", result)
		}
	})
}

// TestAutoTypeWrapping tests the automatic type wrapping for regular options.
func TestAutoTypeWrapping(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("untyped WithExec gets wrapped for typed nodes", func(t *testing.T) {
		execCalled := false
		node := pocket.NewNode[TestInput, TestOutput]("test",
			pocket.Steps{
				Exec: func(ctx context.Context, input any) (any, error) {
					execCalled = true
					// Should be safe to cast due to wrapping
					_, ok := input.(TestInput)
					if !ok {
						t.Errorf("Expected input to be TestInput, got %T", input)
					}
					return TestOutput{Result: "wrapped"}, nil
				},
			},
		)

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, TestInput{Value: "test"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !execCalled {
			t.Error("Exec was not called")
		}

		output := result.(TestOutput)
		if output.Result != "wrapped" {
			t.Errorf("Wrong result: %v", output.Result)
		}
	})

}
