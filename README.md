# Pocket

A minimalist Go framework for building LLM workflows with composable nodes and built-in concurrency patterns. Inspired by [PocketFlow](https://github.com/The-Pocket/PocketFlow), Pocket embraces Go idioms with small interfaces, functional options, and zero dependencies.

## Philosophy

- **Small interfaces**: Single-purpose interfaces that compose naturally
- **Idiomatic Go**: Follows Go best practices and patterns
- **Zero dependencies**: Core has no external dependencies
- **Built-in concurrency**: First-class support for parallel execution
- **Type-safe**: Leverages generics for compile-time safety
- **Functional options**: Clean, extensible configuration

## Installation

```bash
go get github.com/agentstation/pocket
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/agentstation/pocket"
)

func main() {
    // Create a simple processor using a function
    greet := pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
        name := input.(string)
        return fmt.Sprintf("Hello, %s!", name), nil
    })
    
    // Create a node
    node := pocket.NewNode("greeter", greet)
    
    // Create and run a flow
    store := pocket.NewStore()
    flow := pocket.NewFlow(node, store)
    
    result, err := flow.Run(context.Background(), "World")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(result) // "Hello, World!"
}
```

## Core Concepts

### Small, Composable Interfaces

```go
// Process data
type Processor interface {
    Process(ctx context.Context, input any) (output any, err error)
}

// Manage state
type Stateful interface {
    LoadState(ctx context.Context, store Store) (state any, err error)
    SaveState(ctx context.Context, store Store, state any) error
}

// Route to next node
type Router interface {
    Route(ctx context.Context, result any) (next string, err error)
}
```

### Nodes

Nodes combine processing, state management, and routing:

```go
// Create from a function
processor := pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
    // Process input
    return result, nil
})

// Create a node with options
node := pocket.NewNode("myNode", processor,
    pocket.WithRetry(3, time.Second),
    pocket.WithTimeout(30*time.Second),
)

// Connect nodes
node.Connect("success", successNode)
node.Connect("error", errorNode)
node.Default(defaultNode)
```

### Flows

Flows orchestrate node execution:

```go
// Simple flow
flow := pocket.NewFlow(startNode, store)
result, err := flow.Run(ctx, input)

// Using the builder
flow, err := pocket.NewBuilder(store).
    Add(nodeA).
    Add(nodeB).
    Connect("nodeA", "success", "nodeB").
    Start("nodeA").
    Build()
```

## Concurrency Patterns

### Built-in Patterns

Pocket provides idiomatic Go concurrency patterns:

```go
// Run nodes concurrently
results, err := pocket.RunConcurrent(ctx, nodes, store)

// Pipeline - output feeds next input
result, err := pocket.Pipeline(ctx, nodes, store, input)

// Fan-out - process items in parallel
results, err := pocket.FanOut(ctx, processor, store, items)

// Fan-in - aggregate from multiple sources
fanIn := pocket.NewFanIn(aggregator, source1, source2, source3)
result, err := fanIn.Run(ctx, store)
```

### Batch Processing

Type-safe batch operations with generics:

```go
import "github.com/agentstation/pocket/batch"

// Map-reduce pattern
processor := batch.MapReduce(
    extractItems,    // func(ctx, store) ([]T, error)
    transformItem,   // func(ctx, T) (R, error)  
    aggregateResults,// func(ctx, []R) (any, error)
    batch.WithConcurrency(10),
)

// Process each item
batch.ForEach(extractItems, processItem,
    batch.WithConcurrency(5),
)

// Filter items
filtered := batch.Filter(extractItems, predicate)
```

## Design Patterns

### Agent with Think-Act Loop

```go
// Think node decides actions
think := pocket.NewNode("think", &ThinkAgent{})
think.Router = &ThinkAgent{} // Implements Router interface

// Action nodes execute and loop back
research := pocket.NewNode("research", &ResearchAction{})
research.Router = pocket.RouterFunc(func(ctx, result) (string, error) {
    return "think", nil // Loop back
})

// Connect the loop
think.Connect("research", research)
think.Connect("draft", draft)
think.Connect("complete", complete)
```

### Conditional Routing

```go
router := pocket.NewNode("router", processor)
router.Router = pocket.RouterFunc(func(ctx context.Context, result any) (string, error) {
    value := result.(int)
    switch {
    case value > 100:
        return "large", nil
    case value < 0:
        return "negative", nil
    default:
        return "normal", nil
    }
})
```

## Type Safety

### Generic Store Operations

```go
// Type-safe store wrapper
userStore := pocket.NewTypedStore[User](store)

// Compile-time type checking
user := User{ID: "123", Name: "Alice"}
err := userStore.Set(ctx, "user:123", user)

retrieved, exists, err := userStore.Get(ctx, "user:123")
// retrieved is typed as User, not any
```

### Scoped Stores

```go
// Isolated key namespaces
userScope := pocket.NewScopedStore(store, "user")
adminScope := pocket.NewScopedStore(store, "admin")

// Keys are automatically prefixed
userScope.Set("name", "Alice")  // Stored as "user:name"
adminScope.Set("name", "Bob")   // Stored as "admin:name"
```

## Configuration

### Functional Options

```go
node := pocket.NewNode("processor", processor,
    pocket.WithRetry(3, time.Second),
    pocket.WithTimeout(30*time.Second),
    pocket.WithErrorHandler(func(err error) {
        log.Printf("Node error: %v", err)
    }),
)

flow := pocket.NewFlow(start, store,
    pocket.WithLogger(logger),
    pocket.WithTracer(tracer),
)
```

## Examples

- [Chat Bot](examples/chat/main.go) - Multi-agent chat with routing
- [Autonomous Agent](examples/agent/main.go) - Think-act loop pattern  
- [Parallel Processing](examples/parallel/main.go) - Batch document processing

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# Coverage report
go test -cover ./...
```

## Project Structure

```
pocket/
├── pocket.go          # Main API - interfaces and node implementation
├── flow.go           # Flow orchestration and concurrency patterns
├── store.go          # Store implementations and type-safe wrappers
├── doc.go            # Package documentation
├── batch/            # Generic batch processing
├── internal/         # Internal implementation details
└── examples/         # Example applications
```

## Philosophy Comparison

| Aspect | Traditional Approach | Pocket Approach |
|--------|---------------------|-----------------|
| Interfaces | Large, multi-method | Small, focused |
| Concurrency | External libraries | Built-in patterns |
| Configuration | Struct fields | Functional options |
| Type Safety | Interface{} everywhere | Generics where useful |
| Dependencies | Many external | Zero in core |

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Write tests for your changes
4. Ensure all tests pass with race detector
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing`)
7. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [PocketFlow](https://github.com/The-Pocket/PocketFlow)'s minimalist philosophy
- Built with Go's idioms and best practices in mind
- Designed for the modern LLM application stack