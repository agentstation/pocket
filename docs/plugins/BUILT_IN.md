# Built-in Node Reference

This document provides comprehensive documentation for all built-in nodes in Pocket. Each node includes configuration options, examples, and best practices.

## Table of Contents

- [Core Nodes](#core-nodes)
  - [echo](#echo)
  - [delay](#delay)
  - [router](#router)
  - [conditional](#conditional)
- [Data Nodes](#data-nodes)
  - [transform](#transform)
  - [template](#template)
  - [jsonpath](#jsonpath)
  - [validate](#validate)
  - [aggregate](#aggregate)
- [I/O Nodes](#io-nodes)
  - [http](#http)
  - [file](#file)
  - [exec](#exec)
- [Flow Nodes](#flow-nodes)
  - [parallel](#parallel)
- [Script Nodes](#script-nodes)
  - [lua](#lua)

---

## Core Nodes

### echo

Output a message and pass through input data unchanged.

**Category:** core  
**Since:** v0.1.0

#### Configuration

```yaml
type: echo
config:
  message: string  # Message to output (supports templating)
```

#### Example

```yaml
- name: log-step
  type: echo
  config:
    message: "Processing order {{.order_id}} with total: {{.total}}"
```

---

### delay

Add a delay to workflow execution.

**Category:** core  
**Since:** v0.1.0

#### Configuration

```yaml
type: delay
config:
  duration: string  # Duration (e.g., "1s", "500ms", "2m")
```

#### Example

```yaml
- name: rate-limit
  type: delay
  config:
    duration: "1s"  # Wait 1 second between API calls
```

---

### router

Static routing to different nodes based on configuration.

**Category:** core  
**Since:** v0.1.0

#### Configuration

```yaml
type: router
config:
  routes:
    route_name: target_node  # Map of route names to node targets
  default: string           # Default target if no route matches
```

#### Example

```yaml
- name: route-by-type
  type: router
  config:
    routes:
      create: handle-create
      update: handle-update
      delete: handle-delete
    default: handle-unknown
```

---

### conditional

Dynamic routing based on template expressions.

**Category:** core  
**Since:** v0.2.0

#### Configuration

```yaml
type: conditional
config:
  conditions:
    - if: string    # Go template expression
      then: string  # Target node if condition is true
  else: string      # Default target if no conditions match
```

#### Example

```yaml
- name: route-by-score
  type: conditional
  config:
    conditions:
      - if: "{{gt .score 0.9}}"
        then: high-priority
      - if: "{{gt .score 0.5}}"
        then: medium-priority
    else: low-priority
```

---

## Data Nodes

### transform

Transform data using JQ expressions.

**Category:** data  
**Since:** v0.1.0

#### Configuration

```yaml
type: transform
config:
  jq: string  # JQ expression for transformation
```

#### Example

```yaml
- name: extract-items
  type: transform
  config:
    jq: |
      .data.items | map({
        id,
        name: .title,
        price: .cost * 1.2,
        category: .tags[0]
      })
```

---

### template

Render Go templates with input data.

**Category:** data  
**Since:** v0.1.0

#### Configuration

```yaml
type: template
config:
  template: string      # Go template string
  template_file: string # Or path to template file
  output: string        # Output format: "string" (default) or "json"
```

#### Example

```yaml
- name: format-message
  type: template
  config:
    template: |
      Order Summary:
      Customer: {{.customer.name}}
      Items: {{len .items}}
      Total: ${{.total}}
      
      {{range .items}}
      - {{.name}}: ${{.price}}
      {{end}}
```

---

### jsonpath

Extract data using JSONPath expressions.

**Category:** data  
**Since:** v0.2.0

#### Configuration

```yaml
type: jsonpath
config:
  path: string          # JSONPath expression
  default: any          # Default value if path not found
  unwrap: boolean       # Unwrap single-element arrays (default: false)
```

#### Example

```yaml
- name: get-user-email
  type: jsonpath
  config:
    path: "$.users[?(@.active==true)].email"
    unwrap: true
    default: "no-email@example.com"
```

---

### validate

Validate data against JSON Schema.

**Category:** data  
**Since:** v0.2.0

#### Configuration

```yaml
type: validate
config:
  schema: object        # JSON Schema definition
  schema_file: string   # Or path to schema file
  on_error: string      # "fail" (default) or "pass"
```

#### Example

```yaml
- name: validate-order
  type: validate
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
          items:
            type: object
            required: [sku, quantity]
      required: [order_id, items]
```

---

### aggregate

Collect and combine data from multiple sources.

**Category:** data  
**Since:** v0.2.0

#### Configuration

```yaml
type: aggregate
config:
  mode: string     # "array", "object", "merge", or "concat"
  key: string      # Template for object keys (mode: object)
  timeout: string  # Max wait time (e.g., "30s")
  min_items: int   # Minimum items before proceeding
```

#### Example

```yaml
- name: collect-results
  type: aggregate
  config:
    mode: object
    key: "{{.source}}"
    timeout: "5s"
    min_items: 3
```

---

## I/O Nodes

### http

Make HTTP requests with retry and timeout support.

**Category:** io  
**Since:** v0.1.0

#### Configuration

```yaml
type: http
config:
  url: string              # URL (supports templating)
  method: string           # GET, POST, PUT, DELETE, PATCH
  headers: object          # HTTP headers
  body: any                # Request body (for POST/PUT/PATCH)
  timeout: string          # Request timeout (default: "30s")
  retry:
    max_attempts: int      # Maximum retry attempts (default: 3)
    delay: string          # Delay between retries (default: "1s")
```

#### Example

```yaml
- name: call-api
  type: http
  config:
    url: "https://api.example.com/users/{{.user_id}}"
    method: POST
    headers:
      Content-Type: application/json
      Authorization: "Bearer {{.token}}"
    body:
      action: update
      data: "{{.updates}}"
    retry:
      max_attempts: 5
      delay: "2s"
```

---

### file

Read from or write to files with path sandboxing.

**Category:** io  
**Since:** v0.2.0

#### Configuration

```yaml
type: file
config:
  path: string         # File path (supports templating)
  mode: string         # "read", "write", "append", or "list"
  content: string      # Content for write/append (supports templating)
  create_dirs: boolean # Create parent directories (default: false)
  sandbox: string      # Restrict to directory (default: current dir)
```

#### Example

```yaml
- name: save-results
  type: file
  config:
    path: "output/results-{{.timestamp}}.json"
    mode: write
    content: "{{.results | json}}"
    create_dirs: true
```

---

### exec

Execute shell commands with restrictions.

**Category:** io  
**Since:** v0.2.0

#### Configuration

```yaml
type: exec
config:
  command: string         # Command to execute
  args: array            # Command arguments
  env: object            # Environment variables
  timeout: string        # Execution timeout (default: "30s")
  allowed_commands: array # Whitelist of allowed commands
  capture_output: boolean # Capture stdout/stderr (default: true)
```

#### Example

```yaml
- name: process-file
  type: exec
  config:
    command: jq
    args:
      - ".items | length"
      - "data.json"
    timeout: "10s"
    allowed_commands: ["jq", "grep", "sed"]
```

---

## Flow Nodes

### parallel

Execute multiple tasks concurrently.

**Category:** flow  
**Since:** v0.3.0

#### Configuration

```yaml
type: parallel
config:
  tasks: array           # List of tasks to execute
  max_concurrency: int   # Max concurrent tasks (default: unlimited)
  fail_fast: boolean     # Stop on first error (default: true)
  timeout: string        # Overall timeout
```

#### Task Definition

```yaml
tasks:
  - name: string         # Task name
    node: string         # Node type to execute
    config: object       # Node configuration
    input: any           # Input data for the task
```

#### Example

```yaml
- name: fetch-all-data
  type: parallel
  config:
    max_concurrency: 3
    fail_fast: false
    timeout: "30s"
    tasks:
      - name: fetch-users
        node: http
        config:
          url: "https://api.example.com/users"
          
      - name: fetch-orders
        node: http
        config:
          url: "https://api.example.com/orders"
          
      - name: fetch-products
        node: http
        config:
          url: "https://api.example.com/products"
```

---

## Script Nodes

### lua

Execute Lua scripts for custom logic.

**Category:** script  
**Since:** v0.4.0

#### Configuration

```yaml
type: lua
config:
  script: string      # Inline Lua script
  file: string        # Or path to script file
  timeout: string     # Script timeout (default: "30s")
  sandbox: boolean    # Enable sandboxing (default: true)
```

#### Available Functions

- `json_encode(value)` - Encode value as JSON
- `json_decode(json_string)` - Decode JSON string
- `str_trim(string)` - Trim whitespace
- `str_split(string, delimiter)` - Split string
- `str_contains(string, substring)` - Check if contains
- `str_replace(string, old, new, [count])` - Replace occurrences
- `type_of(value)` - Get value type

#### Example

```yaml
- name: process-data
  type: lua
  config:
    script: |
      -- Access input data
      local items = input.items or {}
      
      -- Process items
      local processed = {}
      local total = 0
      
      for i, item in ipairs(items) do
        if item.active then
          local p = {
            id = item.id,
            name = str_trim(item.name),
            value = item.price * (1 - (item.discount or 0))
          }
          table.insert(processed, p)
          total = total + p.value
        end
      end
      
      -- Return result
      return {
        items = processed,
        total = total,
        count = #processed,
        metadata = {
          processed_at = os.time(),
          original_count = #items
        }
      }
```

---

## Best Practices

### 1. Error Handling

Always handle potential errors in your node configurations:

```yaml
# Good: Includes error handling
- name: api-call
  type: http
  config:
    url: "https://api.example.com/data"
    retry:
      max_attempts: 3
    timeout: "10s"
  successors:
    - action: success
      target: process-data
    - action: error
      target: handle-error
```

### 2. Input Validation

Validate inputs before processing:

```yaml
# Validate before processing
- name: validate-input
  type: validate
  config:
    schema:
      type: object
      required: [id, data]
      
- name: process
  type: transform
  config:
    jq: ".data | map(. + {processed: true})"
```

### 3. Use Templates Wisely

Templates are powerful but can be complex:

```yaml
# Good: Clear, simple template
config:
  message: "Processing {{.count}} items for user {{.user_id}}"

# Avoid: Complex logic in templates
config:
  message: "{{if gt .count 10}}Many{{else}}Few{{end}} items"
  # Use conditional node instead
```

### 4. Resource Management

Be mindful of timeouts and resource limits:

```yaml
# Set appropriate timeouts
- name: long-operation
  type: exec
  config:
    command: "./process.sh"
    timeout: "5m"  # Allow enough time
    
# Limit concurrency
- name: batch-process
  type: parallel
  config:
    max_concurrency: 5  # Don't overwhelm the system
```

## Node Comparison

| Feature | Built-in | Lua Script | Future: WASM |
|---------|----------|------------|--------------|
| Performance | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| Flexibility | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| Security | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| Ease of Use | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| Debugging | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |

## See Also

- [Lua Scripting Guide](LUA.md) - Deep dive into Lua scripting
- [Plugin Development](../PLUGIN_DEVELOPMENT.md) - Creating custom nodes
- [Examples](examples/) - Real-world examples