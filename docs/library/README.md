# Pocket Go Library Documentation

Welcome to the Pocket Go library documentation. Pocket is a graph execution engine that you can embed in your Go applications to build composable workflows with type safety and powerful concurrency patterns.

## What is the Pocket Library?

The Pocket library provides:
- A graph execution engine for Go applications
- Type-safe workflow building with generics
- Built-in concurrency patterns
- Extensible node architecture
- State management with bounded stores

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/agentstation/pocket"
)

func main() {
    // Create a simple node
    greet := pocket.NewNode[string, string]("greet",
        pocket.WithExec(func(ctx context.Context, name string) (string, error) {
            return fmt.Sprintf("Hello, %s!", name), nil
        }),
    )
    
    // Create a graph and run it
    store := pocket.NewStore()
    graph := pocket.NewGraph(greet, store)
    
    result, _ := graph.Run(context.Background(), "World")
    fmt.Println(result) // "Hello, World!"
}
```

## Documentation

- [**Getting Started**](getting-started.md) - Build your first Go workflow
- [**API Reference**](api-reference.md) - Complete API documentation
- [**Embedding Guide**](embedding.md) - Integrate Pocket into your application
- [**Type Safety**](../guides/TYPE_SAFETY.md) - Leverage Go's type system
- [**State Management**](../guides/STATE_MANAGEMENT.md) - Work with stores
- [**Error Handling**](../guides/ERROR_HANDLING.md) - Build resilient workflows

## Core Concepts

### Graph Execution Engine

The Pocket library implements a graph execution engine where:
- Nodes are the building blocks of workflows
- Each node follows the Prep→Exec→Post lifecycle
- Graphs can be composed and nested
- Type safety is enforced at compile time

### The Node Interface

```go
type Node interface {
    Name() string
    Prep(ctx context.Context, store StoreReader, input any) (any, error)
    Exec(ctx context.Context, prepData any) (any, error)
    Post(ctx context.Context, store StoreWriter, input, prepData, result any) (any, string, error)
    Connect(action string, next Node)
    Successors() map[string]Node
    InputType() reflect.Type
    OutputType() reflect.Type
}
```

## Next Steps

1. [Get started with the library](getting-started.md)
2. [Learn about type safety](../guides/TYPE_SAFETY.md)
3. [Explore patterns](../patterns/)
4. [View examples](../../examples/)