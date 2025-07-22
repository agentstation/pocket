# Configuration Options

## Overview

This reference documents all configuration options available in Pocket, including node options, store options, graph options, and environment variables.

## Node Configuration

### Basic Options

#### Name
Every node must have a unique name within a workflow.

```go
node := pocket.NewNode[In, Out]("unique-name", opts...)
```

### Lifecycle Options

#### WithPrep
Configure the preparation phase.

```go
pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input In) (any, error) {
    // Preparation logic
    return preparedData, nil
})
```

**Use cases:**
- Input validation
- Loading configuration from store
- Data transformation before processing

#### WithExec
Configure the execution phase.

```go
pocket.WithExec(func(ctx context.Context, prepData any) (Out, error) {
    // Pure business logic
    return result, nil
})
```

**Use cases:**
- Core business logic
- Data processing
- Calculations and transformations

#### WithPost
Configure the post-processing phase.

```go
pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, 
    input In, prep any, exec Out) (Out, string, error) {
    // Post-processing and routing
    return output, "next-node", nil
})
```

**Use cases:**
- Saving results to store
- Determining next node
- Cleanup operations

### Hook Options

#### WithOnSuccess
Execute when node completes successfully.

```go
pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
    store.Set(ctx, "success:count", incrementCounter())
    log.Printf("Success: %v", output)
})
```

#### WithOnFailure
Execute when node fails.

```go
pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, err error) {
    store.Set(ctx, "error:last", err.Error())
    alerting.SendAlert("Node failed", err)
})
```

#### WithOnComplete
Always execute after node completion.

```go
pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
    store.Delete(ctx, "temp:processing")
    releaseResources()
})
```

### Error Handling Options

#### WithRetry
Add retry capability with fixed delay.

```go
pocket.WithRetry(3, time.Second) // 3 attempts, 1 second between
```

**Advanced retry configuration:**
```go
// Custom retry with exponential backoff
pocket.WithRetryConfig(RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    Jitter:       0.1, // 10% jitter
})
```

#### Fallback (in Steps)
Provide alternative behavior on failure. Fallback is now part of the Steps struct and receives prepResult.

```go
pocket.NewNode[In, Out]("node",
    pocket.Steps{
        Exec: func(ctx context.Context, prepResult any) (any, error) {
            // Primary logic
            return process(prepResult)
        },
        Fallback: func(ctx context.Context, prepResult any, err error) (any, error) {
            log.Printf("Primary failed: %v, using fallback", err)
            return getDefaultOutput(prepResult), nil
        },
    },
)
```

#### WithTimeout
Set execution timeout.

```go
pocket.WithTimeout(30 * time.Second)
```

### Validation Options

#### WithInputValidation
Validate input before processing.

```go
pocket.WithInputValidation(func(input In) error {
    if !input.IsValid() {
        return errors.New("invalid input")
    }
    return nil
})
```

#### WithOutputValidation
Validate output before returning.

```go
pocket.WithOutputValidation(func(output Out) error {
    if output.Score < 0 || output.Score > 100 {
        return errors.New("score out of range")
    }
    return nil
})
```

## Store Configuration

### Store Creation Options

#### WithMaxEntries
Set maximum number of entries (LRU eviction).

```go
store := pocket.NewStore(
    pocket.WithMaxEntries(10000),
)
```

**Behavior:**
- When limit exceeded, least recently used entries are evicted
- Access updates entry position in LRU list
- Set operations may trigger eviction

#### WithTTL
Set time-to-live for all entries.

```go
store := pocket.NewStore(
    pocket.WithTTL(5 * time.Minute),
)
```

**Behavior:**
- Entries older than TTL are automatically removed
- TTL checked on access (lazy deletion)
- Can be combined with max entries

#### WithEvictionCallback
Get notified when entries are evicted.

```go
store := pocket.NewStore(
    pocket.WithEvictionCallback(func(key string, value any) {
        log.Printf("Evicted key: %s", key)
        cleanupResources(value)
    }),
)
```

**Called when:**
- LRU eviction occurs
- TTL expiration
- Manual deletion

### Scoped Store Configuration

```go
// Create scoped stores for isolation
userStore := mainStore.Scope("user")
cacheStore := mainStore.Scope("cache")
tempStore := mainStore.Scope("temp")
```

**Behavior:**
- Keys are prefixed with scope name
- Scopes share underlying storage
- Useful for cleanup and organization

## Graph Configuration

### Graph Creation

```go
graph := pocket.NewGraph(startNode, store, opts...)
```

### Graph Options

#### WithLogger
Add logging to graph execution.

```go
graph := pocket.NewGraph(startNode, store,
    pocket.WithLogger(logger),
)
```

**Logs:**
- Node execution start/end
- Routing decisions
- Errors and warnings

#### WithTracer
Add distributed tracing.

```go
graph := pocket.NewGraph(startNode, store,
    pocket.WithTracer(tracer),
)
```

**Traces:**
- Node execution spans
- Store operations
- Graph traversal

#### WithMetrics
Collect execution metrics.

```go
graph := pocket.NewGraph(startNode, store,
    pocket.WithMetrics(metricsCollector),
)
```

**Metrics:**
- Execution duration
- Success/failure rates
- Store operations

#### WithMaxDepth
Prevent infinite loops.

```go
graph := pocket.NewGraph(startNode, store,
    pocket.WithMaxDepth(100), // Maximum 100 node traversals
)
```

## Builder Configuration

### Builder Options

```go
builder := pocket.NewBuilder(store,
    pocket.WithStrictMode(),    // Validate on each operation
    pocket.WithAutoConnect(),   // Auto-connect nodes by name
)
```

#### WithStrictMode
Enable strict validation during building.

```go
builder := pocket.NewBuilder(store, pocket.WithStrictMode())
```

**Validates:**
- Node names are unique
- Connections reference existing nodes
- No cycles in specific cases

#### WithAutoConnect
Automatically connect nodes based on naming.

```go
builder := pocket.NewBuilder(store, pocket.WithAutoConnect())
// Nodes ending with numbers auto-connect to next
builder.Add(step1).Add(step2).Add(step3) // Auto-connects 1→2→3
```

## Concurrent Processing Configuration

### FanOut Options

```go
results, err := pocket.FanOut(ctx, processor, store, items,
    pocket.WithConcurrency(10),        // Max 10 concurrent
    pocket.WithBatchSize(100),         // Process in batches of 100
    pocket.WithOrdered(true),          // Maintain order
    pocket.WithErrorStrategy(Continue), // Continue on errors
)
```

#### Concurrency Control
```go
pocket.WithConcurrency(runtime.NumCPU() * 2)
```

#### Error Strategies
```go
pocket.WithErrorStrategy(pocket.FailFast)    // Stop on first error
pocket.WithErrorStrategy(pocket.Continue)     // Continue, collect errors
pocket.WithErrorStrategy(pocket.BestEffort)   // Return partial results
```

### Pipeline Options

```go
result, err := pocket.Pipeline(ctx, nodes, store, input,
    pocket.WithStageTimeout(5 * time.Second), // Timeout per stage
    pocket.WithStageRetry(2),                  // Retry each stage
)
```

## Environment Variables

### Core Settings

```bash
# Enable debug logging
POCKET_DEBUG=true

# Set default timeout
POCKET_DEFAULT_TIMEOUT=30s

# Set default retry attempts
POCKET_DEFAULT_RETRIES=3

# Enable metrics collection
POCKET_METRICS_ENABLED=true
```

### Store Settings

```bash
# Default store max entries
POCKET_STORE_MAX_ENTRIES=10000

# Default store TTL
POCKET_STORE_TTL=5m

# Enable store persistence (if implemented)
POCKET_STORE_PERSIST=true
POCKET_STORE_PERSIST_PATH=/var/lib/pocket/store
```

### Performance Settings

```bash
# Maximum concurrent operations
POCKET_MAX_CONCURRENT=100

# Default batch size
POCKET_BATCH_SIZE=1000

# Worker pool size
POCKET_WORKER_POOL_SIZE=50
```

## YAML Configuration

### Workflow Configuration

```yaml
# workflow.yaml
name: data-processing
version: 1.0
config:
  timeout: 30s
  retries: 3
  store:
    maxEntries: 10000
    ttl: 5m
  
nodes:
  - id: processor
    config:
      timeout: 10s
      retries: 5
      concurrency: 10
```

### Node Configuration in YAML

```yaml
nodes:
  - id: api-call
    type: processor
    config:
      handler: callAPI
      timeout: ${API_TIMEOUT:10s}
      retries: ${API_RETRIES:3}
      fallback: ${API_FALLBACK:cacheLookup}
      rateLimit:
        requests: 100
        window: 1m
      circuitBreaker:
        maxFailures: 5
        resetTimeout: 30s
```

## Middleware Configuration

### Logging Middleware

```go
loggingConfig := LoggingConfig{
    Level:      InfoLevel,
    LogInput:   true,
    LogOutput:  true,
    LogErrors:  true,
    Sanitizer:  sanitizeFunc,
    Format:     JSONFormat,
}

node = WithLogging(logger, loggingConfig)(node)
```

### Metrics Middleware

```go
metricsConfig := MetricsConfig{
    Namespace:   "pocket",
    Subsystem:   "nodes",
    Buckets:     []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
    Labels:      []string{"node", "status"},
}

node = WithMetrics(collector, metricsConfig)(node)
```

### Rate Limiting Middleware

```go
rateLimitConfig := RateLimitConfig{
    Rate:       100,              // 100 requests
    Burst:      10,               // Allow burst of 10
    Per:        time.Minute,      // Per minute
    KeyFunc:    extractUserID,    // Rate limit per user
    WaitTimeout: 5 * time.Second, // Max wait time
}

node = WithRateLimit(rateLimitConfig)(node)
```

## Advanced Configuration

### Custom Node Configuration

```go
type NodeConfig struct {
    // Basic settings
    Name        string
    Description string
    Version     string
    
    // Performance
    MaxConcurrent   int
    BufferSize      int
    ProcessTimeout  time.Duration
    
    // Resilience
    RetryPolicy     RetryPolicy
    CircuitBreaker  CircuitBreakerConfig
    Fallback        FallbackConfig
    
    // Monitoring
    Metrics         MetricsConfig
    Logging         LoggingConfig
    Tracing         TracingConfig
}

func ConfigureNode(config NodeConfig) pocket.Node {
    node := pocket.NewNode[In, Out](config.Name, baseOptions...)
    
    // Apply configurations
    if config.RetryPolicy.Enabled {
        node = WithRetryPolicy(config.RetryPolicy)(node)
    }
    
    if config.CircuitBreaker.Enabled {
        node = WithCircuitBreaker(config.CircuitBreaker)(node)
    }
    
    return node
}
```

### Dynamic Configuration

```go
// Configuration that can change at runtime
type DynamicConfig struct {
    source ConfigSource
    mu     sync.RWMutex
    values map[string]any
}

func (c *DynamicConfig) Get(key string) any {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.values[key]
}

func (c *DynamicConfig) Watch(key string, callback func(any)) {
    c.source.Subscribe(key, func(value any) {
        c.mu.Lock()
        c.values[key] = value
        c.mu.Unlock()
        callback(value)
    })
}

// Use in nodes
node := pocket.NewNode[In, Out]("dynamic",
    pocket.WithExec(func(ctx context.Context, input In) (Out, error) {
        timeout := config.Get("timeout").(time.Duration)
        // Use dynamic timeout
    }),
)
```

## Best Practices

### 1. Use Environment Variables for Deployment

```go
timeout := getEnvDuration("POCKET_TIMEOUT", 30*time.Second)
maxRetries := getEnvInt("POCKET_MAX_RETRIES", 3)
```

### 2. Validate Configuration Early

```go
func ValidateConfig(config Config) error {
    if config.Timeout <= 0 {
        return errors.New("timeout must be positive")
    }
    if config.MaxRetries < 0 {
        return errors.New("max retries cannot be negative")
    }
    return nil
}
```

### 3. Provide Sensible Defaults

```go
func DefaultConfig() Config {
    return Config{
        Timeout:     30 * time.Second,
        MaxRetries:  3,
        Concurrency: runtime.NumCPU(),
        BufferSize:  1000,
    }
}
```

### 4. Make Configuration Testable

```go
func NewNodeWithConfig(config Config) pocket.Node {
    // Configuration affects behavior
    return pocket.NewNode[In, Out]("configured",
        pocket.WithTimeout(config.Timeout),
        pocket.WithRetry(config.MaxRetries, config.RetryDelay),
    )
}

// Easy to test with different configs
func TestWithConfig(t *testing.T, config Config) {
    node := NewNodeWithConfig(config)
    // Test behavior
}
```

## Summary

Pocket's configuration system provides:

1. **Fine-grained control** over node behavior
2. **Store management** options for memory and performance
3. **Graph-level settings** for observability
4. **Environment variables** for deployment flexibility
5. **YAML support** for declarative configuration

Choose configuration options based on your specific requirements for performance, resilience, and observability.