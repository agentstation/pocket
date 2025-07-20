# Pocket Examples

This directory contains comprehensive examples demonstrating various features and patterns of the Pocket framework. Each example is self-contained and can be run independently.

## Overview

The examples progress from basic concepts to advanced patterns, showing how to build production-ready LLM workflows with Pocket.

## Available Examples

### 1. [Basic Workflow](../../../examples/workflow/)
**Purpose**: Introduction to basic node creation and graph execution

**Key Concepts**:
- Creating nodes with `NewNode`
- Connecting nodes to form workflows
- Running graphs with `Run()`
- Basic error handling

**Run**: `go run examples/workflow/main.go`

---

### 2. [Typed Workflows](../../../examples/typed/)
**Purpose**: Demonstrates compile-time type safety with generics

**Key Concepts**:
- Type-safe nodes with `NewNode[Input, Output]`
- Compile-time type checking
- Graph validation with `ValidateGraph`
- Type conversion and handling

**Run**: `go run examples/typed/main.go`

---

### 3. [Stateful Workflows](../../../examples/stateful/)
**Purpose**: Shows state management across workflow execution

**Key Concepts**:
- Using the Store for state persistence
- Scoped stores for isolation
- State sharing between nodes
- Context-aware store operations

**Run**: `go run examples/stateful/main.go`

---

### 4. [Parallel Processing](../../../examples/parallel/)
**Purpose**: Demonstrates concurrent execution patterns

**Key Concepts**:
- `RunConcurrent` for parallel node execution
- `FanOut` for parallel item processing
- `FanIn` for result aggregation
- Thread-safe operations

**Run**: `go run examples/parallel/main.go`

---

### 5. [Agent Pattern](../../../examples/agent/)
**Purpose**: Implements autonomous agent with think-act loops

**Key Concepts**:
- Decision-making nodes
- Dynamic routing based on analysis
- State accumulation across iterations
- Goal-directed behavior

**Run**: `go run examples/agent/main.go`

---

### 6. [Chat Workflow](../../../examples/chat/)
**Purpose**: Multi-agent chat system with routing

**Key Concepts**:
- Intent classification
- Dynamic agent selection
- Context management
- Response generation pipeline

**Run**: `go run examples/chat/main.go`

---

### 7. [RAG Pattern](../../../examples/rag/)
**Purpose**: Retrieval-Augmented Generation workflow

**Key Concepts**:
- Document retrieval simulation
- Context enhancement
- Response generation with context
- Quality scoring

**Run**: `go run examples/rag/main.go`

---

### 8. [Saga Pattern](../../../examples/saga/)
**Purpose**: Distributed transaction with compensation

**Key Concepts**:
- Transaction steps with rollback
- Compensation on failure
- State tracking for recovery
- Error handling strategies

**Run**: `go run examples/saga/main.go`

---

### 9. [Flow Composition](../../../examples/flow-composition/)
**Purpose**: Shows how to compose complex workflows

**Key Concepts**:
- Graphs as nodes (Graph implements Node)
- Nested workflows
- Subgraph composition
- Workflow reusability

**Run**: `go run examples/flow-composition/main.go`

---

### 10. [YAML Support](../../../examples/yaml/)
**Purpose**: YAML integration for LLM-friendly output

**Key Concepts**:
- YAML marshaling for structured output
- Token efficiency comparison (YAML vs JSON)
- Parsing LLM responses in YAML
- Declarative routing with YAML

**Run**: `go run examples/yaml/main.go`

---

### 11. [Advanced Features](../../../examples/advanced/)
**Purpose**: Comprehensive demonstration of advanced capabilities

**Key Concepts**:
- Circuit breaker pattern
- Fallback chains
- Cleanup hooks and resource management
- Bounded store with LRU/TTL
- Graph composition with YAML output

**Run**: `go run examples/advanced/main.go`

## Running Examples

### Prerequisites

1. Go 1.23+ installed
2. Clone the repository
3. Navigate to the project root

### Running Individual Examples

```bash
# Run from project root
go run examples/<example-name>/main.go

# Or navigate to example directory
cd examples/<example-name>
go run main.go
```

### Building Examples

```bash
# Build all examples
for dir in examples/*/; do
    echo "Building $dir"
    go build -o "$dir/$(basename $dir)" "$dir"
done

# Build specific example
go build -o examples/agent/agent examples/agent/main.go
```

## Learning Path

For beginners, we recommend following this progression:

1. **Start with Basics**
   - workflow → typed → stateful

2. **Learn Concurrency**
   - parallel → flow-composition

3. **Explore Patterns**
   - agent → rag → saga

4. **Advanced Topics**
   - chat → yaml → advanced

## Common Patterns

### Error Handling
```go
graph := pocket.NewGraph(startNode, store)
result, err := graph.Run(ctx, input)
if err != nil {
    log.Printf("Workflow failed: %v", err)
    return
}
```

### Node Creation
```go
// Basic node
node := pocket.NewNode[Input, Output]("name",
    pocket.WithExec(execFunc),
)

// Full lifecycle node
node := pocket.NewNode[Input, Output]("name",
    pocket.WithPrep(prepFunc),
    pocket.WithExec(execFunc),
    pocket.WithPost(postFunc),
)
```

### Store Usage
```go
// Set value
store.Set(ctx, "key", value)

// Get value
if val, exists := store.Get(ctx, "key"); exists {
    // Use val
}
```

### Dynamic Routing
```go
pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, 
    input, prep, exec any) (any, string, error) {
    // Determine next node based on result
    if shouldRoute {
        return result, "next-node", nil
    }
    return result, "other-node", nil
})
```

## Best Practices from Examples

1. **Use Type Safety**: Prefer typed nodes when types are known
2. **Handle Errors Gracefully**: Always check errors and provide context
3. **Leverage Prep/Exec/Post**: Use each phase for its intended purpose
4. **Keep Nodes Focused**: Single responsibility per node
5. **Use Meaningful Names**: Clear node and route names improve readability

## Contributing Examples

When adding new examples:

1. Create a new directory under `/examples/`
2. Include a clear `main.go` with comments
3. Add a README.md if the example is complex
4. Update this documentation
5. Ensure the example is self-contained
6. Include sample output in comments

## Troubleshooting

### Common Issues

**Import errors**: Ensure you're in the project root or have the module in your GOPATH

**Type mismatches**: Check that node input/output types align properly

**Store not found**: Ensure you're passing the store to graph creation

**Context canceled**: Check for proper context handling and timeouts

## Next Steps

After exploring these examples:

1. Read the [Architecture Documentation](../concepts/ARCHITECTURE.md)
2. Study the [Design Patterns](../patterns/)
3. Build your own workflows
4. Contribute improvements or new examples

## Questions?

- Check the [API Reference](../reference/API.md)
- Review the [Getting Started Guide](../guides/GETTING_STARTED.md)
- Explore the [source code](https://github.com/agentstation/pocket)