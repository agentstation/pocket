# Pocket Node Documentation

This section documents all node types available in Pocket - from built-in nodes to custom Lua scripts and WebAssembly plugins.

## Node Categories

### [Built-in Nodes](built-in/)
Pocket includes 14 built-in node types for common operations:

- **Core Nodes** (4): Basic workflow control
  - `echo` - Output messages and pass through data
  - `delay` - Add delays to workflow execution
  - `router` - Route to different nodes based on configuration
  - `conditional` - Dynamic routing based on conditions

- **Data Nodes** (5): Data transformation and validation
  - `transform` - Transform data using JQ expressions
  - `template` - Render Go templates
  - `jsonpath` - Extract data using JSONPath
  - `validate` - Validate against JSON Schema
  - `aggregate` - Collect and combine multiple inputs

- **I/O Nodes** (3): External interactions
  - `http` - Make HTTP requests
  - `file` - File operations
  - `exec` - Execute shell commands

- **Flow Nodes** (1): Control flow
  - `parallel` - Execute multiple operations concurrently

- **Script Nodes** (1): Custom scripting
  - `lua` - Execute Lua scripts

### [Lua Scripts](lua-scripts.md)
Write custom logic in Lua with sandboxed execution:
- Access to input data
- JSON encoding/decoding
- String utilities
- Safe execution environment

### [WebAssembly Plugins](wasm-plugins.md)
Extend Pocket with plugins written in any language:
- TypeScript/JavaScript (via Javy)
- Rust
- Go
- Any language that compiles to WebAssembly

## Quick Reference

| Node Type | Category | Purpose |
|-----------|----------|---------|
| echo | Core | Output messages |
| delay | Core | Add delays |
| router | Core | Static routing |
| conditional | Core | Dynamic routing |
| transform | Data | JQ transformations |
| template | Data | Template rendering |
| jsonpath | Data | Data extraction |
| validate | Data | Schema validation |
| aggregate | Data | Collect inputs |
| http | I/O | HTTP requests |
| file | I/O | File operations |
| exec | I/O | Shell commands |
| parallel | Flow | Concurrent execution |
| lua | Script | Custom logic |

## Using Nodes

### In YAML Workflows
```yaml
nodes:
  - name: my-node
    type: echo
    config:
      message: "Hello!"
```

### In Go Code
```go
node := pocket.NewNode[string, string]("my-node",
    pocket.WithExec(func(ctx context.Context, input string) (string, error) {
        return "Hello, " + input, nil
    }),
)
```

## Next Steps

- [Explore built-in nodes](built-in/)
- [Learn Lua scripting](lua-scripts.md)
- [Create WebAssembly plugins](wasm-plugins.md)
- [View node reference](../NODE_TYPES.md)