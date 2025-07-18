package pocket_test

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
)

const (
	successResult = "success"
)

// ExampleProcessorFunc demonstrates using a simple function as a processor.
func ExampleProcessorFunc() {
	// Create a processor from a function
	uppercase := pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		text, ok := input.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", input)
		}
		return strings.ToUpper(text), nil
	})

	// Use it in a node
	node := pocket.NewNode("uppercase", uppercase)
	store := pocket.NewStore()
	flow := pocket.NewFlow(node, store)

	result, err := flow.Run(context.Background(), "hello world")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output: HELLO WORLD
}

// ExampleBuilder demonstrates the fluent builder API.
func ExampleBuilder() {
	store := pocket.NewStore()

	// Define processors
	validate := pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		email := input.(string)
		if !strings.Contains(email, "@") {
			return nil, fmt.Errorf("invalid email")
		}
		return email, nil
	})

	normalize := pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		email := input.(string)
		return strings.ToLower(strings.TrimSpace(email)), nil
	})

	// Build the flow
	flow, err := pocket.NewBuilder(store).
		Add(pocket.NewNode("validate", validate)).
		Add(pocket.NewNode("normalize", normalize)).
		Connect("validate", "default", "normalize").
		Start("validate").
		Build()

	if err != nil {
		log.Fatal(err)
	}

	result, err := flow.Run(context.Background(), "  USER@EXAMPLE.COM  ")
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
	router := pocket.NewNode("router",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return input, nil
		}),
	)

	// Set up routing logic
	router.Router = pocket.RouterFunc(func(ctx context.Context, result any) (string, error) {
		value := result.(int)
		if value > 100 {
			return "large", nil
		}
		return "small", nil
	})

	// Handler nodes
	largeHandler := pocket.NewNode("large",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Large number: %v", input), nil
		}),
	)

	smallHandler := pocket.NewNode("small",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Small number: %v", input), nil
		}),
	)

	// Connect nodes
	router.Connect("large", largeHandler)
	router.Connect("small", smallHandler)

	// Run with different inputs
	flow := pocket.NewFlow(router, store)

	result1, _ := flow.Run(context.Background(), 50)
	result2, _ := flow.Run(context.Background(), 150)

	fmt.Println(result1)
	fmt.Println(result2)
	// Output:
	// Small number: 50
	// Large number: 150
}

// ExampleFanOut demonstrates parallel processing of items.
func ExampleFanOut() {
	// Create a processor that simulates work
	processor := pocket.NewNode("process",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
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
	flaky := pocket.NewNode("flaky",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			attempts++
			if attempts < 3 {
				return nil, fmt.Errorf("temporary failure %d", attempts)
			}
			return successResult, nil
		}),
		pocket.WithRetry(3, 0), // 3 retries, no delay for example
	)

	store := pocket.NewStore()
	flow := pocket.NewFlow(flaky, store)

	result, err := flow.Run(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result after %d attempts: %v\n", attempts, result)
	// Output: Result after 3 attempts: success
}
