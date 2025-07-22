# Basic Workflow Examples

Simple workflows to help you get started with Pocket. Each example demonstrates a specific concept.

## Available Examples

### 1. Hello World (`hello.yaml`)
The simplest possible workflow - outputs a message.

```yaml
name: hello-world
start: greet

nodes:
  - name: greet
    type: echo
    config:
      message: "Hello from Pocket!"
```

**Run it:** `pocket run hello.yaml`

### 2. Sequential Processing (`sequential.yaml`)
Demonstrates nodes executing one after another with delays.

```yaml
name: sequential-processing
start: step1

nodes:
  - name: step1
    type: echo
    config:
      message: "Starting process..."
      
  - name: wait
    type: delay
    config:
      duration: "2s"
      
  - name: step2
    type: echo
    config:
      message: "Process complete!"

connections:
  - from: step1
    to: wait
  - from: wait
    to: step2
```

### 3. Conditional Routing (`conditional.yaml`)
Shows how to route based on data values.

```yaml
name: conditional-routing
start: check

nodes:
  - name: check
    type: conditional
    config:
      conditions:
        - if: "{{gt .score 80}}"
          then: excellent
        - if: "{{gt .score 60}}"
          then: good
      else: needs-improvement
      
  - name: excellent
    type: echo
    config:
      message: "Excellent score!"
      
  - name: good
    type: echo
    config:
      message: "Good job!"
      
  - name: needs-improvement
    type: echo
    config:
      message: "Keep practicing!"
```

### 4. Static Routing (`router.yaml`)
Demonstrates the router node for predefined paths.

```yaml
name: static-routing
start: router

nodes:
  - name: router
    type: router
    config:
      routes:
        path1: handler1
        path2: handler2
      default: default-handler
```

### 5. Template Rendering (`template-simple.yaml`)
Shows how to use Go templates to format output.

```yaml
name: template-example
start: render

nodes:
  - name: render
    type: template
    config:
      template: |
        Welcome {{.name}}!
        Your account type is: {{.type}}
        Status: {{if .active}}Active{{else}}Inactive{{end}}
```

## Running the Examples

All examples can be found in the `/examples/cli/` directory:

```bash
# Run any example
pocket run examples/cli/hello.yaml

# Run with verbose output to see execution flow
pocket run examples/cli/sequential.yaml --verbose

# Validate without running
pocket run examples/cli/conditional.yaml --dry-run
```

## Key Concepts Demonstrated

1. **Node Types**: Different built-in nodes (echo, delay, router, conditional, template)
2. **Connections**: How to connect nodes explicitly
3. **Routing**: Both static (router) and dynamic (conditional) routing
4. **Templates**: Using Go templates for dynamic content
5. **Configuration**: Various configuration options for each node type

## Next Steps

Once you're comfortable with these basics:
- Try the [Advanced Examples](../advanced/)
- Learn about [Data Processing](../advanced/#data-processing)
- Explore [External Integrations](../advanced/#external-integrations)