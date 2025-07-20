# Error Handling Guide

## Overview

Building resilient workflows requires robust error handling. Pocket provides multiple strategies for handling failures, from simple retries to sophisticated circuit breakers and compensation patterns.

## Error Handling Strategies

### 1. Basic Error Handling

Errors can occur in any phase of node execution:

```go
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input Input) (any, error) {
        // Validation errors
        if !input.IsValid() {
            return nil, fmt.Errorf("invalid input: %v", input)
        }
        return input, nil
    }),
    
    pocket.WithExec(func(ctx context.Context, prepData any) (Output, error) {
        // Business logic errors
        result, err := processData(prepData)
        if err != nil {
            return Output{}, fmt.Errorf("processing failed: %w", err)
        }
        return result, nil
    }),
    
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input Input, prep any, exec Output) (Output, string, error) {
        // State update errors
        if err := store.Set(ctx, "result", exec); err != nil {
            return Output{}, "", fmt.Errorf("failed to save result: %w", err)
        }
        return exec, "next", nil
    }),
)
```

### 2. Fallback Handling

Provide alternative behavior when primary logic fails:

```go
apiCall := pocket.NewNode[Request, Response]("api-call",
    pocket.WithExec(func(ctx context.Context, req Request) (Response, error) {
        // Primary API call
        resp, err := callExternalAPI(req)
        if err != nil {
            return Response{}, err
        }
        return resp, nil
    }),
    pocket.WithFallback(func(ctx context.Context, req Request, originalErr error) (Response, error) {
        // Fallback logic
        log.Printf("API call failed: %v, using fallback", originalErr)
        
        // Try alternative approach
        if cachedResp, exists := getFromCache(req); exists {
            return cachedResp, nil
        }
        
        // Return default response
        return Response{
            Status:  "fallback",
            Message: "Service temporarily unavailable",
            Data:    getDefaultData(),
        }, nil
    }),
)
```

### 3. Retry with Backoff

Automatically retry failed operations:

```go
// Simple retry
resilientNode := pocket.NewNode[Input, Output]("resilient",
    pocket.WithExec(processFunc),
    pocket.WithRetry(3, time.Second), // 3 retries, 1 second between
)

// Custom retry logic
customRetry := pocket.NewNode[Request, Response]("custom-retry",
    pocket.WithExec(func(ctx context.Context, req Request) (Response, error) {
        var lastErr error
        
        for attempt := 1; attempt <= 3; attempt++ {
            resp, err := attemptOperation(req)
            if err == nil {
                return resp, nil
            }
            
            lastErr = err
            
            // Exponential backoff
            if attempt < 3 {
                waitTime := time.Duration(attempt*attempt) * time.Second
                select {
                case <-time.After(waitTime):
                    // Continue to next attempt
                case <-ctx.Done():
                    return Response{}, ctx.Err()
                }
            }
        }
        
        return Response{}, fmt.Errorf("failed after 3 attempts: %w", lastErr)
    }),
)
```

## Circuit Breaker Pattern

Prevent cascading failures with circuit breakers:

```go
import "github.com/agentstation/pocket/fallback"

// Create a circuit breaker
cb := fallback.NewCircuitBreaker("external-service",
    fallback.WithMaxFailures(3),              // Open after 3 failures
    fallback.WithResetTimeout(30*time.Second), // Try again after 30s
    fallback.WithOnStateChange(func(from, to string) {
        log.Printf("Circuit breaker state changed: %s -> %s", from, to)
    }),
)

// Use in a node
protectedNode := pocket.NewNode[Request, Response]("protected",
    pocket.WithExec(func(ctx context.Context, req Request) (Response, error) {
        // Check circuit breaker state
        if !cb.CanExecute() {
            return Response{}, errors.New("circuit breaker open")
        }
        
        // Attempt operation
        resp, err := riskyOperation(req)
        
        // Record result
        if err != nil {
            cb.RecordFailure()
            return Response{}, err
        }
        
        cb.RecordSuccess()
        return resp, nil
    }),
    pocket.WithFallback(func(ctx context.Context, req Request, err error) (Response, error) {
        // Fallback when circuit is open
        return Response{
            Status: "degraded",
            Data:   getDegradedResponse(),
        }, nil
    }),
)
```

## Lifecycle Hooks

Use hooks for cleanup and monitoring:

```go
resourceNode := pocket.NewNode[Request, Response]("resource-handler",
    pocket.WithExec(func(ctx context.Context, req Request) (Response, error) {
        // Acquire resources
        conn, err := acquireConnection()
        if err != nil {
            return Response{}, err
        }
        defer conn.Close()
        
        return processWithConnection(conn, req)
    }),
    
    pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
        // Record success metrics
        metrics.RecordSuccess("resource-handler")
        store.Set(ctx, "last-success", time.Now())
    }),
    
    pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, err error) {
        // Log failure details
        log.Printf("Resource handler failed: %v", err)
        metrics.RecordFailure("resource-handler", err)
        
        // Store error for analysis
        store.Set(ctx, "last-error", map[string]any{
            "error":     err.Error(),
            "timestamp": time.Now(),
        })
    }),
    
    pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
        // Always runs - perfect for cleanup
        store.Delete(ctx, "temp:resource-lock")
        metrics.RecordCompletion("resource-handler")
    }),
)
```

## Error Recovery Patterns

### 1. Compensation Pattern

Undo previous operations on failure:

```go
// Track operations for compensation
type Operation struct {
    Type   string
    Data   any
    Undo   func() error
}

// Node that tracks compensatable operations
createOrder := pocket.NewNode[OrderRequest, Order]("create-order",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        req OrderRequest, prep, order any) (Order, string, error) {
        
        // Track operation for compensation
        ops := []Operation{
            {
                Type: "create-order",
                Data: order,
                Undo: func() error {
                    return deleteOrder(order.(Order).ID)
                },
            },
        }
        
        store.Set(ctx, "compensation:ops", ops)
        return order.(Order), "charge-payment", nil
    }),
)

// Compensation handler
compensate := pocket.NewNode[any, any]("compensate",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
        ops, _ := store.Get(ctx, "compensation:ops")
        return ops, nil
    }),
    pocket.WithExec(func(ctx context.Context, ops any) (any, error) {
        operations := ops.([]Operation)
        
        // Undo in reverse order
        for i := len(operations) - 1; i >= 0; i-- {
            if err := operations[i].Undo(); err != nil {
                log.Printf("Compensation failed for %s: %v", operations[i].Type, err)
            }
        }
        
        return "compensated", nil
    }),
)
```

### 2. Retry with Jitter

Prevent thundering herd with jittered retries:

```go
func withJitteredRetry(maxAttempts int, baseDelay time.Duration) pocket.NodeOption {
    return pocket.WithExec(func(ctx context.Context, input any) (any, error) {
        var lastErr error
        
        for attempt := 1; attempt <= maxAttempts; attempt++ {
            result, err := attemptOperation(input)
            if err == nil {
                return result, nil
            }
            
            lastErr = err
            
            if attempt < maxAttempts {
                // Add jitter to prevent synchronized retries
                jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
                delay := baseDelay*time.Duration(attempt) + jitter
                
                select {
                case <-time.After(delay):
                    continue
                case <-ctx.Done():
                    return nil, ctx.Err()
                }
            }
        }
        
        return nil, fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
    })
}
```

### 3. Bulkhead Pattern

Isolate failures to prevent system-wide impact:

```go
// Create isolated pools for different operations
type BulkheadPool struct {
    semaphore chan struct{}
    name      string
}

func NewBulkheadPool(name string, size int) *BulkheadPool {
    return &BulkheadPool{
        name:      name,
        semaphore: make(chan struct{}, size),
    }
}

func (p *BulkheadPool) Execute(ctx context.Context, fn func() error) error {
    select {
    case p.semaphore <- struct{}{}:
        defer func() { <-p.semaphore }()
        return fn()
    case <-ctx.Done():
        return fmt.Errorf("bulkhead %s: context cancelled", p.name)
    default:
        return fmt.Errorf("bulkhead %s: pool exhausted", p.name)
    }
}

// Use in nodes
criticalPool := NewBulkheadPool("critical", 10)
normalPool := NewBulkheadPool("normal", 50)

criticalNode := pocket.NewNode[Request, Response]("critical-operation",
    pocket.WithExec(func(ctx context.Context, req Request) (Response, error) {
        return Response{}, criticalPool.Execute(ctx, func() error {
            return performCriticalOperation(req)
        })
    }),
)
```

## Error Routing

Route based on error types:

```go
processor := pocket.NewNode[Input, Output]("processor",
    pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
        return processInput(input)
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input Input, prep, exec any) (Output, string, error) {
        
        // Check if there was an error in exec
        if execErr := getExecError(); execErr != nil {
            switch {
            case isRetryable(execErr):
                return Output{}, "retry", execErr
            case isValidationError(execErr):
                return Output{}, "validation-failed", execErr
            case isAuthError(execErr):
                return Output{}, "auth-failed", execErr
            default:
                return Output{}, "general-error", execErr
            }
        }
        
        return exec.(Output), "success", nil
    }),
)

// Connect error handlers
processor.Connect("retry", retryHandler)
processor.Connect("validation-failed", validationErrorHandler)
processor.Connect("auth-failed", authErrorHandler)
processor.Connect("general-error", generalErrorHandler)
processor.Connect("success", successHandler)
```

## Testing Error Scenarios

### 1. Inject Failures

```go
func TestErrorHandling(t *testing.T) {
    // Create a node that fails predictably
    failingNode := pocket.NewNode[Input, Output]("failing",
        pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
            if input.ShouldFail {
                return Output{}, errors.New("injected failure")
            }
            return Output{Success: true}, nil
        }),
        pocket.WithFallback(func(ctx context.Context, input Input, err error) (Output, error) {
            return Output{Fallback: true}, nil
        }),
    )
    
    store := pocket.NewStore()
    graph := pocket.NewGraph(failingNode, store)
    
    // Test failure scenario
    result, err := graph.Run(context.Background(), Input{ShouldFail: true})
    assert.NoError(t, err)
    assert.True(t, result.(Output).Fallback)
}
```

### 2. Timeout Testing

```go
func TestTimeout(t *testing.T) {
    slowNode := pocket.NewNode[Input, Output]("slow",
        pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
            select {
            case <-time.After(5 * time.Second):
                return Output{}, nil
            case <-ctx.Done():
                return Output{}, ctx.Err()
            }
        }),
    )
    
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    
    store := pocket.NewStore()
    graph := pocket.NewGraph(slowNode, store)
    
    _, err := graph.Run(ctx, Input{})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "context deadline exceeded")
}
```

## Best Practices

### 1. Fail Fast for Unrecoverable Errors

```go
pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input Input) (any, error) {
    // Fail fast on critical configuration errors
    config, exists := store.Get(ctx, "required-config")
    if !exists {
        return nil, fmt.Errorf("critical: required configuration missing")
    }
    
    // But handle recoverable issues gracefully
    optional, exists := store.Get(ctx, "optional-config")
    if !exists {
        optional = defaultOptionalConfig()
    }
    
    return map[string]any{
        "config":   config,
        "optional": optional,
    }, nil
})
```

### 2. Provide Context in Error Messages

```go
pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
    result, err := processData(input)
    if err != nil {
        return Output{}, fmt.Errorf("failed to process data for user %s: %w", 
            input.UserID, err)
    }
    return result, nil
})
```

### 3. Use Structured Errors

```go
type ProcessingError struct {
    Op         string
    Input      any
    Err        error
    Retryable  bool
}

func (e *ProcessingError) Error() string {
    return fmt.Sprintf("%s failed: %v", e.Op, e.Err)
}

func (e *ProcessingError) Unwrap() error {
    return e.Err
}

// Usage
return Output{}, &ProcessingError{
    Op:        "transform",
    Input:     input,
    Err:       err,
    Retryable: true,
}
```

### 4. Monitor Error Rates

```go
errorMonitor := pocket.NewNode[Input, Output]("monitored",
    pocket.WithExec(processFunc),
    pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, err error) {
        // Track error metrics
        errorType := classifyError(err)
        metrics.IncrementCounter("errors", map[string]string{
            "node":  "monitored",
            "type":  errorType,
        })
        
        // Alert on high error rates
        if metrics.GetErrorRate("monitored") > 0.1 { // 10% error rate
            alerting.Send("High error rate in monitored node")
        }
    }),
)
```

### 5. Document Error Behavior

```go
// ProcessPaymentNode processes payment transactions.
//
// Errors:
//   - ErrInsufficientFunds: User lacks funds (non-retryable)
//   - ErrPaymentGatewayTimeout: Gateway timeout (retryable)
//   - ErrInvalidCard: Card validation failed (non-retryable)
//
// The node implements exponential backoff for retryable errors
// and falls back to alternative payment methods when available.
func ProcessPaymentNode() pocket.Node {
    // Implementation...
}
```

## Summary

Effective error handling in Pocket involves:

1. **Multiple strategies**: Fallbacks, retries, circuit breakers
2. **Lifecycle hooks**: Success, failure, and completion handlers
3. **Recovery patterns**: Compensation, bulkheads, jittered retries
4. **Error routing**: Different paths for different failure types
5. **Testing**: Inject failures to verify resilience

By combining these approaches, you can build workflows that gracefully handle failures and maintain system stability under adverse conditions.