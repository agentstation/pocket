# State Management Guide

## Overview

State management in Pocket is handled through the Store interface, which provides thread-safe storage for sharing data between nodes in a workflow. This guide covers store operations, scoping, bounded stores, and best practices.

## The Store Interface

### Basic Store Operations

```go
// Create a store
store := pocket.NewStore()

// Set a value
err := store.Set(ctx, "user:123", User{
    ID:   "123",
    Name: "Alice",
})

// Get a value
value, exists := store.Get(ctx, "user:123")
if exists {
    user := value.(User)
    fmt.Printf("Found user: %s\n", user.Name)
}

// Delete a value
err = store.Delete(ctx, "user:123")
```

### Read/Write Separation

Pocket enforces read/write separation through different interfaces:

```go
// StoreReader - used in Prep phase
type StoreReader interface {
    Get(ctx context.Context, key string) (value any, exists bool)
    Scope(prefix string) Store
}

// StoreWriter - used in Post phase
type StoreWriter interface {
    Store // Includes Get, Set, Delete, Scope
}

// Example usage in node lifecycle
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input Input) (any, error) {
        // Can only read
        config, _ := store.Get(ctx, "config")
        // store.Set(...) // Would not compile!
        return config, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, 
        input Input, prep any, exec Output) (Output, string, error) {
        // Can read and write
        store.Set(ctx, "result", exec)
        return exec, "next", nil
    }),
)
```

## Store Scoping

Scopes provide namespaced storage within a store:

```go
// Create a parent store
parentStore := pocket.NewStore()

// Create scoped stores
userStore := parentStore.Scope("user")
configStore := parentStore.Scope("config")

// Keys are automatically prefixed
userStore.Set(ctx, "123", userData)    // Actually stores as "user:123"
configStore.Set(ctx, "app", appConfig) // Actually stores as "config:app"

// Scopes can be nested
adminStore := userStore.Scope("admin") // Prefix: "user:admin:"
adminStore.Set(ctx, "001", adminData)  // Stores as "user:admin:001"
```

### Scope Use Cases

```go
// 1. Multi-tenant workflows
func ProcessTenant(tenantID string, parentStore pocket.Store) pocket.Node {
    // Each tenant gets isolated storage
    tenantStore := parentStore.Scope("tenant:" + tenantID)
    
    return pocket.NewNode[Request, Response]("process",
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            req Request, prep, exec any) (Response, string, error) {
            
            // All operations are scoped to this tenant
            store.Set(ctx, "lastRequest", req)
            store.Set(ctx, "processedAt", time.Now())
            
            return exec.(Response), "done", nil
        }),
    )
}

// 2. Workflow isolation
func CreateIsolatedWorkflow(name string, parentStore pocket.Store) *pocket.Graph {
    // Each workflow instance gets its own scope
    workflowStore := parentStore.Scope("workflow:" + name)
    return pocket.NewGraph(startNode, workflowStore)
}

// 3. Component namespacing
authStore := store.Scope("auth")
cacheStore := store.Scope("cache")
metricsStore := store.Scope("metrics")
```

## Bounded Stores

Bounded stores provide memory management with LRU eviction and TTL:

```go
// Create a bounded store
boundedStore := pocket.NewStore(
    pocket.WithMaxEntries(1000),              // Maximum 1000 entries
    pocket.WithTTL(5 * time.Minute),          // Entries expire after 5 minutes
    pocket.WithEvictionCallback(func(key string, value any) {
        log.Printf("Evicted entry: %s", key)
    }),
)

// Use it normally
boundedStore.Set(ctx, "key", "value")

// When limits are exceeded:
// - Oldest entries are evicted (LRU)
// - Expired entries are removed automatically
// - Eviction callback is called
```

### Bounded Store Patterns

```go
// 1. Cache with automatic cleanup
cacheStore := pocket.NewStore(
    pocket.WithMaxEntries(10000),
    pocket.WithTTL(time.Hour),
    pocket.WithEvictionCallback(func(key string, value any) {
        // Log cache evictions for monitoring
        metrics.CacheEviction(key)
    }),
)

// 2. Session storage
sessionStore := pocket.NewStore(
    pocket.WithMaxEntries(5000),
    pocket.WithTTL(30 * time.Minute), // 30-minute sessions
    pocket.WithEvictionCallback(func(key string, value any) {
        session := value.(Session)
        // Cleanup session resources
        cleanupSession(session)
    }),
)

// 3. Rate limiting
rateLimitStore := pocket.NewStore(
    pocket.WithMaxEntries(100000),
    pocket.WithTTL(time.Minute), // Reset limits every minute
)
```

## Type-Safe Storage

Use TypedStore for compile-time type safety:

```go
// Create a typed store for User objects
userStore := pocket.NewTypedStore[User](store)

// Type-safe operations
user := User{ID: "123", Name: "Alice"}
err := userStore.Set(ctx, "user:123", user)

// Retrieved value is already typed
retrieved, exists, err := userStore.Get(ctx, "user:123")
if exists {
    // 'retrieved' is User, not any
    fmt.Printf("User name: %s\n", retrieved.Name)
}

// Custom typed store wrapper
type UserRepository struct {
    store pocket.TypedStore[User]
}

func (r *UserRepository) Save(ctx context.Context, user User) error {
    return r.store.Set(ctx, user.ID, user)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (User, bool, error) {
    return r.store.Get(ctx, id)
}
```

## State Patterns

### 1. Workflow Context Pattern

Share context throughout a workflow:

```go
type WorkflowContext struct {
    RequestID   string
    UserID      string
    StartTime   time.Time
    Metadata    map[string]any
}

// Initialize context at workflow start
initContext := pocket.NewNode[Request, Request]("init-context",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        req Request, prep, exec any) (Request, string, error) {
        
        workflowCtx := WorkflowContext{
            RequestID: req.ID,
            UserID:    req.UserID,
            StartTime: time.Now(),
            Metadata:  make(map[string]any),
        }
        
        store.Set(ctx, "workflow:context", workflowCtx)
        return req, "process", nil
    }),
)

// Access context in other nodes
processNode := pocket.NewNode[Request, Response]("process",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, req Request) (any, error) {
        workflowCtx, _ := store.Get(ctx, "workflow:context")
        
        return map[string]any{
            "request": req,
            "context": workflowCtx.(WorkflowContext),
        }, nil
    }),
)
```

### 2. Accumulator Pattern

Collect results across multiple nodes:

```go
// Node that adds to accumulator
collectNode := pocket.NewNode[Item, Item]("collect",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        item Item, prep, exec any) (Item, string, error) {
        
        // Get current accumulator
        var items []Item
        if existing, exists := store.Get(ctx, "accumulated:items"); exists {
            items = existing.([]Item)
        }
        
        // Add new item
        items = append(items, item)
        store.Set(ctx, "accumulated:items", items)
        
        return item, "next", nil
    }),
)

// Final node processes accumulated results
finalizeNode := pocket.NewNode[any, Result]("finalize",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
        items, _ := store.Get(ctx, "accumulated:items")
        return items, nil
    }),
    pocket.WithExec(func(ctx context.Context, items any) (Result, error) {
        return processAllItems(items.([]Item)), nil
    }),
)
```

### 3. Checkpoint Pattern

Save progress for recovery:

```go
// Save checkpoint after each major step
checkpointNode := pocket.NewNode[Data, Data]("checkpoint",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        data Data, prep, exec any) (Data, string, error) {
        
        checkpoint := Checkpoint{
            Step:      "processing-complete",
            Data:      data,
            Timestamp: time.Now(),
        }
        
        store.Set(ctx, "checkpoint:latest", checkpoint)
        store.Set(ctx, fmt.Sprintf("checkpoint:%d", time.Now().Unix()), checkpoint)
        
        return data, "next", nil
    }),
)

// Recovery node
recoverNode := pocket.NewNode[Request, Data]("recover",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, req Request) (any, error) {
        // Check for existing checkpoint
        if checkpoint, exists := store.Get(ctx, "checkpoint:latest"); exists {
            return checkpoint.(Checkpoint), nil
        }
        return nil, nil
    }),
    pocket.WithExec(func(ctx context.Context, checkpoint any) (Data, error) {
        if checkpoint != nil {
            // Resume from checkpoint
            return checkpoint.(Checkpoint).Data, nil
        }
        // Start fresh
        return Data{}, nil
    }),
)
```

### 4. Transaction Pattern

Implement transactional semantics:

```go
type Transaction struct {
    ID      string
    Changes []Change
    Status  string
}

// Begin transaction
beginTx := pocket.NewNode[Request, Transaction]("begin-tx",
    pocket.WithExec(func(ctx context.Context, req Request) (Transaction, error) {
        return Transaction{
            ID:      generateTxID(),
            Changes: []Change{},
            Status:  "pending",
        }, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        req Request, prep, tx any) (Transaction, string, error) {
        
        transaction := tx.(Transaction)
        store.Set(ctx, "tx:current", transaction)
        store.Set(ctx, "tx:"+transaction.ID, transaction)
        
        return transaction, "execute", nil
    }),
)

// Commit or rollback
commitTx := pocket.NewNode[Result, FinalResult]("commit-tx",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, result Result) (any, error) {
        tx, _ := store.Get(ctx, "tx:current")
        return map[string]any{
            "result": result,
            "tx":     tx.(Transaction),
        }, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        result Result, prep, exec any) (FinalResult, string, error) {
        
        data := prep.(map[string]any)
        tx := data["tx"].(Transaction)
        
        if result.Success {
            tx.Status = "committed"
            store.Set(ctx, "tx:"+tx.ID, tx)
            return FinalResult{Success: true}, "done", nil
        }
        
        // Rollback
        tx.Status = "rolled-back"
        store.Set(ctx, "tx:"+tx.ID, tx)
        // Undo changes...
        
        return FinalResult{Success: false}, "done", nil
    }),
)
```

## Best Practices

### 1. Use Clear Key Naming

```go
// Good: hierarchical, descriptive keys
store.Set(ctx, "user:123:profile", profile)
store.Set(ctx, "cache:api:user:123", userData)
store.Set(ctx, "session:abc123:data", sessionData)

// Avoid: ambiguous keys
store.Set(ctx, "data", data)      // What data?
store.Set(ctx, "123", user)       // What does 123 represent?
store.Set(ctx, "temp", temp)      // When to clean up?
```

### 2. Clean Up Temporary Data

```go
// Use Post phase to clean up
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input Input, prep, exec any) (Output, string, error) {
        
        // Clean up temporary data
        store.Delete(ctx, "temp:processing:"+input.ID)
        store.Delete(ctx, "temp:validation:"+input.ID)
        
        return exec.(Output), "done", nil
    }),
)

// Or use OnComplete hook
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
        // Always runs, even on error
        store.Delete(ctx, "temp:lock:processor")
    }),
)
```

### 3. Document Store Dependencies

```go
// ProcessOrderNode requires the following store entries:
// - "config:pricing" (PricingConfig): Current pricing configuration
// - "user:{userID}:profile" (UserProfile): User profile data
// 
// Produces:
// - "order:{orderID}" (Order): Processed order
// - "metrics:orders:daily" (OrderMetrics): Updated daily metrics
func ProcessOrderNode() pocket.Node {
    // Implementation...
}
```

### 4. Handle Missing Data Gracefully

```go
pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input Input) (any, error) {
    // Provide defaults for missing data
    config, exists := store.Get(ctx, "config")
    if !exists {
        config = defaultConfig()
    }
    
    // Or fail fast if required
    required, exists := store.Get(ctx, "required:data")
    if !exists {
        return nil, errors.New("required data not found in store")
    }
    
    return map[string]any{
        "config":   config,
        "required": required,
    }, nil
})
```

### 5. Use Scopes for Isolation

```go
// Each workflow run gets its own scope
func RunWorkflow(workflowID string, parentStore pocket.Store) {
    runID := generateRunID()
    runStore := parentStore.Scope(fmt.Sprintf("run:%s:%s", workflowID, runID))
    
    graph := pocket.NewGraph(startNode, runStore)
    // All state is isolated to this run
}
```

## Advanced Patterns

### Store Middleware

Wrap stores with additional functionality:

```go
type LoggingStore struct {
    inner  pocket.Store
    logger Logger
}

func (s *LoggingStore) Set(ctx context.Context, key string, value any) error {
    s.logger.Debug("Store set", "key", key, "type", fmt.Sprintf("%T", value))
    return s.inner.Set(ctx, key, value)
}

func (s *LoggingStore) Get(ctx context.Context, key string) (any, bool) {
    value, exists := s.inner.Get(ctx, key)
    s.logger.Debug("Store get", "key", key, "exists", exists)
    return value, exists
}

// Usage
store := &LoggingStore{
    inner:  pocket.NewStore(),
    logger: logger,
}
```

### Persistent Store Adapter

```go
type PersistentStore struct {
    memory pocket.Store
    db     Database
}

func (s *PersistentStore) Set(ctx context.Context, key string, value any) error {
    // Write to memory first
    if err := s.memory.Set(ctx, key, value); err != nil {
        return err
    }
    
    // Async persist to database
    go func() {
        data, _ := json.Marshal(value)
        s.db.Save(key, data)
    }()
    
    return nil
}

func (s *PersistentStore) Get(ctx context.Context, key string) (any, bool) {
    // Try memory first
    if value, exists := s.memory.Get(ctx, key); exists {
        return value, true
    }
    
    // Fallback to database
    if data, err := s.db.Load(key); err == nil {
        var value any
        json.Unmarshal(data, &value)
        s.memory.Set(ctx, key, value) // Cache it
        return value, true
    }
    
    return nil, false
}
```

## Summary

Effective state management in Pocket involves:

1. **Understanding store operations** and read/write separation
2. **Using scopes** for isolation and organization
3. **Leveraging bounded stores** for memory management
4. **Applying patterns** for common scenarios
5. **Following best practices** for maintainable workflows

The store system provides the foundation for building stateful, coordinated workflows while maintaining clean separation of concerns through the Prep/Exec/Post lifecycle.