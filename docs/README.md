# Pocket Documentation Hub

Welcome to the comprehensive documentation for Pocket, a graph execution engine for building LLM workflows.

## 🚀 Quick Start by Use Case

### Use Case 1: Run Workflows with CLI (No Coding Required)

Perfect for most users who want to run workflows defined in YAML:

1. **[Install Pocket CLI](cli/installation.md)** - Get up and running in minutes
2. **[Getting Started Tutorial](cli/getting-started.md)** - Create your first workflow
3. **[YAML Schema Reference](cli/yaml-schema.md)** - Define workflows in YAML
4. **[Command Reference](cli/command-reference.md)** - All CLI commands

**Example:**
```bash
# Install via Homebrew
brew install agentstation/tap/pocket

# Run a workflow
pocket run workflow.yaml
```

### Use Case 2: Embed in Go Applications

For developers building Go applications:

1. **[Library Getting Started](library/getting-started.md)** - Build workflows in Go
2. **[Embedding Guide](library/embedding.md)** - Integrate into your application
3. **[API Reference](library/api-reference.md)** - Complete API documentation

**Example:**
```go
import "github.com/agentstation/pocket"

node := pocket.NewNode[string, string]("process",
    pocket.WithExec(processFunc),
)
```

## 📚 Documentation Structure

### 🖥️ CLI Documentation
Everything you need to use Pocket as a command-line tool.

- **[CLI Overview](cli/)** - Introduction to the CLI
- **[Installation](cli/installation.md)** - Installation methods
- **[Getting Started](cli/getting-started.md)** - First workflow tutorial
- **[Command Reference](cli/command-reference.md)** - All commands and options
- **[YAML Schema](cli/yaml-schema.md)** - Complete YAML reference
- **[Configuration](cli/configuration.md)** - Config files and environment
- **[Plugin Management](cli/plugins.md)** - Installing and using plugins
- **[Troubleshooting](cli/troubleshooting.md)** - Common issues and solutions

### 📦 Library Documentation
For embedding Pocket in your Go applications.

- **[Library Overview](library/)** - Introduction to the Go library
- **[Getting Started](library/getting-started.md)** - Build your first workflow
- **[API Reference](library/api-reference.md)** - Complete API documentation
- **[Embedding Guide](library/embedding.md)** - Integration patterns

### 🎯 Core Concepts
Understand the fundamental architecture of Pocket's graph execution engine.

- **[Architecture Overview](concepts/ARCHITECTURE.md)** - Graph-based workflow design
- **[Prep/Exec/Post Pattern](concepts/PREP_EXEC_POST.md)** - The three-phase lifecycle
- **[Node Interface Design](concepts/NODE_INTERFACE.md)** - How nodes work
- **[Graph Composition](concepts/GRAPH_COMPOSITION.md)** - Building complex workflows

### 🔌 Node Types
Documentation for all available node types.

- **[Node Types Overview](nodes/)** - All node categories
- **[Built-in Nodes](NODE_TYPES.md)** - 14 built-in node types
- **[Lua Scripting](nodes/lua-scripts.md)** - Custom logic with Lua
- **[WebAssembly Plugins](nodes/wasm-plugins.md)** - Plugins in any language

### 🔄 Workflow Examples
Real-world workflow examples organized by complexity.

- **[Example Workflows](workflows/)** - Catalog of examples
- **[Basic Workflows](workflows/basic/)** - Simple starting points
- **[Advanced Workflows](workflows/advanced/)** - Complex patterns
- **[Real-World Examples](workflows/real-world/)** - Production patterns

### 📖 Guides
Step-by-step guides for specific topics.

- **[Type Safety Guide](guides/TYPE_SAFETY.md)** - Leveraging Go's type system
- **[State Management](guides/STATE_MANAGEMENT.md)** - Working with stores
- **[Error Handling](guides/ERROR_HANDLING.md)** - Building resilient workflows
- **[Testing Workflows](guides/TESTING.md)** - Testing best practices

### 🔧 Patterns
Common patterns for building sophisticated workflows.

- **[Concurrency Patterns](patterns/CONCURRENCY.md)** - Parallel execution
- **[Agent Patterns](patterns/AGENT_PATTERNS.md)** - Building LLM agents
- **[Workflow Patterns](patterns/WORKFLOW_PATTERNS.md)** - Complex flows
- **[Batch Processing](patterns/BATCH_PROCESSING.md)** - Processing large datasets

### 🚀 Advanced Topics
Deep dives into advanced features.

- **[Middleware System](advanced/MIDDLEWARE.md)** - Hooks and lifecycle
- **[YAML Integration](advanced/YAML_INTEGRATION.md)** - Declarative workflows
- **[Performance Optimization](advanced/PERFORMANCE.md)** - Optimization guide
- **[Custom Nodes](advanced/CUSTOM_NODES.md)** - Creating custom nodes

### 🛠️ Development
For contributors and plugin developers.

- **[Plugin Development](development/plugin-development.md)** - Create plugins
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute
- **[Architecture](development/architecture.md)** - Internal design
- **[Go Style Guide](GO_STYLE_GUIDE.md)** - Code style guidelines

### 📋 Reference
Detailed reference documentation.

- **[API Reference](reference/API.md)** - Complete API documentation
- **[Node Reference](NODE_REFERENCE.md)** - Auto-generated node reference
- **[Configuration Options](reference/CONFIGURATION.md)** - All config options
- **[Plugin System](PLUGIN_SYSTEM.md)** - Plugin architecture

## 🎓 Learning Paths

### For CLI Users

1. Start with [CLI Installation](cli/installation.md)
2. Follow the [Getting Started Tutorial](cli/getting-started.md)
3. Learn the [YAML Schema](cli/yaml-schema.md)
4. Explore [Workflow Examples](workflows/)
5. Install [Plugins](cli/plugins.md) for extended functionality

### For Go Developers

1. Begin with [Library Getting Started](library/getting-started.md)
2. Understand [Core Architecture](concepts/ARCHITECTURE.md)
3. Learn the [Prep/Exec/Post Pattern](concepts/PREP_EXEC_POST.md)
4. Master [Type Safety](guides/TYPE_SAFETY.md)
5. Study [Embedding Patterns](library/embedding.md)

### For Plugin Developers

1. Read [Plugin System Overview](PLUGIN_SYSTEM.md)
2. Choose your approach:
   - [Lua Scripts](nodes/lua-scripts.md) for simple logic
   - [WebAssembly](nodes/wasm-plugins.md) for complex plugins
3. Follow [Plugin Development Guide](development/plugin-development.md)
4. See [Plugin Examples](../plugins/examples/)

## 🔍 Quick Links

- **[Main README](../README.md)** - Project overview
- **[GitHub Repository](https://github.com/agentstation/pocket)** - Source code
- **[Issues](https://github.com/agentstation/pocket/issues)** - Report bugs
- **[Examples](../examples/)** - Code examples
- **[pkg.go.dev](https://pkg.go.dev/github.com/agentstation/pocket)** - Go package docs

## 📝 Contributing to Documentation

We welcome documentation improvements! If you find errors or have suggestions:

1. Open an issue describing the improvement
2. Submit a PR with your changes
3. Ensure examples are tested and working

See our [Contributing Guide](CONTRIBUTING.md) for more details.