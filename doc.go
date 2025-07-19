/*
Package pocket provides a minimalist framework for building LLM workflows
using composable nodes in a directed graph structure.

Key features:
  - Small, composable interfaces
  - Type-safe operations with generics
  - Built-in concurrency patterns
  - Functional options for configuration
  - Zero external dependencies in core

Basic usage:

	// Create a simple processor
	greet := pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		name := input.(string)
		return fmt.Sprintf("Hello, %s!", name), nil
	})

	// Create a node
	node := pocket.NewNode[any, any]("greeter", greet)

	// Create and run a flow
	store := pocket.NewStore()
	flow := pocket.NewFlow(node, store)
	result, err := flow.Run(context.Background(), "World")

Building complex flows:

	// Use the builder API
	builder := pocket.NewBuilder(store).
		Add(pocket.NewNode[any, any]("fetch", fetchData)).
		Add(pocket.NewNode[any, any]("process", processData)).
		Add(pocket.NewNode[any, any]("save", saveData)).
		Connect("fetch", "success", "process").
		Connect("process", "success", "save").
		Start("fetch")

	flow, err := builder.Build()

Concurrent patterns:

	// Fan-out processing
	results, err := pocket.FanOut(ctx, processNode, store, items)

	// Pipeline
	result, err := pocket.Pipeline(ctx, nodes, store, input)

	// Concurrent execution
	results, err := pocket.RunConcurrent(ctx, nodes, store)

Type-safe operations:

	// Create a typed store
	userStore := pocket.NewTypedStore[User](store)

	// Type-safe get/set
	user, exists, err := userStore.Get(ctx, "user:123")
	err = userStore.Set(ctx, "user:123", newUser)
*/
package pocket
