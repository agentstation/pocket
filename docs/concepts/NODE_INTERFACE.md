# Node Interface Design

## Overview

In Pocket, `Node` is an interface, not a struct. This fundamental design decision enables powerful composition patterns and extensibility while maintaining a simple API.

## The Node Interface

```go
type Node interface {
    // Identity
    Name() string
    
    // Lifecycle methods (Prep/Exec/Post pattern)
    Prep(ctx context.Context, store StoreReader, input any) (prepResult any, err error)
    Exec(ctx context.Context, prepResult any) (execResult any, err error)
    Post(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error)
    
    // Graph construction
    Connect(action string, next Node) Node
    Successors() map[string]Node
    
    // Type information
    InputType() reflect.Type
    OutputType() reflect.Type
}
```

## Why Interface Over Struct?

### 1. Polymorphism

Different implementations can be used interchangeably:

```go
var processor Node

// Can be a simple node
processor = pocket.NewNode[Input, Output]("simple", ...)

// Can be a graph
processor = pocket.NewGraph(startNode, store)

// Can be a custom implementation
processor = &CustomProcessor{...}

// All work the same way
result, err := processor.Exec(ctx, input)
```

### 2. Composition Over Inheritance

Interfaces enable clean composition without inheritance hierarchies:

```go
// Compose behaviors through embedding
type LoggedNode struct {
    Node
    logger Logger
}

func (n *LoggedNode) Exec(ctx context.Context, input any) (any, error) {
    n.logger.Info("Executing node", "name", n.Name())
    result, err := n.Node.Exec(ctx, input)
    n.logger.Info("Execution complete", "error", err)
    return result, err
}
```

### 3. Natural Graph Composition

Since `Graph` implements `Node`, graphs can be nested naturally:

```go
// Create a sub-workflow
subWorkflow := pocket.NewGraph(validationStart, store)

// Use it directly as a node in a larger workflow
mainWorkflow := pocket.NewNode[any, any]("main",
    pocket.WithExec(mainLogic),
)
mainWorkflow.Connect("validate", subWorkflow)  // Direct connection!
```

## Built-in Implementations

### 1. Standard Node (node struct)

The default implementation created by `pocket.NewNode()`:

```go
// Internal implementation (not exposed)
type node struct {
    name       string
    prep       PrepFunc
    exec       ExecFunc
    post       PostFunc
    successors map[string]Node
    // ... other fields
}

// Created via public API
myNode := pocket.NewNode[In, Out]("processor",
    pocket.WithExec(processFunc),
)
```

### 2. Graph as Node

Graphs implement the Node interface:

```go
// Graph implements Node
type Graph struct {
    start Node
    store Store
    // ... other fields
}

func (g *Graph) Name() string { return g.start.Name() }
func (g *Graph) Exec(ctx context.Context, input any) (any, error) {
    // Executes the entire graph
    return g.Run(ctx, input)
}
```

This enables powerful composition:

```go
// Build complex nested workflows
authGraph := buildAuthWorkflow()
processGraph := buildProcessingWorkflow()
notifyGraph := buildNotificationWorkflow()

// Compose them
mainNode := pocket.NewNode[Request, Response]("main",
    pocket.WithPost(func(ctx context.Context, store StoreWriter, req Request, prep, exec any) (Response, string, error) {
        if !req.Authenticated {
            return Response{}, "auth", nil
        }
        return Response{}, "process", nil
    }),
)

mainNode.Connect("auth", authGraph)      // Graph as node
mainNode.Connect("process", processGraph) // Graph as node
processGraph.Connect("notify", notifyGraph)
```

## Creating Custom Nodes

### Basic Custom Node

```go
type RateLimitedNode struct {
    name    string
    inner   Node
    limiter *rate.Limiter
}

func (n *RateLimitedNode) Name() string {
    return n.name
}

func (n *RateLimitedNode) Exec(ctx context.Context, input any) (any, error) {
    // Wait for rate limit
    if err := n.limiter.Wait(ctx); err != nil {
        return nil, err
    }
    
    // Execute inner node
    return n.inner.Exec(ctx, input)
}

func (n *RateLimitedNode) Prep(ctx context.Context, store StoreReader, input any) (any, error) {
    return n.inner.Prep(ctx, store, input)
}

func (n *RateLimitedNode) Post(ctx context.Context, store StoreWriter, input, prep, exec any) (any, string, error) {
    return n.inner.Post(ctx, store, input, prep, exec)
}

func (n *RateLimitedNode) Connect(action string, next Node) Node {
    n.inner.Connect(action, next)
    return n
}

func (n *RateLimitedNode) Successors() map[string]Node {
    return n.inner.Successors()
}

func (n *RateLimitedNode) InputType() reflect.Type {
    return n.inner.InputType()
}

func (n *RateLimitedNode) OutputType() reflect.Type {
    return n.inner.OutputType()
}
```

### Advanced Custom Node

```go
// A node that can dynamically choose between multiple strategies
type StrategyNode struct {
    name       string
    strategies map[string]Node
    selector   func(context.Context, any) string
    successors map[string]Node
}

func (n *StrategyNode) Exec(ctx context.Context, input any) (any, error) {
    // Select strategy based on input
    strategyName := n.selector(ctx, input)
    
    strategy, exists := n.strategies[strategyName]
    if !exists {
        return nil, fmt.Errorf("unknown strategy: %s", strategyName)
    }
    
    return strategy.Exec(ctx, input)
}

// Usage
strategyNode := &StrategyNode{
    name: "adaptive-processor",
    strategies: map[string]Node{
        "fast":     fastProcessor,
        "accurate": accurateProcessor,
        "balanced": balancedProcessor,
    },
    selector: func(ctx context.Context, input any) string {
        // Choose based on context or input
        if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < time.Second {
            return "fast"
        }
        return "accurate"
    },
}
```

## Interface Benefits in Practice

### 1. Testing

Interfaces make testing trivial:

```go
type MockNode struct {
    mock.Mock
}

func (m *MockNode) Exec(ctx context.Context, input any) (any, error) {
    args := m.Called(ctx, input)
    return args.Get(0), args.Error(1)
}

// In tests
mockNode := new(MockNode)
mockNode.On("Exec", mock.Anything, testInput).Return(testOutput, nil)

// Use mock in place of real node
graph := pocket.NewGraph(mockNode, store)
```

### 2. Middleware Pattern

Wrap nodes with cross-cutting concerns:

```go
func WithMetrics(inner Node, metrics MetricsCollector) Node {
    return &MetricsNode{
        inner:   inner,
        metrics: metrics,
    }
}

func WithLogging(inner Node, logger Logger) Node {
    return &LoggedNode{
        inner:  inner,
        logger: logger,
    }
}

// Chain middleware
node := pocket.NewNode[In, Out]("processor", opts...)
node = WithLogging(node, logger)
node = WithMetrics(node, metrics)
node = WithRetry(node, retryPolicy)
```

### 3. Dynamic Behavior

Change behavior at runtime:

```go
type ToggleNode struct {
    primary   Node
    secondary Node
    useSecondary func() bool
}

func (n *ToggleNode) Exec(ctx context.Context, input any) (any, error) {
    if n.useSecondary() {
        return n.secondary.Exec(ctx, input)
    }
    return n.primary.Exec(ctx, input)
}

// Usage - feature flags, A/B testing, etc.
toggle := &ToggleNode{
    primary:   stableImplementation,
    secondary: experimentalImplementation,
    useSecondary: func() bool {
        return featureFlags.IsEnabled("use-experimental")
    },
}
```

## Type System Integration

The interface design works seamlessly with Go's type system:

### Generic Constraints

```go
// Type-safe node creation
func ProcessorNode[In any, Out any](name string, fn func(In) Out) Node {
    return pocket.NewNode[In, Out](name,
        pocket.WithExec(func(ctx context.Context, input In) (Out, error) {
            return fn(input), nil
        }),
    )
}

// Usage with type inference
upperNode := ProcessorNode("upper", strings.ToUpper)
```

### Type Validation

```go
// ValidateGraph works with any Node implementation
func ValidateGraph(start Node) error {
    visited := make(map[string]bool)
    return validateNode(start, visited)
}

func validateNode(node Node, visited map[string]bool) error {
    if visited[node.Name()] {
        return nil
    }
    visited[node.Name()] = true
    
    outputType := node.OutputType()
    
    for action, next := range node.Successors() {
        inputType := next.InputType()
        
        if !isTypeCompatible(outputType, inputType) {
            return fmt.Errorf("type mismatch: %s outputs %v but %s expects %v",
                node.Name(), outputType, next.Name(), inputType)
        }
        
        if err := validateNode(next, visited); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Design Patterns

### 1. Decorator Pattern

```go
type CachedNode struct {
    Node
    cache Cache
}

func (n *CachedNode) Exec(ctx context.Context, input any) (any, error) {
    key := fmt.Sprintf("%s:%v", n.Name(), input)
    
    if cached, exists := n.cache.Get(key); exists {
        return cached, nil
    }
    
    result, err := n.Node.Exec(ctx, input)
    if err == nil {
        n.cache.Set(key, result)
    }
    
    return result, err
}
```

### 2. Adapter Pattern

```go
// Adapt external services to Node interface
type HTTPServiceNode struct {
    name     string
    endpoint string
    client   *http.Client
}

func (n *HTTPServiceNode) Exec(ctx context.Context, input any) (any, error) {
    data, _ := json.Marshal(input)
    resp, err := n.client.Post(n.endpoint, "application/json", bytes.NewReader(data))
    // ... handle response
}
```

### 3. Composite Pattern

```go
// Already built-in! Graph implements Node
subGraph := pocket.NewGraph(startNode, store)
mainGraph.Connect("process", subGraph)
```

## Best Practices

### 1. Keep Interfaces Small

The Node interface is intentionally focused. Avoid adding methods that aren't universally needed.

### 2. Use Composition

Instead of creating complex nodes, compose simple ones:

```go
// Instead of one complex node
complexNode := NewComplexNode(...)

// Compose simple nodes
validate := pocket.NewNode[In, Valid]("validate", ...)
transform := pocket.NewNode[Valid, Out]("transform", ...)
validate.Connect("valid", transform)
```

### 3. Document Custom Implementations

When creating custom nodes, clearly document:
- What the node does
- Expected input/output types
- Any side effects
- Thread safety guarantees

### 4. Leverage Type Information

Implement InputType() and OutputType() for better validation:

```go
func (n *MyNode) InputType() reflect.Type {
    return reflect.TypeOf((*MyInput)(nil)).Elem()
}

func (n *MyNode) OutputType() reflect.Type {
    return reflect.TypeOf((*MyOutput)(nil)).Elem()
}
```

## Summary

The Node interface design provides:

1. **Flexibility**: Multiple implementations for different needs
2. **Composability**: Graphs as nodes enables hierarchical workflows
3. **Extensibility**: Easy to add new node types
4. **Testability**: Simple to mock and test
5. **Type Safety**: Works with Go's type system

This interface-based architecture is key to Pocket's power - it enables complex workflows while keeping the API surface minimal and intuitive.