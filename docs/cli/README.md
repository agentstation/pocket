# Pocket CLI Documentation

Welcome to the Pocket CLI documentation. Pocket is a graph execution engine that allows you to run workflow graphs defined in YAML or JSON files, making it language-agnostic and perfect for LLM-directed workflows.

## What is the Pocket CLI?

The Pocket CLI is a command-line tool that:
- Executes workflow graphs from YAML/JSON files
- Manages plugins written in any language (via WebAssembly)
- Provides built-in nodes for common operations
- Supports Lua scripting for custom logic

## Quick Start

```bash
# Install Pocket CLI
go install github.com/agentstation/pocket/cmd/pocket@latest

# Run your first workflow
pocket run hello.yaml

# List available nodes
pocket nodes list

# Install a plugin
pocket plugins install ./my-plugin

# Run with verbose output
pocket run workflow.yaml --verbose
```

## Documentation

- [**Installation Guide**](installation.md) - Detailed installation instructions
- [**Getting Started**](getting-started.md) - Create and run your first workflow
- [**Command Reference**](command-reference.md) - Complete CLI command documentation
- [**YAML Schema**](yaml-schema.md) - Workflow file format reference
- [**Configuration**](configuration.md) - Config files and environment variables
- [**Plugin Management**](plugins.md) - Installing and using plugins
- [**Troubleshooting**](troubleshooting.md) - Common issues and solutions

## Core Concepts

### Graph Execution Engine

Pocket is a graph execution engine, which means:
- Workflows are directed graphs of nodes
- Each node performs a specific operation
- Nodes are connected by edges that route data
- Execution flows through the graph based on node outputs

### Workflow Files

Workflows are defined in YAML files:

```yaml
name: my-workflow
start: first-node

nodes:
  - name: first-node
    type: echo
    config:
      message: "Hello, Pocket!"
    
  - name: second-node
    type: transform
    config:
      jq: ".message | ascii_upcase"

connections:
  - from: first-node
    to: second-node
```

### Node Types

Pocket provides 14 built-in node types:
- **Core**: echo, delay, router, conditional
- **Data**: transform, template, jsonpath, validate, aggregate
- **I/O**: http, file, exec
- **Flow**: parallel
- **Script**: lua

Plus support for:
- Custom Lua scripts
- WebAssembly plugins in any language

## Next Steps

1. [Install Pocket CLI](installation.md)
2. [Create your first workflow](getting-started.md)
3. [Explore built-in nodes](../nodes/README.md)
4. [Learn about plugins](plugins.md)