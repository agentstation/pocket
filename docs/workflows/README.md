# Pocket Workflow Examples

This directory contains example workflows organized by complexity and use case. Each example includes a complete YAML workflow file and explanation.

## Example Categories

### [Basic Workflows](basic/)
Simple workflows to get started:
- Hello World
- Sequential processing
- Conditional routing
- Error handling

### [Advanced Workflows](advanced/)
Complex patterns and techniques:
- Parallel processing
- Data aggregation
- External API integration
- State management
- Nested workflows

### [Real-World Examples](real-world/)
Production-ready workflow patterns:
- LLM agent workflows
- Data processing pipelines
- API orchestration
- File processing
- Monitoring and alerting

## Quick Start Examples

### Hello World
```yaml
name: hello-world
start: greet

nodes:
  - name: greet
    type: echo
    config:
      message: "Hello, Pocket!"
```

### Conditional Routing
```yaml
name: conditional-example
start: check-value

nodes:
  - name: check-value
    type: conditional
    config:
      conditions:
        - if: "{{gt .value 100}}"
          then: high-value
        - if: "{{gt .value 50}}"
          then: medium-value
      else: low-value
```

### API Integration
```yaml
name: api-workflow
start: fetch-data

nodes:
  - name: fetch-data
    type: http
    config:
      url: "https://api.example.com/data"
      method: GET
      
  - name: process-data
    type: transform
    config:
      jq: ".items | map({id, name, value})"

connections:
  - from: fetch-data
    to: process-data
```

## Running Examples

```bash
# Run any example
pocket run workflows/basic/hello.yaml

# Run with verbose output
pocket run workflows/advanced/parallel.yaml --verbose

# Validate without running
pocket run workflows/real-world/agent.yaml --dry-run
```

## Creating Your Own Workflows

1. Start with a basic example
2. Modify the configuration
3. Test with `--dry-run`
4. Run and iterate

## Learn More

- [YAML Schema Reference](../cli/yaml-schema.md)
- [Built-in Nodes](../NODE_TYPES.md)
- [Plugin Development](../development/plugin-development.md)