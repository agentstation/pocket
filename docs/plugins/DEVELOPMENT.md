# Pocket Plugin Development Guide

This guide explains how to create custom nodes for the Pocket framework. Whether you're adding a new built-in node or preparing for future plugin support, this guide covers the essential concepts and patterns.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Creating a Built-in Node](#creating-a-built-in-node)
- [Node Lifecycle](#node-lifecycle)
- [Best Practices](#best-practices)
- [Testing Your Node](#testing-your-node)
- [Future Plugin System](#future-plugin-system)

## Overview

Pocket's plugin system is designed around the `NodeBuilder` interface, which provides a clean separation between node configuration (metadata) and runtime behavior (implementation). This architecture enables:

- **Type Safety**: Nodes can specify their input/output types
- **Validation**: Automatic config validation using JSON Schema
- **Documentation**: Self-documenting nodes with examples
- **Discoverability**: CLI integration for listing and inspecting nodes

## Architecture

### Key Interfaces

```go
// NodeBuilder creates nodes and provides metadata
type NodeBuilder interface {
    Metadata() NodeMetadata
    Build(def *yaml.NodeDefinition) (pocket.Node, error)
}

// NodeMetadata describes a node type
type NodeMetadata struct {
    Type         string                 `json:"type"`
    Category     string                 `json:"category"`
    Description  string                 `json:"description"`
    InputSchema  map[string]interface{} `json:"inputSchema,omitempty"`
    OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
    ConfigSchema map[string]interface{} `json:"configSchema"`
    Examples     []Example              `json:"examples,omitempty"`
    Since        string                 `json:"since,omitempty"`
}
```

### Node Categories

Nodes are organized into categories for better organization:

- **core**: Fundamental workflow control (echo, delay, router, conditional)
- **data**: Data manipulation and transformation (transform, template, jsonpath, validate, aggregate)
- **io**: Input/output operations (http, file, exec)
- **flow**: Advanced flow control (parallel, retry, cache)

## Creating a Built-in Node

Here's a step-by-step guide to creating a new built-in node:

### 1. Design Your Node

First, determine:
- What problem does your node solve?
- What configuration options does it need?
- What are the input and output types?
- Which category does it belong to?

### 2. Implement the NodeBuilder

Create your node builder in `builtin/builders.go`:

```go
// MyNodeBuilder builds custom processing nodes
type MyNodeBuilder struct {
    Verbose bool
}

func (b *MyNodeBuilder) Metadata() NodeMetadata {
    return NodeMetadata{
        Type:        "mynode",
        Category:    "data",
        Description: "Processes data in a custom way",
        ConfigSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "mode": map[string]interface{}{
                    "type":        "string",
                    "description": "Processing mode",
                    "enum":        []interface{}{"fast", "accurate"},
                    "default":     "fast",
                },
                "threshold": map[string]interface{}{
                    "type":        "number",
                    "description": "Processing threshold",
                    "minimum":     0,
                    "maximum":     1,
                },
            },
            "required": []string{"threshold"},
        },
        Examples: []Example{
            {
                Name:        "Basic processing",
                Description: "Process with default settings",
                Config: map[string]interface{}{
                    "threshold": 0.5,
                },
            },
        },
        Since: "1.0.0",
    }
}

func (b *MyNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    // Extract configuration
    threshold, ok := def.Config["threshold"].(float64)
    if !ok {
        return nil, fmt.Errorf("threshold must be a number")
    }
    
    mode, _ := def.Config["mode"].(string)
    if mode == "" {
        mode = "fast"
    }
    
    // Create the node with Prep/Exec/Post lifecycle
    return pocket.NewNode[any, any](def.Name,
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
            // Validation and preparation
            if b.Verbose {
                log.Printf("[%s] Preparing with mode=%s, threshold=%f", def.Name, mode, threshold)
            }
            
            // Load any required state
            config, _ := store.Get(ctx, "config")
            
            return map[string]interface{}{
                "input":     input,
                "config":    config,
                "threshold": threshold,
            }, nil
        }),
        
        pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
            // Core processing logic (pure function)
            data := prepData.(map[string]interface{})
            
            // Your custom processing here
            result := processData(data["input"], data["threshold"].(float64), mode)
            
            return result, nil
        }),
        
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
            // Save state and determine routing
            store.Set(ctx, "lastResult", result)
            
            // Determine next node based on result
            if isSuccessful(result) {
                return result, "success", nil
            }
            return result, "failure", nil
        }),
    ), nil
}
```

### 3. Register Your Node

Add your node to the registry in `builtin/registry.go`:

```go
func RegisterAll(loader *yaml.Loader, verbose bool) *Registry {
    registry := NewRegistry()
    
    // ... existing registrations ...
    
    // Register your node
    registry.Register(&MyNodeBuilder{Verbose: verbose})
    
    // ... rest of function ...
}
```

### 4. Add CLI Support

Update `cmd/pocket/nodes.go` to include your node in the CLI:

```go
func getBuiltinNodes() []builtin.NodeMetadata {
    return []builtin.NodeMetadata{
        // ... existing nodes ...
        (&builtin.MyNodeBuilder{}).Metadata(),
    }
}
```

### 5. Write Tests

Create comprehensive tests in `builtin/builders_test.go`:

```go
func TestMyNode(t *testing.T) {
    ctx := context.Background()
    store := pocket.NewStore()
    
    t.Run("basic processing", func(t *testing.T) {
        builder := &MyNodeBuilder{}
        def := &yaml.NodeDefinition{
            Name: "test-mynode",
            Config: map[string]interface{}{
                "threshold": 0.5,
                "mode":      "fast",
            },
        }
        
        node, err := builder.Build(def)
        if err != nil {
            t.Fatalf("Failed to build node: %v", err)
        }
        
        graph := pocket.NewGraph(node, store)
        result, err := graph.Run(ctx, testInput)
        if err != nil {
            t.Fatalf("Failed to run graph: %v", err)
        }
        
        // Verify result
        if !isExpectedResult(result) {
            t.Errorf("Unexpected result: %v", result)
        }
    })
    
    t.Run("validation", func(t *testing.T) {
        // Test config validation
        def := &yaml.NodeDefinition{
            Name: "test-mynode",
            Config: map[string]interface{}{
                // Missing required threshold
            },
        }
        
        _, err := builder.Build(def)
        if err == nil {
            t.Error("Expected validation error")
        }
    })
}
```

### 6. Create Example Workflow

Add an example workflow in `examples/cli/`:

```yaml
name: mynode-example
description: Demonstrates custom node processing
version: 1.0.0

nodes:
  - name: prepare
    type: echo
    config:
      message: "Starting processing..."
      
  - name: process
    type: mynode
    config:
      threshold: 0.7
      mode: accurate
      
  - name: success
    type: echo
    config:
      message: "Processing successful!"
      
  - name: failure
    type: echo
    config:
      message: "Processing failed!"

connections:
  - from: prepare
    to: process
    
  - from: process
    to: success
    action: success
    
  - from: process
    to: failure
    action: failure

start: prepare
```

## Node Lifecycle

Understanding the Prep/Exec/Post lifecycle is crucial for building effective nodes:

### Prep Phase
- **Purpose**: Validate input, load configuration, prepare data
- **Store Access**: Read-only (StoreReader)
- **Best For**: 
  - Input validation
  - Loading configuration from store
  - Data transformation for exec phase

### Exec Phase
- **Purpose**: Core business logic execution
- **Store Access**: None (pure function)
- **Best For**:
  - Computation
  - API calls
  - Data processing
  - Any stateless operations

### Post Phase
- **Purpose**: Save results, determine routing
- **Store Access**: Read/write (StoreWriter)
- **Best For**:
  - Storing results in state
  - Determining next node
  - Cleanup operations
  - Logging/metrics

## Best Practices

### 1. Configuration Design

- Use JSON Schema for validation
- Provide sensible defaults
- Use enums for restricted values
- Document all options clearly

```go
ConfigSchema: map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "timeout": map[string]interface{}{
            "type":        "string",
            "description": "Operation timeout",
            "default":     "30s",
            "pattern":     "^[0-9]+(s|m|h)$",
        },
    },
}
```

### 2. Error Handling

- Return descriptive errors
- Include context in error messages
- Handle edge cases gracefully

```go
if err != nil {
    return nil, fmt.Errorf("failed to process data: %w", err)
}
```

### 3. Logging

- Use verbose flag for debug output
- Log at appropriate levels
- Include node name in logs

```go
if b.Verbose {
    log.Printf("[%s] Processing with config: %+v", def.Name, config)
}
```

### 4. Type Safety

- Use generics when types are known
- Validate type assertions
- Provide clear type documentation

```go
// Type-safe node
pocket.NewNode[InputType, OutputType](name, ...)

// Dynamic typing
pocket.NewNode[any, any](name, ...)
```

### 5. Testing

- Test all configuration options
- Test error cases
- Test with various input types
- Verify routing logic

## Testing Your Node

### Unit Tests

Test each phase independently:

```go
// Test Prep phase
prepResult, err := node.Prep(ctx, store, input)

// Test Exec phase
execResult, err := node.Exec(ctx, prepResult)

// Test Post phase
output, next, err := node.Post(ctx, store, input, prepResult, execResult)
```

### Integration Tests

Test the complete workflow:

```go
graph := pocket.NewGraph(node, store)
result, err := graph.Run(ctx, input)
```

### Validation Tests

Ensure config validation works:

```go
// Should fail with invalid config
_, err := builder.Build(invalidDef)
if err == nil {
    t.Error("Expected validation error")
}
```

## Future Plugin System

While current nodes are built-in, the architecture supports future plugin systems:

### Phase 2: Lua Scripting (Planned)
- Embedded Lua interpreter
- Sandboxed execution
- Access to workflow data

### Phase 3: External Plugins (Planned)
- WebAssembly (WASM) support
- RPC-based plugins
- Language-agnostic development

### Plugin Interface (Future)

```go
// Future plugin interface
type Plugin interface {
    NodeBuilder
    Initialize(config PluginConfig) error
    Shutdown() error
}
```

## Examples

### Simple Processing Node

```go
type SimpleNodeBuilder struct{}

func (b *SimpleNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    return pocket.NewNode[any, any](def.Name,
        pocket.WithExec(func(ctx context.Context, input any) (any, error) {
            // Simple transformation
            return transform(input), nil
        }),
    ), nil
}
```

### Stateful Node

```go
type StatefulNodeBuilder struct{}

func (b *StatefulNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    return pocket.NewNode[any, any](def.Name,
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
            // Load state
            state, _ := store.Get(ctx, "nodeState")
            return map[string]interface{}{
                "input": input,
                "state": state,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
            // Process with state
            data := prepData.(map[string]interface{})
            return processWithState(data["input"], data["state"]), nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
            // Update state
            store.Set(ctx, "nodeState", result)
            return result, "next", nil
        }),
    ), nil
}
```

### Routing Node

```go
type RouterNodeBuilder struct{}

func (b *RouterNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    routes := def.Config["routes"].(map[string]interface{})
    
    return pocket.NewNode[any, any](def.Name,
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
            // Dynamic routing based on result
            route := determineRoute(result, routes)
            return result, route, nil
        }),
    ), nil
}
```

## Conclusion

Building nodes for Pocket is straightforward once you understand the core concepts:

1. Implement the `NodeBuilder` interface
2. Define comprehensive metadata
3. Follow the Prep/Exec/Post lifecycle
4. Write thorough tests
5. Document with examples

The plugin system is designed to grow with your needs, from simple built-in nodes to future support for scripting and external plugins.

For more examples, see the existing node implementations in `builtin/builders.go`.