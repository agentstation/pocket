# Embedding Pocket in Your Application

This guide covers how to embed the Pocket graph execution engine into your Go applications, from simple integrations to advanced patterns.

## Overview

Pocket is designed to be embedded as a library in larger applications. The graph execution engine can power:
- Workflow automation systems
- Decision engines
- Data processing pipelines
- LLM agent orchestration
- Business process management

## Basic Integration

### Simple Embedding

```go
package main

import (
    "context"
    "github.com/agentstation/pocket"
)

type WorkflowEngine struct {
    store *pocket.Store
    graphs map[string]*pocket.Graph
}

func NewWorkflowEngine() *WorkflowEngine {
    return &WorkflowEngine{
        store: pocket.NewStore(),
        graphs: make(map[string]*pocket.Graph),
    }
}

func (w *WorkflowEngine) RegisterWorkflow(name string, startNode pocket.Node) {
    graph := pocket.NewGraph(startNode, w.store)
    w.graphs[name] = graph
}

func (w *WorkflowEngine) Execute(ctx context.Context, workflow string, input any) (any, error) {
    graph, exists := w.graphs[workflow]
    if !exists {
        return nil, fmt.Errorf("workflow %s not found", workflow)
    }
    return graph.Run(ctx, input)
}
```

### Integration with HTTP Server

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/agentstation/pocket"
)

type Server struct {
    engine *WorkflowEngine
}

func (s *Server) HandleWorkflow(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Workflow string          `json:"workflow"`
        Input    json.RawMessage `json:"input"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    var input any
    if err := json.Unmarshal(req.Input, &input); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    result, err := s.engine.Execute(r.Context(), req.Workflow, input)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]any{
        "success": true,
        "result":  result,
    })
}

func main() {
    engine := NewWorkflowEngine()
    
    // Register workflows
    engine.RegisterWorkflow("process-order", createOrderWorkflow())
    engine.RegisterWorkflow("user-onboarding", createOnboardingWorkflow())
    
    server := &Server{engine: engine}
    http.HandleFunc("/workflow", server.HandleWorkflow)
    http.ListenAndServe(":8080", nil)
}
```

## Advanced Patterns

### Workflow Repository Pattern

Create a repository for managing workflows:

```go
type WorkflowRepository interface {
    Get(ctx context.Context, id string) (*WorkflowDefinition, error)
    Save(ctx context.Context, def *WorkflowDefinition) error
    List(ctx context.Context, filter WorkflowFilter) ([]*WorkflowDefinition, error)
}

type WorkflowDefinition struct {
    ID          string
    Name        string
    Description string
    Version     string
    Graph       *pocket.Graph
    Metadata    map[string]any
}

type WorkflowService struct {
    repo    WorkflowRepository
    store   pocket.Store
    cache   map[string]*pocket.Graph
    mu      sync.RWMutex
}

func (s *WorkflowService) Execute(ctx context.Context, workflowID string, input any) (any, error) {
    // Check cache
    s.mu.RLock()
    graph, cached := s.cache[workflowID]
    s.mu.RUnlock()
    
    if !cached {
        // Load from repository
        def, err := s.repo.Get(ctx, workflowID)
        if err != nil {
            return nil, fmt.Errorf("workflow not found: %w", err)
        }
        
        graph = def.Graph
        
        // Update cache
        s.mu.Lock()
        s.cache[workflowID] = graph
        s.mu.Unlock()
    }
    
    // Execute with isolated store scope
    scopedStore := s.store.Scope(fmt.Sprintf("workflow:%s", workflowID))
    return graph.RunWithStore(ctx, scopedStore, input)
}
```

### Dynamic Workflow Building

Build workflows dynamically from configuration:

```go
type NodeConfig struct {
    Name   string                 `json:"name"`
    Type   string                 `json:"type"`
    Config map[string]any         `json:"config"`
    Next   map[string]string      `json:"next"`
}

type WorkflowConfig struct {
    Nodes []NodeConfig `json:"nodes"`
    Start string       `json:"start"`
}

func BuildWorkflow(config WorkflowConfig, registry NodeRegistry) (*pocket.Graph, error) {
    store := pocket.NewStore()
    builder := pocket.NewBuilder(store)
    
    // Create nodes
    nodes := make(map[string]pocket.Node)
    for _, nc := range config.Nodes {
        node, err := registry.CreateNode(nc.Type, nc.Name, nc.Config)
        if err != nil {
            return nil, fmt.Errorf("create node %s: %w", nc.Name, err)
        }
        nodes[nc.Name] = node
        builder.Add(node)
    }
    
    // Connect nodes
    for _, nc := range config.Nodes {
        for action, target := range nc.Next {
            builder.Connect(nc.Name, action, target)
        }
    }
    
    // Set start node
    builder.Start(config.Start)
    
    return builder.Build()
}
```

### Plugin System Integration

Extend your application with plugins:

```go
type Plugin interface {
    Name() string
    Version() string
    CreateNodes() []pocket.Node
}

type PluginManager struct {
    plugins map[string]Plugin
    nodes   map[string]pocket.Node
}

func (pm *PluginManager) Load(plugin Plugin) error {
    if _, exists := pm.plugins[plugin.Name()]; exists {
        return fmt.Errorf("plugin %s already loaded", plugin.Name())
    }
    
    pm.plugins[plugin.Name()] = plugin
    
    // Register plugin nodes
    for _, node := range plugin.CreateNodes() {
        pm.nodes[node.Name()] = node
    }
    
    return nil
}

func (pm *PluginManager) GetNode(name string) (pocket.Node, bool) {
    node, exists := pm.nodes[name]
    return node, exists
}

// Use in your application
func createWorkflowWithPlugins(pm *PluginManager) *pocket.Graph {
    builder := pocket.NewBuilder(pocket.NewStore())
    
    // Get node from plugin
    if customNode, exists := pm.GetNode("custom-processor"); exists {
        builder.Add(customNode)
    }
    
    // Continue building workflow...
    return builder.Build()
}
```

## Production Considerations

### Observability

Add monitoring and tracing:

```go
type ObservableEngine struct {
    engine  *WorkflowEngine
    metrics MetricsCollector
    tracer  Tracer
}

func (o *ObservableEngine) Execute(ctx context.Context, workflow string, input any) (any, error) {
    // Start trace
    span, ctx := o.tracer.Start(ctx, "workflow.execute")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("workflow.name", workflow),
    )
    
    // Record metrics
    timer := o.metrics.Timer("workflow.duration", "workflow", workflow)
    defer timer.Stop()
    
    // Execute
    result, err := o.engine.Execute(ctx, workflow, input)
    
    if err != nil {
        o.metrics.Inc("workflow.errors", "workflow", workflow)
        span.RecordError(err)
    } else {
        o.metrics.Inc("workflow.success", "workflow", workflow)
    }
    
    return result, err
}
```

### Resource Management

Manage resources effectively:

```go
type ManagedEngine struct {
    engine       *WorkflowEngine
    maxConcurrent int
    sem          *semaphore.Weighted
    timeout      time.Duration
}

func NewManagedEngine(maxConcurrent int, timeout time.Duration) *ManagedEngine {
    return &ManagedEngine{
        engine:        NewWorkflowEngine(),
        maxConcurrent: maxConcurrent,
        sem:          semaphore.NewWeighted(int64(maxConcurrent)),
        timeout:      timeout,
    }
}

func (m *ManagedEngine) Execute(ctx context.Context, workflow string, input any) (any, error) {
    // Acquire semaphore
    if err := m.sem.Acquire(ctx, 1); err != nil {
        return nil, fmt.Errorf("acquire semaphore: %w", err)
    }
    defer m.sem.Release(1)
    
    // Apply timeout
    ctx, cancel := context.WithTimeout(ctx, m.timeout)
    defer cancel()
    
    return m.engine.Execute(ctx, workflow, input)
}
```

### Multi-Tenancy

Support multiple tenants:

```go
type TenantEngine struct {
    engines map[string]*WorkflowEngine
    mu      sync.RWMutex
}

func (t *TenantEngine) GetEngine(tenantID string) *WorkflowEngine {
    t.mu.RLock()
    engine, exists := t.engines[tenantID]
    t.mu.RUnlock()
    
    if !exists {
        t.mu.Lock()
        defer t.mu.Unlock()
        
        // Double-check after acquiring write lock
        if engine, exists = t.engines[tenantID]; !exists {
            // Create tenant-specific engine with isolated store
            engine = &WorkflowEngine{
                store: pocket.NewStore(
                    pocket.WithMaxEntries(1000),
                    pocket.WithTTL(30 * time.Minute),
                ),
                graphs: make(map[string]*pocket.Graph),
            }
            t.engines[tenantID] = engine
        }
    }
    
    return engine
}

func (t *TenantEngine) Execute(ctx context.Context, tenantID, workflow string, input any) (any, error) {
    engine := t.GetEngine(tenantID)
    return engine.Execute(ctx, workflow, input)
}
```

## Testing Embedded Workflows

### Unit Testing

Test individual workflows:

```go
func TestOrderProcessingWorkflow(t *testing.T) {
    // Create test engine
    engine := NewWorkflowEngine()
    engine.RegisterWorkflow("process-order", createOrderWorkflow())
    
    tests := []struct {
        name     string
        input    any
        wantErr  bool
        validate func(t *testing.T, result any)
    }{
        {
            name: "valid order",
            input: Order{
                ID:     "123",
                Amount: 100.00,
                Items:  []Item{{SKU: "ABC", Qty: 1}},
            },
            validate: func(t *testing.T, result any) {
                processed := result.(ProcessedOrder)
                assert.Equal(t, "completed", processed.Status)
            },
        },
        {
            name: "invalid order",
            input: Order{
                ID:     "",
                Amount: -10,
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.Execute(context.Background(), "process-order", tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            if tt.validate != nil {
                tt.validate(t, result)
            }
        })
    }
}
```

### Integration Testing

Test the full system:

```go
func TestWorkflowAPI(t *testing.T) {
    // Create test server
    engine := NewWorkflowEngine()
    engine.RegisterWorkflow("test", createTestWorkflow())
    
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        s := &Server{engine: engine}
        s.HandleWorkflow(w, r)
    }))
    defer server.Close()
    
    // Test API
    payload := map[string]any{
        "workflow": "test",
        "input":    map[string]any{"value": 42},
    }
    
    body, _ := json.Marshal(payload)
    resp, err := http.Post(server.URL+"/workflow", "application/json", bytes.NewReader(body))
    require.NoError(t, err)
    defer resp.Body.Close()
    
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var result map[string]any
    json.NewDecoder(resp.Body).Decode(&result)
    assert.True(t, result["success"].(bool))
}
```

## Best Practices

1. **Isolate Workflows**: Use scoped stores to isolate workflow state
2. **Version Workflows**: Include versioning in your workflow definitions
3. **Monitor Performance**: Add metrics and tracing
4. **Handle Errors**: Implement proper error handling and recovery
5. **Test Thoroughly**: Test both individual nodes and complete workflows
6. **Document Workflows**: Maintain clear documentation for each workflow
7. **Resource Limits**: Set appropriate timeouts and concurrency limits
8. **Security**: Validate inputs and sanitize outputs

## Next Steps

- Explore [Advanced Patterns](../patterns/)
- Learn about [Performance Optimization](../advanced/PERFORMANCE.md)
- Study [Middleware Integration](../advanced/MIDDLEWARE.md)
- See [Production Examples](../../examples/)