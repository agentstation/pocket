# Go Node Development Guide

This guide explains how to create custom nodes in Go for the Pocket framework. Whether you're extending Pocket for your organization or contributing to the project, this guide will help you build robust, type-safe nodes.

## Table of Contents

- [Overview](#overview)
- [Node Architecture](#node-architecture)
- [Creating a Simple Node](#creating-a-simple-node)
- [Node Builder Pattern](#node-builder-pattern)
- [Advanced Features](#advanced-features)
- [Testing Your Node](#testing-your-node)
- [Contributing to Pocket](#contributing-to-pocket)

## Overview

Pocket nodes are the building blocks of workflows. Each node:
- Implements the `pocket.Node` interface
- Follows the Prep/Exec/Post lifecycle
- Can be strongly typed using generics
- Connects to other nodes to form graphs

## Node Architecture

### The Node Interface

```go
type Node interface {
    Name() string
    Prep(ctx context.Context, store StoreReader, input any) (any, error)
    Exec(ctx context.Context, prepData any) (any, error)
    Post(ctx context.Context, store StoreWriter, input, prepData, result any) (any, string, error)
    Connect(action string, next Node)
    Successors() map[string]Node
    InputType() reflect.Type
    OutputType() reflect.Type
}
```

### Lifecycle Phases

1. **Prep** - Validate input, load data from store (read-only access)
2. **Exec** - Core business logic (no store access, pure function)
3. **Post** - Process results, update store, determine routing

## Creating a Simple Node

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/agentstation/pocket"
)

// WordCountNode counts words in text
type WordCountNode struct {
    node pocket.Node
}

func NewWordCountNode(name string) *WordCountNode {
    n := &WordCountNode{}
    
    n.node = pocket.NewNode[string, map[string]int](name,
        pocket.WithExec(func(ctx context.Context, text string) (map[string]int, error) {
            words := strings.Fields(text)
            
            counts := make(map[string]int)
            for _, word := range words {
                word = strings.ToLower(word)
                counts[word]++
            }
            
            return counts, nil
        }),
    )
    
    return n
}

// Delegate Node interface methods
func (n *WordCountNode) Name() string { return n.node.Name() }
func (n *WordCountNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
    return n.node.Prep(ctx, store, input)
}
// ... implement other Node methods
```

### Using Functional Options

```go
func NewAdvancedNode(name string) pocket.Node {
    return pocket.NewNode[Input, Output](name,
        // Prep phase: validate and prepare data
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, in Input) (any, error) {
            // Validate input
            if in.Value == "" {
                return nil, fmt.Errorf("value cannot be empty")
            }
            
            // Load configuration from store
            config, _ := store.Get(ctx, "config")
            
            return map[string]any{
                "input": in,
                "config": config,
            }, nil
        }),
        
        // Exec phase: core logic
        pocket.WithExec(func(ctx context.Context, prepData any) (Output, error) {
            data := prepData.(map[string]any)
            input := data["input"].(Input)
            
            // Process data
            result := processData(input)
            
            return Output{
                Result: result,
                Status: "completed",
            }, nil
        }),
        
        // Post phase: routing and state updates
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, 
            input Input, prepData any, result Output) (Output, string, error) {
            
            // Update store
            store.Set(ctx, "lastResult", result)
            
            // Determine next node
            next := "success"
            if result.Status == "error" {
                next = "error"
            }
            
            return result, next, nil
        }),
        
        // Additional options
        pocket.WithRetry(3, time.Second),
        pocket.WithTimeout(30 * time.Second),
    )
}
```

## Node Builder Pattern

For integration with Pocket's plugin system, implement the NodeBuilder interface:

### NodeBuilder Interface

```go
type NodeBuilder interface {
    Metadata() NodeMetadata
    Build(def *yaml.NodeDefinition) (pocket.Node, error)
}

type NodeMetadata struct {
    Type         string                 `json:"type"`
    Category     string                 `json:"category"`
    Description  string                 `json:"description"`
    ConfigSchema map[string]interface{} `json:"configSchema"`
    Examples     []Example              `json:"examples"`
    Since        string                 `json:"since,omitempty"`
}
```

### Complete Example: URL Validator Node

```go
package builtin

import (
    "context"
    "fmt"
    "net/url"
    "time"
    
    "github.com/agentstation/pocket"
    "github.com/agentstation/pocket/yaml"
)

// URLValidatorBuilder builds URL validation nodes
type URLValidatorBuilder struct {
    Verbose bool
}

func (b *URLValidatorBuilder) Metadata() NodeMetadata {
    return NodeMetadata{
        Type:        "url-validator",
        Category:    "validation",
        Description: "Validates and parses URLs",
        ConfigSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "schemes": map[string]interface{}{
                    "type":        "array",
                    "items":       map[string]interface{}{"type": "string"},
                    "default":     []string{"http", "https"},
                    "description": "Allowed URL schemes",
                },
                "require_host": map[string]interface{}{
                    "type":        "boolean",
                    "default":     true,
                    "description": "Require hostname in URL",
                },
                "timeout": map[string]interface{}{
                    "type":        "string",
                    "default":     "5s",
                    "description": "Validation timeout",
                },
            },
        },
        Examples: []Example{
            {
                Name:        "Basic validation",
                Description: "Validate HTTP/HTTPS URLs",
                Config: map[string]interface{}{
                    "schemes":      []string{"http", "https"},
                    "require_host": true,
                },
            },
        },
        Since: "v0.5.0",
    }
}

func (b *URLValidatorBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    // Parse configuration
    schemes := []string{"http", "https"}
    if s, ok := def.Config["schemes"].([]interface{}); ok {
        schemes = make([]string, len(s))
        for i, v := range s {
            schemes[i] = fmt.Sprint(v)
        }
    }
    
    requireHost := true
    if r, ok := def.Config["require_host"].(bool); ok {
        requireHost = r
    }
    
    timeoutStr := "5s"
    if t, ok := def.Config["timeout"].(string); ok {
        timeoutStr = t
    }
    timeout, err := time.ParseDuration(timeoutStr)
    if err != nil {
        return nil, fmt.Errorf("invalid timeout: %w", err)
    }
    
    // Build node with configuration
    return pocket.NewNode[string, URLInfo](def.Name,
        pocket.WithExec(func(ctx context.Context, input string) (URLInfo, error) {
            // Parse URL
            u, err := url.Parse(input)
            if err != nil {
                return URLInfo{}, fmt.Errorf("invalid URL: %w", err)
            }
            
            // Validate scheme
            validScheme := false
            for _, scheme := range schemes {
                if u.Scheme == scheme {
                    validScheme = true
                    break
                }
            }
            if !validScheme {
                return URLInfo{}, fmt.Errorf("invalid scheme %s, allowed: %v", 
                    u.Scheme, schemes)
            }
            
            // Validate host
            if requireHost && u.Host == "" {
                return URLInfo{}, fmt.Errorf("URL must have a host")
            }
            
            // Return parsed info
            return URLInfo{
                Original: input,
                Scheme:   u.Scheme,
                Host:     u.Host,
                Path:     u.Path,
                Query:    u.Query(),
                Valid:    true,
            }, nil
        }),
        pocket.WithTimeout(timeout),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            input string, prep, result any) (any, string, error) {
            
            info := result.(URLInfo)
            
            // Log if verbose
            if b.Verbose {
                fmt.Printf("[%s] Validated URL: %s (scheme=%s, host=%s)\n",
                    def.Name, input, info.Scheme, info.Host)
            }
            
            // Route based on validation
            if info.Valid {
                return info, "valid", nil
            }
            return info, "invalid", nil
        }),
    ), nil
}

// URLInfo contains parsed URL information
type URLInfo struct {
    Original string              `json:"original"`
    Scheme   string              `json:"scheme"`
    Host     string              `json:"host"`
    Path     string              `json:"path"`
    Query    url.Values          `json:"query"`
    Valid    bool                `json:"valid"`
}
```

## Advanced Features

### Error Handling

```go
node := pocket.NewNode[Input, Output]("processor",
    pocket.WithExec(processData),
    pocket.WithErrorHandler(func(ctx context.Context, err error) error {
        // Log error
        log.Printf("Processing error: %v", err)
        
        // Wrap with context
        return fmt.Errorf("node %s failed: %w", "processor", err)
    }),
    pocket.WithFallback(func(ctx context.Context, store pocket.StoreWriter, 
        input Input) (Output, error) {
        // Return default output on error
        return Output{
            Status: "failed",
            Error:  "processing failed, using fallback",
        }, nil
    }),
)
```

### Cleanup Hooks

```go
node := pocket.NewNode[Input, Output]("resource-user",
    pocket.WithExec(useResource),
    pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
        // Always runs, even on panic
        releaseResource()
    }),
    pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, 
        result Output) {
        // Runs only on success
        store.Set(ctx, "lastSuccess", time.Now())
    }),
    pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, 
        err error) {
        // Runs only on failure
        store.Set(ctx, "lastError", err.Error())
    }),
)
```

### Store Integration

```go
pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, in Input) (any, error) {
    // Read from store (read-only in Prep)
    config, exists := store.Get(ctx, "config")
    if !exists {
        return nil, fmt.Errorf("missing configuration")
    }
    
    // Use scoped stores for organization
    userStore := store.Scope("user")
    userData, _ := userStore.Get(ctx, fmt.Sprintf("%d", in.UserID))
    
    return PrepData{
        Input:    in,
        Config:   config.(Config),
        UserData: userData,
    }, nil
}),

pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
    input Input, prep PrepData, result Output) (Output, string, error) {
    
    // Write to store (read-write in Post)
    store.Set(ctx, "lastProcessed", result)
    
    // Use TTL for temporary data
    userStore := store.Scope("user")
    userStore.Set(ctx, fmt.Sprintf("session:%d", input.UserID), 
        map[string]any{
            "result": result,
            "time":   time.Now(),
        })
    
    return result, "next", nil
}),
```

## Testing Your Node

### Unit Testing

```go
func TestURLValidator(t *testing.T) {
    ctx := context.Background()
    store := pocket.NewStore()
    
    // Create builder
    builder := &URLValidatorBuilder{Verbose: false}
    
    // Build node
    node, err := builder.Build(&yaml.NodeDefinition{
        Name: "test-validator",
        Type: "url-validator",
        Config: map[string]interface{}{
            "schemes": []string{"https"},
            "require_host": true,
        },
    })
    require.NoError(t, err)
    
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid https", "https://example.com", false},
        {"invalid http", "http://example.com", true},
        {"no host", "https://", true},
        {"malformed", "not a url", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create a simple graph
            graph := pocket.NewGraph(node, store)
            
            // Run the node
            result, err := graph.Run(ctx, tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                info := result.(URLInfo)
                assert.True(t, info.Valid)
            }
        })
    }
}
```

### Testing Lifecycle Phases

```go
func TestNodeLifecycle(t *testing.T) {
    ctx := context.Background()
    store := pocket.NewStore()
    
    // Set up test data in store
    store.Set(ctx, "config", map[string]string{
        "mode": "test",
    })
    
    node := pocket.NewNode[string, string]("lifecycle-test",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, 
            input string) (any, error) {
            config, _ := store.Get(ctx, "config")
            return map[string]any{
                "input": input,
                "config": config,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (string, error) {
            data := prepData.(map[string]any)
            return fmt.Sprintf("processed: %s", data["input"]), nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            input string, prep, result any) (string, string, error) {
            store.Set(ctx, "lastResult", result)
            return result.(string), "done", nil
        }),
    )
    
    // Test individual phases
    prepResult, err := node.Prep(ctx, store, "test input")
    assert.NoError(t, err)
    assert.NotNil(t, prepResult)
    
    execResult, err := node.Exec(ctx, prepResult)
    assert.NoError(t, err)
    assert.Equal(t, "processed: test input", execResult)
    
    postResult, next, err := node.Post(ctx, store, "test input", 
        prepResult, execResult)
    assert.NoError(t, err)
    assert.Equal(t, "done", next)
    assert.Equal(t, execResult, postResult)
    
    // Verify store was updated
    saved, exists := store.Get(ctx, "lastResult")
    assert.True(t, exists)
    assert.Equal(t, execResult, saved)
}
```

## Contributing to Pocket

### Adding a New Built-in Node

1. **Implement the NodeBuilder** in `builtin/builders.go`:

```go
// SentimentAnalyzerBuilder analyzes text sentiment
type SentimentAnalyzerBuilder struct {
    Verbose bool
}

func (b *SentimentAnalyzerBuilder) Metadata() NodeMetadata {
    return NodeMetadata{
        Type:        "sentiment",
        Category:    "ai",
        Description: "Analyzes text sentiment",
        // ... full metadata
    }
}

func (b *SentimentAnalyzerBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    // Implementation
}
```

2. **Register in the registry** (`builtin/registry.go`):

```go
func RegisterAll(loader *yaml.Loader, verbose bool) *Registry {
    registry := NewRegistry()
    
    // ... existing registrations
    
    // Add your node
    registry.Register(&SentimentAnalyzerBuilder{Verbose: verbose})
    
    // ... rest of function
}
```

3. **Add to CLI** (`cmd/pocket/nodes.go`):

```go
func getBuiltinNodes() []builtin.NodeMetadata {
    return []builtin.NodeMetadata{
        // ... existing nodes
        (&builtin.SentimentAnalyzerBuilder{}).Metadata(),
    }
}
```

4. **Write comprehensive tests** in `builtin/builders_test.go`

5. **Update documentation** in `docs/NODE_TYPES.md`

### Code Style Guidelines

Follow Pocket's conventions:

1. **Use functional options** for configuration
2. **Implement all three lifecycle phases** when needed
3. **Add meaningful error messages** with context
4. **Write comprehensive tests** including edge cases
5. **Document configuration options** in metadata
6. **Use type safety** where possible

### Submitting a PR

1. Fork the repository
2. Create a feature branch
3. Write tests first (TDD)
4. Implement your node
5. Update documentation
6. Run tests and linter:
   ```bash
   make test
   make lint
   ```
7. Submit PR with clear description

## Best Practices

### 1. Type Safety

Use generics for compile-time safety:

```go
// Good: Type-safe
node := pocket.NewNode[OrderInput, OrderOutput]("process-order", ...)

// Avoid: Dynamic typing unless necessary
node := pocket.NewNode[any, any]("process-order", ...)
```

### 2. Error Context

Always add context to errors:

```go
if err := validateOrder(order); err != nil {
    return nil, fmt.Errorf("order validation failed for ID %s: %w", 
        order.ID, err)
}
```

### 3. Resource Management

Use cleanup hooks for resources:

```go
pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
    if conn != nil {
        conn.Close()
    }
})
```

### 4. Configuration Validation

Validate configuration in Build():

```go
func (b *Builder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    timeout, err := time.ParseDuration(def.Config["timeout"].(string))
    if err != nil {
        return nil, fmt.Errorf("invalid timeout: %w", err)
    }
    
    if timeout < time.Second {
        return nil, fmt.Errorf("timeout must be at least 1 second")
    }
    
    // ... build node
}
```

## See Also

- [Architecture Overview](../concepts/ARCHITECTURE.md) - Core concepts
- [Testing Guide](../guides/TESTING.md) - Testing best practices
- [API Reference](../library/api-reference.md) - Complete API documentation
- [Built-in Nodes](../NODE_TYPES.md) - Existing node implementations