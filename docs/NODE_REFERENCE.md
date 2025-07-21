# Pocket Node Reference

This document provides a comprehensive reference for all built-in nodes in the Pocket framework.

## Table of Contents

- [Core Nodes](#core-nodes)
  - [conditional](#conditional)
  - [delay](#delay)
  - [echo](#echo)
  - [router](#router)
- [Data Nodes](#data-nodes)
  - [aggregate](#aggregate)
  - [jsonpath](#jsonpath)
  - [template](#template)
  - [transform](#transform)
  - [validate](#validate)
- [Io Nodes](#io-nodes)
  - [exec](#exec)
  - [file](#file)
  - [http](#http)
- [Flow Nodes](#flow-nodes)
  - [parallel](#parallel)
- [Script Nodes](#script-nodes)
  - [lua](#lua)

---

## Core Nodes

### conditional

Routes to different nodes based on conditions

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "conditions": {
      "items": {
        "properties": {
          "if": {
            "type": "string"
          },
          "then": {
            "type": "string"
          }
        },
        "required": [
          "if",
          "then"
        ],
        "type": "object"
      },
      "type": "array"
    },
    "else": {
      "description": "Default route if no conditions match",
      "type": "string"
    }
  },
  "required": [
    "conditions"
  ],
  "type": "object"
}
```

**Properties:**

- **conditions** *(required)*: 
  - Type: `array`
- **else**: Default route if no conditions match
  - Type: `string`

#### Examples

**Example 1: Route by score**

```yaml
type: conditional
config:
  conditions: [map[if:{{gt .score 0.8}} then:high] map[if:{{gt .score 0.5}} then:medium]]
  else: low
```

**Example 2: Route by type**

```yaml
type: conditional
config:
  conditions: [map[if:{{eq .type "error"}} then:error-handler] map[if:{{eq .type "warning"}} then:warning-handler]]
  else: success
```

---

### delay

Delays execution for a specified duration

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "duration": {
      "default": "1s",
      "description": "Duration to delay (e.g., '1s', '500ms')",
      "pattern": "^[0-9]+[a-z]+$",
      "type": "string"
    }
  },
  "type": "object"
}
```

**Properties:**

- **duration**: Duration to delay (e.g., '1s', '500ms')
  - Type: `string`
  - Default: `1s`

#### Examples

**Example 1: Simple delay**

Delay for 1 second

```yaml
type: delay
config:
  duration: 1s
```

**Example 2: Short delay**

Delay for 500 milliseconds

```yaml
type: delay
config:
  duration: 500ms
```

---

### echo

Outputs a message and passes through input

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "message": {
      "default": "Hello from echo node",
      "description": "Message to output",
      "type": "string"
    }
  },
  "type": "object"
}
```

**Properties:**

- **message**: Message to output
  - Type: `string`
  - Default: `Hello from echo node`

#### Output Schema

```json
{
  "properties": {
    "input": {
      "type": [
        "null",
        "object",
        "string",
        "number",
        "boolean",
        "array"
      ]
    },
    "message": {
      "type": "string"
    },
    "node": {
      "type": "string"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Simple echo**

Output a message

```yaml
type: echo
config:
  message: Hello, World!
```

Output:
```json
{
  "input": null,
  "message": "Hello, World!",
  "node": "echo1"
}
```

**Example 2: Echo with input**

Echo message with input passthrough

```yaml
type: echo
config:
  message: Processing complete
```

Input:
```json
{
  "data": "test"
}
```

Output:
```json
{
  "input": {
    "data": "test"
  },
  "message": "Processing complete",
  "node": "echo2"
}
```

---

### router

Routes to a specific node based on configuration

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "route": {
      "default": "default",
      "description": "The route/action to take",
      "type": "string"
    }
  },
  "type": "object"
}
```

**Properties:**

- **route**: The route/action to take
  - Type: `string`
  - Default: `default`

#### Examples

**Example 1: Simple routing**

Route to a specific action

```yaml
type: router
config:
  route: success
```

**Example 2: Default routing**

Use default route

```yaml
type: router
config:
```

---

## Data Nodes

### aggregate

Collects and combines data from multiple inputs

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "count": {
      "description": "Number of inputs to collect before continuing",
      "minimum": 1,
      "type": "integer"
    },
    "key": {
      "description": "Key to use for object mode (supports templates)",
      "type": "string"
    },
    "mode": {
      "default": "array",
      "description": "How to aggregate inputs: array (collect all), object (key-value pairs), merge (deep merge objects), concat (concatenate arrays)",
      "enum": [
        "array",
        "object",
        "merge",
        "concat"
      ],
      "type": "string"
    },
    "partial": {
      "default": false,
      "description": "Allow partial results if timeout occurs",
      "type": "boolean"
    },
    "timeout": {
      "default": "30s",
      "description": "Maximum time to wait for all inputs",
      "type": "string"
    }
  },
  "type": "object"
}
```

**Properties:**

- **count**: Number of inputs to collect before continuing
  - Type: `integer`
- **key**: Key to use for object mode (supports templates)
  - Type: `string`
- **mode**: How to aggregate inputs: array (collect all), object (key-value pairs), merge (deep merge objects), concat (concatenate arrays)
  - Type: `string`
  - Default: `array`
- **partial**: Allow partial results if timeout occurs
  - Type: `boolean`
  - Default: `false`
- **timeout**: Maximum time to wait for all inputs
  - Type: `string`
  - Default: `30s`

#### Output Schema

```json
{
  "properties": {
    "complete": {
      "description": "Whether all expected inputs were received",
      "type": "boolean"
    },
    "count": {
      "description": "Number of items collected",
      "type": "integer"
    },
    "data": {
      "description": "Aggregated data (array, object, or merged result)"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Collect array of results**

Aggregate multiple inputs into an array

```yaml
type: aggregate
config:
  mode: array
  count: 3
```

Output:
```json
{
  "complete": true,
  "count": 3,
  "data": [
    "input1",
    "input2",
    "input3"
  ]
}
```

**Example 2: Build object from inputs**

Create object with dynamic keys

```yaml
type: aggregate
config:
  mode: object
  key: {{.type}}
```

Output:
```json
{
  "complete": true,
  "count": 2,
  "data": {
    "product": {
      "name": "Widget",
      "type": "product"
    },
    "user": {
      "name": "Alice",
      "type": "user"
    }
  }
}
```

**Example 3: Merge objects deeply**

Deep merge multiple objects

```yaml
type: aggregate
config:
  mode: merge
```

Output:
```json
{
  "complete": true,
  "count": 3,
  "data": {
    "role": "admin",
    "settings": {
      "lang": "en",
      "theme": "dark"
    },
    "user": "Alice"
  }
}
```

---

### jsonpath

Extracts data from JSON using JSONPath expressions

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "default": {
      "description": "Default value if path not found"
    },
    "multiple": {
      "default": false,
      "description": "Return all matches as array (true) or first match only (false)",
      "type": "boolean"
    },
    "path": {
      "description": "JSONPath expression to extract data",
      "type": "string"
    },
    "unwrap": {
      "default": true,
      "description": "Unwrap single-element arrays",
      "type": "boolean"
    }
  },
  "required": [
    "path"
  ],
  "type": "object"
}
```

**Properties:**

- **default**: Default value if path not found
- **multiple**: Return all matches as array (true) or first match only (false)
  - Type: `boolean`
  - Default: `false`
- **path** *(required)*: JSONPath expression to extract data
  - Type: `string`
- **unwrap**: Unwrap single-element arrays
  - Type: `boolean`
  - Default: `true`

#### Output Schema

```json
{
  "description": "Extracted value(s) from the JSONPath query"
}
```

#### Examples

**Example 1: Extract user name**

Get user name from nested object

```yaml
type: jsonpath
config:
  path: $.user.name
```

Input:
```json
{
  "user": {
    "age": 30,
    "name": "Alice"
  }
}
```

Output:
```json
"Alice"
```

**Example 2: Extract all prices**

Get all prices from array of items

```yaml
type: jsonpath
config:
  path: $.items[*].price
  multiple: true
```

Input:
```json
{
  "items": [
    {
      "name": "Book",
      "price": 10.99
    },
    {
      "name": "Pen",
      "price": 2.5
    }
  ]
}
```

Output:
```json
[
  10.99,
  2.5
]
```

**Example 3: Extract with default**

Use default value when path not found

```yaml
type: jsonpath
config:
  default: Not found
  path: $.missing.field
```

Input:
```json
{
  "other": "data"
}
```

Output:
```json
"Not found"
```

---

### template

Renders Go templates with input data

**Since:** 1.0.0

#### Configuration

```json
{
  "oneOf": [
    {
      "required": [
        "template"
      ]
    },
    {
      "required": [
        "file"
      ]
    }
  ],
  "properties": {
    "file": {
      "description": "Path to template file (alternative to inline template)",
      "type": "string"
    },
    "output_format": {
      "default": "string",
      "description": "Output format for the rendered template",
      "enum": [
        "string",
        "json",
        "yaml"
      ],
      "type": "string"
    },
    "template": {
      "description": "Go template string to render",
      "type": "string"
    }
  },
  "type": "object"
}
```

**Properties:**

- **file**: Path to template file (alternative to inline template)
  - Type: `string`
- **output_format**: Output format for the rendered template
  - Type: `string`
  - Default: `string`
- **template**: Go template string to render
  - Type: `string`

#### Examples

**Example 1: Simple greeting**

Render a greeting message

```yaml
type: template
config:
  template: Hello, {{.name}}! Your score is {{.score}}.
```

Input:
```json
{
  "name": "Alice",
  "score": 95
}
```

Output:
```json
"Hello, Alice! Your score is 95."
```

**Example 2: JSON output**

Render template and output as JSON

```yaml
type: template
config:
  template: {"message": "Welcome {{.user}}", "timestamp": "{{.time}}"}
  output_format: json
```

Input:
```json
{
  "time": "2024-01-01T00:00:00Z",
  "user": "Bob"
}
```

Output:
```json
{
  "message": "Welcome Bob",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

### transform

Transforms input data

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {},
  "type": "object"
}
```

**Properties:**


#### Output Schema

```json
{
  "properties": {
    "node": {
      "type": "string"
    },
    "original": {
      "type": [
        "null",
        "object",
        "string",
        "number",
        "boolean",
        "array"
      ]
    },
    "timestamp": {
      "format": "date-time",
      "type": "string"
    },
    "transformed": {
      "type": "boolean"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Simple transform**

Wrap input with metadata

```yaml
type: transform
config:
```

Input:
```json
{
  "value": 42
}
```

Output:
```json
{
  "node": "transform1",
  "original": {
    "value": 42
  },
  "timestamp": "2024-01-01T00:00:00Z",
  "transformed": true
}
```

---

### validate

Validates data against schema

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "fail_on_error": {
      "default": true,
      "description": "Whether to fail the node on validation error",
      "type": "boolean"
    },
    "schema": {
      "description": "JSON Schema to validate against",
      "type": "object"
    }
  },
  "required": [
    "schema"
  ],
  "type": "object"
}
```

**Properties:**

- **fail_on_error**: Whether to fail the node on validation error
  - Type: `boolean`
  - Default: `true`
- **schema** *(required)*: JSON Schema to validate against
  - Type: `object`

#### Output Schema

```json
{
  "properties": {
    "data": {},
    "errors": {
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "valid": {
      "type": "boolean"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Validate user data**

```yaml
type: validate
config:
  schema:
    type: object
    properties:
      name:
        type: string
      age:
        type: integer
        minimum: 0
    required: [name, age]
```

Input:
```json
{
  "age": 30,
  "name": "Alice"
}
```

Output:
```json
{
  "data": {
    "age": 30,
    "name": "Alice"
  },
  "errors": [],
  "valid": true
}
```

**Example 2: Handle validation errors**

```yaml
type: validate
config:
  schema:
    type: object
    properties:
      email:
        type: string
        format: email
  fail_on_error: false
```

Input:
```json
{
  "email": "not-an-email"
}
```

Output:
```json
{
  "data": {
    "email": "not-an-email"
  },
  "errors": [
    "email: Does not match format 'email'"
  ],
  "valid": false
}
```

---

## Io Nodes

### exec

Executes external commands

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "args": {
      "description": "Command arguments",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "command": {
      "description": "Command to execute",
      "type": "string"
    },
    "dir": {
      "description": "Working directory",
      "type": "string"
    },
    "env": {
      "additionalProperties": {
        "type": "string"
      },
      "description": "Environment variables",
      "type": "object"
    },
    "timeout": {
      "default": "30s",
      "description": "Command timeout",
      "type": "string"
    }
  },
  "required": [
    "command"
  ],
  "type": "object"
}
```

**Properties:**

- **args**: Command arguments
  - Type: `array`
- **command** *(required)*: Command to execute
  - Type: `string`
- **dir**: Working directory
  - Type: `string`
- **env**: Environment variables
  - Type: `object`
- **timeout**: Command timeout
  - Type: `string`
  - Default: `30s`

#### Output Schema

```json
{
  "properties": {
    "code": {
      "description": "Exit code",
      "type": "integer"
    },
    "stderr": {
      "description": "Standard error output",
      "type": "string"
    },
    "stdout": {
      "description": "Standard output",
      "type": "string"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Run simple command**

```yaml
type: exec
config:
  command: echo
  args: ["Hello, World!"]
```

Output:
```json
{
  "code": 0,
  "stderr": "",
  "stdout": "Hello, World!\n"
}
```

**Example 2: Run with environment**

```yaml
type: exec
config:
  command: sh
  args: ["-c", "echo $MESSAGE"]
  env:
    MESSAGE: "Hello from env"
```

Output:
```json
{
  "code": 0,
  "stderr": "",
  "stdout": "Hello from env\n"
}
```

---

### file

File operations (read, write, append)

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "content": {
      "description": "Content to write (for write/append operations)",
      "type": "string"
    },
    "encoding": {
      "default": "utf-8",
      "description": "File encoding",
      "type": "string"
    },
    "operation": {
      "default": "read",
      "description": "Operation to perform",
      "enum": [
        "read",
        "write",
        "append",
        "delete",
        "exists"
      ],
      "type": "string"
    },
    "path": {
      "description": "File path",
      "type": "string"
    }
  },
  "required": [
    "path"
  ],
  "type": "object"
}
```

**Properties:**

- **content**: Content to write (for write/append operations)
  - Type: `string`
- **encoding**: File encoding
  - Type: `string`
  - Default: `utf-8`
- **operation**: Operation to perform
  - Type: `string`
  - Default: `read`
- **path** *(required)*: File path
  - Type: `string`

#### Output Schema

```json
{
  "oneOf": [
    {
      "description": "File content (for read)",
      "type": "string"
    },
    {
      "description": "Success status (for write/append/delete)",
      "type": "boolean"
    },
    {
      "description": "Existence status (for exists)",
      "type": "boolean"
    }
  ]
}
```

#### Examples

**Example 1: Read file**

```yaml
type: file
config:
  path: /tmp/data.txt
  operation: read
```

Output:
```json
"File contents here..."
```

**Example 2: Write file**

```yaml
type: file
config:
  path: /tmp/output.txt
  operation: write
  content: "Hello, World!"
```

Output:
```json
true
```

**Example 3: Check existence**

```yaml
type: file
config:
  path: /tmp/check.txt
  operation: exists
```

Output:
```json
false
```

---

### http

Makes HTTP requests

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "body": {
      "description": "Request body"
    },
    "headers": {
      "additionalProperties": {
        "type": "string"
      },
      "description": "Request headers",
      "type": "object"
    },
    "method": {
      "default": "GET",
      "description": "HTTP method",
      "enum": [
        "GET",
        "POST",
        "PUT",
        "DELETE",
        "PATCH",
        "HEAD",
        "OPTIONS"
      ],
      "type": "string"
    },
    "params": {
      "additionalProperties": {
        "type": "string"
      },
      "description": "Query parameters",
      "type": "object"
    },
    "timeout": {
      "default": "30s",
      "description": "Request timeout",
      "type": "string"
    },
    "url": {
      "description": "Request URL",
      "type": "string"
    }
  },
  "required": [
    "url"
  ],
  "type": "object"
}
```

**Properties:**

- **body**: Request body
- **headers**: Request headers
  - Type: `object`
- **method**: HTTP method
  - Type: `string`
  - Default: `GET`
- **params**: Query parameters
  - Type: `object`
- **timeout**: Request timeout
  - Type: `string`
  - Default: `30s`
- **url** *(required)*: Request URL
  - Type: `string`

#### Output Schema

```json
{
  "properties": {
    "body": {
      "description": "Response body"
    },
    "headers": {
      "additionalProperties": {
        "type": "string"
      },
      "description": "Response headers",
      "type": "object"
    },
    "status": {
      "description": "HTTP status code",
      "type": "integer"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Simple GET request**

```yaml
type: http
config:
  url: https://api.example.com/users
```

Output:
```json
{
  "body": [
    {
      "id": 1,
      "name": "Alice"
    }
  ],
  "headers": {
    "content-type": "application/json"
  },
  "status": 200
}
```

**Example 2: POST with data**

```yaml
type: http
config:
  url: https://api.example.com/users
  method: POST
  headers:
    Content-Type: application/json
  body:
    name: Bob
    email: bob@example.com
```

Output:
```json
{
  "body": {
    "email": "bob@example.com",
    "id": 2,
    "name": "Bob"
  },
  "headers": {
    "content-type": "application/json"
  },
  "status": 201
}
```

---

## Flow Nodes

### parallel

Execute multiple tasks concurrently

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "tasks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string"
          },
          "node": {
            "type": "string"
          },
          "config": {
            "type": "object"
          },
          "input": {}
        },
        "required": ["name", "node"]
      }
    },
    "max_concurrency": {
      "type": "integer",
      "description": "Maximum concurrent tasks"
    },
    "fail_fast": {
      "type": "boolean",
      "default": true,
      "description": "Stop on first error"
    },
    "timeout": {
      "type": "string",
      "description": "Overall timeout"
    }
  },
  "required": ["tasks"],
  "type": "object"
}
```

#### Examples

**Example 1: Fetch data in parallel**

```yaml
type: parallel
config:
  max_concurrency: 3
  tasks:
    - name: fetch-users
      node: http
      config:
        url: https://api.example.com/users
    - name: fetch-orders
      node: http
      config:
        url: https://api.example.com/orders
```

**Example 2: Process files concurrently**

```yaml
type: parallel
config:
  fail_fast: false
  tasks:
    - name: process-csv
      node: file
      config:
        path: data.csv
        operation: read
    - name: process-json
      node: file
      config:
        path: data.json
        operation: read
    - name: process-xml
      node: file
      config:
        path: data.xml
        operation: read
```

---

## Script Nodes

### lua

Execute Lua scripts for custom logic

**Since:** 1.0.0

#### Configuration

```json
{
  "oneOf": [
    {
      "required": ["script"]
    },
    {
      "required": ["file"]
    }
  ],
  "properties": {
    "file": {
      "description": "Path to Lua script file",
      "type": "string"
    },
    "sandbox": {
      "default": true,
      "description": "Enable sandboxing",
      "type": "boolean"
    },
    "script": {
      "description": "Inline Lua script",
      "type": "string"
    },
    "timeout": {
      "default": "30s",
      "description": "Script execution timeout",
      "type": "string"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Filter high scores**

```yaml
type: lua
config:
  script: |
    if input.score > 0.8 then
      return {status = "high", data = input}
    else
      return {status = "low", data = input}
    end
```

**Example 2: Transform data with utilities**

```yaml
type: lua
config:
  script: |
    local result = {
      name = str_trim(input.name),
      data = json_decode(input.json_string),
      timestamp = os.time()
    }
    return result
```

**Example 3: Complex data processing**

```yaml
type: lua
config:
  script: |
    -- Calculate statistics
    local items = input.items or {}
    local total = 0
    local count = #items
    
    for _, item in ipairs(items) do
      total = total + (item.value or 0)
    end
    
    local avg = count > 0 and (total / count) or 0
    
    -- Filter and transform
    local processed = {}
    for _, item in ipairs(items) do
      if item.value > avg then
        table.insert(processed, {
          id = item.id,
          value = item.value,
          above_average = true
        })
      end
    end
    
    return {
      total = total,
      count = count,
      average = avg,
      above_average_items = processed
    }
```

**Example 4: Use external script file**

```yaml
type: lua
config:
  file: scripts/process_order.lua
  timeout: 45s
```