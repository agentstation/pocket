# Custom Nodes

## Overview

While Pocket provides a comprehensive `NewNode` function for creating nodes, you can implement custom nodes by implementing the `Node` interface. This enables specialized behavior, integration with external systems, and advanced patterns.

## The Node Interface

To create a custom node, implement all methods of the Node interface:

```go
type Node interface {
    // Identity
    Name() string
    
    // Lifecycle methods
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

## Basic Custom Node

### Simple Implementation

```go
type UppercaseNode struct {
    name       string
    successors map[string]pocket.Node
}

func NewUppercaseNode(name string) pocket.Node {
    return &UppercaseNode{
        name:       name,
        successors: make(map[string]pocket.Node),
    }
}

func (n *UppercaseNode) Name() string {
    return n.name
}

func (n *UppercaseNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
    // Validate input is string
    str, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf("expected string, got %T", input)
    }
    return str, nil
}

func (n *UppercaseNode) Exec(ctx context.Context, prepResult any) (any, error) {
    str := prepResult.(string)
    return strings.ToUpper(str), nil
}

func (n *UppercaseNode) Post(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
    result := exec.(string)
    
    // Store result
    store.Set(ctx, "uppercase:result", result)
    
    // Route based on result
    if len(result) > 10 {
        return result, "long", nil
    }
    return result, "short", nil
}

func (n *UppercaseNode) Connect(action string, next pocket.Node) pocket.Node {
    n.successors[action] = next
    return n
}

func (n *UppercaseNode) Successors() map[string]pocket.Node {
    return n.successors
}

func (n *UppercaseNode) InputType() reflect.Type {
    return reflect.TypeOf("")
}

func (n *UppercaseNode) OutputType() reflect.Type {
    return reflect.TypeOf("")
}
```

## Advanced Custom Nodes

### Stateful Node

A node that maintains internal state:

```go
type CounterNode struct {
    name       string
    count      int64
    threshold  int64
    successors map[string]pocket.Node
    mu         sync.Mutex
}

func NewCounterNode(name string, threshold int64) pocket.Node {
    return &CounterNode{
        name:       name,
        threshold:  threshold,
        successors: make(map[string]pocket.Node),
    }
}

func (n *CounterNode) Exec(ctx context.Context, input any) (any, error) {
    n.mu.Lock()
    defer n.mu.Unlock()
    
    n.count++
    
    return map[string]int64{
        "count":     n.count,
        "threshold": n.threshold,
    }, nil
}

func (n *CounterNode) Post(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
    result := exec.(map[string]int64)
    
    if result["count"] >= result["threshold"] {
        // Reset counter
        n.mu.Lock()
        n.count = 0
        n.mu.Unlock()
        
        return result, "threshold-reached", nil
    }
    
    return result, "counting", nil
}

// ... implement other methods
```

### External Service Node

Integrate with external services:

```go
type HTTPServiceNode struct {
    name       string
    client     *http.Client
    endpoint   string
    method     string
    successors map[string]pocket.Node
}

func NewHTTPServiceNode(name, endpoint, method string) pocket.Node {
    return &HTTPServiceNode{
        name:     name,
        client:   &http.Client{Timeout: 30 * time.Second},
        endpoint: endpoint,
        method:   method,
        successors: make(map[string]pocket.Node),
    }
}

func (n *HTTPServiceNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
    // Prepare request
    data, err := json.Marshal(input)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal input: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, n.method, n.endpoint, bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    
    // Add headers from store
    if headers, exists := store.Get(ctx, "http:headers"); exists {
        for k, v := range headers.(map[string]string) {
            req.Header.Set(k, v)
        }
    }
    
    return req, nil
}

func (n *HTTPServiceNode) Exec(ctx context.Context, prepResult any) (any, error) {
    req := prepResult.(*http.Request)
    
    resp, err := n.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    return HTTPResponse{
        StatusCode: resp.StatusCode,
        Headers:    resp.Header,
        Body:       body,
    }, nil
}

func (n *HTTPServiceNode) Post(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
    response := exec.(HTTPResponse)
    
    // Store response
    store.Set(ctx, "http:lastResponse", response)
    
    // Route based on status code
    switch {
    case response.StatusCode >= 200 && response.StatusCode < 300:
        return response, "success", nil
    case response.StatusCode >= 400 && response.StatusCode < 500:
        return response, "client-error", nil
    case response.StatusCode >= 500:
        return response, "server-error", nil
    default:
        return response, "unknown", nil
    }
}
```

### Composite Node

A node that wraps multiple nodes:

```go
type CompositeNode struct {
    name       string
    nodes      []pocket.Node
    strategy   CompositeStrategy
    successors map[string]pocket.Node
}

type CompositeStrategy int

const (
    Sequential CompositeStrategy = iota
    Parallel
    FirstSuccess
)

func NewCompositeNode(name string, strategy CompositeStrategy, nodes ...pocket.Node) pocket.Node {
    return &CompositeNode{
        name:       name,
        nodes:      nodes,
        strategy:   strategy,
        successors: make(map[string]pocket.Node),
    }
}

func (n *CompositeNode) Exec(ctx context.Context, input any) (any, error) {
    switch n.strategy {
    case Sequential:
        return n.execSequential(ctx, input)
    case Parallel:
        return n.execParallel(ctx, input)
    case FirstSuccess:
        return n.execFirstSuccess(ctx, input)
    default:
        return nil, errors.New("unknown strategy")
    }
}

func (n *CompositeNode) execSequential(ctx context.Context, input any) (any, error) {
    current := input
    
    for _, node := range n.nodes {
        result, err := node.Exec(ctx, current)
        if err != nil {
            return nil, fmt.Errorf("%s failed: %w", node.Name(), err)
        }
        current = result
    }
    
    return current, nil
}

func (n *CompositeNode) execParallel(ctx context.Context, input any) (any, error) {
    results := make([]any, len(n.nodes))
    errors := make([]error, len(n.nodes))
    
    var wg sync.WaitGroup
    for i, node := range n.nodes {
        wg.Add(1)
        go func(idx int, nd pocket.Node) {
            defer wg.Done()
            results[idx], errors[idx] = nd.Exec(ctx, input)
        }(i, node)
    }
    
    wg.Wait()
    
    // Check for errors
    for i, err := range errors {
        if err != nil {
            return nil, fmt.Errorf("node %s failed: %w", n.nodes[i].Name(), err)
        }
    }
    
    return results, nil
}
```

## Specialized Node Types

### Stream Processing Node

Process data streams:

```go
type StreamNode struct {
    name       string
    bufferSize int
    processor  func([]any) (any, error)
    buffer     []any
    successors map[string]pocket.Node
    mu         sync.Mutex
}

func NewStreamNode(name string, bufferSize int, processor func([]any) (any, error)) pocket.Node {
    return &StreamNode{
        name:       name,
        bufferSize: bufferSize,
        processor:  processor,
        buffer:     make([]any, 0, bufferSize),
        successors: make(map[string]pocket.Node),
    }
}

func (n *StreamNode) Exec(ctx context.Context, input any) (any, error) {
    n.mu.Lock()
    defer n.mu.Unlock()
    
    // Add to buffer
    n.buffer = append(n.buffer, input)
    
    // Process if buffer is full
    if len(n.buffer) >= n.bufferSize {
        result, err := n.processor(n.buffer)
        n.buffer = n.buffer[:0] // Clear buffer
        return result, err
    }
    
    // Return partial result
    return map[string]any{
        "buffered": len(n.buffer),
        "pending":  true,
    }, nil
}

func (n *StreamNode) Post(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
    result := exec.(map[string]any)
    
    if pending, ok := result["pending"].(bool); ok && pending {
        return result, "buffering", nil
    }
    
    return result, "processed", nil
}
```

### Machine Learning Node

Integrate ML models:

```go
type MLModelNode struct {
    name       string
    modelPath  string
    model      MLModel
    successors map[string]pocket.Node
}

func NewMLModelNode(name, modelPath string) (pocket.Node, error) {
    model, err := loadModel(modelPath)
    if err != nil {
        return nil, err
    }
    
    return &MLModelNode{
        name:       name,
        modelPath:  modelPath,
        model:      model,
        successors: make(map[string]pocket.Node),
    }, nil
}

func (n *MLModelNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
    // Prepare features
    features, err := extractFeatures(input)
    if err != nil {
        return nil, fmt.Errorf("feature extraction failed: %w", err)
    }
    
    // Normalize features
    if normalizer, exists := store.Get(ctx, "ml:normalizer"); exists {
        features = normalizer.(Normalizer).Transform(features)
    }
    
    return features, nil
}

func (n *MLModelNode) Exec(ctx context.Context, features any) (any, error) {
    // Run inference
    prediction, err := n.model.Predict(features)
    if err != nil {
        return nil, fmt.Errorf("prediction failed: %w", err)
    }
    
    return Prediction{
        Value:      prediction,
        Confidence: n.model.Confidence(),
        ModelID:    n.modelPath,
    }, nil
}

func (n *MLModelNode) Post(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
    prediction := exec.(Prediction)
    
    // Store prediction for monitoring
    store.Set(ctx, "ml:lastPrediction", prediction)
    
    // Route based on confidence
    if prediction.Confidence < 0.5 {
        return prediction, "low-confidence", nil
    }
    
    return prediction, "high-confidence", nil
}
```

## Node Decorators

### Caching Decorator

Add caching to any node:

```go
type CachingNode struct {
    pocket.Node
    cache  Cache
    keyGen func(any) string
    ttl    time.Duration
}

func WithCaching(node pocket.Node, cache Cache, ttl time.Duration) pocket.Node {
    return &CachingNode{
        Node:   node,
        cache:  cache,
        ttl:    ttl,
        keyGen: defaultKeyGen,
    }
}

func (n *CachingNode) Exec(ctx context.Context, input any) (any, error) {
    key := n.keyGen(input)
    
    // Check cache
    if cached, exists := n.cache.Get(key); exists {
        if entry, ok := cached.(CacheEntry); ok && time.Since(entry.Time) < n.ttl {
            return entry.Value, nil
        }
    }
    
    // Execute wrapped node
    result, err := n.Node.Exec(ctx, input)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    n.cache.Set(key, CacheEntry{
        Value: result,
        Time:  time.Now(),
    })
    
    return result, nil
}
```

### Monitoring Decorator

Add monitoring to nodes:

```go
type MonitoredNode struct {
    pocket.Node
    metrics MetricsCollector
}

func WithMonitoring(node pocket.Node, metrics MetricsCollector) pocket.Node {
    return &MonitoredNode{
        Node:    node,
        metrics: metrics,
    }
}

func (n *MonitoredNode) Exec(ctx context.Context, input any) (any, error) {
    start := time.Now()
    
    result, err := n.Node.Exec(ctx, input)
    
    duration := time.Since(start)
    n.metrics.RecordDuration(n.Name(), "exec", duration)
    
    if err != nil {
        n.metrics.IncrementCounter(n.Name(), "errors")
    } else {
        n.metrics.IncrementCounter(n.Name(), "success")
    }
    
    return result, err
}
```

## Testing Custom Nodes

### Unit Testing

```go
func TestCustomNode(t *testing.T) {
    node := NewCustomNode("test")
    
    // Test Prep
    store := pocket.NewStore()
    prepResult, err := node.Prep(context.Background(), store, testInput)
    assert.NoError(t, err)
    assert.NotNil(t, prepResult)
    
    // Test Exec
    execResult, err := node.Exec(context.Background(), prepResult)
    assert.NoError(t, err)
    assert.Equal(t, expectedResult, execResult)
    
    // Test Post
    output, next, err := node.Post(context.Background(), store, testInput, prepResult, execResult)
    assert.NoError(t, err)
    assert.Equal(t, "success", next)
    assert.Equal(t, execResult, output)
}
```

### Integration Testing

```go
func TestCustomNodeIntegration(t *testing.T) {
    // Create custom node
    customNode := NewStreamNode("streamer", 5, aggregateFunc)
    
    // Connect to other nodes
    customNode.Connect("processed", processedHandler)
    customNode.Connect("buffering", bufferingHandler)
    
    // Create graph
    graph := pocket.NewGraph(customNode, pocket.NewStore())
    
    // Test stream processing
    for i := 0; i < 10; i++ {
        result, err := graph.Run(context.Background(), fmt.Sprintf("item-%d", i))
        assert.NoError(t, err)
        
        if i < 4 {
            // Should be buffering
            assert.True(t, result.(map[string]any)["pending"].(bool))
        } else if i == 4 {
            // Should process batch
            assert.False(t, result.(map[string]any)["pending"].(bool))
        }
    }
}
```

## Best Practices

### 1. Implement All Interface Methods

Even if not used, implement all methods:

```go
func (n *CustomNode) InputType() reflect.Type {
    return reflect.TypeOf((*any)(nil)).Elem() // any type
}

func (n *CustomNode) OutputType() reflect.Type {
    return reflect.TypeOf((*any)(nil)).Elem() // any type
}
```

### 2. Thread Safety

Make nodes thread-safe:

```go
type SafeNode struct {
    mu    sync.RWMutex
    state NodeState
}

func (n *SafeNode) getState() NodeState {
    n.mu.RLock()
    defer n.mu.RUnlock()
    return n.state
}

func (n *SafeNode) setState(state NodeState) {
    n.mu.Lock()
    defer n.mu.Unlock()
    n.state = state
}
```

### 3. Error Handling

Provide detailed error context:

```go
func (n *CustomNode) Exec(ctx context.Context, input any) (any, error) {
    result, err := n.process(input)
    if err != nil {
        return nil, fmt.Errorf("node %s exec failed for input %v: %w", n.Name(), input, err)
    }
    return result, nil
}
```

### 4. Documentation

Document node behavior:

```go
// StreamNode processes inputs in batches. It buffers inputs until
// the buffer size is reached, then processes the entire batch.
// 
// Inputs are expected to be serializable values.
// Outputs are either:
//   - Partial results with "pending: true" while buffering
//   - Batch processing results when buffer is full
//
// Routes:
//   - "buffering": Still collecting inputs
//   - "processed": Batch has been processed
type StreamNode struct {
    // ...
}
```

### 5. Validate Inputs

Always validate in Prep:

```go
func (n *CustomNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
    // Type check
    typed, ok := input.(ExpectedType)
    if !ok {
        return nil, fmt.Errorf("invalid input type: expected %T, got %T", ExpectedType{}, input)
    }
    
    // Validate
    if err := typed.Validate(); err != nil {
        return nil, fmt.Errorf("input validation failed: %w", err)
    }
    
    return typed, nil
}
```

## Summary

Custom nodes in Pocket enable:

1. **Specialized behavior** not covered by standard nodes
2. **External service integration** with custom protocols
3. **Stateful processing** with internal state management
4. **Advanced patterns** like streaming and batching
5. **Domain-specific logic** tailored to your needs

By implementing the Node interface, you can extend Pocket to handle any use case while maintaining compatibility with the framework's features.