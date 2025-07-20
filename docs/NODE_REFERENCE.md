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

Validates data against JSON Schema

**Since:** 1.0.0

#### Configuration

```json
{
  "oneOf": [
    {
      "required": [
        "schema"
      ]
    },
    {
      "required": [
        "schema_file"
      ]
    }
  ],
  "properties": {
    "fail_on_error": {
      "default": true,
      "description": "Return error on validation failure (true) or continue with validation result (false)",
      "type": "boolean"
    },
    "schema": {
      "description": "JSON Schema to validate against",
      "type": "object"
    },
    "schema_file": {
      "description": "Path to JSON Schema file (alternative to inline schema)",
      "type": "string"
    }
  },
  "type": "object"
}
```

**Properties:**

- **fail_on_error**: Return error on validation failure (true) or continue with validation result (false)
  - Type: `boolean`
  - Default: `true`
- **schema**: JSON Schema to validate against
  - Type: `object`
- **schema_file**: Path to JSON Schema file (alternative to inline schema)
  - Type: `string`

#### Output Schema

```json
{
  "properties": {
    "data": {
      "description": "The original input data"
    },
    "errors": {
      "items": {
        "properties": {
          "description": {
            "type": "string"
          },
          "field": {
            "type": "string"
          },
          "type": {
            "type": "string"
          }
        },
        "type": "object"
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

**Example 1: Validate user object**

Ensure user data matches expected schema

```yaml
type: validate
config:
  schema:
    properties:
      name:
        type: string
      email:
        type: string
        format: email
      age:
        type: integer
        minimum: 0
    required: [name email]
    type: object
```

Input:
```json
{
  "age": 30,
  "email": "alice@example.com",
  "name": "Alice"
}
```

Output:
```json
{
  "data": {
    "age": 30,
    "email": "alice@example.com",
    "name": "Alice"
  },
  "errors": [],
  "valid": true
}
```

**Example 2: Validation failure**

Handle invalid data gracefully

```yaml
type: validate
config:
  schema:
    type: object
    properties:
      score:
        type: number
        minimum: 0
        maximum: 100
    required: [score]
  fail_on_error: false
```

Input:
```json
{
  "score": 150
}
```

Output:
```json
{
  "data": {
    "score": 150
  },
  "errors": [
    {
      "description": "Must be less than or equal to 100",
      "field": "score",
      "type": "number_gte"
    }
  ],
  "valid": false
}
```

---

## Io Nodes

### exec

Executes shell commands with restrictions

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "allowed_commands": {
      "description": "List of allowed commands (whitelist)",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "args": {
      "description": "Command arguments",
      "items": {
        "type": "string"
      },
      "type": "array"
    },
    "capture_output": {
      "default": true,
      "description": "Whether to capture command output",
      "type": "boolean"
    },
    "command": {
      "description": "Command to execute",
      "type": "string"
    },
    "env": {
      "additionalProperties": {
        "type": "string"
      },
      "description": "Environment variables to set",
      "type": "object"
    },
    "timeout": {
      "default": "30s",
      "description": "Execution timeout",
      "type": "string"
    },
    "working_dir": {
      "description": "Working directory for command",
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

- **allowed_commands**: List of allowed commands (whitelist)
  - Type: `array`
- **args**: Command arguments
  - Type: `array`
- **capture_output**: Whether to capture command output
  - Type: `boolean`
  - Default: `true`
- **command** *(required)*: Command to execute
  - Type: `string`
- **env**: Environment variables to set
  - Type: `object`
- **timeout**: Execution timeout
  - Type: `string`
  - Default: `30s`
- **working_dir**: Working directory for command
  - Type: `string`

#### Output Schema

```json
{
  "properties": {
    "duration": {
      "type": "string"
    },
    "exit_code": {
      "type": "integer"
    },
    "stderr": {
      "type": "string"
    },
    "stdout": {
      "type": "string"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: List files**

List files in current directory

```yaml
type: exec
config:
  command: ls
  args:
    - -la
```

**Example 2: Run with timeout**

Execute command with timeout

```yaml
type: exec
config:
  command: sleep
  args:
    - 5
  timeout: 2s
```

**Example 3: Restricted commands**

Only allow specific commands

```yaml
type: exec
config:
  command: echo
  args:
    - Hello, World!
  allowed_commands: [echo ls cat]
```

---

### file

Reads or writes files with path restrictions

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "allow_absolute": {
      "default": false,
      "description": "Allow absolute paths outside base directory",
      "type": "boolean"
    },
    "base_dir": {
      "description": "Base directory for sandboxing (defaults to current working directory)",
      "type": "string"
    },
    "content": {
      "description": "Content to write (for write/append operations)",
      "type": "string"
    },
    "create_dirs": {
      "default": false,
      "description": "Create parent directories if they don't exist",
      "type": "boolean"
    },
    "encoding": {
      "default": "utf-8",
      "description": "File encoding",
      "type": "string"
    },
    "operation": {
      "default": "read",
      "description": "File operation to perform",
      "enum": [
        "read",
        "write",
        "append",
        "exists",
        "list"
      ],
      "type": "string"
    },
    "path": {
      "description": "File path (relative to working directory or absolute if allowed)",
      "type": "string"
    }
  },
  "required": [
    "operation",
    "path"
  ],
  "type": "object"
}
```

**Properties:**

- **allow_absolute**: Allow absolute paths outside base directory
  - Type: `boolean`
  - Default: `false`
- **base_dir**: Base directory for sandboxing (defaults to current working directory)
  - Type: `string`
- **content**: Content to write (for write/append operations)
  - Type: `string`
- **create_dirs**: Create parent directories if they don't exist
  - Type: `boolean`
  - Default: `false`
- **encoding**: File encoding
  - Type: `string`
  - Default: `utf-8`
- **operation** *(required)*: File operation to perform
  - Type: `string`
  - Default: `read`
- **path** *(required)*: File path (relative to working directory or absolute if allowed)
  - Type: `string`

#### Output Schema

```json
{
  "properties": {
    "content": {
      "description": "File content (for read operations)",
      "type": "string"
    },
    "exists": {
      "description": "Whether the file exists",
      "type": "boolean"
    },
    "files": {
      "description": "List of files (for list operation)",
      "items": {
        "properties": {
          "isDir": {
            "type": "boolean"
          },
          "modified": {
            "format": "date-time",
            "type": "string"
          },
          "name": {
            "type": "string"
          },
          "path": {
            "type": "string"
          },
          "size": {
            "type": "integer"
          }
        },
        "type": "object"
      },
      "type": "array"
    },
    "modified": {
      "description": "Last modification time",
      "format": "date-time",
      "type": "string"
    },
    "path": {
      "description": "Resolved file path",
      "type": "string"
    },
    "size": {
      "description": "File size in bytes",
      "type": "integer"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: Read file**

Read contents of a text file

```yaml
type: file
config:
  operation: read
  path: config.json
```

Output:
```json
{
  "content": "{\"version\": \"1.0.0\"}",
  "exists": true,
  "modified": "2024-01-15T10:30:00Z",
  "path": "/app/config.json",
  "size": 20
}
```

**Example 2: Write file**

Write content to a file

```yaml
type: file
config:
  operation: write
  path: output/result.txt
  content: Processing complete
  create_dirs: true
```

Output:
```json
{
  "exists": true,
  "modified": "2024-01-15T10:35:00Z",
  "path": "/app/output/result.txt",
  "size": 19
}
```

**Example 3: List directory**

List files in a directory

```yaml
type: file
config:
  operation: list
  path: data/
```

Output:
```json
{
  "exists": true,
  "files": [
    {
      "isDir": false,
      "modified": "2024-01-14T09:00:00Z",
      "name": "file1.txt",
      "path": "/app/data/file1.txt",
      "size": 1024
    }
  ],
  "path": "/app/data"
}
```

---

### http

Makes HTTP requests with retry and timeout support

**Since:** 1.0.0

#### Configuration

```json
{
  "properties": {
    "body": {
      "description": "Request body (for POST/PUT/PATCH)",
      "type": [
        "string",
        "object"
      ]
    },
    "headers": {
      "description": "HTTP headers",
      "type": "object"
    },
    "method": {
      "default": "GET",
      "enum": [
        "GET",
        "POST",
        "PUT",
        "DELETE",
        "PATCH"
      ],
      "type": "string"
    },
    "retry": {
      "properties": {
        "delay": {
          "default": "1s",
          "type": "string"
        },
        "max_attempts": {
          "default": 3,
          "type": "integer"
        }
      },
      "type": "object"
    },
    "timeout": {
      "default": "30s",
      "description": "Request timeout",
      "type": "string"
    },
    "url": {
      "description": "URL to request (supports templating)",
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

- **body**: Request body (for POST/PUT/PATCH)
- **headers**: HTTP headers
  - Type: `object`
- **method**: 
  - Type: `string`
  - Default: `GET`
- **retry**: 
  - Type: `object`
- **timeout**: Request timeout
  - Type: `string`
  - Default: `30s`
- **url** *(required)*: URL to request (supports templating)
  - Type: `string`

#### Output Schema

```json
{
  "properties": {
    "body": {
      "type": [
        "object",
        "string"
      ]
    },
    "headers": {
      "type": "object"
    },
    "status": {
      "type": "integer"
    }
  },
  "type": "object"
}
```

#### Examples

**Example 1: GET request**

```yaml
type: http
config:
  url: https://api.example.com/data
  method: GET
```

**Example 2: POST with retry**

```yaml
type: http
config:
  url: https://api.example.com/submit
  method: POST
  headers:
    Content-Type: application/json
  body:
    key: value
  retry:
    max_attempts: 5
    delay: 2s
```

---

