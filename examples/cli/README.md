# Pocket CLI Examples (Experimental)

> **⚠️ Experimental Feature**: The Pocket CLI and YAML workflow format are experimental features. The YAML schema and available node types may change in future versions.

This directory contains example YAML workflow definitions that can be executed using the `pocket` CLI tool.

## Running Examples

First, build the pocket CLI:
```bash
make build
```

Then run any example:
```bash
# Run hello world example
./bin/pocket run examples/cli/hello.yaml

# Run with verbose output
./bin/pocket run examples/cli/hello.yaml --verbose

# Dry run (validate without executing)
./bin/pocket run examples/cli/hello.yaml --dry-run
```

## Available Examples

### hello.yaml
A simple hello world workflow that demonstrates:
- Basic node definition
- Sequential execution
- Echo and transform node types

```bash
./bin/pocket run examples/cli/hello.yaml
```

### sequential.yaml
Demonstrates sequential processing with delays:
- Multiple steps in sequence
- Delay nodes for timing
- Transform nodes for data processing

```bash
./bin/pocket run examples/cli/sequential.yaml -v
```

### router.yaml
Shows routing capabilities:
- Router node type
- Branching paths
- Path convergence

```bash
./bin/pocket run examples/cli/router.yaml --verbose
```

## CLI Options

```bash
# Show version information
./bin/pocket version

# Run with different store configurations
./bin/pocket run workflow.yaml --store-type bounded --max-entries 1000 --ttl 5m

# Get help
./bin/pocket --help
```

## YAML Workflow Structure (Experimental)

> **Note**: This YAML format is experimental and may change. Future versions may support additional fields and node types.

### Complete Workflow Schema

```yaml
# Required: Workflow name
name: string

# Optional: Human-readable description
description: string

# Optional: Version string
version: string

# Required: Name of the starting node
start: string

# Optional: Metadata for the workflow
metadata:
  key: value

# Required: List of nodes in the workflow
nodes:
  - name: string          # Required: Unique node identifier
    type: string          # Required: Node type (echo, delay, transform, router)
    description: string   # Optional: Node description
    config:              # Optional: Node-specific configuration
      key: value
    timeout: string      # Optional: Node timeout (e.g., "30s", "5m")
    retry:               # Optional: Retry configuration
      max_attempts: int
      delay: string
      multiplier: float
      max_delay: string

# Optional: Connections between nodes
connections:
  - from: string         # Required: Source node name
    to: string           # Required: Target node name
    action: string       # Optional: Action/route name (default: "default")
```

### Minimal Example

```yaml
name: minimal-workflow
start: only-node

nodes:
  - name: only-node
    type: echo
    config:
      message: "This is the simplest possible workflow"
```

### Full Example

```yaml
name: complete-example
description: Demonstrates all YAML features
version: "2.0.0"
start: validate

metadata:
  author: "Your Name"
  tags: ["example", "demo"]

nodes:
  - name: validate
    type: echo
    description: "Validates input data"
    config:
      message: "Validating..."
    timeout: "10s"
    retry:
      max_attempts: 3
      delay: "1s"
      multiplier: 2.0
      max_delay: "10s"
    
  - name: process
    type: transform
    description: "Processes validated data"
    
  - name: route
    type: router
    config:
      default_route: "success"
    
  - name: success
    type: echo
    config:
      message: "Processing successful!"
    
  - name: failure
    type: echo
    config:
      message: "Processing failed!"

connections:
  - from: validate
    to: process
    action: default
    
  - from: process
    to: route
    action: default
    
  - from: route
    to: success
    action: success
    
  - from: route
    to: failure
    action: failure
```

### Available Node Types

1. **echo** - Outputs a message
   ```yaml
   type: echo
   config:
     message: "Your message here"
   ```

2. **delay** - Pauses execution
   ```yaml
   type: delay
   config:
     duration: "1s"  # or "500ms", "2m", etc.
   ```

3. **transform** - Transforms input data
   ```yaml
   type: transform
   ```

4. **router** - Routes to different paths
   ```yaml
   type: router
   config:
     default_route: "path-name"
   ```

## Validation Rules

The CLI validates workflows before execution:

1. **Required Fields**:
   - `name` must be non-empty
   - `start` must reference an existing node
   - Each node must have a unique `name` and `type`

2. **Node References**:
   - All connections must reference existing nodes
   - The start node must exist in the nodes list
   - Connection actions are optional (default: "default")

3. **Type Validation**:
   - Node types must be supported (echo, delay, transform, router)
   - Config values must match expected types for each node
   - Timeout and delay values must be valid durations

## Error Messages

Common errors and their meanings:

- `file not found: <path>` - The YAML file doesn't exist
- `parse YAML: <error>` - Invalid YAML syntax
- `invalid workflow: graph name is required` - Missing required field
- `invalid workflow: start node not found` - Start references non-existent node
- `create node <name>: unknown node type` - Unsupported node type

## Future Node Types (Planned)

The CLI is extensible and will support additional node types:
- `http` - Make HTTP requests
- `conditional` - Route based on conditions
- `aggregate` - Collect and combine results
- `parallel` - Run multiple operations concurrently
- `llm` - Integrate with language models
- `validator` - Validate data structure

> **Note**: As this is an experimental feature, the node type registry and YAML schema may change significantly in future versions.

## Tips

1. Use `--verbose` flag to see detailed execution logs
2. Use `--dry-run` to validate workflows before running
3. Node names must be unique within a workflow
4. All connections must reference existing nodes
5. The `start` node must exist in the nodes list