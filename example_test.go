package pocket_test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/agentstation/pocket"
)

// Using constants from lifecycle_test.go and options_test.go

// ExampleNode demonstrates using the Prep/Exec/Post lifecycle.
func ExampleNode() {
	// Create a node with lifecycle steps
	uppercase := pocket.NewNode[any, any]("uppercase",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate input is a string
			text, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("expected string, got %T", input)
			}
			return text, nil
		}),
		pocket.WithExec(func(ctx context.Context, text any) (any, error) {
			// Transform to uppercase
			return strings.ToUpper(text.(string)), nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, text, result any) (any, string, error) {
			// Return result and routing
			return result, doneRoute, nil
		}),
	)

	store := pocket.NewStore()
	graph := pocket.NewGraph(uppercase, store)

	result, err := graph.Run(context.Background(), "hello world")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output: HELLO WORLD
}

// ExampleBuilder demonstrates the fluent builder API.
func ExampleBuilder() {
	store := pocket.NewStore()

	// Define nodes with lifecycle
	validate := pocket.NewNode[any, any]("validate",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			email, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("expected string")
			}
			return email, nil
		}),
		pocket.WithExec(func(ctx context.Context, email any) (any, error) {
			if !strings.Contains(email.(string), "@") {
				return nil, fmt.Errorf("invalid email")
			}
			return email, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			return result, defaultRoute, nil
		}),
	)

	normalize := pocket.NewNode[any, any]("normalize",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			email := input.(string)
			return strings.ToLower(strings.TrimSpace(email)), nil
		}),
	)

	// Build the graph
	graph, err := pocket.NewBuilder(store).
		Add(validate).
		Add(normalize).
		Connect("validate", "default", "normalize").
		Start("validate").
		Build()

	if err != nil {
		log.Fatal(err)
	}

	result, err := graph.Run(context.Background(), "  USER@EXAMPLE.COM  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output: user@example.com
}

// ExampleNode_routing demonstrates conditional routing between nodes.
func ExampleNode_routing() {
	store := pocket.NewStore()

	// Router node that checks input
	router := pocket.NewNode[any, any]("router",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			value := result.(int)
			if value > 100 {
				return result, "large", nil
			}
			return result, "small", nil
		}),
	)

	// Handler nodes
	largeHandler := pocket.NewNode[any, any]("large",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Large number: %v", input), nil
		}),
	)

	smallHandler := pocket.NewNode[any, any]("small",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Small number: %v", input), nil
		}),
	)

	// Connect nodes
	router.Connect("large", largeHandler)
	router.Connect("small", smallHandler)

	// Run with different inputs
	graph := pocket.NewGraph(router, store)

	result1, _ := graph.Run(context.Background(), 50)
	result2, _ := graph.Run(context.Background(), 150)

	fmt.Println(result1)
	fmt.Println(result2)
	// Output:
	// Small number: 50
	// Large number: 150
}

// ExampleFanOut demonstrates parallel processing of items.
func ExampleFanOut() {
	// Create a processor that simulates work
	processor := pocket.NewNode[any, any]("process",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			num := input.(int)
			return num * num, nil
		}),
	)

	store := pocket.NewStore()
	items := []int{1, 2, 3, 4, 5}

	// Process items concurrently
	results, err := pocket.FanOut(context.Background(), processor, store, items)
	if err != nil {
		log.Fatal(err)
	}

	// Results maintain order
	for i, result := range results {
		fmt.Printf("%d -> %v\n", items[i], result)
	}
	// Output:
	// 1 -> 1
	// 2 -> 4
	// 3 -> 9
	// 4 -> 16
	// 5 -> 25
}

// ExamplePipeline demonstrates sequential processing.
func ExamplePipeline() {
	store := pocket.NewStore()

	// Create a pipeline of transformations
	double := pocket.NewNode[any, any]("double",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input.(int) * 2, nil
		}),
	)

	addTen := pocket.NewNode[any, any]("addTen",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return input.(int) + 10, nil
		}),
	)

	toString := pocket.NewNode[any, any]("toString",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Result: %d", input.(int)), nil
		}),
	)

	nodes := []*pocket.Node{double, addTen, toString}

	result, err := pocket.Pipeline(context.Background(), nodes, store, 5)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output: Result: 20
}

// ExampleTypedStore demonstrates type-safe storage.
func ExampleTypedStore() {
	type User struct {
		ID   string
		Name string
	}

	// Create a typed store
	store := pocket.NewStore()
	userStore := pocket.NewTypedStore[User](store)

	ctx := context.Background()

	// Store a user
	user := User{ID: "123", Name: "Alice"}
	err := userStore.Set(ctx, "user:123", user)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve with type safety
	retrieved, exists, err := userStore.Get(ctx, "user:123")
	if err != nil {
		log.Fatal(err)
	}

	if exists {
		fmt.Printf("Found user: %+v\n", retrieved)
	}
	// Output: Found user: {ID:123 Name:Alice}
}

// ExampleWithRetry demonstrates retry configuration.
func ExampleWithRetry() {
	attempts := 0

	// Create a node that fails twice before succeeding
	flaky := pocket.NewNode[any, any]("flaky",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			attempts++
			if attempts < 3 {
				return nil, fmt.Errorf("temporary failure %d", attempts)
			}
			return "success", nil
		}),
		pocket.WithRetry(2, 10*time.Millisecond), // Retry up to 2 times
	)

	store := pocket.NewStore()
	graph := pocket.NewGraph(flaky, store)

	result, err := graph.Run(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result after %d attempts: %v\n", attempts, result)
	// Output: Result after 3 attempts: success
}

// Example_lifecycle demonstrates the full Prep/Exec/Post lifecycle.
func Example_lifecycle() {
	// Create a node that uses all three steps
	processor := pocket.NewNode[any, any]("processor",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Prepare: validate and transform input
			data := input.(map[string]int)
			if len(data) == 0 {
				return nil, fmt.Errorf("empty data")
			}
			return data, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Execute: calculate sum
			m := data.(map[string]int)
			sum := 0
			for _, v := range m {
				sum += v
			}
			return sum, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, sum any) (any, string, error) {
			// Post: decide routing based on result
			total := sum.(int)
			if total > 100 {
				return fmt.Sprintf("High total: %d", total), "high", nil
			}
			return fmt.Sprintf("Low total: %d", total), "low", nil
		}),
	)

	store := pocket.NewStore()
	graph := pocket.NewGraph(processor, store)

	result, err := graph.Run(context.Background(), map[string]int{
		"a": 10,
		"b": 20,
		"c": 30,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output: Low total: 60
}
