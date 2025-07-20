# Middleware System

## Overview

Pocket's middleware system allows you to wrap nodes with cross-cutting concerns like logging, metrics, authentication, and more. Middleware provides a clean way to add functionality without modifying node implementations.

## Understanding Middleware

### The Middleware Pattern

Middleware in Pocket follows the decorator pattern:

```go
// Middleware is a function that wraps a node
type Middleware func(pocket.Node) pocket.Node

// Apply middleware to a node
node := pocket.NewNode[Input, Output]("processor", ...)
wrappedNode := withLogging(withMetrics(withAuth(node)))
```

### Creating Custom Middleware

Basic middleware structure:

```go
func WithCustomMiddleware(config Config) Middleware {
    return func(inner pocket.Node) pocket.Node {
        return &customMiddlewareNode{
            Node:   inner,
            config: config,
        }
    }
}

type customMiddlewareNode struct {
    pocket.Node
    config Config
}

// Override methods as needed
func (n *customMiddlewareNode) Exec(ctx context.Context, input any) (any, error) {
    // Before execution
    log.Printf("Executing node: %s", n.Name())
    
    // Call inner node
    result, err := n.Node.Exec(ctx, input)
    
    // After execution
    log.Printf("Completed node: %s, error: %v", n.Name(), err)
    
    return result, err
}
```

## Built-in Middleware

### Logging Middleware

Add structured logging to nodes:

```go
import "github.com/agentstation/pocket/middleware"

// Simple logging
loggingMiddleware := middleware.WithLogging(logger)

// Detailed logging with request/response
detailedLogging := middleware.WithDetailedLogging(logger, middleware.LogConfig{
    LogInput:  true,
    LogOutput: true,
    LogErrors: true,
    Sanitizer: func(data any) any {
        // Remove sensitive data
        return sanitize(data)
    },
})

// Apply to node
node := pocket.NewNode[Request, Response]("api-call", ...)
loggedNode := detailedLogging(node)
```

### Metrics Middleware

Collect performance metrics:

```go
// Basic metrics
metricsMiddleware := middleware.WithMetrics(metricsCollector)

// Custom metrics
customMetrics := middleware.WithCustomMetrics(func(node pocket.Node) pocket.Node {
    return &metricsNode{
        Node: node,
        histogram: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "node_duration_seconds",
                Help: "Node execution duration",
            },
            []string{"node_name", "status"},
        ),
    }
})

type metricsNode struct {
    pocket.Node
    histogram *prometheus.HistogramVec
}

func (n *metricsNode) Exec(ctx context.Context, input any) (any, error) {
    timer := prometheus.NewTimer(n.histogram.WithLabelValues(n.Name(), "pending"))
    
    result, err := n.Node.Exec(ctx, input)
    
    status := "success"
    if err != nil {
        status = "error"
    }
    
    timer.ObserveDuration()
    n.histogram.WithLabelValues(n.Name(), status).Observe(timer.ObserveDuration().Seconds())
    
    return result, err
}
```

### Timing Middleware

Track execution time:

```go
timingMiddleware := middleware.WithTiming(func(nodeName string, phase string, duration time.Duration) {
    log.Printf("[TIMING] %s.%s took %v", nodeName, phase, duration)
})

// Or with metrics
timingWithMetrics := middleware.WithTiming(func(nodeName string, phase string, duration time.Duration) {
    metrics.RecordNodeDuration(nodeName, phase, duration)
})
```

### Retry Middleware

Add retry logic to any node:

```go
// Simple retry
retryMiddleware := middleware.WithRetry(3, time.Second)

// Advanced retry with backoff
advancedRetry := middleware.WithAdvancedRetry(middleware.RetryConfig{
    MaxAttempts: 5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay: 10 * time.Second,
    Multiplier: 2,
    Jitter: 0.1,
    RetryableErrors: func(err error) bool {
        // Only retry specific errors
        return errors.Is(err, ErrTemporary) || errors.Is(err, context.DeadlineExceeded)
    },
})
```

## Lifecycle Hooks

### Hook System

Hooks provide points to inject behavior during node execution:

```go
hookManager := node.NewHookManager()

// Register hooks for specific events
hookManager.Register(loggingHook, node.EventPrep, node.EventExec, node.EventPost)
hookManager.Register(metricsHook, node.EventSuccess, node.EventError)
hookManager.Register(tracingHook) // Global hook for all events

// Apply hooks to node
hookedNode := node.WithHooks(hookManager)(originalNode)
```

### Creating Custom Hooks

```go
type CustomHook struct {
    name string
    logger Logger
}

func (h *CustomHook) Name() string {
    return h.name
}

func (h *CustomHook) Execute(ctx context.Context, event *node.Event) error {
    switch event.Type {
    case node.EventPrep:
        if event.Phase == "before" {
            h.logger.Info("Starting prep phase", "node", event.NodeName)
        }
        
    case node.EventExec:
        if event.Phase == "after" && event.Error != nil {
            h.logger.Error("Exec failed", "node", event.NodeName, "error", event.Error)
        }
        
    case node.EventSuccess:
        h.logger.Info("Node completed successfully", "node", event.NodeName)
        
    case node.EventRoute:
        next := event.Metadata["next"].(string)
        h.logger.Info("Routing", "from", event.NodeName, "to", next)
    }
    
    return nil
}
```

## Advanced Middleware Patterns

### Composable Middleware

Build complex middleware from simpler pieces:

```go
// Middleware composer
func Compose(middlewares ...Middleware) Middleware {
    return func(node pocket.Node) pocket.Node {
        // Apply in reverse order so first middleware is outermost
        for i := len(middlewares) - 1; i >= 0; i-- {
            node = middlewares[i](node)
        }
        return node
    }
}

// Usage
combined := Compose(
    WithLogging(logger),
    WithMetrics(metrics),
    WithRetry(3, time.Second),
    WithTimeout(30 * time.Second),
)

protectedNode := combined(node)
```

### Conditional Middleware

Apply middleware based on conditions:

```go
func ConditionalMiddleware(condition func() bool, middleware Middleware) Middleware {
    return func(node pocket.Node) pocket.Node {
        if condition() {
            return middleware(node)
        }
        return node
    }
}

// Usage
debugLogging := ConditionalMiddleware(
    func() bool { return os.Getenv("DEBUG") == "true" },
    WithDetailedLogging(logger, config),
)

productionRetry := ConditionalMiddleware(
    func() bool { return os.Getenv("ENV") == "production" },
    WithRetry(5, time.Second),
)
```

### Context-Aware Middleware

Access and modify context:

```go
type contextKey string

const (
    RequestIDKey contextKey = "requestID"
    UserIDKey    contextKey = "userID"
)

func WithRequestID() Middleware {
    return func(node pocket.Node) pocket.Node {
        return &requestIDNode{Node: node}
    }
}

type requestIDNode struct {
    pocket.Node
}

func (n *requestIDNode) Exec(ctx context.Context, input any) (any, error) {
    // Extract or generate request ID
    requestID, ok := ctx.Value(RequestIDKey).(string)
    if !ok {
        requestID = generateRequestID()
        ctx = context.WithValue(ctx, RequestIDKey, requestID)
    }
    
    // Add to logs
    log.Printf("[%s] Executing node: %s", requestID, n.Name())
    
    return n.Node.Exec(ctx, input)
}
```

### Authentication Middleware

Add authentication checks:

```go
func WithAuth(authService AuthService) Middleware {
    return func(node pocket.Node) pocket.Node {
        return &authNode{
            Node:        node,
            authService: authService,
        }
    }
}

type authNode struct {
    pocket.Node
    authService AuthService
}

func (n *authNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
    // Extract auth token from context
    token, ok := ctx.Value("authToken").(string)
    if !ok {
        return nil, errors.New("missing auth token")
    }
    
    // Validate token
    user, err := n.authService.ValidateToken(token)
    if err != nil {
        return nil, fmt.Errorf("authentication failed: %w", err)
    }
    
    // Add user to context
    ctx = context.WithValue(ctx, "user", user)
    
    // Continue with original prep
    return n.Node.Prep(ctx, store, input)
}
```

### Circuit Breaker Middleware

Protect against cascading failures:

```go
func WithCircuitBreaker(name string, config CircuitBreakerConfig) Middleware {
    cb := newCircuitBreaker(name, config)
    
    return func(node pocket.Node) pocket.Node {
        return &circuitBreakerNode{
            Node:    node,
            breaker: cb,
        }
    }
}

type circuitBreakerNode struct {
    pocket.Node
    breaker *circuitBreaker
}

func (n *circuitBreakerNode) Exec(ctx context.Context, input any) (any, error) {
    // Check circuit state
    if !n.breaker.Allow() {
        return nil, errors.New("circuit breaker open")
    }
    
    // Execute with circuit breaker protection
    result, err := n.Node.Exec(ctx, input)
    
    if err != nil {
        n.breaker.RecordFailure()
    } else {
        n.breaker.RecordSuccess()
    }
    
    return result, err
}
```

## Middleware for Different Phases

### Phase-Specific Middleware

Apply middleware to specific lifecycle phases:

```go
type PhaseMiddleware struct {
    prep Middleware
    exec Middleware
    post Middleware
}

func (m *PhaseMiddleware) Apply(node pocket.Node) pocket.Node {
    return &phaseNode{
        Node:     node,
        prepNode: m.prep(node),
        execNode: m.exec(node),
        postNode: m.post(node),
    }
}

type phaseNode struct {
    pocket.Node
    prepNode pocket.Node
    execNode pocket.Node
    postNode pocket.Node
}

func (n *phaseNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
    // Use prep-specific middleware
    return n.prepNode.Prep(ctx, store, input)
}

func (n *phaseNode) Exec(ctx context.Context, prepResult any) (any, error) {
    // Use exec-specific middleware
    return n.execNode.Exec(ctx, prepResult)
}

func (n *phaseNode) Post(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
    // Use post-specific middleware
    return n.postNode.Post(ctx, store, input, prep, exec)
}
```

## Testing Middleware

### Mock Middleware for Testing

```go
func WithMockResponse(response any, err error) Middleware {
    return func(node pocket.Node) pocket.Node {
        return &mockNode{
            Node:     node,
            response: response,
            err:      err,
        }
    }
}

type mockNode struct {
    pocket.Node
    response any
    err      error
}

func (n *mockNode) Exec(ctx context.Context, input any) (any, error) {
    // Return mock response instead of calling inner node
    return n.response, n.err
}

// Usage in tests
testNode := WithMockResponse(expectedResponse, nil)(realNode)
```

### Spy Middleware

Capture calls for verification:

```go
type SpyMiddleware struct {
    calls []CallInfo
    mu    sync.Mutex
}

type CallInfo struct {
    Method    string
    Input     any
    Output    any
    Error     error
    Timestamp time.Time
}

func (s *SpyMiddleware) Wrap(node pocket.Node) pocket.Node {
    return &spyNode{
        Node: node,
        spy:  s,
    }
}

func (s *SpyMiddleware) GetCalls() []CallInfo {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    return append([]CallInfo{}, s.calls...)
}

type spyNode struct {
    pocket.Node
    spy *SpyMiddleware
}

func (n *spyNode) Exec(ctx context.Context, input any) (any, error) {
    result, err := n.Node.Exec(ctx, input)
    
    n.spy.mu.Lock()
    n.spy.calls = append(n.spy.calls, CallInfo{
        Method:    "Exec",
        Input:     input,
        Output:    result,
        Error:     err,
        Timestamp: time.Now(),
    })
    n.spy.mu.Unlock()
    
    return result, err
}
```

## Best Practices

### 1. Order Matters

```go
// Correct order: retry wraps timeout
node = WithTimeout(30*time.Second)(node)
node = WithRetry(3, time.Second)(node)

// Incorrect: timeout wraps retry (timeout might fire during retries)
node = WithRetry(3, time.Second)(node)
node = WithTimeout(30*time.Second)(node)
```

### 2. Keep Middleware Focused

```go
// Good: Single responsibility
WithLogging(logger)
WithMetrics(collector)
WithAuth(service)

// Avoid: Kitchen sink middleware
WithEverything(logger, metrics, auth, retry, timeout)
```

### 3. Make Middleware Configurable

```go
type LoggingConfig struct {
    Level      LogLevel
    LogInput   bool
    LogOutput  bool
    Sanitizer  func(any) any
    Fields     map[string]any
}

func WithConfigurableLogging(config LoggingConfig) Middleware {
    // Implementation uses config
}
```

### 4. Document Middleware Behavior

```go
// WithRateLimit adds rate limiting to a node.
// It limits execution to maxRequests per window duration.
// Excess requests are rejected with ErrRateLimitExceeded.
// The rate limit is shared across all instances of the wrapped node.
func WithRateLimit(maxRequests int, window time.Duration) Middleware {
    // Implementation
}
```

### 5. Test Middleware Thoroughly

```go
func TestRetryMiddleware(t *testing.T) {
    attempts := 0
    failingNode := pocket.NewNode[string, string]("failing",
        pocket.WithExec(func(ctx context.Context, input string) (string, error) {
            attempts++
            if attempts < 3 {
                return "", errors.New("temporary failure")
            }
            return "success", nil
        }),
    )
    
    retryNode := WithRetry(3, 10*time.Millisecond)(failingNode)
    
    result, err := retryNode.Exec(context.Background(), "test")
    assert.NoError(t, err)
    assert.Equal(t, "success", result)
    assert.Equal(t, 3, attempts)
}
```

## Summary

Pocket's middleware system provides:

1. **Cross-cutting concerns** without modifying node logic
2. **Composable decorators** for building complex behaviors
3. **Lifecycle hooks** for fine-grained control
4. **Built-in patterns** for common needs (logging, metrics, retry)
5. **Testability** through mock and spy middleware

Middleware enables clean separation of concerns, making your workflows more maintainable and your nodes more focused on their core responsibilities.