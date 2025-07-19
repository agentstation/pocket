package testutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agentstation/pocket"
)

// Fixtures provides common test fixtures.
type Fixtures struct{}

// NewFixtures creates a new fixtures helper.
func NewFixtures() *Fixtures {
	return &Fixtures{}
}

// SimpleNode creates a simple pass-through node.
func (f *Fixtures) SimpleNode(name string) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}),
	)
}

// TransformNode creates a node that transforms input.
func (f *Fixtures) TransformNode(name string, transform func(any) any) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return transform(input), nil
		}),
	)
}

// ErrorNode creates a node that always errors.
func (f *Fixtures) ErrorNode(name string, err error) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return nil, err
		}),
	)
}

// DelayNode creates a node that delays execution.
func (f *Fixtures) DelayNode(name string, delay time.Duration) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			select {
			case <-time.After(delay):
				return input, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}),
	)
}

// CounterNode creates a node that counts executions.
func (f *Fixtures) CounterNode(name string) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			key := fmt.Sprintf("counter:%s", name)
			count, _ := store.Get(ctx, key)

			currentCount := 0
			if c, ok := count.(int); ok {
				currentCount = c
			}

			return map[string]interface{}{
				"input":        input,
				"currentCount": currentCount,
				"key":          key,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			data := prepData.(map[string]interface{})
			currentCount := data["currentCount"].(int)
			currentCount++
			data["newCount"] = currentCount
			return data, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			data := exec.(map[string]interface{})
			key := data["key"].(string)
			newCount := data["newCount"].(int)
			_ = store.Set(ctx, key, newCount)
			return newCount, defaultRoute, nil
		}),
	)
}

// ConditionalNode creates a node with conditional routing.
func (f *Fixtures) ConditionalNode(name string, condition func(any) bool) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			if condition(exec) {
				return exec, "true", nil
			}
			return exec, "false", nil
		}),
	)
}

// StoreNode creates a node that stores its input.
func (f *Fixtures) StoreNode(name, key string) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return map[string]interface{}{
				"key":   key,
				"value": input,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			data := exec.(map[string]interface{})
			_ = store.Set(ctx, data["key"].(string), data["value"])
			return data["value"], defaultRoute, nil
		}),
	)
}

// LoadNode creates a node that loads from store.
func (f *Fixtures) LoadNode(name, key string) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			value, exists := store.Get(ctx, key)
			if !exists {
				return nil, fmt.Errorf("key %s not found", key)
			}
			return value, nil
		}),
	)
}

// ValidatorNode creates a node that validates input.
func (f *Fixtures) ValidatorNode(name string, validate func(any) error) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			if err := validate(input); err != nil {
				return nil, err
			}
			return input, nil
		}),
	)
}

// AggregatorNode creates a node that aggregates values.
func (f *Fixtures) AggregatorNode(name string, aggregate func([]any) any) *pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			key := fmt.Sprintf("aggregator:%s:values", name)

			// Get existing values
			existing, _ := store.Get(ctx, key)
			values := []any{}
			if v, ok := existing.([]any); ok {
				values = v
			}

			return map[string]interface{}{
				"input":  input,
				"values": values,
				"key":    key,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			data := prepData.(map[string]interface{})
			values := data["values"].([]any)
			input := data["input"]

			// Add new value
			values = append(values, input)

			// Return aggregated result
			result := aggregate(values)

			return map[string]interface{}{
				"key":    data["key"],
				"values": values,
				"result": result,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			data := exec.(map[string]interface{})
			key := data["key"].(string)
			values := data["values"].([]any)
			result := data["result"]

			_ = store.Set(ctx, key, values)

			return result, defaultRoute, nil
		}),
	)
}

// Graphs provides common graph patterns.

// LinearGraph creates a linear graph of nodes.
func (f *Fixtures) LinearGraph(store pocket.Store, nodes ...*pocket.Node) *pocket.Graph {
	if len(nodes) == 0 {
		return nil
	}

	// Connect nodes in sequence
	for i := 0; i < len(nodes)-1; i++ {
		nodes[i].Connect("default", nodes[i+1])
	}

	return pocket.NewGraph(nodes[0], store)
}

// BranchingGraph creates a graph with conditional branching.
func (f *Fixtures) BranchingGraph(store pocket.Store, condition, trueBranch, falseBranch *pocket.Node) *pocket.Graph {
	condition.Connect("true", trueBranch)
	condition.Connect("false", falseBranch)

	return pocket.NewGraph(condition, store)
}

// LoopGraph creates a graph with a loop.
func (f *Fixtures) LoopGraph(store pocket.Store, body *pocket.Node, maxIterations int) *pocket.Graph {
	counter := f.CounterNode("loop_counter")

	check := pocket.NewNode[any, any]("loop_check",
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			count, _ := store.Get(ctx, "counter:loop_counter")
			if c, ok := count.(int); ok && c < maxIterations {
				return exec, "continue", nil
			}
			return exec, "done", nil
		}),
	)

	// Connect: counter -> check -> body -> counter (loop)
	counter.Connect("default", check)
	check.Connect("continue", body)
	body.Connect("default", counter)

	return pocket.NewGraph(counter, store)
}

// containsIgnoreCase is a helper function for case-insensitive string contains check.
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Data provides test data fixtures.

// User represents a test user.
type User struct {
	ID    string
	Name  string
	Email string
	Age   int
}

// SampleUsers returns sample user data.
func (f *Fixtures) SampleUsers() []User {
	return []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com", Age: 30},
		{ID: "2", Name: "Bob", Email: "bob@example.com", Age: 25},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com", Age: 35},
	}
}

// Message represents a test message.
type Message struct {
	ID        string
	Content   string
	Author    string
	Timestamp time.Time
}

// SampleMessages returns sample message data.
func (f *Fixtures) SampleMessages() []Message {
	now := time.Now()
	return []Message{
		{ID: "m1", Content: "Hello world", Author: "Alice", Timestamp: now.Add(-1 * time.Hour)},
		{ID: "m2", Content: "How are you?", Author: "Bob", Timestamp: now.Add(-30 * time.Minute)},
		{ID: "m3", Content: "Great, thanks!", Author: "Alice", Timestamp: now},
	}
}

// TestData represents generic test data.
type TestData struct {
	String string
	Int    int
	Float  float64
	Bool   bool
	Time   time.Time
	List   []string
	Map    map[string]any
}

// SampleTestData returns sample test data.
func (f *Fixtures) SampleTestData() TestData {
	return TestData{
		String: "test",
		Int:    42,
		Float:  3.14,
		Bool:   true,
		Time:   time.Now(),
		List:   []string{"a", "b", "c"},
		Map: map[string]any{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		},
	}
}

// Scenarios provides complete test scenarios.

// Scenario represents a test scenario.
type Scenario struct {
	Name        string
	Description string
	Graph       *pocket.Graph
	Input       any
	Expected    any
	ShouldError bool
}

// ChatBotScenario creates a chat bot test scenario.
func (f *Fixtures) ChatBotScenario(store pocket.Store) Scenario {
	// Intent classifier
	classifier := pocket.NewNode[any, any]("classifier",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			msg, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("expected string input")
			}

			// Simple classification
			if containsIgnoreCase(msg, "hello") || containsIgnoreCase(msg, "hi") {
				return "greeting", nil
			}
			if containsIgnoreCase(msg, "help") {
				return "help", nil
			}
			return "general", nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			intent := exec.(string)
			return input, intent, nil
		}),
	)

	// Response handlers
	greeting := pocket.NewNode[any, any]("greeting",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return "Hello! How can I help you today?", nil
		}),
	)

	help := pocket.NewNode[any, any]("help",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return "I can help you with various tasks. Just ask!", nil
		}),
	)

	general := pocket.NewNode[any, any]("general",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return "I understand. Please tell me more.", nil
		}),
	)

	// Connect nodes
	classifier.Connect("greeting", greeting)
	classifier.Connect("help", help)
	classifier.Connect("general", general)

	return Scenario{
		Name:        "ChatBot",
		Description: "Simple chat bot with intent classification",
		Graph:       pocket.NewGraph(classifier, store),
		Input:       "hello there",
		Expected:    "Hello! How can I help you today?",
		ShouldError: false,
	}
}

// DataPipelineScenario creates a data pipeline test scenario.
func (f *Fixtures) DataPipelineScenario(store pocket.Store) Scenario {
	// Validate
	validate := f.ValidatorNode("validate", func(input any) error {
		data, ok := input.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map input")
		}
		if _, exists := data["value"]; !exists {
			return fmt.Errorf("missing required field: value")
		}
		return nil
	})

	// Transform
	transform := f.TransformNode("transform", func(input any) any {
		data := input.(map[string]any)
		value := data["value"].(float64)
		data["doubled"] = value * 2
		data["squared"] = value * value
		return data
	})

	// Store result
	storeResult := f.StoreNode("store", "pipeline_result")

	// Connect pipeline
	validate.Connect("default", transform)
	transform.Connect("default", storeResult)

	return Scenario{
		Name:        "DataPipeline",
		Description: "Data validation, transformation, and storage",
		Graph:       pocket.NewGraph(validate, store),
		Input:       map[string]any{"value": 5.0},
		Expected:    map[string]any{"value": 5.0, "doubled": 10.0, "squared": 25.0},
		ShouldError: false,
	}
}
