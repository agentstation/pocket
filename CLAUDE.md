# Pocket Framework Documentation

## Overview

Pocket is a Go implementation of PocketFlow's Prep/Exec/Post workflow pattern, enhanced with type safety, built-in concurrency, and idiomatic Go patterns. This document provides comprehensive information about the framework's architecture, design decisions, and implementation details.

## Core Philosophy: Prep/Exec/Post Lifecycle

Every node in Pocket follows a three-phase lifecycle:

1. **Prep Phase**: Data preparation, validation, and state loading
2. **Exec Phase**: Core business logic execution
3. **Post Phase**: Result processing, state updates, and routing decisions

This structured approach provides:
- Clear separation of concerns
- Predictable execution flow
- Easy testing and debugging
- Natural error boundaries

## Architecture

### Node Structure

```go
type Node struct {
    Name       string
    Prep       PrepFunc
    Exec       ExecFunc
    Post       PostFunc
    InputType  reflect.Type  // Optional type validation
    OutputType reflect.Type  // Optional type validation
    successors map[string]*Node
    opts       nodeOptions
}
```

### Lifecycle Functions

```go
type PrepFunc func(ctx context.Context, store Store, input any) (prepResult any, err error)
type ExecFunc func(ctx context.Context, store Store, prepResult any) (execResult any, err error)
type PostFunc func(ctx context.Context, store Store, input, prepResult, execResult any) (output any, next string, err error)
```

Key aspects:
- **Prep** receives the original input and prepares data for execution
- **Exec** receives the prep result and performs the main logic
- **Post** receives all three values (input, prep result, exec result) and decides routing

### Store Interface

```go
type Store interface {
    Get(ctx context.Context, key string) (value any, exists bool)
    Set(ctx context.Context, key string, value any) error
    Delete(ctx context.Context, key string) error
    Scope(prefix string) Store
}
```

The Store is context-aware and supports scoping for isolation between concurrent operations.

## Design Decisions

### 1. Prep/Exec/Post as Primary Pattern

We fully adopted PocketFlow's lifecycle pattern because:
- It naturally models most workflow patterns (think-act, ETL, validation-process-route)
- Provides clear phases for different concerns
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

### 4. Type Safety with Generics

NewNode provides compile-time type checking while maintaining flexibility:
```go
// NewNode with generic options provides type safety out of the box!
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithPrep(func(ctx context.Context, store pocket.Store, in Input) (any, error) {
        // Type-safe prep function - Go infers the type
        return processedInput, nil
    }),
    pocket.WithExec(func(ctx context.Context, store pocket.Store, in Input) (Output, error) {
        // Type-safe exec function - no wrapper needed
        return Output{Result: process(in)}, nil
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

## Implementation Details

### Flow Execution

1. Flow starts at the designated start node
2. For each node:
   - Execute Prep phase (with retry support)
   - Execute Exec phase (with retry support)
   - Execute Post phase (no retry for routing decisions)
   - Post returns the next node name
   - Flow continues to the next node or ends

### Error Handling

- Each phase can be retried independently
- Timeouts apply to the entire lifecycle
- Custom error handlers can be attached to nodes
- Errors include context about which phase failed

### State Management

- Store is thread-safe using sync.RWMutex
- Scoped stores share data but have key prefixes
- TypedStore provides type-safe wrappers
- Store passed through all lifecycle phases

### Type Validation

Optional type validation ensures compatibility:
```go
func ValidateFlow(start *Node) error {
    // Traverses graph checking InputType/OutputType compatibility
    // Returns error if types don't match
}
```

## Advanced Features

### Flow Composition

Flows can be converted to nodes for composition:
```go
subFlow := pocket.NewFlow(startNode, store)
compositeNode := subFlow.AsNode("sub-workflow")
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
- Circuit breaker pattern in `internal/fallback`
- Fallback chains with multiple strategies

### Cleanup Hooks

Lifecycle hooks for resource management:
- `WithOnSuccess`: Runs after successful execution
- `WithOnFailure`: Runs after failed execution  
- `WithOnComplete`: Always runs (even on panic)

### Memory Management

Advanced store implementations:
- `BoundedStore`: Size limits with eviction policies (LRU, LFU, FIFO, TTL)
- `MultiTieredStore`: Hot/cold storage tiers
- `ShardedStore`: Distributed storage across shards

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

### 1. Keep Phases Focused

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
    pocket.WithExec(func(ctx context.Context, store pocket.Store, order Order) (Invoice, error) {
        // Process order and return invoice - types are inferred!
        return createInvoice(order), nil
    }),
)
```

For dynamic typing, be explicit with `[any, any]`:
```go
flexible := pocket.NewNode[any, any]("flexible",
    pocket.WithExec(func(ctx context.Context, store pocket.Store, input any) (any, error) {
        // Handle any input type
        return processAny(input), nil
    }),
)
```

### 4. Handle Errors Appropriately

- Use retries for transient failures
- Set reasonable timeouts
- Log errors with context

### 5. Design for Testability

Each phase can be tested independently:
```go
// Test prep phase
result, err := node.Prep(ctx, mockStore, input)

// Test exec phase
result, err := node.Exec(ctx, mockStore, prepResult)

// Test post phase
output, next, err := node.Post(ctx, mockStore, input, prepResult, execResult)
```

## Migration from Simple Processors

If migrating from a simple processor pattern:

Old pattern:
```go
func Process(ctx context.Context, input any) (any, error) {
    // All logic mixed together
}
```

New pattern:
```go
node := pocket.NewNode[any, any]("processor",
    pocket.WithPrep(func(ctx context.Context, store pocket.Store, input any) (any, error) {
        // Validation and prep
        return input, nil
    }),
    pocket.WithExec(func(ctx context.Context, store pocket.Store, input any) (any, error) {
        // Core logic
        return result, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.Store, input, prep, result any) (any, string, error) {
        // Routing decision
        return result, "done", nil
    }),
)
```

## Performance Considerations

1. **Lifecycle Overhead**: Minimal - three function calls vs one
2. **Type Validation**: Only runs if types are specified
3. **Store Operations**: O(1) with mutex overhead
4. **Concurrency**: Uses sync.Pool where appropriate
5. **Memory**: Efficient reuse of nodes across flows

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
    flow := pocket.NewFlow(node, store)
    result, err := flow.Run(ctx, input)
}
```

### Integration Testing Flows

```go
func TestFlow(t *testing.T) {
    // Build complete flow
    flow, err := pocket.NewBuilder(store).
        Add(node1).
        Add(node2).
        Connect("node1", "success", "node2").
        Start("node1").
        Build()
    
    // Run flow
    result, err := flow.Run(ctx, input)
}
```

## Debugging

1. **Enable Logging**: Use WithLogger option
2. **Add Error Handlers**: Use WithErrorHandler on nodes
3. **Validate Types**: Run ValidateFlow before execution
4. **Check Store State**: Inspect store between phases
5. **Trace Execution**: Use WithTracer for distributed tracing

## Future Enhancements

Potential areas for enhancement:
1. Middleware support for cross-cutting concerns
2. Flow visualization tools
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
| State Management | External | Integrated Store |
| Error Handling | Basic | Retries, timeouts, handlers |
| Language | Python | Go |

## Conclusion

Pocket brings PocketFlow's elegant Prep/Exec/Post pattern to Go while adding:
- Type safety through generics
- Built-in concurrency patterns
- Integrated state management
- Comprehensive error handling
- Idiomatic Go APIs

The framework maintains simplicity while providing power and flexibility for building complex LLM workflows.