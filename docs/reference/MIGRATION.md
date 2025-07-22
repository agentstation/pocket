# Migration Guide

## Overview

This guide helps you migrate to Pocket from other workflow frameworks or upgrade between Pocket versions.

## From Traditional Workflow Engines

### From Apache Airflow

#### Airflow DAG to Pocket Graph

**Airflow:**
```python
from airflow import DAG
from airflow.operators.python_operator import PythonOperator

dag = DAG('example_dag')

def extract():
    return fetch_data()

def transform(data):
    return process_data(data)

def load(data):
    save_data(data)

extract_task = PythonOperator(task_id='extract', python_callable=extract, dag=dag)
transform_task = PythonOperator(task_id='transform', python_callable=transform, dag=dag)
load_task = PythonOperator(task_id='load', python_callable=load, dag=dag)

extract_task >> transform_task >> load_task
```

**Pocket equivalent:**
```go
// Define nodes
extract := pocket.NewNode[any, RawData]("extract",
    pocket.WithExec(func(ctx context.Context, _ any) (RawData, error) {
        return fetchData()
    }),
)

transform := pocket.NewNode[RawData, ProcessedData]("transform",
    pocket.WithExec(func(ctx context.Context, data RawData) (ProcessedData, error) {
        return processData(data)
    }),
)

load := pocket.NewNode[ProcessedData, any]("load",
    pocket.WithExec(func(ctx context.Context, data ProcessedData) (any, error) {
        return saveData(data)
    }),
)

// Connect nodes
extract.Connect("default", transform)
transform.Connect("default", load)

// Create and run graph
graph := pocket.NewGraph(extract, pocket.NewStore())
result, err := graph.Run(context.Background(), nil)
```

#### Key Differences

| Airflow | Pocket |
|---------|---------|
| Python-based | Go-based |
| Task scheduling focus | Data flow focus |
| External orchestrator | In-process execution |
| DAG with tasks | Graph with nodes |
| XCom for data passing | Direct type-safe passing |

### From Temporal

#### Temporal Workflow to Pocket

**Temporal:**
```go
func ProcessOrderWorkflow(ctx workflow.Context, order Order) error {
    // Validate
    var validationResult ValidationResult
    err := workflow.ExecuteActivity(ctx, ValidateOrder, order).Get(ctx, &validationResult)
    if err != nil {
        return err
    }
    
    // Process payment
    var paymentResult PaymentResult
    err = workflow.ExecuteActivity(ctx, ProcessPayment, order).Get(ctx, &paymentResult)
    if err != nil {
        return err
    }
    
    // Ship
    return workflow.ExecuteActivity(ctx, ShipOrder, order).Get(ctx, nil)
}
```

**Pocket equivalent:**
```go
// Define nodes for each activity
validate := pocket.NewNode[Order, ValidationResult]("validate",
    pocket.WithExec(validateOrder),
)

processPayment := pocket.NewNode[Order, PaymentResult]("process-payment",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, order Order) (any, error) {
        validation, _ := store.Get(ctx, "validation:result")
        return map[string]any{
            "order":      order,
            "validation": validation,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, data any) (PaymentResult, error) {
        d := data.(map[string]any)
        return processPayment(d["order"].(Order))
    }),
)

ship := pocket.NewNode[Order, ShipmentResult]("ship",
    pocket.WithExec(shipOrder),
)

// Connect workflow
validate.Connect("valid", processPayment)
processPayment.Connect("success", ship)
```

#### Key Differences

| Temporal | Pocket |
|----------|---------|
| Distributed execution | In-process by default |
| Activity-based | Node-based |
| Durable execution | Stateless execution |
| External service | Library |
| Workflow as code | Graph-based |

### From AWS Step Functions

#### Step Functions to Pocket

**Step Functions (JSON):**
```json
{
  "StartAt": "ValidateInput",
  "States": {
    "ValidateInput": {
      "Type": "Task",
      "Resource": "arn:aws:lambda:validate",
      "Next": "ProcessData"
    },
    "ProcessData": {
      "Type": "Parallel",
      "Branches": [
        {
          "StartAt": "ProcessA",
          "States": {
            "ProcessA": {
              "Type": "Task",
              "Resource": "arn:aws:lambda:processA",
              "End": true
            }
          }
        },
        {
          "StartAt": "ProcessB",
          "States": {
            "ProcessB": {
              "Type": "Task",
              "Resource": "arn:aws:lambda:processB",
              "End": true
            }
          }
        }
      ],
      "Next": "Aggregate"
    }
  }
}
```

**Pocket equivalent:**
```go
// Validation node
validate := pocket.NewNode[Input, ValidatedInput]("validate",
    pocket.WithExec(validateInput),
)

// Parallel processing
processA := pocket.NewNode[ValidatedInput, ResultA]("processA",
    pocket.WithExec(processDataA),
)

processB := pocket.NewNode[ValidatedInput, ResultB]("processB",
    pocket.WithExec(processDataB),
)

// Fan-out for parallel execution
parallelProcess := pocket.NewNode[ValidatedInput, []any]("parallel",
    pocket.WithExec(func(ctx context.Context, input ValidatedInput) ([]any, error) {
        // Run A and B in parallel
        results := make([]any, 2)
        errors := make([]error, 2)
        
        var wg sync.WaitGroup
        wg.Add(2)
        
        go func() {
            defer wg.Done()
            results[0], errors[0] = processA.Exec(ctx, input)
        }()
        
        go func() {
            defer wg.Done()
            results[1], errors[1] = processB.Exec(ctx, input)
        }()
        
        wg.Wait()
        
        // Check errors
        for _, err := range errors {
            if err != nil {
                return nil, err
            }
        }
        
        return results, nil
    }),
)

// Connect
validate.Connect("default", parallelProcess)
```

## From Node-RED

### Node-RED Flow to Pocket

**Node-RED concepts map to Pocket:**

| Node-RED | Pocket |
|----------|---------|
| Flow | Graph |
| Node | Node |
| Message | Typed data |
| Wire | Connection |
| Context | Store |

**Example migration:**

```javascript
// Node-RED function node
msg.payload = msg.payload.toUpperCase();
return msg;
```

**Pocket equivalent:**
```go
upperNode := pocket.NewNode[string, string]("uppercase",
    pocket.WithExec(func(ctx context.Context, input string) (string, error) {
        return strings.ToUpper(input), nil
    }),
)
```

## Version Migration

### From Pocket v0.x to v1.x

#### Major Changes

1. **Node is now an interface**
   ```go
   // Old: Node was a struct
   node := &pocket.Node{Name: "processor"}
   
   // New: Node is an interface
   node := pocket.NewNode[In, Out]("processor", opts...)
   ```

2. **Simplified store API**
   ```go
   // Old: Separate bounded store
   store := stores.NewBoundedStore(1000)
   
   // New: Built-in bounded functionality
   store := pocket.NewStore(
       pocket.WithMaxEntries(1000),
       pocket.WithTTL(5 * time.Minute),
   )
   ```

3. **Graph implements Node**
   ```go
   // Old: AsNode() required
   subGraph := graph.AsNode("sub")
   
   // New: Direct usage
   mainNode.Connect("default", subGraph)
   ```

#### Migration Steps

1. **Update imports**
   ```go
   // Remove internal imports
   // import "github.com/agentstation/pocket/internal/..."
   
   // Use public APIs
   import "github.com/agentstation/pocket"
   import "github.com/agentstation/pocket/middleware"
   ```

2. **Update node creation**
   ```go
   // Old
   node := pocket.TypedNode[In, Out]("name", processor)
   
   // New
   node := pocket.NewNode[In, Out]("name",
       pocket.WithExec(processor),
   )
   ```

3. **Update store usage**
   ```go
   // Old
   store := stores.NewBoundedStore(1000)
   stats := store.GetStats()
   
   // New
   store := pocket.NewStore(pocket.WithMaxEntries(1000))
   // Stats removed - use metrics middleware
   ```

### Backward Compatibility

Most v0.x code works without changes:

```go
// This still works
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithExec(func(ctx context.Context, in Input) (Output, error) {
        return process(in), nil
    }),
)

graph := pocket.NewGraph(node, store)
result, err := graph.Run(ctx, input)
```

## Best Practices for Migration

### 1. Incremental Migration

Migrate one workflow at a time:

```go
// Start with simple workflows
func MigrateSimpleWorkflow() *pocket.Graph {
    // Port node by node
    node1 := migrateNode1()
    node2 := migrateNode2()
    
    // Connect
    node1.Connect("default", node2)
    
    return pocket.NewGraph(node1, pocket.NewStore())
}
```

### 2. Type Safety First

Add types during migration:

```go
// Instead of any/interface{}
old := pocket.NewNode[any, any]("processor", ...)

// Use specific types
new := pocket.NewNode[OrderRequest, OrderResponse]("processor", ...)
```

### 3. Test During Migration

Create parallel tests:

```go
func TestMigration(t *testing.T) {
    // Test old implementation
    oldResult := runOldWorkflow(input)
    
    // Test new implementation
    newResult := runPocketWorkflow(input)
    
    // Compare results
    assert.Equal(t, oldResult, newResult)
}
```

### 4. Leverage Pocket Features

Take advantage of Pocket's features:

```go
// Add type validation
if err := pocket.ValidateGraph(startNode); err != nil {
    log.Fatal("Invalid workflow:", err)
}

// Add resilience
node = pocket.NewNode[In, Out]("resilient",
    pocket.Steps{
        Exec: processor,
        Fallback: fallbackProcessor, // Receives prepResult and error
    },
    pocket.WithRetry(3, time.Second),
)

// Add observability
node = middleware.WithLogging(logger)(node)
node = middleware.WithMetrics(metrics)(node)
```

## Common Patterns Translation

### Error Handling

**Traditional try-catch:**
```python
try:
    result = process(data)
except Exception as e:
    result = fallback(data)
```

**Pocket pattern:**
```go
node := pocket.NewNode[Data, Result]("processor",
    pocket.Steps{
        Exec: func(ctx context.Context, prepResult any) (any, error) {
            data := prepResult.(Data)
            return process(data)
        },
        Fallback: func(ctx context.Context, prepResult any, err error) (any, error) {
            data := prepResult.(Data)
            return fallback(data), nil
        },
    },
)
```

### State Management

**Global state:**
```python
global_state = {}

def process(data):
    global_state['last'] = data
    return transform(data)
```

**Pocket pattern:**
```go
node := pocket.NewNode[Data, Result]("processor",
    pocket.WithExec(func(ctx context.Context, data Data) (Result, error) {
        return transform(data), nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input Data, prep, result any) (Result, string, error) {
        
        store.Set(ctx, "last", input)
        return result.(Result), "next", nil
    }),
)
```

### Conditional Logic

**If-else chains:**
```python
if condition1:
    result = process1(data)
elif condition2:
    result = process2(data)
else:
    result = process3(data)
```

**Pocket pattern:**
```go
router := pocket.NewNode[Data, Result]("router",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        data Data, prep, exec any) (Result, string, error) {
        
        if condition1(data) {
            return exec.(Result), "process1", nil
        } else if condition2(data) {
            return exec.(Result), "process2", nil
        }
        return exec.(Result), "process3", nil
    }),
)

router.Connect("process1", processor1)
router.Connect("process2", processor2)
router.Connect("process3", processor3)
```

## Troubleshooting Migration

### Common Issues

1. **Type Mismatches**
   ```go
   // Error: type mismatch at connection
   // Solution: Ensure connected nodes have compatible types
   err := pocket.ValidateGraph(startNode)
   ```

2. **Missing Connections**
   ```go
   // Error: no route "success" from node "processor"
   // Solution: Add missing connection
   processor.Connect("success", nextNode)
   ```

3. **Store Access in Exec**
   ```go
   // Error: cannot access store in Exec
   // Solution: Move store operations to Prep or Post
   ```

4. **Circular Dependencies**
   ```go
   // Error: circular reference detected
   // Solution: Redesign workflow to avoid cycles
   ```

## Migration Checklist

- [ ] Identify workflow patterns in existing system
- [ ] Map concepts to Pocket equivalents
- [ ] Define types for inputs and outputs
- [ ] Create nodes for each processing step
- [ ] Implement Prep/Exec/Post logic
- [ ] Connect nodes to form workflow
- [ ] Add error handling and retries
- [ ] Add observability (logging, metrics)
- [ ] Validate graph structure
- [ ] Test with production-like data
- [ ] Compare results with original system
- [ ] Plan rollout strategy

## Summary

Migrating to Pocket involves:

1. **Understanding the conceptual mapping** from your current framework
2. **Leveraging type safety** for more reliable workflows
3. **Adopting the Prep/Exec/Post pattern** for clean separation of concerns
4. **Taking advantage of built-in features** like retries and fallbacks
5. **Testing thoroughly** during migration

The effort invested in migration pays off through improved type safety, better testability, and cleaner code organization.