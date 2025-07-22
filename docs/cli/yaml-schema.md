# YAML Workflow Schema Reference

This document provides a complete reference for the Pocket workflow YAML schema.

## Schema Overview

```yaml
# Required fields
name: string          # Workflow identifier
start: string         # Starting node name

# Optional fields
description: string   # Human-readable description
version: string       # Semantic version (e.g., "1.0.0")
metadata: map         # Arbitrary metadata

# Required: Define workflow nodes
nodes: []Node         # List of node definitions

# Optional: Define connections
connections: []Connection  # Explicit node connections
```

## Complete Schema Definition

### Root Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique workflow identifier |
| `description` | string | No | Human-readable description |
| `version` | string | No | Semantic version (e.g., "1.0.0") |
| `start` | string | Yes | Name of the starting node |
| `metadata` | object | No | Arbitrary key-value metadata |
| `nodes` | array[Node] | Yes | List of node definitions |
| `connections` | array[Connection] | No | Explicit node connections |

### Node Schema

```yaml
nodes:
  - name: string         # Required: Unique node identifier
    type: string         # Required: Node type (echo, http, etc.)
    description: string  # Optional: Node description
    config: object      # Optional: Type-specific configuration
    timeout: duration   # Optional: Execution timeout
    retry: RetryConfig  # Optional: Retry configuration
    fallback: object    # Optional: Fallback configuration
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique node identifier within workflow |
| `type` | string | Yes | Node type (see [Built-in Nodes](#built-in-node-types)) |
| `description` | string | No | Human-readable description |
| `config` | object | No | Type-specific configuration |
| `timeout` | duration | No | Execution timeout (e.g., "30s", "5m") |
| `retry` | RetryConfig | No | Retry configuration |
| `fallback` | object | No | Fallback behavior on error |

### Connection Schema

```yaml
connections:
  - from: string    # Required: Source node name
    to: string      # Required: Target node name
    action: string  # Optional: Action/route name (default: "default")
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `from` | string | Yes | Source node name |
| `to` | string | Yes | Target node name |
| `action` | string | No | Action/route identifier (default: "default") |

### RetryConfig Schema

```yaml
retry:
  max_attempts: integer  # Maximum retry attempts
  delay: duration       # Initial delay between retries
  multiplier: float     # Backoff multiplier
  max_delay: duration   # Maximum delay between retries
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_attempts` | integer | 3 | Maximum number of retry attempts |
| `delay` | duration | "1s" | Initial delay between retries |
| `multiplier` | float | 2.0 | Backoff multiplier for exponential backoff |
| `max_delay` | duration | "1m" | Maximum delay between retries |

## Built-in Node Types

### Core Nodes

#### echo
Output a message and pass through input.

```yaml
type: echo
config:
  message: string  # Message to output (supports templating)
```

#### delay
Pause execution for a specified duration.

```yaml
type: delay
config:
  duration: duration  # Duration to wait (e.g., "2s", "500ms")
```

#### router
Route to different nodes based on static configuration.

```yaml
type: router
config:
  routes:           # Map of route names to target nodes
    route1: node1
    route2: node2
  default: string   # Default target if no route matches
```

#### conditional
Dynamic routing based on conditions.

```yaml
type: conditional
config:
  conditions:       # Array of condition rules
    - if: string    # Go template expression
      then: string  # Target node if true
  else: string      # Default target if no conditions match
```

### Data Nodes

#### transform
Transform data using JQ expressions.

```yaml
type: transform
config:
  jq: string  # JQ expression for transformation
```

#### template
Render Go templates with input data.

```yaml
type: template
config:
  template: string      # Go template string (or)
  template_file: string # Path to template file
  output_format: string # "string" (default), "json", or "yaml"
```

#### jsonpath
Extract data using JSONPath expressions.

```yaml
type: jsonpath
config:
  path: string          # JSONPath expression
  default: any          # Default if path not found
  multiple: boolean     # Return all matches (default: false)
  unwrap: boolean       # Unwrap single-element arrays (default: true)
```

#### validate
Validate data against JSON Schema.

```yaml
type: validate
config:
  schema: object        # JSON Schema definition (or)
  schema_file: string   # Path to schema file
  fail_on_error: boolean # Fail node on validation error (default: true)
```

#### aggregate
Collect and combine data from multiple inputs.

```yaml
type: aggregate
config:
  mode: string          # "array", "object", "merge", "concat"
  key: string           # Template for object keys (mode: object)
  count: integer        # Number of inputs to collect
  timeout: duration     # Max wait time
  partial: boolean      # Allow partial results (default: false)
```

### I/O Nodes

#### http
Make HTTP requests.

```yaml
type: http
config:
  url: string           # Request URL (supports templating)
  method: string        # HTTP method (default: "GET")
  headers: object       # Request headers
  params: object        # Query parameters
  body: any            # Request body
  timeout: duration    # Request timeout (default: "30s")
  retry:               # HTTP-specific retry config
    max_attempts: integer
    delay: duration
```

#### file
File operations (read, write, append, delete, exists).

```yaml
type: file
config:
  path: string          # File path (supports templating)
  operation: string     # "read", "write", "append", "delete", "exists"
  content: string       # Content for write/append
  encoding: string      # File encoding (default: "utf-8")
  create_dirs: boolean  # Create parent directories (default: false)
```

#### exec
Execute shell commands.

```yaml
type: exec
config:
  command: string       # Command to execute
  args: []string        # Command arguments
  dir: string          # Working directory
  env: object          # Environment variables
  timeout: duration    # Execution timeout (default: "30s")
  capture_output: boolean # Capture stdout/stderr (default: true)
```

### Flow Nodes

#### parallel
Execute multiple tasks concurrently.

```yaml
type: parallel
config:
  tasks:                # Array of task definitions
    - name: string      # Task name
      node: string      # Node type to execute
      config: object    # Node configuration
      input: any        # Input for the task
  max_concurrency: integer # Max concurrent tasks
  fail_fast: boolean    # Stop on first error (default: true)
  timeout: duration     # Overall timeout
```

### Script Nodes

#### lua
Execute Lua scripts.

```yaml
type: lua
config:
  script: string        # Inline Lua script (or)
  file: string         # Path to script file
  timeout: duration    # Script timeout (default: "30s")
  sandbox: boolean     # Enable sandboxing (default: true)
```

## Data Types

### Duration Format

Duration strings follow Go's duration format:
- `"300ms"` - 300 milliseconds
- `"1.5s"` - 1.5 seconds
- `"2m"` - 2 minutes
- `"1h30m"` - 1 hour 30 minutes

### Template Syntax

Go template expressions in strings:
- `"Hello, {{.name}}"` - Variable substitution
- `"{{.price | printf \"%.2f\"}}"` - Formatting
- `"{{if .premium}}Premium{{else}}Standard{{end}}"` - Conditionals

### JSONPath Syntax

Standard JSONPath expressions:
- `"$.user.name"` - Object navigation
- `"$.items[0].price"` - Array access
- `"$.items[*].name"` - All items
- `"$.items[?(@.price > 100)]"` - Filtering

## Connection Patterns

### Implicit Connections

If no explicit connections are defined, nodes connect sequentially:

```yaml
nodes:
  - name: first
    type: echo
  - name: second  # Implicitly connected from first
    type: echo
  - name: third   # Implicitly connected from second
    type: echo
```

### Explicit Connections

Define specific routing:

```yaml
nodes:
  - name: start
    type: conditional
  - name: path-a
    type: echo
  - name: path-b
    type: echo
  - name: finish
    type: echo

connections:
  - from: start
    to: path-a
    action: "true"
  - from: start
    to: path-b
    action: "false"
  - from: path-a
    to: finish
  - from: path-b
    to: finish
```

### Action-Based Routing

Nodes can route based on actions:

```yaml
connections:
  - from: validator
    to: processor
    action: valid
  - from: validator
    to: error-handler
    action: invalid
  - from: processor
    to: success
    action: success
  - from: processor
    to: retry
    action: failure
```

## Complete Example

```yaml
name: order-processing-pipeline
description: Complete order processing workflow with error handling
version: "1.0.0"
start: receive-order

metadata:
  author: "Platform Team"
  category: "e-commerce"
  tags: ["orders", "payments", "critical"]

nodes:
  - name: receive-order
    type: echo
    description: "Log order receipt"
    config:
      message: "Order received: {{.order_id}}"
      
  - name: validate-order
    type: validate
    description: "Validate order structure"
    config:
      schema:
        type: object
        properties:
          order_id:
            type: string
            pattern: "^ORD-[0-9]+$"
          items:
            type: array
            minItems: 1
          customer:
            type: object
            required: ["id", "email"]
        required: ["order_id", "items", "customer"]
    timeout: "5s"
    
  - name: check-inventory
    type: http
    description: "Check inventory availability"
    config:
      url: "https://api.inventory.internal/check"
      method: POST
      headers:
        Content-Type: "application/json"
        X-API-Key: "${INVENTORY_API_KEY}"
      body: "{{.items}}"
    timeout: "10s"
    retry:
      max_attempts: 3
      delay: "2s"
      
  - name: process-payment
    type: parallel
    description: "Process payment and fraud check in parallel"
    config:
      tasks:
        - name: charge-payment
          node: http
          config:
            url: "https://api.payments.internal/charge"
            method: POST
        - name: fraud-check
          node: http
          config:
            url: "https://api.fraud.internal/check"
            method: POST
      fail_fast: true
      timeout: "30s"
      
  - name: fulfill-order
    type: http
    description: "Send to fulfillment"
    config:
      url: "https://api.fulfillment.internal/orders"
      method: POST
      
  - name: send-confirmation
    type: template
    description: "Generate confirmation"
    config:
      template: |
        Order {{.order_id}} confirmed!
        Items: {{len .items}}
        Total: ${{.total}}
        
        Thank you for your order!
        
  - name: handle-error
    type: template
    config:
      template: "Error processing order {{.order_id}}: {{.error}}"

connections:
  - from: receive-order
    to: validate-order
    
  - from: validate-order
    to: check-inventory
    action: valid
    
  - from: validate-order
    to: handle-error
    action: invalid
    
  - from: check-inventory
    to: process-payment
    action: available
    
  - from: check-inventory
    to: handle-error
    action: unavailable
    
  - from: process-payment
    to: fulfill-order
    action: success
    
  - from: process-payment
    to: handle-error
    action: failure
    
  - from: fulfill-order
    to: send-confirmation
```

## Validation Rules

1. **Unique Node Names**: All node names must be unique within a workflow
2. **Valid Start Node**: The `start` field must reference an existing node
3. **Valid Connections**: Both `from` and `to` must reference existing nodes
4. **Required Fields**: `name` and `start` at root level, `name` and `type` for nodes
5. **Type Existence**: Node types must be registered (built-in or plugin)
6. **Config Validation**: Node configurations must match their schema

## Best Practices

1. **Use Descriptive Names**: Node names should describe their purpose
2. **Add Descriptions**: Use the description field for documentation
3. **Set Timeouts**: Always set appropriate timeouts for I/O operations
4. **Handle Errors**: Define error paths and fallback nodes
5. **Version Workflows**: Use semantic versioning for workflow versions
6. **Metadata**: Include author, purpose, and tags in metadata
7. **Modular Design**: Break complex workflows into smaller, reusable parts
8. **Test Incrementally**: Use `--dry-run` to validate before running