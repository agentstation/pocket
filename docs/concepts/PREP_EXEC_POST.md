# The Prep/Exec/Post Pattern

## Introduction

The Prep/Exec/Post pattern is the heart of Pocket's execution model. Every node in a Pocket workflow follows this three-phase lifecycle, with an optional Fallback phase for error recovery, ensuring clean separation of concerns and predictable execution.

## The Phases

### 1. Prep Phase - Preparation and Validation

The Prep phase is responsible for:
- **Reading state** from the store (read-only access)
- **Validating input** data
- **Preparing data** for the execution phase
- **Gathering context** needed for processing

```go
func (n *node) Prep(ctx context.Context, store StoreReader, input any) (prepResult any, err error) {
    // Read configuration or state
    config, _ := store.Get(ctx, "config")
    
    // Validate input
    if !isValid(input) {
        return nil, errors.New("invalid input")
    }
    
    // Prepare data for execution
    return map[string]any{
        "input":  input,
        "config": config,
    }, nil
}
```

**Key constraint**: The store parameter is `StoreReader`, providing only read access. This enforces that Prep cannot modify state.

### 2. Exec Phase - Pure Business Logic

The Exec phase contains:
- **Pure functions** with no side effects
- **Business logic** implementation
- **Data transformation** and processing
- **Decision making** based on inputs

```go
func (n *node) Exec(ctx context.Context, prepResult any) (execResult any, err error) {
    // Pure business logic - no store access
    data := prepResult.(map[string]any)
    
    // Process data
    result := processBusinessLogic(data["input"], data["config"])
    
    // Return results
    return result, nil
}
```

**Key constraint**: No store access at all. This enforces functional purity and makes the business logic easily testable.

### 3. Post Phase - State Updates and Routing

The Post phase handles:
- **Writing results** to the store
- **Updating state** based on execution results
- **Routing decisions** for the next node
- **Cleanup operations** if needed

```go
func (n *node) Post(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error) {
    // Write results to store
    store.Set(ctx, "lastResult", execResult)
    
    // Determine routing based on results
    if execResult.(Result).Success {
        return execResult, "success", nil
    }
    
    return execResult, "failure", nil
}
```

**Key capability**: Full read/write access to the store via `StoreWriter`.

### 4. Fallback Phase (Optional) - Error Recovery

The Fallback phase provides graceful error handling when Exec fails:
- **Receives prepResult** (not the original input) for consistency with Exec
- **Handles errors** from the Exec phase
- **Provides alternative results** when primary logic fails
- **No store access** to maintain purity like Exec

```go
// Fallback is part of the Steps struct
pocket.Steps{
    Exec: func(ctx context.Context, prepResult any) (any, error) {
        // Primary logic that might fail
        data := prepResult.(PreparedData)
        return riskyOperation(data)
    },
    Fallback: func(ctx context.Context, prepResult any, execErr error) (any, error) {
        // Fallback receives same prepResult as Exec
        data := prepResult.(PreparedData)
        
        log.Printf("Primary operation failed: %v, using fallback", execErr)
        
        // Return safe default based on prepared data
        return SafeResult{
            Value: data.DefaultValue,
            Source: "fallback",
            OriginalError: execErr.Error(),
        }, nil
    },
}
```

**Key constraints**: 
- Fallback receives `prepResult`, not the original input
- No store access (like Exec) to maintain functional purity
- Only runs if Exec returns an error

## Why This Pattern?

### 1. Separation of Concerns

Each phase has a single, clear responsibility:
- **Prep**: Gather what you need
- **Exec**: Do the work
- **Post**: Handle the results

This separation makes code easier to understand, test, and maintain.

### 2. Testability

The Exec phase is pure, making it trivial to test:

```go
func TestBusinessLogic(t *testing.T) {
    // No mocking needed - just call the function
    input := PreparedData{Value: 42}
    result, err := processBusinessLogic(input)
    
    assert.NoError(t, err)
    assert.Equal(t, 84, result.Value)
}
```

### 3. Predictable State Management

State mutations only occur in designated phases:
- **Prep**: Read-only access prevents accidental mutations
- **Exec**: No access enforces purity
- **Post**: Explicit write access for intentional changes

### 4. Concurrency Safety

The pattern enables safe concurrent execution:
- Multiple Prep phases can run in parallel (read-only)
- Exec phases have no shared state
- Post phases can be synchronized when needed

## Common Patterns

### 1. Configuration Loading

```go
pocket.WithPrep(func(ctx context.Context, store StoreReader, input any) (any, error) {
    // Load configuration during prep
    config, exists := store.Get(ctx, "app:config")
    if !exists {
        config = defaultConfig()
    }
    
    return map[string]any{
        "input":  input,
        "config": config,
    }, nil
})
```

### 2. Validation and Enrichment

```go
pocket.WithPrep(func(ctx context.Context, store StoreReader, input any) (any, error) {
    user := input.(User)
    
    // Validate
    if user.Email == "" {
        return nil, errors.New("email required")
    }
    
    // Enrich with additional data
    profile, _ := store.Get(ctx, "profile:" + user.ID)
    
    return EnrichedUser{
        User:    user,
        Profile: profile,
    }, nil
})
```

### 3. Pure Processing

```go
pocket.WithExec(func(ctx context.Context, prepResult any) (any, error) {
    data := prepResult.(EnrichedUser)
    
    // Pure business logic
    score := calculateRiskScore(data.User, data.Profile)
    recommendation := generateRecommendation(score)
    
    return ProcessResult{
        Score:          score,
        Recommendation: recommendation,
    }, nil
})
```

### 4. Conditional Routing

```go
pocket.WithPost(func(ctx context.Context, store StoreWriter, input, prep, exec any) (any, string, error) {
    result := exec.(ProcessResult)
    
    // Save results
    store.Set(ctx, "lastScore", result.Score)
    
    // Route based on score
    switch {
    case result.Score > 80:
        return result, "high-risk", nil
    case result.Score > 50:
        return result, "medium-risk", nil
    default:
        return result, "low-risk", nil
    }
})
```

### 5. Error Recovery with Fallback

```go
pocket.NewNode[Request, Response]("api-call",
    pocket.Steps{
        Prep: func(ctx context.Context, store StoreReader, req Request) (any, error) {
            // Prepare request with retry count
            retries, _ := store.Get(ctx, "retries:" + req.ID)
            return map[string]any{
                "request": req,
                "retries": retries,
                "timeout": 5 * time.Second,
            }, nil
        },
        Exec: func(ctx context.Context, prepResult any) (any, error) {
            // Primary API call that might fail
            data := prepResult.(map[string]any)
            req := data["request"].(Request)
            timeout := data["timeout"].(time.Duration)
            
            return callExternalAPI(req, timeout)
        },
        Fallback: func(ctx context.Context, prepResult any, err error) (any, error) {
            // Fallback receives prepResult, not original input
            data := prepResult.(map[string]any)
            req := data["request"].(Request)
            
            log.Printf("API call failed for %s: %v", req.ID, err)
            
            // Return cached or default response
            return Response{
                ID:     req.ID,
                Status: "fallback",
                Data:   getDefaultData(),
                Error:  err.Error(),
            }, nil
        },
    },
)
```

## Best Practices

### 1. Keep Prep Lightweight

The Prep phase should focus on gathering data, not heavy processing:

```go
// Good: Quick data gathering
pocket.WithPrep(func(ctx context.Context, store StoreReader, input any) (any, error) {
    config, _ := store.Get(ctx, "config")
    return struct {
        Input  any
        Config any
    }{input, config}, nil
})

// Avoid: Heavy processing in Prep
pocket.WithPrep(func(ctx context.Context, store StoreReader, input any) (any, error) {
    // Don't do complex calculations here
    result := expensiveCalculation(input)
    return result, nil
})
```

### 2. Keep Exec Pure

The Exec phase should have no side effects:

```go
// Good: Pure function
pocket.WithExec(func(ctx context.Context, prep any) (any, error) {
    return transform(prep), nil
})

// Avoid: Side effects in Exec
pocket.WithExec(func(ctx context.Context, prep any) (any, error) {
    // Don't do I/O operations here
    saveToDatabase(prep)  // Wrong!
    return prep, nil
})
```

### 3. Use Post for All State Changes

Consolidate state mutations in the Post phase:

```go
pocket.WithPost(func(ctx context.Context, store StoreWriter, input, prep, exec any) (any, string, error) {
    // All state changes happen here
    store.Set(ctx, "processed", exec)
    store.Set(ctx, "timestamp", time.Now())
    
    // Clear temporary data
    store.Delete(ctx, "temp:processing")
    
    return exec, "next", nil
})
```

### 4. Handle Errors Appropriately

Each phase can return errors with different implications:

```go
// Prep error: Invalid input or missing configuration
pocket.WithPrep(func(ctx context.Context, store StoreReader, input any) (any, error) {
    if input == nil {
        return nil, errors.New("input required")
    }
    return input, nil
})

// Exec error: Business logic failure
pocket.WithExec(func(ctx context.Context, prep any) (any, error) {
    result, err := process(prep)
    if err != nil {
        return nil, fmt.Errorf("processing failed: %w", err)
    }
    return result, nil
})

// Post error: State update or routing failure
pocket.WithPost(func(ctx context.Context, store StoreWriter, input, prep, exec any) (any, string, error) {
    if err := store.Set(ctx, "result", exec); err != nil {
        return nil, "", fmt.Errorf("failed to save result: %w", err)
    }
    return exec, "next", nil
})
```

## Advanced Usage

### 1. Skipping Phases

You can skip phases by not providing handlers:

```go
// Exec-only node (common for simple transformations)
simple := pocket.NewNode[string, string]("uppercase",
    pocket.WithExec(func(ctx context.Context, input string) (string, error) {
        return strings.ToUpper(input), nil
    }),
)
```

### 2. Complex State Management

Use Prep and Post together for sophisticated state handling:

```go
complex := pocket.NewNode[Request, Response]("processor",
    pocket.WithPrep(func(ctx context.Context, store StoreReader, req Request) (any, error) {
        // Check if we have cached results
        if cached, exists := store.Get(ctx, "cache:" + req.ID); exists {
            return map[string]any{
                "request": req,
                "cached":  cached,
            }, nil
        }
        
        return map[string]any{
            "request": req,
            "cached":  nil,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prep any) (Response, error) {
        data := prep.(map[string]any)
        
        // Return cached if available
        if cached := data["cached"]; cached != nil {
            return cached.(Response), nil
        }
        
        // Otherwise process
        return processRequest(data["request"].(Request)), nil
    }),
    pocket.WithPost(func(ctx context.Context, store StoreWriter, req Request, prep any, resp Response) (Response, string, error) {
        data := prep.(map[string]any)
        
        // Cache new results
        if data["cached"] == nil {
            store.Set(ctx, "cache:" + req.ID, resp)
        }
        
        return resp, "done", nil
    }),
)
```

### 3. Conditional Execution

Use Prep to determine if Exec should run:

```go
conditional := pocket.NewNode[Task, Result]("conditional",
    pocket.WithPrep(func(ctx context.Context, store StoreReader, task Task) (any, error) {
        // Check if task should be processed
        if !shouldProcess(task) {
            return map[string]any{
                "skip": true,
                "task": task,
            }, nil
        }
        
        return map[string]any{
            "skip": false,
            "task": task,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prep any) (Result, error) {
        data := prep.(map[string]any)
        
        if data["skip"].(bool) {
            return Result{Skipped: true}, nil
        }
        
        return processTask(data["task"].(Task)), nil
    }),
)
```

## Summary

The Prep/Exec/Post pattern (with optional Fallback) provides:

1. **Clear separation** of read, compute, write, and error-recovery operations
2. **Enhanced testability** through pure business logic in Exec and Fallback
3. **Predictable state management** with controlled mutations in designated phases
4. **Improved concurrency** through phase-based isolation
5. **Better maintainability** with organized code structure
6. **Graceful error handling** through the optional Fallback phase

This pattern is fundamental to Pocket's design philosophy: make the complex simple by breaking it into well-defined, composable parts.