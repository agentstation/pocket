# Pocket Framework Documentation

## Overview

Pocket is a Go implementation of PocketFlow's Prep/Exec/Post workflow pattern, enhanced with type safety, built-in concurrency, and idiomatic Go patterns. This document provides comprehensive information about the framework's architecture, design decisions, and implementation details.

## Core Philosophy: Prep/Exec/Post Lifecycle

Every node in Pocket follows a three-step lifecycle:

1. **Prep Step**: Data preparation, validation, and state loading
2. **Exec Step**: Core business logic execution
3. **Post Step**: Result processing, state updates, and routing decisions

This structured approach provides:
- Clear separation of concerns
- Predictable execution flow
- Easy testing and debugging
- Natural error boundaries

## Architecture

### Node as Interface

Node is now an interface, not a struct. This fundamental change enables powerful composition patterns:

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

The concrete implementation is internal:

```go
type node struct {
    name       string
    prep       PrepFunc
    exec       ExecFunc
    post       PostFunc
    inputType  reflect.Type
    outputType reflect.Type
    successors map[string]Node  // Now stores Node interface, not *node
    opts       nodeOptions
}
```

Key benefits:
- **Graph implements Node**: Enables natural composition
- **Custom implementations**: Users can create their own Node types
- **Interface-based connections**: More flexible graph structures
- **Backward compatibility**: `pocket.NewNode()` still works as before

### Lifecycle Functions with Read/Write Separation

```go
type PrepFunc func(ctx context.Context, store StoreReader, input any) (prepResult any, err error)
type ExecFunc func(ctx context.Context, prepResult any) (execResult any, err error)
type PostFunc func(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error)
```

Key aspects:
- **Prep** receives the original input and a read-only store (StoreReader)
- **Exec** receives only the prep result - no store access for pure functions
- **Post** receives all values and a read-write store (StoreWriter) for state mutations

This enforces the read/write separation at the type level.

### Store Interfaces

The store now uses separate interfaces for read and write operations:

```go
type StoreReader interface {
    Get(ctx context.Context, key string) (value any, exists bool)
}

type StoreWriter interface {
    StoreReader
    Set(ctx context.Context, key string, value any) error
    Delete(ctx context.Context, key string) error
}

type Store interface {
    StoreWriter
    Scope(prefix string) Store
}
```

The Store implementation now has built-in bounded functionality:

```go
// Create a bounded store with LRU eviction and TTL
store := pocket.NewStore(
    pocket.WithMaxEntries(10000),
    pocket.WithTTL(30 * time.Minute),
    pocket.WithEvictionCallback(func(key string, value any) {
        log.Printf("Evicted: %s", key)
    }),
)
```

Features:
- **LRU eviction**: When max entries exceeded
- **TTL support**: Automatic expiration
- **Context-aware**: All operations use context
- **Thread-safe**: Safe for concurrent use
- **Scoping**: Create isolated key namespaces

## Design Decisions

### 1. Prep/Exec/Post as Primary Pattern

We fully adopted PocketFlow's lifecycle pattern because:
- It naturally models most workflow patterns (think-act, ETL, validation-process-route)
- Provides clear steps for different concerns
- Enables better optimization and caching strategies
- Makes workflows more testable

### 2. Context-First Store

All Store operations require context to:
- Support cancellation and timeouts
- Enable distributed tracing
- Allow request-scoped values
- Follow Go best practices

### 3. Functional Options Pattern for Nodes

Nodes are created using functional options for clean, composable configuration:
```go
node := pocket.NewNode[any, any]("processor",
    pocket.WithPrep(prepFunc),
    pocket.WithExec(execFunc),
    pocket.WithPost(postFunc),
    pocket.WithRetry(3, time.Second),
)
```

Global defaults can be set for all nodes:
```go
pocket.SetDefaults(
    pocket.WithDefaultPrep(globalPrepFunc),
    pocket.WithDefaultExec(globalExecFunc),
    pocket.WithDefaultPost(globalPostFunc),
)
```

### 4. Type Safety with Generics

NewNode provides compile-time type checking while maintaining flexibility:
```go
// NewNode with generic options provides type safety out of the box!
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, in Input) (any, error) {
        // Type-safe prep function - store is read-only
        data, _ := store.Get(ctx, "config")
        return processedInput, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (Output, error) {
        // Type-safe exec function - pure, no store access
        return Output{Result: process(prepData)}, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, in Input, prep any, result Output) (Output, string, error) {
        // Post has full read/write access
        store.Set(ctx, "lastResult", result)
        return result, "next", nil
    }),
    pocket.WithRetry(3, time.Second),  // Regular options work!
    pocket.WithTimeout(5*time.Second),
)
```

All lifecycle options (`WithPrep`, `WithExec`, `WithPost`) are now generic by default, providing compile-time type safety when used with typed nodes. For untyped nodes, use `NewNode[any, any]` to make the dynamic typing explicit.

### 5. Built-in Concurrency Patterns

Instead of external libraries, we provide idiomatic Go patterns:
- `RunConcurrent`: Execute multiple nodes in parallel
- `Pipeline`: Sequential processing with output chaining
- `FanOut`: Process items concurrently
- `FanIn`: Aggregate from multiple sources

### 6. Graph as Node

Graphs now implement the Node interface, enabling powerful composition:

```go
// Graph implements Node
type Graph struct {
    start Node
    store Store
    // ... other fields
}

func (g *Graph) Name() string { return g.name }
func (g *Graph) Prep(ctx context.Context, store StoreReader, input any) (any, error) {
    // Delegates to start node
}
func (g *Graph) Exec(ctx context.Context, prepData any) (any, error) {
    // Runs the graph execution
}
func (g *Graph) Post(ctx context.Context, store StoreWriter, input, prepData, result any) (any, string, error) {
    // Returns graph result
}
```

This means:
- Graphs can be used anywhere a Node is expected
- Natural composition without wrapper functions
- `AsNode()` method retained for backward compatibility

## Implementation Details

### Graph Execution

1. Graph starts at the designated start node
2. For each node:
   - Execute Prep step (with retry support)
   - Execute Exec step (with retry support)
   - Execute Post step (no retry for routing decisions)
   - Post returns the next node name
   - Graph continues to the next node or ends

### Error Handling

- Each step can be retried independently
- Timeouts apply to the entire lifecycle
- Custom error handlers can be attached to nodes
- Errors include context about which step failed

### State Management

- Store is thread-safe using sync.RWMutex
- Scoped stores share data but have key prefixes
- TypedStore provides type-safe wrappers
- Store passed through all lifecycle steps

### Type Validation

Optional type validation ensures compatibility:
```go
func ValidateGraph(start *Node) error {
    // Traverses graph checking InputType/OutputType compatibility
    // Returns error if types don't match
}
```

## Advanced Features

### Graph Composition

Since graphs implement the Node interface, composition is natural:
```go
// Create a sub-graph
subGraph := pocket.NewGraph(startNode, store)

// Use it directly as a node - no conversion needed!
mainNode.Connect("action", subGraph)

// AsNode() still works for backward compatibility
compositeNode := subGraph.AsNode("sub-workflow")  // Optional, returns the graph itself
```

### YAML Support

Better token efficiency for LLM interactions:
```go
yamlNode := pocket.NewNode[any, any]("output",
    pocket.WithExec(func(ctx context.Context, store pocket.Store, input any) (any, error) {
        // Convert result to YAML format
        return convertToYAML(input), nil
    }),
)
```

### Fallback Mechanisms

- Node-level fallbacks with `WithFallback`
- Circuit breaker pattern in `fallback`
- Fallback chains with multiple strategies

### Cleanup Hooks

Lifecycle hooks for resource management:
- `WithOnSuccess`: Runs after successful execution
- `WithOnFailure`: Runs after failed execution  
- `WithOnComplete`: Always runs (even on panic)

### Memory Management

The core Store now includes bounded functionality:
- LRU eviction when max entries exceeded
- TTL-based expiration
- Eviction callbacks
- Thread-safe with scoping support

## Usage Patterns

### Agent Pattern (Think-Act Loop)

```go
think := pocket.NewNode[any, any]("think",
    pocket.WithPrep(loadTaskState),
    pocket.WithExec(analyzeAndDecide),
    pocket.WithPost(routeToAction),
)

// Actions loop back to think
action.Connect("think", think)
```

### ETL Pattern

```go
extract := pocket.NewNode[any, any]("extract",
    pocket.WithPrep(validateSource),
    pocket.WithExec(extractData),
    pocket.WithPost(routeByDataType),
)

transform := pocket.NewNode[any, any]("transform",
    pocket.WithPrep(validateData),
    pocket.WithExec(transformData),
    pocket.WithPost(routeToLoad),
)

load := pocket.NewNode[any, any]("load",
    pocket.WithPrep(prepareDestination),
    pocket.WithExec(loadData),
    pocket.WithPost(finalizeAndRoute),
)
```

### Saga Pattern (with Compensation)

```go
action := pocket.NewNode[any, any]("action",
    pocket.WithExec(performAction),
    pocket.WithPost(func(ctx context.Context, store pocket.Store, input, prep, result any) (any, string, error) {
        if isSuccess(result) {
            return result, "next", nil
        }
        return result, "compensate", nil
    }),
)

compensate := pocket.NewNode[any, any]("compensate",
    pocket.WithPrep(loadSagaState),
    pocket.WithExec(rollbackAction),
    pocket.WithPost(routeAfterCompensation),
)
```

## Best Practices

### 1. Keep Steps Focused

- **Prep**: Only validation and data preparation
- **Exec**: Only core business logic
- **Post**: Only routing and state updates

### 2. Use Scoped Stores

For concurrent operations, use scoped stores:
```go
userStore := store.Scope("user")
orderStore := store.Scope("order")
```

### 3. Leverage Type Safety

When types are known, use typed nodes:
```go
processor := pocket.NewNode[Order, Invoice]("processor",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, order Order) (any, error) {
        // Read-only access to store
        config, _ := store.Get(ctx, "invoiceConfig")
        return map[string]any{"order": order, "config": config}, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (Invoice, error) {
        // Pure function - process order and return invoice
        data := prepData.(map[string]any)
        return createInvoice(data["order"].(Order), data["config"]), nil
    }),
)
```

For dynamic typing, be explicit with `[any, any]`:
```go
flexible := pocket.NewNode[any, any]("flexible",
    pocket.WithExec(func(ctx context.Context, input any) (any, error) {
        // Handle any input type - exec has no store access
        return processAny(input), nil
    }),
)
```

### 4. Handle Errors Appropriately

- Use retries for transient failures
- Set reasonable timeouts
- Log errors with context

### 5. Design for Testability

Each step can be tested independently:
```go
// Test prep step
result, err := node.Prep(ctx, mockStore, input)

// Test exec step
result, err := node.Exec(ctx, mockStore, prepResult)

// Test post step
output, next, err := node.Post(ctx, mockStore, input, prepResult, execResult)
```

## Migration Guide

### No Migration Needed!

The interface-based architecture maintains full backward compatibility:

```go
// This code still works exactly as before:
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
        return processInput(input), nil
    }),
)

// Graphs still work the same:
graph := pocket.NewGraph(node, store)
result, err := graph.Run(ctx, input)

// AsNode() still works but is now optional:
subGraph.AsNode("name")  // Returns the graph itself since it implements Node
```

### What's New

1. **Direct graph composition** - no AsNode() needed:
```go
mainNode.Connect("success", subGraph)  // Works directly!
```

2. **Built-in store bounds**:
```go
store := pocket.NewStore(
    pocket.WithMaxEntries(1000),
    pocket.WithTTL(5 * time.Minute),
)
```

3. **Interface-based extensibility** - create custom Node implementations:
```go
type CustomNode struct {
    // your fields
}

func (c *CustomNode) Name() string { return c.name }
func (c *CustomNode) Prep(ctx context.Context, store StoreReader, input any) (any, error) {
    // custom prep logic
}
// ... implement other methods
```

## Performance Considerations

1. **Lifecycle Overhead**: Minimal - three function calls vs one
2. **Type Validation**: Only runs if types are specified
3. **Store Operations**: O(1) with mutex overhead
4. **Concurrency**: Uses sync.Pool where appropriate
5. **Memory**: Efficient reuse of nodes across graphs

## Testing

### Unit Testing Nodes

```go
func TestNode(t *testing.T) {
    store := pocket.NewStore()
    ctx := context.Background()
    
    node := pocket.NewNode[any, any]("test",
        pocket.WithPrep(prepFunc),
        pocket.WithExec(execFunc),
        pocket.WithPost(postFunc),
    )
    
    // Test lifecycle
    graph := pocket.NewGraph(node, store)
    result, err := graph.Run(ctx, input)
}
```

### Integration Testing Graphs

```go
func TestGraph(t *testing.T) {
    // Build complete graph
    graph, err := pocket.NewBuilder(store).
        Add(node1).
        Add(node2).
        Connect("node1", "success", "node2").
        Start("node1").
        Build()
    
    // Run graph
    result, err := graph.Run(ctx, input)
}
```

## Debugging

1. **Enable Logging**: Use WithLogger option
2. **Add Error Handlers**: Use WithErrorHandler on nodes
3. **Validate Types**: Run ValidateFlow before execution
4. **Check Store State**: Inspect store between steps
5. **Trace Execution**: Use WithTracer for distributed tracing

## Implementation Benefits

The interface-based architecture provides:

1. **Natural Composition**: Graphs are nodes, enabling nested workflows
2. **Type Safety**: Interface contracts ensure correctness
3. **Extensibility**: Custom node implementations possible
4. **Zero Migration**: Existing code continues to work
5. **Clean Separation**: Read/write store interfaces enforce proper access

## Future Enhancements

Potential areas for enhancement:
1. Middleware support for cross-cutting concerns
2. Graph visualization tools
3. Persistent store implementations
4. Distributed execution support
5. Advanced routing strategies

## Contributing

When contributing:
1. Maintain the Prep/Exec/Post pattern
2. Keep the API simple and idiomatic
3. Add tests for new features
4. Update documentation
5. Follow Go best practices

## Comparison with PocketFlow

| Feature | PocketFlow | Pocket (Go) |
|---------|------------|-------------|
| Core Pattern | Prep/Exec/Post | Prep/Exec/Post |
| Type Safety | No | Optional with generics |
| Concurrency | External | Built-in patterns |
| State Management | External | Integrated Store with bounds |
| Error Handling | Basic | Retries, timeouts, handlers |
| Language | Python | Go |
| Architecture | Class-based | Interface-based |
| Composition | Manual | Natural (Graph implements Node) |
| Store Access | Unrestricted | Read/Write separation |

## Conclusion

Pocket brings PocketFlow's elegant Prep/Exec/Post pattern to Go while adding:
- Type safety through generics
- Built-in concurrency patterns
- Integrated state management
- Comprehensive error handling
- Idiomatic Go APIs

The framework maintains simplicity while providing power and flexibility for building complex LLM workflows.