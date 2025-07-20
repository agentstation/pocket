# Concurrency Patterns

## Overview

Pocket leverages Go's concurrency primitives to provide powerful patterns for parallel execution. This guide covers fan-out, fan-in, pipeline, and other concurrent processing patterns.

## Built-in Concurrency Functions

### Fan-Out Pattern

Distribute work across multiple concurrent processors:

```go
// Process items in parallel
items := []string{"item1", "item2", "item3", "item4", "item5"}

processor := pocket.NewNode[string, ProcessedItem]("processor",
    pocket.WithExec(func(ctx context.Context, item string) (ProcessedItem, error) {
        // Simulate processing
        time.Sleep(100 * time.Millisecond)
        return ProcessedItem{
            Original: item,
            Result:   strings.ToUpper(item),
        }, nil
    }),
)

// Fan-out processing
results, err := pocket.FanOut(ctx, processor, store, items)
if err != nil {
    log.Fatal(err)
}

// Results are in the same order as inputs
for i, result := range results {
    fmt.Printf("Item %s -> %s\n", items[i], result.Result)
}
```

### Fan-In Pattern

Aggregate results from multiple sources:

```go
// Multiple source nodes
source1 := pocket.NewNode[any, []Data]("source1",
    pocket.WithExec(func(ctx context.Context, _ any) ([]Data, error) {
        return fetchFromSource1()
    }),
)

source2 := pocket.NewNode[any, []Data]("source2",
    pocket.WithExec(func(ctx context.Context, _ any) ([]Data, error) {
        return fetchFromSource2()
    }),
)

source3 := pocket.NewNode[any, []Data]("source3",
    pocket.WithExec(func(ctx context.Context, _ any) ([]Data, error) {
        return fetchFromSource3()
    }),
)

// Aggregator node
aggregator := pocket.NewNode[[][]Data, AggregatedResult]("aggregator",
    pocket.WithExec(func(ctx context.Context, allData [][]Data) (AggregatedResult, error) {
        var combined []Data
        for _, data := range allData {
            combined = append(combined, data...)
        }
        
        return AggregatedResult{
            TotalItems: len(combined),
            Data:       combined,
        }, nil
    }),
)

// Create fan-in
fanIn := pocket.NewFanIn(aggregator, source1, source2, source3)
result, err := fanIn.Run(ctx, store)
```

### Pipeline Pattern

Chain operations where each output feeds the next input:

```go
// Pipeline stages
fetch := pocket.NewNode[URL, RawData]("fetch",
    pocket.WithExec(func(ctx context.Context, url URL) (RawData, error) {
        return fetchData(url)
    }),
)

parse := pocket.NewNode[RawData, ParsedData]("parse",
    pocket.WithExec(func(ctx context.Context, raw RawData) (ParsedData, error) {
        return parseData(raw)
    }),
)

transform := pocket.NewNode[ParsedData, TransformedData]("transform",
    pocket.WithExec(func(ctx context.Context, parsed ParsedData) (TransformedData, error) {
        return transformData(parsed)
    }),
)

save := pocket.NewNode[TransformedData, SaveResult]("save",
    pocket.WithExec(func(ctx context.Context, data TransformedData) (SaveResult, error) {
        return saveData(data)
    }),
)

// Execute pipeline
nodes := []pocket.Node{fetch, parse, transform, save}
result, err := pocket.Pipeline(ctx, nodes, store, URL("https://example.com/data"))
```

### RunConcurrent Pattern

Execute independent nodes in parallel:

```go
// Independent operations
checkInventory := pocket.NewNode[Order, InventoryStatus]("check-inventory",
    pocket.WithExec(checkInventoryFunc),
)

validatePayment := pocket.NewNode[Order, PaymentStatus]("validate-payment",
    pocket.WithExec(validatePaymentFunc),
)

checkShipping := pocket.NewNode[Order, ShippingStatus]("check-shipping",
    pocket.WithExec(checkShippingFunc),
)

// Run all checks concurrently
nodes := []pocket.Node{checkInventory, validatePayment, checkShipping}
results, err := pocket.RunConcurrent(ctx, nodes, store)

// Process results
inventoryStatus := results[0].(InventoryStatus)
paymentStatus := results[1].(PaymentStatus)
shippingStatus := results[2].(ShippingStatus)
```

## Custom Concurrency Patterns

### Worker Pool Pattern

Limit concurrent executions with a worker pool:

```go
type WorkerPool struct {
    workers   int
    taskQueue chan Task
    results   chan Result
}

func NewWorkerPool(workers int) *WorkerPool {
    return &WorkerPool{
        workers:   workers,
        taskQueue: make(chan Task, workers*2),
        results:   make(chan Result, workers*2),
    }
}

func (p *WorkerPool) Start(ctx context.Context, processor pocket.Node) {
    var wg sync.WaitGroup
    
    // Start workers
    for i := 0; i < p.workers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for {
                select {
                case task, ok := <-p.taskQueue:
                    if !ok {
                        return
                    }
                    
                    // Process task
                    result, err := processor.Exec(ctx, task)
                    p.results <- Result{
                        Task:   task,
                        Output: result,
                        Error:  err,
                    }
                    
                case <-ctx.Done():
                    return
                }
            }
        }(i)
    }
    
    // Wait for completion
    go func() {
        wg.Wait()
        close(p.results)
    }()
}

// Usage
pool := NewWorkerPool(5) // 5 concurrent workers
processor := createProcessorNode()

ctx := context.Background()
pool.Start(ctx, processor)

// Submit tasks
for _, task := range tasks {
    pool.taskQueue <- task
}
close(pool.taskQueue)

// Collect results
var results []Result
for result := range pool.results {
    results = append(results, result)
}
```

### Scatter-Gather Pattern

Send requests to multiple services and gather responses:

```go
func ScatterGather(ctx context.Context, request Request, services []ServiceNode) ([]Response, error) {
    responses := make([]Response, len(services))
    errors := make([]error, len(services))
    
    var wg sync.WaitGroup
    
    for i, service := range services {
        wg.Add(1)
        go func(idx int, svc ServiceNode) {
            defer wg.Done()
            
            // Create timeout context for each service
            svcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
            defer cancel()
            
            result, err := svc.Exec(svcCtx, request)
            if err != nil {
                errors[idx] = err
                return
            }
            
            responses[idx] = result.(Response)
        }(i, service)
    }
    
    wg.Wait()
    
    // Check for errors
    var errs []error
    for i, err := range errors {
        if err != nil {
            errs = append(errs, fmt.Errorf("service %d: %w", i, err))
        }
    }
    
    if len(errs) > 0 {
        return responses, fmt.Errorf("scatter-gather errors: %v", errs)
    }
    
    return responses, nil
}
```

### Rate-Limited Concurrency

Control request rate while maintaining concurrency:

```go
type RateLimitedProcessor struct {
    processor pocket.Node
    limiter   *rate.Limiter
}

func (p *RateLimitedProcessor) ProcessBatch(ctx context.Context, items []Item) ([]Result, error) {
    results := make([]Result, len(items))
    errors := make([]error, len(items))
    
    var wg sync.WaitGroup
    
    for i, item := range items {
        wg.Add(1)
        
        // Wait for rate limit
        if err := p.limiter.Wait(ctx); err != nil {
            errors[i] = err
            wg.Done()
            continue
        }
        
        go func(idx int, data Item) {
            defer wg.Done()
            
            result, err := p.processor.Exec(ctx, data)
            if err != nil {
                errors[idx] = err
                return
            }
            
            results[idx] = result.(Result)
        }(i, item)
    }
    
    wg.Wait()
    
    // Return results with any errors
    return results, combineErrors(errors)
}

// Usage
processor := RateLimitedProcessor{
    processor: createProcessorNode(),
    limiter:   rate.NewLimiter(rate.Limit(10), 1), // 10 requests per second
}

results, err := processor.ProcessBatch(ctx, items)
```

### Batch Processing with Concurrency

Process items in batches with controlled concurrency:

```go
func ProcessInBatches(ctx context.Context, items []Item, batchSize int, processor pocket.Node) error {
    store := pocket.NewStore()
    
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        batch := items[i:end]
        
        // Process batch concurrently
        results, err := pocket.FanOut(ctx, processor, store, batch)
        if err != nil {
            return fmt.Errorf("batch %d failed: %w", i/batchSize, err)
        }
        
        // Handle results
        for j, result := range results {
            log.Printf("Processed item %d: %v", i+j, result)
        }
        
        // Optional: Add delay between batches
        if end < len(items) {
            time.Sleep(100 * time.Millisecond)
        }
    }
    
    return nil
}
```

## Advanced Patterns

### Dynamic Fan-Out

Adjust concurrency based on system load:

```go
type DynamicFanOut struct {
    minWorkers int
    maxWorkers int
    load       *LoadMonitor
}

func (d *DynamicFanOut) Process(ctx context.Context, items []Item, processor pocket.Node) ([]Result, error) {
    // Determine optimal worker count
    currentLoad := d.load.GetSystemLoad()
    workerCount := d.calculateWorkers(currentLoad, len(items))
    
    log.Printf("Processing %d items with %d workers (load: %.2f)", 
        len(items), workerCount, currentLoad)
    
    // Create channels
    taskChan := make(chan Item, len(items))
    resultChan := make(chan IndexedResult, len(items))
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < workerCount; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            for item := range taskChan {
                result, err := processor.Exec(ctx, item)
                resultChan <- IndexedResult{
                    Index:  item.Index,
                    Result: result,
                    Error:  err,
                }
            }
        }()
    }
    
    // Send tasks
    for i, item := range items {
        item.Index = i
        taskChan <- item
    }
    close(taskChan)
    
    // Wait for workers
    go func() {
        wg.Wait()
        close(resultChan)
    }()
    
    // Collect results
    results := make([]Result, len(items))
    for res := range resultChan {
        results[res.Index] = res.Result
    }
    
    return results, nil
}

func (d *DynamicFanOut) calculateWorkers(load float64, itemCount int) int {
    // High load: use minimum workers
    if load > 0.8 {
        return d.minWorkers
    }
    
    // Low load: scale up workers
    if load < 0.3 {
        desired := itemCount / 10
        if desired > d.maxWorkers {
            return d.maxWorkers
        }
        if desired < d.minWorkers {
            return d.minWorkers
        }
        return desired
    }
    
    // Medium load: proportional scaling
    scale := 1.0 - load
    workers := int(float64(d.minWorkers) + scale*float64(d.maxWorkers-d.minWorkers))
    return workers
}
```

### Concurrent Graph Execution

Execute independent subgraphs concurrently:

```go
func ExecuteSubgraphsConcurrently(ctx context.Context, subgraphs map[string]*pocket.Graph, inputs map[string]any) (map[string]any, error) {
    results := make(map[string]any)
    var mu sync.Mutex
    var wg sync.WaitGroup
    
    errChan := make(chan error, len(subgraphs))
    
    for name, graph := range subgraphs {
        wg.Add(1)
        go func(graphName string, g *pocket.Graph, input any) {
            defer wg.Done()
            
            result, err := g.Run(ctx, input)
            if err != nil {
                errChan <- fmt.Errorf("subgraph %s: %w", graphName, err)
                return
            }
            
            mu.Lock()
            results[graphName] = result
            mu.Unlock()
        }(name, graph, inputs[name])
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return results, fmt.Errorf("subgraph errors: %v", errs)
    }
    
    return results, nil
}
```

### Stream Processing

Process continuous streams with back-pressure:

```go
type StreamProcessor struct {
    processor   pocket.Node
    bufferSize  int
    concurrency int
}

func (s *StreamProcessor) ProcessStream(ctx context.Context, input <-chan Item) <-chan Result {
    output := make(chan Result, s.bufferSize)
    
    // Create worker pool
    workers := make(chan struct{}, s.concurrency)
    for i := 0; i < s.concurrency; i++ {
        workers <- struct{}{}
    }
    
    go func() {
        defer close(output)
        
        for {
            select {
            case item, ok := <-input:
                if !ok {
                    return
                }
                
                // Wait for available worker
                <-workers
                
                go func(data Item) {
                    defer func() { workers <- struct{}{} }()
                    
                    result, err := s.processor.Exec(ctx, data)
                    
                    select {
                    case output <- Result{Data: result, Error: err}:
                    case <-ctx.Done():
                        return
                    }
                }(item)
                
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return output
}

// Usage
processor := &StreamProcessor{
    processor:   createProcessorNode(),
    bufferSize:  100,
    concurrency: 10,
}

// Process stream
inputStream := make(chan Item)
outputStream := processor.ProcessStream(ctx, inputStream)

// Send items
go func() {
    defer close(inputStream)
    for _, item := range items {
        select {
        case inputStream <- item:
        case <-ctx.Done():
            return
        }
    }
}()

// Consume results
for result := range outputStream {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
    } else {
        log.Printf("Result: %v", result.Data)
    }
}
```

## Performance Considerations

### 1. Context Cancellation

Always respect context cancellation:

```go
func ProcessWithContext(ctx context.Context, items []Item) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Process item
            if err := processItem(item); err != nil {
                return err
            }
        }
    }
    return nil
}
```

### 2. Goroutine Lifecycle Management

Ensure goroutines are properly cleaned up:

```go
type Manager struct {
    wg sync.WaitGroup
}

func (m *Manager) Start(ctx context.Context) {
    m.wg.Add(1)
    go func() {
        defer m.wg.Done()
        m.run(ctx)
    }()
}

func (m *Manager) Stop() {
    m.wg.Wait()
}
```

### 3. Channel Buffer Sizes

Choose appropriate buffer sizes:

```go
// Unbuffered - synchronous
ch := make(chan Item)

// Small buffer - reduce contention
ch := make(chan Item, 10)

// Large buffer - decouple producers/consumers
ch := make(chan Item, 1000)
```

### 4. Avoid Goroutine Leaks

Always ensure goroutines can exit:

```go
// Bad: Potential goroutine leak
go func() {
    for item := range inputChan { // May block forever
        process(item)
    }
}()

// Good: Can be cancelled
go func() {
    for {
        select {
        case item, ok := <-inputChan:
            if !ok {
                return
            }
            process(item)
        case <-ctx.Done():
            return
        }
    }
}()
```

## Best Practices

### 1. Use Appropriate Patterns

- **Fan-Out**: When you have independent items to process
- **Fan-In**: When aggregating from multiple sources
- **Pipeline**: When you have sequential transformations
- **Worker Pool**: When you need to limit concurrency

### 2. Monitor Goroutine Count

```go
func MonitorGoroutines() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        log.Printf("Active goroutines: %d", runtime.NumGoroutine())
    }
}
```

### 3. Handle Errors Appropriately

```go
type ConcurrentResult struct {
    Index  int
    Result any
    Error  error
}

func ProcessConcurrently(items []Item) ([]any, error) {
    results := make([]ConcurrentResult, len(items))
    
    // Process all items, collecting errors
    var hasErrors bool
    for i, res := range results {
        if res.Error != nil {
            hasErrors = true
            log.Printf("Item %d failed: %v", i, res.Error)
        }
    }
    
    if hasErrors {
        return nil, errors.New("some items failed processing")
    }
    
    // Extract successful results
    output := make([]any, len(results))
    for i, res := range results {
        output[i] = res.Result
    }
    
    return output, nil
}
```

### 4. Test Concurrent Code

```go
func TestConcurrentExecution(t *testing.T) {
    // Use race detector
    // go test -race
    
    processor := createProcessorNode()
    items := generateTestItems(100)
    
    results, err := pocket.FanOut(context.Background(), processor, store, items)
    
    assert.NoError(t, err)
    assert.Len(t, results, len(items))
    
    // Verify order preservation
    for i, result := range results {
        assert.Equal(t, expectedResult(items[i]), result)
    }
}
```

## Summary

Pocket's concurrency patterns enable:

1. **Efficient parallel processing** with fan-out/fan-in
2. **Sequential pipelines** for data transformation
3. **Controlled concurrency** with worker pools
4. **Dynamic scaling** based on load
5. **Stream processing** with back-pressure

By leveraging these patterns and Go's concurrency primitives, you can build high-performance workflows that scale with your workload.