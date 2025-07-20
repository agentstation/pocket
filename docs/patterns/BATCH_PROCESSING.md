# Batch Processing Patterns

## Overview

Batch processing patterns enable efficient handling of large datasets by processing items in groups rather than individually. This guide covers batch collection, parallel processing, streaming, and optimization techniques.

## Basic Batch Processing

### Simple Batch Processor

Process items in fixed-size batches:

```go
import "github.com/agentstation/pocket/batch"

// Process items in batches of 100
processor := pocket.NewNode[Item, ProcessedItem]("item-processor",
    pocket.WithExec(func(ctx context.Context, item Item) (ProcessedItem, error) {
        return processItem(item), nil
    }),
)

// Batch processing function
func ProcessInBatches(ctx context.Context, items []Item, batchSize int) ([]ProcessedItem, error) {
    var allResults []ProcessedItem
    store := pocket.NewStore()
    
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        batch := items[i:end]
        
        // Process batch in parallel
        results, err := pocket.FanOut(ctx, processor, store, batch)
        if err != nil {
            return nil, fmt.Errorf("batch %d failed: %w", i/batchSize, err)
        }
        
        allResults = append(allResults, results...)
        
        // Log progress
        log.Printf("Processed batch %d/%d", i/batchSize+1, (len(items)+batchSize-1)/batchSize)
    }
    
    return allResults, nil
}
```

### Map-Reduce Pattern

Transform and aggregate data in batches:

```go
// Map-reduce for data aggregation
mapReduce := batch.MapReduce(
    // Extract function - get items to process
    func(ctx context.Context, store pocket.Store) ([]RawData, error) {
        return fetchRawData()
    },
    
    // Map function - transform each item
    func(ctx context.Context, data RawData) (MappedData, error) {
        return MappedData{
            Key:   extractKey(data),
            Value: transformValue(data),
        }, nil
    },
    
    // Reduce function - aggregate results
    func(ctx context.Context, mapped []MappedData) (AggregatedResult, error) {
        groups := make(map[string][]float64)
        
        // Group by key
        for _, m := range mapped {
            groups[m.Key] = append(groups[m.Key], m.Value)
        }
        
        // Aggregate each group
        results := make(map[string]float64)
        for key, values := range groups {
            results[key] = aggregate(values)
        }
        
        return AggregatedResult{
            Aggregates: results,
            Count:      len(mapped),
        }, nil
    },
    
    batch.WithConcurrency(10),
)

// Execute map-reduce
graph := pocket.NewGraph(mapReduce, pocket.NewStore())
result, err := graph.Run(ctx, nil)
```

## Stream Processing

### Continuous Batch Processing

Process continuous streams in batches:

```go
type StreamBatcher struct {
    batchSize     int
    flushInterval time.Duration
    processor     pocket.Node
}

func (b *StreamBatcher) ProcessStream(ctx context.Context, input <-chan Item) <-chan BatchResult {
    output := make(chan BatchResult, 10)
    
    go func() {
        defer close(output)
        
        ticker := time.NewTicker(b.flushInterval)
        defer ticker.Stop()
        
        batch := make([]Item, 0, b.batchSize)
        
        for {
            select {
            case item, ok := <-input:
                if !ok {
                    // Process remaining items
                    if len(batch) > 0 {
                        b.processBatch(ctx, batch, output)
                    }
                    return
                }
                
                batch = append(batch, item)
                
                // Process when batch is full
                if len(batch) >= b.batchSize {
                    b.processBatch(ctx, batch, output)
                    batch = make([]Item, 0, b.batchSize)
                }
                
            case <-ticker.C:
                // Flush partial batch on timeout
                if len(batch) > 0 {
                    b.processBatch(ctx, batch, output)
                    batch = make([]Item, 0, b.batchSize)
                }
                
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return output
}

func (b *StreamBatcher) processBatch(ctx context.Context, batch []Item, output chan<- BatchResult) {
    store := pocket.NewStore()
    results, err := pocket.FanOut(ctx, b.processor, store, batch)
    
    output <- BatchResult{
        Items:   batch,
        Results: results,
        Error:   err,
    }
}
```

### Window-Based Processing

Process data in time or count-based windows:

```go
type WindowProcessor struct {
    windowSize     time.Duration
    slideInterval  time.Duration
    aggregateFunc  func([]Event) AggregateResult
}

func (w *WindowProcessor) ProcessWindowed(ctx context.Context, events <-chan Event) <-chan WindowResult {
    output := make(chan WindowResult, 10)
    
    go func() {
        defer close(output)
        
        // Sliding window
        window := []Event{}
        windowStart := time.Now()
        
        ticker := time.NewTicker(w.slideInterval)
        defer ticker.Stop()
        
        for {
            select {
            case event, ok := <-events:
                if !ok {
                    // Process final window
                    if len(window) > 0 {
                        w.processWindow(window, windowStart, output)
                    }
                    return
                }
                
                // Add to window
                window = append(window, event)
                
                // Remove old events
                cutoff := time.Now().Add(-w.windowSize)
                newWindow := []Event{}
                for _, e := range window {
                    if e.Timestamp.After(cutoff) {
                        newWindow = append(newWindow, e)
                    }
                }
                window = newWindow
                
            case <-ticker.C:
                // Process current window
                if len(window) > 0 {
                    w.processWindow(window, windowStart, output)
                    windowStart = time.Now()
                }
                
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return output
}

func (w *WindowProcessor) processWindow(events []Event, start time.Time, output chan<- WindowResult) {
    result := w.aggregateFunc(events)
    
    output <- WindowResult{
        WindowStart: start,
        WindowEnd:   time.Now(),
        EventCount:  len(events),
        Result:      result,
    }
}
```

## Parallel Batch Processing

### Worker Pool Pattern

Process batches with a fixed number of workers:

```go
type BatchWorkerPool struct {
    workers      int
    batchSize    int
    processor    pocket.Node
}

func (p *BatchWorkerPool) ProcessDataset(ctx context.Context, dataset []Item) ([]Result, error) {
    // Create channels
    batches := make(chan []Item, p.workers*2)
    results := make(chan IndexedBatchResult, p.workers*2)
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < p.workers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            p.worker(ctx, workerID, batches, results)
        }(i)
    }
    
    // Send batches
    go func() {
        defer close(batches)
        
        for i := 0; i < len(dataset); i += p.batchSize {
            end := i + p.batchSize
            if end > len(dataset) {
                end = len(dataset)
            }
            
            select {
            case batches <- dataset[i:end]:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    // Collect results
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Aggregate results in order
    return p.aggregateResults(results, len(dataset))
}

func (p *BatchWorkerPool) worker(ctx context.Context, id int, batches <-chan []Item, results chan<- IndexedBatchResult) {
    store := pocket.NewStore()
    
    for batch := range batches {
        select {
        case <-ctx.Done():
            return
        default:
            // Process batch
            batchResults, err := pocket.FanOut(ctx, p.processor, store, batch)
            
            results <- IndexedBatchResult{
                BatchIndex: getBatchIndex(batch),
                Results:    batchResults,
                Error:      err,
                WorkerID:   id,
            }
        }
    }
}
```

### Dynamic Batch Sizing

Adjust batch size based on performance:

```go
type AdaptiveBatchProcessor struct {
    minBatch      int
    maxBatch      int
    targetLatency time.Duration
    processor     pocket.Node
}

func (a *AdaptiveBatchProcessor) Process(ctx context.Context, items []Item) error {
    batchSize := a.minBatch
    store := pocket.NewStore()
    
    for i := 0; i < len(items); {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        batch := items[i:end]
        
        // Measure processing time
        start := time.Now()
        _, err := pocket.FanOut(ctx, a.processor, store, batch)
        duration := time.Since(start)
        
        if err != nil {
            // Reduce batch size on error
            batchSize = max(a.minBatch, batchSize/2)
            log.Printf("Error processing batch, reducing size to %d: %v", batchSize, err)
            continue
        }
        
        // Adjust batch size based on latency
        itemLatency := duration / time.Duration(len(batch))
        
        if itemLatency < a.targetLatency/2 {
            // Can increase batch size
            batchSize = min(a.maxBatch, batchSize*2)
            log.Printf("Increasing batch size to %d (latency: %v)", batchSize, itemLatency)
        } else if itemLatency > a.targetLatency {
            // Should decrease batch size
            batchSize = max(a.minBatch, batchSize*3/4)
            log.Printf("Decreasing batch size to %d (latency: %v)", batchSize, itemLatency)
        }
        
        i = end
    }
    
    return nil
}
```

## Batch Aggregation Patterns

### Time-Based Batching

Collect items for a specific duration:

```go
type TimedBatcher struct {
    duration  time.Duration
    processor pocket.Node
}

func (t *TimedBatcher) CreateBatchNode() pocket.Node {
    return pocket.NewNode[Item, BatchResult]("timed-batcher",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, item Item) (any, error) {
            // Get or create current batch
            batchKey := "current-batch"
            batch, exists := store.Get(ctx, batchKey)
            if !exists {
                batch = &TimedBatch{
                    Items:     []Item{},
                    StartTime: time.Now(),
                }
            }
            
            return map[string]any{
                "item":  item,
                "batch": batch,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (BatchResult, error) {
            data := prepData.(map[string]any)
            item := data["item"].(Item)
            batch := data["batch"].(*TimedBatch)
            
            batch.Items = append(batch.Items, item)
            
            // Check if batch period expired
            if time.Since(batch.StartTime) >= t.duration {
                // Process complete batch
                return processBatch(batch.Items), nil
            }
            
            // Continue collecting
            return BatchResult{Partial: true, Batch: batch}, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            item Item, prep, result any) (BatchResult, string, error) {
            
            res := result.(BatchResult)
            
            if res.Partial {
                // Save partial batch
                store.Set(ctx, "current-batch", res.Batch)
                return res, "collect", nil
            }
            
            // Clear batch and process
            store.Delete(ctx, "current-batch")
            return res, "process", nil
        }),
    )
}
```

### Count-Based Batching

Collect a specific number of items:

```go
type CountBatcher struct {
    targetCount int
    processor   pocket.Node
}

func (c *CountBatcher) BatchAndProcess(ctx context.Context, items <-chan Item) error {
    batch := make([]Item, 0, c.targetCount)
    store := pocket.NewStore()
    
    for item := range items {
        batch = append(batch, item)
        
        if len(batch) >= c.targetCount {
            // Process full batch
            if err := c.processBatch(ctx, batch, store); err != nil {
                return err
            }
            
            // Reset batch
            batch = make([]Item, 0, c.targetCount)
        }
    }
    
    // Process remaining items
    if len(batch) > 0 {
        return c.processBatch(ctx, batch, store)
    }
    
    return nil
}

func (c *CountBatcher) processBatch(ctx context.Context, batch []Item, store pocket.Store) error {
    results, err := pocket.FanOut(ctx, c.processor, store, batch)
    if err != nil {
        return fmt.Errorf("batch processing failed: %w", err)
    }
    
    log.Printf("Processed batch of %d items, results: %v", len(batch), results)
    return nil
}
```

## Optimization Techniques

### Memory-Efficient Processing

Process large datasets without loading everything into memory:

```go
type ChunkedProcessor struct {
    chunkSize int
    processor pocket.Node
}

func (c *ChunkedProcessor) ProcessFile(ctx context.Context, filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    chunk := make([]string, 0, c.chunkSize)
    store := pocket.NewStore()
    
    for scanner.Scan() {
        line := scanner.Text()
        chunk = append(chunk, line)
        
        if len(chunk) >= c.chunkSize {
            // Process chunk
            items := c.parseChunk(chunk)
            if _, err := pocket.FanOut(ctx, c.processor, store, items); err != nil {
                return err
            }
            
            // Clear chunk
            chunk = make([]string, 0, c.chunkSize)
        }
    }
    
    // Process remaining
    if len(chunk) > 0 {
        items := c.parseChunk(chunk)
        _, err := pocket.FanOut(ctx, c.processor, store, items)
        return err
    }
    
    return scanner.Err()
}
```

### Batch Retry Strategy

Retry failed batches with exponential backoff:

```go
type BatchRetryProcessor struct {
    maxRetries int
    baseDelay  time.Duration
    processor  pocket.Node
}

func (r *BatchRetryProcessor) ProcessWithRetry(ctx context.Context, batch []Item) ([]Result, error) {
    var lastErr error
    store := pocket.NewStore()
    
    for attempt := 0; attempt < r.maxRetries; attempt++ {
        results, err := pocket.FanOut(ctx, r.processor, store, batch)
        if err == nil {
            return results, nil
        }
        
        lastErr = err
        
        // Check for partial success
        partialResults, failedItems := r.extractPartialResults(results, batch, err)
        
        if len(failedItems) == 0 {
            // All items eventually succeeded
            return partialResults, nil
        }
        
        // Retry only failed items
        batch = failedItems
        
        // Exponential backoff
        if attempt < r.maxRetries-1 {
            delay := r.baseDelay * time.Duration(1<<uint(attempt))
            log.Printf("Batch retry %d/%d after %v for %d failed items", 
                attempt+1, r.maxRetries, delay, len(failedItems))
            
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
    
    return nil, fmt.Errorf("batch processing failed after %d retries: %w", r.maxRetries, lastErr)
}
```

### Progress Tracking

Track and report batch processing progress:

```go
type ProgressTracker struct {
    totalItems    int
    processedItems int
    startTime     time.Time
    mu            sync.Mutex
}

func (p *ProgressTracker) Update(processed int) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.processedItems += processed
    
    // Calculate metrics
    progress := float64(p.processedItems) / float64(p.totalItems) * 100
    elapsed := time.Since(p.startTime)
    rate := float64(p.processedItems) / elapsed.Seconds()
    eta := time.Duration(float64(p.totalItems-p.processedItems) / rate * float64(time.Second))
    
    log.Printf("Progress: %.1f%% (%d/%d) | Rate: %.1f items/sec | ETA: %v",
        progress, p.processedItems, p.totalItems, rate, eta)
}

// Use with batch processor
tracker := &ProgressTracker{
    totalItems: len(items),
    startTime:  time.Now(),
}

for i := 0; i < len(items); i += batchSize {
    // Process batch...
    
    tracker.Update(len(batch))
}
```

## Best Practices

### 1. Error Handling

Handle partial batch failures gracefully:

```go
type BatchError struct {
    BatchIndex     int
    FailedItems    []int
    PartialResults []Result
    Errors         []error
}

func processBatchWithErrorHandling(batch []Item) (*BatchResult, *BatchError) {
    results := make([]Result, len(batch))
    errors := make([]error, len(batch))
    failedIndices := []int{}
    
    for i, item := range batch {
        result, err := processItem(item)
        if err != nil {
            errors[i] = err
            failedIndices = append(failedIndices, i)
        } else {
            results[i] = result
        }
    }
    
    if len(failedIndices) > 0 {
        return nil, &BatchError{
            FailedItems:    failedIndices,
            PartialResults: results,
            Errors:         errors,
        }
    }
    
    return &BatchResult{Results: results}, nil
}
```

### 2. Resource Management

Control resource usage during batch processing:

```go
type ResourceLimitedBatcher struct {
    maxMemory     int64
    maxGoroutines int
    monitor       *ResourceMonitor
}

func (r *ResourceLimitedBatcher) CanProcessBatch(batchSize int) bool {
    estimatedMemory := int64(batchSize) * r.averageItemSize()
    currentMemory := r.monitor.GetMemoryUsage()
    
    return currentMemory+estimatedMemory < r.maxMemory &&
           runtime.NumGoroutine() < r.maxGoroutines
}
```

### 3. Batch Metrics

Collect and analyze batch processing metrics:

```go
type BatchMetrics struct {
    BatchesProcessed   int64
    ItemsProcessed     int64
    FailedBatches      int64
    TotalDuration      time.Duration
    AverageBatchSize   float64
    AverageLatency     time.Duration
}

func (m *BatchMetrics) RecordBatch(size int, duration time.Duration, success bool) {
    atomic.AddInt64(&m.BatchesProcessed, 1)
    atomic.AddInt64(&m.ItemsProcessed, int64(size))
    
    if !success {
        atomic.AddInt64(&m.FailedBatches, 1)
    }
    
    // Update averages (simplified - use proper averaging in production)
    m.AverageBatchSize = float64(m.ItemsProcessed) / float64(m.BatchesProcessed)
    m.AverageLatency = time.Duration(int64(duration) / int64(size))
}
```

## Summary

Batch processing patterns in Pocket enable:

1. **Efficient data processing** through batching and parallelization
2. **Stream processing** with windowing and continuous batching
3. **Adaptive strategies** that adjust to workload characteristics
4. **Resource optimization** with memory-efficient processing
5. **Robust error handling** with retry and partial success support

These patterns help process large datasets efficiently while maintaining system stability and providing visibility into processing progress.