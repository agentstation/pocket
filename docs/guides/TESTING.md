# Testing Guide

## Overview

Testing Pocket workflows involves validating individual nodes, testing graph connections, and ensuring workflows behave correctly under various conditions. This guide covers testing strategies, patterns, and best practices.

## Testing Individual Nodes

### Basic Node Testing

Test each phase of a node independently:

```go
func TestNode(t *testing.T) {
    // Create the node
    processor := pocket.NewNode[Input, Output]("processor",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input Input) (any, error) {
            config, _ := store.Get(ctx, "config")
            return map[string]any{
                "input":  input,
                "config": config,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (Output, error) {
            data := prepData.(map[string]any)
            return processData(data["input"].(Input), data["config"]), nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            input Input, prep, exec any) (Output, string, error) {
            store.Set(ctx, "result", exec)
            return exec.(Output), "next", nil
        }),
    )
    
    // Test with a mock store
    store := pocket.NewStore()
    store.Set(context.Background(), "config", testConfig)
    
    // Create and run graph
    graph := pocket.NewGraph(processor, store)
    result, err := graph.Run(context.Background(), Input{Value: "test"})
    
    assert.NoError(t, err)
    assert.Equal(t, expectedOutput, result)
    
    // Verify store state
    saved, exists := store.Get(context.Background(), "result")
    assert.True(t, exists)
    assert.Equal(t, result, saved)
}
```

### Testing Node Phases Separately

```go
func TestNodePrep(t *testing.T) {
    prepFunc := func(ctx context.Context, store pocket.StoreReader, input Input) (any, error) {
        if input.Value == "" {
            return nil, errors.New("value required")
        }
        config, _ := store.Get(ctx, "config")
        return map[string]any{
            "input":  input,
            "config": config,
        }, nil
    }
    
    // Test valid input
    store := pocket.NewStore()
    store.Set(context.Background(), "config", "test-config")
    
    result, err := prepFunc(context.Background(), store, Input{Value: "test"})
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    // Test invalid input
    _, err = prepFunc(context.Background(), store, Input{Value: ""})
    assert.Error(t, err)
}

func TestNodeExec(t *testing.T) {
    execFunc := func(ctx context.Context, prepData any) (Output, error) {
        data := prepData.(map[string]any)
        input := data["input"].(Input)
        
        if input.Value == "error" {
            return Output{}, errors.New("processing error")
        }
        
        return Output{
            Result: strings.ToUpper(input.Value),
        }, nil
    }
    
    // Test successful execution
    prepData := map[string]any{
        "input": Input{Value: "hello"},
    }
    
    result, err := execFunc(context.Background(), prepData)
    assert.NoError(t, err)
    assert.Equal(t, "HELLO", result.Result)
    
    // Test error case
    prepData["input"] = Input{Value: "error"}
    _, err = execFunc(context.Background(), prepData)
    assert.Error(t, err)
}
```

## Testing Type Safety

### Compile-Time Type Testing

```go
func TestTypeSafety(t *testing.T) {
    // This test verifies that type mismatches are caught
    userProcessor := pocket.NewNode[User, ProcessedUser]("process", 
        pocket.WithExec(func(ctx context.Context, user User) (ProcessedUser, error) {
            return ProcessedUser{
                ID:   user.ID,
                Name: strings.ToUpper(user.Name),
            }, nil
        }),
    )
    
    // This should work
    graph := pocket.NewGraph(userProcessor, pocket.NewStore())
    result, err := graph.Run(context.Background(), User{ID: "123", Name: "alice"})
    
    assert.NoError(t, err)
    processed := result.(ProcessedUser)
    assert.Equal(t, "ALICE", processed.Name)
    
    // This should fail at runtime with wrong input type
    _, err = graph.Run(context.Background(), "not a user")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid input type")
}
```

### ValidateGraph Testing

```go
func TestGraphValidation(t *testing.T) {
    // Create nodes with specific types
    fetchUser := pocket.NewNode[UserID, User]("fetch",
        pocket.WithExec(func(ctx context.Context, id UserID) (User, error) {
            return User{ID: string(id)}, nil
        }),
    )
    
    processUser := pocket.NewNode[User, ProcessedUser]("process",
        pocket.WithExec(func(ctx context.Context, user User) (ProcessedUser, error) {
            return ProcessedUser{User: user}, nil
        }),
    )
    
    // This should validate successfully
    fetchUser.Connect("default", processUser)
    err := pocket.ValidateGraph(fetchUser)
    assert.NoError(t, err)
    
    // Create incompatible node
    wrongNode := pocket.NewNode[Product, ProductResult]("wrong",
        pocket.WithExec(func(ctx context.Context, p Product) (ProductResult, error) {
            return ProductResult{}, nil
        }),
    )
    
    // This should fail validation
    fetchUser.Connect("error", wrongNode)
    err = pocket.ValidateGraph(fetchUser)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "type mismatch")
}
```

## Testing Workflows

### Integration Testing

Test complete workflows end-to-end:

```go
func TestCompleteWorkflow(t *testing.T) {
    // Build a multi-node workflow
    validate := pocket.NewNode[Order, ValidatedOrder]("validate",
        pocket.WithExec(validateOrder),
    )
    
    calculateTax := pocket.NewNode[ValidatedOrder, TaxedOrder]("tax",
        pocket.WithExec(func(ctx context.Context, order ValidatedOrder) (TaxedOrder, error) {
            return TaxedOrder{
                Order: order,
                Tax:   order.Total * 0.08,
            }, nil
        }),
    )
    
    processPayment := pocket.NewNode[TaxedOrder, Receipt]("payment",
        pocket.WithExec(func(ctx context.Context, order TaxedOrder) (Receipt, error) {
            return Receipt{
                OrderID: order.ID,
                Total:   order.Total + order.Tax,
                Status:  "paid",
            }, nil
        }),
    )
    
    // Connect nodes
    validate.Connect("valid", calculateTax)
    calculateTax.Connect("default", processPayment)
    
    // Test the workflow
    store := pocket.NewStore()
    graph := pocket.NewGraph(validate, store)
    
    testOrder := Order{
        ID:    "test-123",
        Items: []Item{{Name: "Widget", Price: 100}},
        Total: 100,
    }
    
    result, err := graph.Run(context.Background(), testOrder)
    assert.NoError(t, err)
    
    receipt := result.(Receipt)
    assert.Equal(t, "paid", receipt.Status)
    assert.Equal(t, 108.0, receipt.Total) // 100 + 8% tax
}
```

### Testing Error Paths

```go
func TestErrorHandling(t *testing.T) {
    // Node that can fail
    riskyNode := pocket.NewNode[Request, Response]("risky",
        pocket.WithExec(func(ctx context.Context, req Request) (Response, error) {
            if req.ShouldFail {
                return Response{}, errors.New("simulated failure")
            }
            return Response{Success: true}, nil
        }),
        pocket.WithFallback(func(ctx context.Context, req Request, err error) (Response, error) {
            return Response{
                Success:  false,
                Fallback: true,
                Error:    err.Error(),
            }, nil
        }),
    )
    
    errorHandler := pocket.NewNode[Response, FinalResult]("error-handler",
        pocket.WithExec(func(ctx context.Context, resp Response) (FinalResult, error) {
            return FinalResult{
                Handled: true,
                Message: "Error was handled",
            }, nil
        }),
    )
    
    successHandler := pocket.NewNode[Response, FinalResult]("success-handler",
        pocket.WithExec(func(ctx context.Context, resp Response) (FinalResult, error) {
            return FinalResult{
                Handled: true,
                Message: "Success",
            }, nil
        }),
    )
    
    // Route based on success/failure
    riskyNode.Connect("error", errorHandler)
    riskyNode.Connect("success", successHandler)
    
    graph := pocket.NewGraph(riskyNode, pocket.NewStore())
    
    // Test failure path
    failResult, err := graph.Run(context.Background(), Request{ShouldFail: true})
    assert.NoError(t, err) // Fallback prevents error
    
    // Test success path
    successResult, err := graph.Run(context.Background(), Request{ShouldFail: false})
    assert.NoError(t, err)
    assert.True(t, successResult.(Response).Success)
}
```

## Testing Patterns

### Table-Driven Tests

```go
func TestProcessor(t *testing.T) {
    processor := createProcessorNode()
    
    tests := []struct {
        name     string
        input    Input
        setup    func(*pocket.Store)
        expected Output
        wantErr  bool
    }{
        {
            name:  "valid input",
            input: Input{Value: "test"},
            setup: func(s *pocket.Store) {
                s.Set(context.Background(), "config", "default")
            },
            expected: Output{Result: "TEST"},
            wantErr:  false,
        },
        {
            name:  "empty input",
            input: Input{Value: ""},
            setup: func(s *pocket.Store) {},
            expected: Output{},
            wantErr:  true,
        },
        {
            name:  "with special config",
            input: Input{Value: "test"},
            setup: func(s *pocket.Store) {
                s.Set(context.Background(), "config", "special")
            },
            expected: Output{Result: "SPECIAL:TEST"},
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            store := pocket.NewStore()
            if tt.setup != nil {
                tt.setup(store)
            }
            
            graph := pocket.NewGraph(processor, store)
            result, err := graph.Run(context.Background(), tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Mocking External Dependencies

```go
type ExternalService interface {
    FetchData(id string) (Data, error)
}

type MockService struct {
    mock.Mock
}

func (m *MockService) FetchData(id string) (Data, error) {
    args := m.Called(id)
    return args.Get(0).(Data), args.Error(1)
}

func TestWithMockService(t *testing.T) {
    mockService := new(MockService)
    
    // Create node with injected service
    fetchNode := pocket.NewNode[string, Data]("fetch",
        pocket.WithExec(func(ctx context.Context, id string) (Data, error) {
            return mockService.FetchData(id)
        }),
    )
    
    // Set up mock expectations
    expectedData := Data{ID: "123", Value: "test"}
    mockService.On("FetchData", "123").Return(expectedData, nil)
    mockService.On("FetchData", "404").Return(Data{}, errors.New("not found"))
    
    graph := pocket.NewGraph(fetchNode, pocket.NewStore())
    
    // Test successful fetch
    result, err := graph.Run(context.Background(), "123")
    assert.NoError(t, err)
    assert.Equal(t, expectedData, result)
    
    // Test error case
    _, err = graph.Run(context.Background(), "404")
    assert.Error(t, err)
    
    mockService.AssertExpectations(t)
}
```

### Testing Concurrent Workflows

```go
func TestConcurrentExecution(t *testing.T) {
    // Create a node that tracks concurrent executions
    var activeCount int32
    var maxActive int32
    
    concurrentNode := pocket.NewNode[int, int]("concurrent",
        pocket.WithExec(func(ctx context.Context, input int) (int, error) {
            // Track concurrent executions
            current := atomic.AddInt32(&activeCount, 1)
            defer atomic.AddInt32(&activeCount, -1)
            
            // Update max if needed
            for {
                max := atomic.LoadInt32(&maxActive)
                if current <= max || atomic.CompareAndSwapInt32(&maxActive, max, current) {
                    break
                }
            }
            
            // Simulate work
            time.Sleep(10 * time.Millisecond)
            
            return input * 2, nil
        }),
    )
    
    // Run multiple workflows concurrently
    store := pocket.NewStore()
    graph := pocket.NewGraph(concurrentNode, store)
    
    var wg sync.WaitGroup
    results := make([]int, 10)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            result, err := graph.Run(context.Background(), idx)
            assert.NoError(t, err)
            results[idx] = result.(int)
        }(i)
    }
    
    wg.Wait()
    
    // Verify results
    for i := 0; i < 10; i++ {
        assert.Equal(t, i*2, results[i])
    }
    
    // Verify concurrent execution occurred
    assert.Greater(t, int(maxActive), 1, "Expected concurrent execution")
}
```

## Testing State Management

### Store Isolation Testing

```go
func TestStoreIsolation(t *testing.T) {
    parentStore := pocket.NewStore()
    
    // Create scoped stores
    workflow1Store := parentStore.Scope("workflow1")
    workflow2Store := parentStore.Scope("workflow2")
    
    // Node that uses store
    storeNode := pocket.NewNode[string, string]("store-user",
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            input string, prep, exec any) (string, string, error) {
            
            store.Set(ctx, "data", input)
            return input, "done", nil
        }),
    )
    
    // Run workflows with different stores
    graph1 := pocket.NewGraph(storeNode, workflow1Store)
    graph2 := pocket.NewGraph(storeNode, workflow2Store)
    
    graph1.Run(context.Background(), "data1")
    graph2.Run(context.Background(), "data2")
    
    // Verify isolation
    data1, exists1 := workflow1Store.Get(context.Background(), "data")
    data2, exists2 := workflow2Store.Get(context.Background(), "data")
    
    assert.True(t, exists1)
    assert.True(t, exists2)
    assert.Equal(t, "data1", data1)
    assert.Equal(t, "data2", data2)
    
    // Verify parent store has both
    parentData1, _ := parentStore.Get(context.Background(), "workflow1:data")
    parentData2, _ := parentStore.Get(context.Background(), "workflow2:data")
    assert.Equal(t, "data1", parentData1)
    assert.Equal(t, "data2", parentData2)
}
```

### Testing Bounded Stores

```go
func TestBoundedStore(t *testing.T) {
    evictedKeys := []string{}
    
    store := pocket.NewStore(
        pocket.WithMaxEntries(3),
        pocket.WithEvictionCallback(func(key string, value any) {
            evictedKeys = append(evictedKeys, key)
        }),
    )
    
    // Add entries up to limit
    store.Set(context.Background(), "1", "one")
    store.Set(context.Background(), "2", "two")
    store.Set(context.Background(), "3", "three")
    
    assert.Len(t, evictedKeys, 0)
    
    // Add one more - should trigger eviction
    store.Set(context.Background(), "4", "four")
    
    assert.Len(t, evictedKeys, 1)
    assert.Equal(t, "1", evictedKeys[0]) // LRU eviction
    
    // Verify remaining entries
    _, exists1 := store.Get(context.Background(), "1")
    _, exists4 := store.Get(context.Background(), "4")
    
    assert.False(t, exists1) // Evicted
    assert.True(t, exists4)   // Still present
}
```

## Performance Testing

### Benchmarking Nodes

```go
func BenchmarkNode(b *testing.B) {
    processor := pocket.NewNode[Input, Output]("bench",
        pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
            // Simulate some work
            return Output{
                Result: strings.ToUpper(input.Value),
            }, nil
        }),
    )
    
    store := pocket.NewStore()
    graph := pocket.NewGraph(processor, store)
    input := Input{Value: "benchmark test string"}
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := graph.Run(context.Background(), input)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkConcurrentWorkflows(b *testing.B) {
    processor := createProcessorNode()
    store := pocket.NewStore()
    graph := pocket.NewGraph(processor, store)
    
    b.RunParallel(func(pb *testing.PB) {
        input := Input{Value: "test"}
        for pb.Next() {
            _, err := graph.Run(context.Background(), input)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

### Load Testing

```go
func TestHighLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    processor := createProcessorNode()
    store := pocket.NewStore(
        pocket.WithMaxEntries(10000),
        pocket.WithTTL(time.Minute),
    )
    graph := pocket.NewGraph(processor, store)
    
    // Track metrics
    var successCount int64
    var errorCount int64
    startTime := time.Now()
    
    // Run many concurrent requests
    var wg sync.WaitGroup
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            input := Input{Value: fmt.Sprintf("test-%d", id)}
            _, err := graph.Run(context.Background(), input)
            
            if err != nil {
                atomic.AddInt64(&errorCount, 1)
            } else {
                atomic.AddInt64(&successCount, 1)
            }
        }(i)
    }
    
    wg.Wait()
    duration := time.Since(startTime)
    
    // Report results
    t.Logf("Load test completed in %v", duration)
    t.Logf("Success: %d, Errors: %d", successCount, errorCount)
    t.Logf("Throughput: %.2f requests/second", 
        float64(successCount+errorCount)/duration.Seconds())
    
    // Verify acceptable error rate
    errorRate := float64(errorCount) / float64(successCount+errorCount)
    assert.Less(t, errorRate, 0.01, "Error rate should be less than 1%")
}
```

## Best Practices

### 1. Test at Multiple Levels

```go
// Unit test individual functions
func TestProcessingLogic(t *testing.T) {
    result := processData(testInput)
    assert.Equal(t, expectedOutput, result)
}

// Integration test nodes
func TestNodeIntegration(t *testing.T) {
    node := createNode()
    graph := pocket.NewGraph(node, pocket.NewStore())
    // Test node in context
}

// End-to-end test workflows
func TestWorkflowE2E(t *testing.T) {
    workflow := buildCompleteWorkflow()
    // Test entire workflow
}
```

### 2. Use Test Fixtures

```go
func setupTestData(t *testing.T) (*pocket.Store, Input) {
    store := pocket.NewStore()
    
    // Set up common test data
    store.Set(context.Background(), "config", TestConfig{
        Timeout: time.Second,
        Retries: 3,
    })
    
    input := Input{
        ID:    "test-123",
        Value: "test data",
    }
    
    return store, input
}

func TestWithFixtures(t *testing.T) {
    store, input := setupTestData(t)
    
    // Use in test...
}
```

### 3. Test Error Conditions

```go
func TestAllErrorPaths(t *testing.T) {
    tests := []struct {
        name        string
        input       Input
        prepareErr  error
        execErr     error
        expectedErr string
    }{
        {
            name:        "prep failure",
            input:       Input{},
            prepareErr:  errors.New("prep failed"),
            expectedErr: "prep failed",
        },
        {
            name:        "exec failure",
            input:       Input{Value: "test"},
            execErr:     errors.New("exec failed"),
            expectedErr: "exec failed",
        },
    }
    
    // Test each error condition...
}
```

### 4. Use Context for Timeouts

```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    slowNode := pocket.NewNode[Input, Output]("slow",
        pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
            select {
            case <-time.After(time.Second):
                return Output{}, errors.New("should not reach here")
            case <-ctx.Done():
                return Output{}, ctx.Err()
            }
        }),
    )
    
    graph := pocket.NewGraph(slowNode, pocket.NewStore())
    _, err := graph.Run(ctx, Input{})
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "deadline exceeded")
}
```

## Summary

Testing Pocket workflows effectively requires:

1. **Unit testing** individual node phases
2. **Integration testing** complete workflows
3. **Type safety verification** with ValidateGraph
4. **Error path testing** for resilience
5. **Performance testing** for production readiness
6. **Proper mocking** of external dependencies

By following these patterns and practices, you can build reliable, well-tested workflows that behave predictably in production.