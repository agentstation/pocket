# Pocket Plugin System

## Overview

The Pocket plugin system provides an extensible architecture for adding new node types to the framework. The system has been implemented in three phases:

1. **Phase 1**: Built-in nodes with metadata and CLI integration (✅ Complete)
2. **Phase 2**: Lua scripting for custom logic (✅ Complete)  
3. **Phase 3**: WebAssembly plugins for any language (✅ Complete)

## Phase 1 Status: ✅ Complete

### Implemented Features

#### Core Infrastructure
- **Node Registry**: Central registration system for all node types
- **Metadata System**: Rich metadata for documentation and validation
- **Builder Pattern**: Clean separation between node definition and implementation
- **CLI Integration**: `pocket nodes list` and `pocket nodes info <type>` commands

#### Built-in Nodes (14 types)

**Note**: These built-in nodes are native to the Pocket framework and are NOT plugins. They provide core functionality out of the box. For detailed documentation on all built-in node types, see the [Node Types Reference](NODE_TYPES.md).

**Core Nodes (4)**
- `echo` - Output messages and pass through input
- `delay` - Delay execution for specified duration
- `router` - Static routing to named actions
- `conditional` - Dynamic routing based on template expressions

**Data Nodes (5)**
- `transform` - Transform input data
- `template` - Render Go templates with input data
- `jsonpath` - Extract data using JSONPath expressions
- `validate` - Validate data against JSON Schema
- `aggregate` - Collect and combine data from multiple inputs

**I/O Nodes (3)**
- `http` - Make HTTP requests with retry and timeout support
- `file` - Read/write files with sandboxing and path restrictions
- `exec` - Execute shell commands with restrictions and timeout support

**Flow Nodes (1)**
- `parallel` - Execute multiple operations in parallel with concurrency control

**Script Nodes (1)**
- `lua` - Execute Lua scripts for custom logic

### Key Design Decisions

1. **Registry Pattern**: All nodes register through a central registry, enabling dynamic discovery and consistent management.

2. **Metadata-Driven**: Each node provides comprehensive metadata including:
   - Type and category classification
   - Configuration schema (JSON Schema)
   - Input/output schemas
   - Usage examples
   - Version information

3. **Builder Interface**: Clean separation between node configuration and runtime behavior:
   ```go
   type NodeBuilder interface {
       Metadata() NodeMetadata
       Build(def *yaml.NodeDefinition) (pocket.Node, error)
   }
   ```

4. **Security First**: File operations include sandboxing by default, HTTP nodes support timeouts and retries.

## Usage Examples

### List Available Nodes
```bash
$ pocket nodes list

Core:
-----
  conditional          Routes to different nodes based on conditions
  delay                Delays execution for a specified duration
  echo                 Outputs a message and passes through input
  router               Routes to a specific node based on configuration

Data:
-----
  aggregate            Collects and combines data from multiple inputs
  jsonpath             Extracts data from JSON using JSONPath expressions
  template             Renders Go templates with input data
  transform            Transforms input data
  validate             Validates data against JSON Schema

Io:
---
  exec                 Executes shell commands with restrictions
  file                 Reads or writes files with path restrictions
  http                 Makes HTTP requests with retry and timeout support

Flow:
-----
  parallel             Executes multiple operations in parallel

Script:
-------
  lua                  Execute Lua scripts for custom logic

Total: 14 node types
```

### Get Node Details
```bash
$ pocket nodes info template

Node Type: template
Category: data
Description: Renders Go templates with input data

Configuration:
  {
    "properties": {
      "file": {
        "description": "Path to template file (alternative to inline template)",
        "type": "string"
      },
      "output_format": {
        "default": "string",
        "description": "Output format for the rendered template",
        "enum": ["string", "json", "yaml"],
        "type": "string"
      },
      "template": {
        "description": "Go template string to render",
        "type": "string"
      }
    },
    "type": "object"
  }

Examples:
  1. Simple greeting
     Render a greeting message
     Config:
       template: Hello, {{.name}}! Your score is {{.score}}.
```

### Example Workflows

See the `examples/cli/` directory for complete workflow examples:
- `conditional-routing.yaml` - Dynamic routing based on data
- `template-simple.yaml` - Template rendering
- `http-api.yaml` - HTTP API integration
- `jsonpath-extract.yaml` - JSON data extraction
- `validate-api-response.yaml` - Data validation
- `aggregate-data.yaml` - Data aggregation
- `file-operations.yaml` - File I/O with sandboxing
- `exec-commands.yaml` - Command execution with restrictions
- `parallel-tasks.yaml` - Parallel task execution

## Architecture Benefits

1. **Extensibility**: New node types can be added without modifying core framework
2. **Discoverability**: CLI commands provide easy access to available nodes
3. **Type Safety**: Optional validation through schema definitions
4. **Documentation**: Self-documenting through metadata
5. **Testing**: Comprehensive test coverage for all nodes

## Phase 2 Status: ✅ Complete

### Lua Scripting Integration
- **Embedded Lua interpreter** via Shopify/go-lua
- **Sandboxed execution** with restricted functions
- **JSON support** with encode/decode functions
- **String utilities** for common operations
- **Script timeout** support for safety
- **File-based scripts** in addition to inline
- **Lua node type** available as a built-in node

Lua scripts can be used through the `lua` node type in workflows. For detailed documentation on Lua scripting, see the [Node Types Reference](NODE_TYPES.md#lua).

## Phase 3 Status: ✅ Complete

### WebAssembly Plugin Support
- **WASM Runtime**: Integrated wazero (pure Go, no CGO)
- **Plugin Architecture**: Complete lifecycle support (prep/exec/post)
- **TypeScript SDK**: Full SDK with Javy integration
- **Example Plugins**: TypeScript, Rust, and Go examples
- **Security**: Sandboxing with memory limits and permissions
- **CLI Integration**: Unified `pocket plugins` commands
- **Documentation**: Comprehensive guides and API reference

### Plugin CLI Commands
- `pocket plugins list` - List installed plugins
- `pocket plugins install <path>` - Install plugin from directory
- `pocket plugins remove <name>` - Remove installed plugin
- `pocket plugins info <name>` - Show plugin details
- `pocket plugins validate <path>` - Validate plugin manifest

For detailed plugin documentation, see:
- [WebAssembly Plugin Guide](PLUGINS.md) - Creating WASM plugins
- [Plugin Development Guide](plugins/DEVELOPMENT.md) - General plugin development
- [Plugin SDK API Reference](plugins/SDK_API.md) - TypeScript SDK reference

For built-in node documentation, see:
- [Node Types Reference](NODE_TYPES.md) - All 14 built-in node types

## Development Guide

### Adding a New Built-in Node

1. Implement the NodeBuilder interface in `nodes/builders.go`:
```go
type MyNodeBuilder struct {
    Verbose bool
}

func (b *MyNodeBuilder) Metadata() NodeMetadata {
    return NodeMetadata{
        Type:        "mynode",
        Category:    "data",
        Description: "My custom node",
        // ... schemas and examples
    }
}

func (b *MyNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
    // Create and return the node
}
```

2. Register in `nodes/registry.go`:
```go
registry.Register(&MyNodeBuilder{Verbose: verbose})
```

3. The node will be automatically discovered through the registry.

4. Write tests in `nodes/builders_test.go`

5. Create an example workflow in `examples/cli/`

## Conclusion

The Pocket plugin system provides a comprehensive solution for extending workflow capabilities:

- **Built-in nodes** (14 types) cover common use cases with full system access
- **Lua scripting** enables custom logic in a sandboxed environment
- **WebAssembly plugins** allow development in TypeScript, Rust, or Go with security boundaries

The architecture balances simplicity with power, enabling both simple configurations and complex integrations while maintaining security and performance.