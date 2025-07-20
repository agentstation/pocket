# CLI and YAML Workflows (Experimental)

> **⚠️ Experimental Feature**: The Pocket CLI and YAML workflow format are experimental. APIs, schemas, and features may change in future versions without notice.

## Overview

The Pocket CLI allows you to define and execute workflows using YAML files, making it easy to create, test, and share workflows without writing Go code.

## Installation

```bash
# From source
go install github.com/agentstation/pocket/cmd/pocket@latest

# Or build locally
make build
```

## Basic Usage

```bash
# Execute a workflow
pocket run workflow.yaml

# With options
pocket run workflow.yaml --verbose --dry-run

# Check version
pocket version
```

## YAML Workflow Format

### Schema Overview

```yaml
name: string              # Required: Workflow identifier
description: string       # Optional: Human-readable description
version: string          # Optional: Version (e.g., "1.0.0")
start: string            # Required: Starting node name
metadata: map            # Optional: Arbitrary metadata
nodes: []NodeDefinition  # Required: List of nodes
connections: []Connection # Optional: Node connections
```

### Node Definition

```yaml
nodes:
  - name: string         # Required: Unique identifier
    type: string         # Required: Node type
    description: string  # Optional: Node description
    config: map         # Optional: Type-specific config
    timeout: duration   # Optional: Execution timeout
    retry:              # Optional: Retry configuration
      max_attempts: int
      delay: duration
      multiplier: float
      max_delay: duration
```

### Connection Definition

```yaml
connections:
  - from: string        # Required: Source node
    to: string          # Required: Target node
    action: string      # Optional: Route/action name
```

## Built-in Node Types

### echo
Outputs a message and passes through input data.

```yaml
type: echo
config:
  message: "Your message here"
```

### delay
Pauses execution for a specified duration.

```yaml
type: delay
config:
  duration: "2s"  # Supports: ms, s, m, h
```

### transform
Wraps input data with metadata and timestamp.

```yaml
type: transform
# No config required
```

### router
Routes to different nodes based on configuration.

```yaml
type: router
config:
  default_route: "next-node"
```

## Complete Example

```yaml
name: data-processor
description: Process data with validation and routing
version: "1.0.0"
start: validate

metadata:
  author: "Team Name"
  environment: "production"

nodes:
  - name: validate
    type: echo
    config:
      message: "Validating input..."
    timeout: "30s"
    retry:
      max_attempts: 3
      delay: "1s"
      
  - name: transform
    type: transform
    description: "Transform validated data"
    
  - name: router
    type: router
    config:
      default_route: "success"
      
  - name: success
    type: echo
    config:
      message: "✓ Processing complete"
      
  - name: error
    type: echo
    config:
      message: "✗ Processing failed"

connections:
  - from: validate
    to: transform
    
  - from: transform
    to: router
    
  - from: router
    to: success
    action: success
    
  - from: router
    to: error
    action: error
```

## CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--verbose`, `-v` | Enable detailed logging | false |
| `--dry-run` | Validate without executing | false |
| `--store-type` | Store implementation (memory, bounded) | memory |
| `--max-entries` | Max entries for bounded store | 10000 |
| `--ttl` | TTL for store entries | 0 (no expiration) |

## Validation

The CLI performs validation before execution:

1. **YAML Syntax**: File must be valid YAML
2. **Required Fields**: `name` and `start` are required
3. **Node Validation**: 
   - Unique names
   - Supported types
   - Valid configuration
4. **Connection Validation**:
   - Source and target nodes must exist
   - Actions are optional (default: "default")

## Extending the CLI

### Custom Node Types

Currently, custom node types must be added to the CLI source code. Future versions will support plugins.

```go
loader.RegisterNodeType("custom", func(def *yaml.NodeDefinition) (pocket.Node, error) {
    return pocket.NewNode[any, any](def.Name,
        pocket.WithExec(func(ctx context.Context, input any) (any, error) {
            // Custom logic here
            return input, nil
        }),
    ), nil
})
```

## Limitations

Current limitations of the experimental CLI:

1. **No Input Support**: Workflows start with nil input
2. **Limited Node Types**: Only 4 built-in types
3. **No Plugin System**: Custom nodes require recompilation
4. **Basic Output**: Results printed as YAML to stdout
5. **No State Persistence**: In-memory stores only

## Future Plans

Planned enhancements for the CLI:

- Plugin system for custom node types
- Input from files, stdin, or command line
- Output formatting options (JSON, table, etc.)
- Persistent state stores
- Interactive mode
- Workflow composition and includes
- Integration with LLM providers
- Web UI for workflow design

## Migration Path

As this is experimental, breaking changes may occur. To minimize impact:

1. **Version Lock**: Pin to specific Pocket versions
2. **Test Coverage**: Validate workflows with each upgrade
3. **Schema Version**: Use the `version` field to track schema changes
4. **Gradual Migration**: New features will be opt-in when possible

## Examples

See the [CLI examples directory](../../examples/cli/) for more workflow examples:

- `hello.yaml` - Basic workflow demonstration
- `sequential.yaml` - Sequential processing with delays
- `router.yaml` - Conditional routing

## Troubleshooting

### Common Issues

**"unknown node type"**: The node type isn't registered. Check spelling and available types.

**"start node not found"**: The `start` field references a non-existent node.

**"parse YAML" errors**: Check YAML syntax, especially indentation.

**No output**: Ensure your workflow ends with a node that produces output.

### Debug Tips

1. Use `--verbose` to see execution flow
2. Use `--dry-run` to validate without running
3. Start with simple workflows and build up
4. Check example workflows for patterns

## Feedback

As this is an experimental feature, we welcome feedback:

- Report issues on [GitHub](https://github.com/agentstation/pocket/issues)
- Share workflow examples
- Suggest new node types
- Propose schema improvements

Remember: This feature is experimental and may change significantly in future versions.