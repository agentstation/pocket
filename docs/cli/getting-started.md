# Getting Started with Pocket CLI

This guide will help you create and run your first Pocket workflow in under 5 minutes.

## Prerequisites

- Pocket CLI installed ([Installation Guide](installation.md))
- A text editor
- Basic understanding of YAML

## Your First Workflow

### Step 1: Create a Hello World Workflow

Create a file named `hello.yaml`:

```yaml
name: hello-world
description: My first Pocket workflow
version: "1.0.0"
start: greet

nodes:
  - name: greet
    type: echo
    config:
      message: "Hello from Pocket!"
```

### Step 2: Run the Workflow

```bash
pocket run hello.yaml
```

You should see:
```
Hello from Pocket!
```

Congratulations! You've run your first Pocket workflow.

## Understanding the Workflow

Let's break down what happened:

- **name**: Identifies your workflow
- **start**: The first node to execute
- **nodes**: List of operations to perform
- **type: echo**: Built-in node that outputs a message

## A More Complex Example

Let's create a workflow that processes data through multiple steps:

### Step 1: Create the Workflow

Create `process-data.yaml`:

```yaml
name: data-processor
description: Process data through multiple steps
start: fetch

nodes:
  - name: fetch
    type: echo
    config:
      message: '{"name": "pocket user", "score": 85}'
      
  - name: parse
    type: transform
    config:
      jq: ". | fromjson"
      
  - name: check-score
    type: conditional
    config:
      conditions:
        - if: "{{gt .score 90}}"
          then: excellent
        - if: "{{gt .score 70}}"
          then: good
      else: needs-improvement
      
  - name: excellent
    type: template
    config:
      template: "🌟 Excellent work, {{.name}}! Score: {{.score}}"
      
  - name: good
    type: template
    config:
      template: "👍 Good job, {{.name}}! Score: {{.score}}"
      
  - name: needs-improvement
    type: template
    config:
      template: "📚 Keep studying, {{.name}}. Score: {{.score}}"

connections:
  - from: fetch
    to: parse
  - from: parse
    to: check-score
```

### Step 2: Run It

```bash
pocket run process-data.yaml --verbose
```

This workflow:
1. Starts with JSON data
2. Parses it into a structure
3. Checks the score with conditions
4. Routes to different responses
5. Formats output with templates

## Working with Real Data

### HTTP Example

Create `fetch-weather.yaml`:

```yaml
name: weather-checker
start: fetch-weather

nodes:
  - name: fetch-weather
    type: http
    config:
      url: "https://api.open-meteo.com/v1/forecast"
      method: GET
      params:
        latitude: "40.7128"
        longitude: "-74.0060"
        current_weather: "true"
        
  - name: extract-temp
    type: jsonpath
    config:
      path: "$.current_weather.temperature"
      
  - name: format-output
    type: template
    config:
      template: "Current temperature in NYC: {{.}}°C"

connections:
  - from: fetch-weather
    to: extract-temp
  - from: extract-temp
    to: format-output
```

Run it:
```bash
pocket run fetch-weather.yaml
```

### File Processing Example

Create `process-file.yaml`:

```yaml
name: file-processor
start: read

nodes:
  - name: read
    type: file
    config:
      path: "./data.txt"
      operation: read
      
  - name: transform
    type: transform
    config:
      jq: ". | ascii_upcase"
      
  - name: write
    type: file
    config:
      path: "./output.txt"
      operation: write

connections:
  - from: read
    to: transform
  - from: transform
    to: write
```

## Using Plugins

### Lua Script Example

Create a Lua script at `~/.pocket/scripts/custom.lua`:

```lua
-- @name: custom-processor
-- @description: Custom data processing

function exec(input)
    local data = input.data or "none"
    return {
        processed = true,
        original = data,
        timestamp = os.time()
    }
end
```

Use it in a workflow:

```yaml
name: lua-example
start: process

nodes:
  - name: process
    type: custom-processor
    config:
      data: "test input"
```

## Debugging Workflows

### Verbose Mode

See detailed execution information:

```bash
pocket run workflow.yaml --verbose
```

### Dry Run

Validate without executing:

```bash
pocket run workflow.yaml --dry-run
```

### Viewing Node Information

```bash
# List all available nodes
pocket nodes list

# Get details about a specific node
pocket nodes info http
```

## Common Patterns

### Error Handling

```yaml
nodes:
  - name: risky-operation
    type: http
    config:
      url: "https://api.example.com/data"
    timeout: "5s"
    retry:
      max_attempts: 3
      delay: "1s"
```

### Parallel Processing

```yaml
nodes:
  - name: parallel-tasks
    type: parallel
    config:
      tasks:
        - name: task1
          node: http
          config:
            url: "https://api1.example.com"
        - name: task2
          node: http
          config:
            url: "https://api2.example.com"
```

## Next Steps

Now that you understand the basics:

1. [Explore all built-in nodes](../nodes/built-in/)
2. [Learn the YAML schema](yaml-schema.md)
3. [Install plugins](plugins.md)
4. [View more examples](../workflows/)
5. [Read the command reference](command-reference.md)

## Tips for Success

1. **Start Simple**: Begin with echo nodes to understand flow
2. **Use Verbose Mode**: `--verbose` helps debug issues
3. **Check Examples**: The `examples/cli/` directory has many patterns
4. **Validate First**: Use `--dry-run` before running complex workflows
5. **Modular Design**: Break complex workflows into smaller pieces