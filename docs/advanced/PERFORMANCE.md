# Performance Optimization

## Overview

This guide covers performance optimization techniques for Pocket workflows, including benchmarking, profiling, memory management, and scaling strategies.

## Benchmarking

### Basic Benchmarks

Create benchmarks for your nodes:

```go
func BenchmarkNode(b *testing.B) {
    node := pocket.NewNode[Input, Output]("processor",
        pocket.WithExec(processFunc),
    )
    
    store := pocket.NewStore()
    graph := pocket.NewGraph(node, store)
    
    input := generateTestInput()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := graph.Run(context.Background(), input)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Benchmark with different input sizes
func BenchmarkNodeSizes(b *testing.B) {
    sizes := []int{10, 100, 1000, 10000}
    
    for _, size := range sizes {
        b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
            input := generateInputOfSize(size)
            benchmarkWithInput(b, input)
        })
    }
}
```

### Parallel Benchmarks

Test concurrent performance:

```go
func BenchmarkConcurrent(b *testing.B) {
    node := createNode()
    store := pocket.NewStore()
    
    b.RunParallel(func(pb *testing.PB) {
        graph := pocket.NewGraph(node, store)
        input := generateInput()
        
        for pb.Next() {
            _, err := graph.Run(context.Background(), input)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

// Compare sequential vs parallel
func BenchmarkParallelVsSequential(b *testing.B) {
    items := generateItems(1000)
    processor := createProcessor()
    store := pocket.NewStore()
    
    b.Run("Sequential", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            for _, item := range items {
                processor.Exec(context.Background(), item)
            }
        }
    })
    
    b.Run("Parallel", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            pocket.FanOut(context.Background(), processor, store, items)
        }
    })
}
```

### Memory Benchmarks

Track memory allocations:

```go
func BenchmarkMemory(b *testing.B) {
    node := createNode()
    store := pocket.NewStore()
    graph := pocket.NewGraph(node, store)
    
    b.ReportAllocs()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, _ = graph.Run(context.Background(), testInput)
    }
    
    // Report custom metrics
    b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
    b.ReportMetric(float64(runtime.MemStats.Alloc)/float64(b.N), "bytes/op")
}
```

## Profiling

### CPU Profiling

Profile CPU usage:

```go
import (
    "runtime/pprof"
    "os"
)

func profileCPU() {
    f, err := os.Create("cpu.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
    
    // Run your workflow
    runWorkflow()
}

// Analyze with: go tool pprof cpu.prof
```

### Memory Profiling

Track memory usage:

```go
func profileMemory() {
    f, err := os.Create("mem.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    // Run workflow
    runWorkflow()
    
    runtime.GC()
    pprof.WriteHeapProfile(f)
}

// Analyze with: go tool pprof mem.prof
```

### Execution Tracing

Trace execution flow:

```go
import "runtime/trace"

func traceExecution() {
    f, err := os.Create("trace.out")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    trace.Start(f)
    defer trace.Stop()
    
    // Run workflow
    runWorkflow()
}

// Analyze with: go tool trace trace.out
```

## Memory Optimization

### Reduce Allocations

Minimize memory allocations:

```go
// Pre-allocate slices
func optimizedBatch(items []Item) []Result {
    // Good: pre-allocate with capacity
    results := make([]Result, 0, len(items))
    
    for _, item := range items {
        results = append(results, process(item))
    }
    
    return results
}

// Reuse buffers
var bufferPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func processWithBuffer(data []byte) string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    
    buf.Reset()
    buf.Write(data)
    // Process...
    
    return buf.String()
}
```

### Efficient Data Structures

Choose appropriate data structures:

```go
// Use pointers for large structs
type LargeData struct {
    // Many fields...
}

// Good: pass pointer
func processLarge(data *LargeData) error {
    // Process without copying
}

// Avoid: pass by value
func processLargeBad(data LargeData) error {
    // Copies entire struct
}

// Use string builder for concatenation
func buildString(parts []string) string {
    var sb strings.Builder
    sb.Grow(estimateSize(parts)) // Pre-allocate
    
    for _, part := range parts {
        sb.WriteString(part)
    }
    
    return sb.String()
}
```

### Store Memory Management

Optimize store usage:

```go
// Use bounded stores
store := pocket.NewStore(
    pocket.WithMaxEntries(10000),
    pocket.WithTTL(5 * time.Minute),
)

// Clean up after use
cleanupNode := pocket.NewNode[Result, Result]("cleanup",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input, prep, result any) (Result, string, error) {
        
        // Clean up temporary data
        store.Delete(ctx, "temp:*")
        
        return result.(Result), "done", nil
    }),
)

// Use scoped stores for isolation
workflowStore := parentStore.Scope("workflow:" + workflowID)
// Scoped data is easier to clean up
```

## Concurrency Optimization

### Optimal Worker Count

Find the right concurrency level:

```go
func findOptimalWorkers(workload []Item) int {
    cpuCount := runtime.NumCPU()
    itemCount := len(workload)
    
    // Heuristics
    if itemCount < cpuCount {
        return itemCount
    }
    
    if isIOBound(workload) {
        return cpuCount * 2 // More workers for I/O
    }
    
    return cpuCount // CPU-bound work
}

// Adaptive concurrency
type AdaptiveConcurrency struct {
    min, max int
    current  int
    mu       sync.Mutex
}

func (a *AdaptiveConcurrency) AdjustWorkers(latency time.Duration) int {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    if latency > targetLatency {
        a.current = max(a.min, a.current-1)
    } else if latency < targetLatency/2 {
        a.current = min(a.max, a.current+1)
    }
    
    return a.current
}
```

### Goroutine Pool

Reuse goroutines:

```go
type WorkerPool struct {
    workers   int
    taskQueue chan func()
    wg        sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
    p := &WorkerPool{
        workers:   workers,
        taskQueue: make(chan func(), workers*2),
    }
    
    p.start()
    return p
}

func (p *WorkerPool) start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go func() {
            defer p.wg.Done()
            for task := range p.taskQueue {
                task()
            }
        }()
    }
}

func (p *WorkerPool) Submit(task func()) {
    p.taskQueue <- task
}

func (p *WorkerPool) Stop() {
    close(p.taskQueue)
    p.wg.Wait()
}
```

### Channel Optimization

Efficient channel usage:

```go
// Buffer channels appropriately
ch := make(chan Item, 100) // Reduce contention

// Use select with default for non-blocking
select {
case ch <- item:
    // Sent successfully
default:
    // Channel full, handle accordingly
}

// Batch channel operations
func batchReceive(ch <-chan Item, maxBatch int, timeout time.Duration) []Item {
    batch := make([]Item, 0, maxBatch)
    timer := time.NewTimer(timeout)
    
    for {
        select {
        case item := <-ch:
            batch = append(batch, item)
            if len(batch) >= maxBatch {
                return batch
            }
        case <-timer.C:
            return batch
        }
    }
}
```

## Caching Strategies

### Node Result Caching

Cache expensive computations:

```go
type CachedNode struct {
    pocket.Node
    cache *lru.Cache
    ttl   time.Duration
}

func WithCache(size int, ttl time.Duration) func(pocket.Node) pocket.Node {
    cache, _ := lru.New(size)
    
    return func(node pocket.Node) pocket.Node {
        return &CachedNode{
            Node:  node,
            cache: cache,
            ttl:   ttl,
        }
    }
}

func (n *CachedNode) Exec(ctx context.Context, input any) (any, error) {
    // Generate cache key
    key := generateKey(n.Name(), input)
    
    // Check cache
    if cached, ok := n.cache.Get(key); ok {
        entry := cached.(cacheEntry)
        if time.Since(entry.timestamp) < n.ttl {
            return entry.value, nil
        }
    }
    
    // Execute and cache
    result, err := n.Node.Exec(ctx, input)
    if err == nil {
        n.cache.Add(key, cacheEntry{
            value:     result,
            timestamp: time.Now(),
        })
    }
    
    return result, err
}
```

### Distributed Caching

Use external cache for scale:

```go
type RedisCache struct {
    client *redis.Client
    prefix string
}

func (c *RedisCache) Get(ctx context.Context, key string) (any, error) {
    data, err := c.client.Get(ctx, c.prefix+key).Bytes()
    if err != nil {
        return nil, err
    }
    
    var value any
    err = json.Unmarshal(data, &value)
    return value, err
}

func (c *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    
    return c.client.Set(ctx, c.prefix+key, data, ttl).Err()
}
```

## Scaling Strategies

### Horizontal Scaling

Distribute work across instances:

```go
type DistributedProcessor struct {
    instances []string
    client    *http.Client
}

func (d *DistributedProcessor) Process(ctx context.Context, items []Item) ([]Result, error) {
    // Partition work
    partitions := partitionItems(items, len(d.instances))
    
    // Process in parallel on different instances
    var wg sync.WaitGroup
    results := make([][]Result, len(d.instances))
    errors := make([]error, len(d.instances))
    
    for i, partition := range partitions {
        wg.Add(1)
        go func(idx int, items []Item) {
            defer wg.Done()
            results[idx], errors[idx] = d.processOnInstance(ctx, d.instances[idx], items)
        }(i, partition)
    }
    
    wg.Wait()
    
    // Combine results
    return combineResults(results, errors)
}
```

### Load Balancing

Distribute requests evenly:

```go
type LoadBalancer struct {
    nodes    []pocket.Node
    counter  uint64
    strategy LoadBalanceStrategy
}

func (lb *LoadBalancer) Exec(ctx context.Context, input any) (any, error) {
    node := lb.selectNode()
    return node.Exec(ctx, input)
}

func (lb *LoadBalancer) selectNode() pocket.Node {
    switch lb.strategy {
    case RoundRobin:
        n := atomic.AddUint64(&lb.counter, 1)
        return lb.nodes[n%uint64(len(lb.nodes))]
        
    case LeastConnections:
        return lb.getLeastLoadedNode()
        
    case Random:
        return lb.nodes[rand.Intn(len(lb.nodes))]
        
    default:
        return lb.nodes[0]
    }
}
```

## Optimization Checklist

### Pre-Deployment

1. **Benchmark critical paths**
   ```go
   go test -bench=. -benchmem
   ```

2. **Profile under load**
   ```go
   go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
   ```

3. **Check for goroutine leaks**
   ```go
   func TestNoGoroutineLeaks(t *testing.T) {
       before := runtime.NumGoroutine()
       runWorkflow()
       time.Sleep(100 * time.Millisecond)
       after := runtime.NumGoroutine()
       
       if after > before {
           t.Errorf("Goroutine leak: %d -> %d", before, after)
       }
   }
   ```

4. **Validate memory usage**
   ```go
   var m runtime.MemStats
   runtime.ReadMemStats(&m)
   log.Printf("Alloc: %v MB", m.Alloc/1024/1024)
   ```

### Production Monitoring

```go
// Metrics collection
type PerformanceMonitor struct {
    nodeMetrics map[string]*NodeMetrics
    mu          sync.RWMutex
}

type NodeMetrics struct {
    Executions   int64
    TotalLatency time.Duration
    Errors       int64
    LastExec     time.Time
}

func (p *PerformanceMonitor) RecordExecution(node string, duration time.Duration, err error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    metrics := p.nodeMetrics[node]
    if metrics == nil {
        metrics = &NodeMetrics{}
        p.nodeMetrics[node] = metrics
    }
    
    metrics.Executions++
    metrics.TotalLatency += duration
    metrics.LastExec = time.Now()
    
    if err != nil {
        metrics.Errors++
    }
}

func (p *PerformanceMonitor) GetStats(node string) NodeStats {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    metrics := p.nodeMetrics[node]
    if metrics == nil {
        return NodeStats{}
    }
    
    return NodeStats{
        AvgLatency: time.Duration(int64(metrics.TotalLatency) / metrics.Executions),
        ErrorRate:  float64(metrics.Errors) / float64(metrics.Executions),
        Throughput: float64(metrics.Executions) / time.Since(metrics.LastExec).Seconds(),
    }
}
```

## Best Practices

### 1. Measure Before Optimizing

```go
// Always benchmark first
func BenchmarkBefore(b *testing.B) {
    // Original implementation
}

func BenchmarkAfter(b *testing.B) {
    // Optimized implementation
}
```

### 2. Set Performance Goals

```go
// Define SLOs
const (
    MaxLatency     = 100 * time.Millisecond
    MinThroughput  = 1000 // requests per second
    MaxMemoryUsage = 1 << 30 // 1GB
)
```

### 3. Use Context for Cancellation

```go
func (n *Node) Exec(ctx context.Context, input any) (any, error) {
    // Check context regularly
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Long operation
    for i := 0; i < iterations; i++ {
        if i%100 == 0 {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            default:
            }
        }
        
        // Process...
    }
}
```

### 4. Avoid Premature Optimization

Focus on:
- Correctness first
- Clean, maintainable code
- Optimize based on profiling data
- Consider trade-offs

## Summary

Performance optimization in Pocket involves:

1. **Benchmarking** to establish baselines
2. **Profiling** to identify bottlenecks
3. **Memory optimization** to reduce allocations
4. **Concurrency tuning** for optimal parallelism
5. **Caching** to avoid redundant work
6. **Scaling strategies** for production loads

Always measure, profile, and validate optimizations with real workloads before deploying to production.