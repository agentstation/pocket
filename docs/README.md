# Pocket Documentation

Welcome to the comprehensive documentation for Pocket, a minimalist LLM decision graph framework for Go.

## 📚 Documentation Structure

### 🎯 Core Concepts
Understand the fundamental architecture and design principles of Pocket.

- [**Architecture Overview**](concepts/ARCHITECTURE.md) - Graph-based workflow design and core principles
- [**Prep/Exec/Post Pattern**](concepts/PREP_EXEC_POST.md) - The three-phase lifecycle pattern explained
- [**Node Interface Design**](concepts/NODE_INTERFACE.md) - How Node as interface enables composition
- [**Graph Composition**](concepts/GRAPH_COMPOSITION.md) - Building complex workflows with nested graphs

### 📖 Guides
Step-by-step guides for common tasks and features.

- [**Getting Started**](guides/GETTING_STARTED.md) - Expanded tutorial for building your first workflow
- [**Type Safety Guide**](guides/TYPE_SAFETY.md) - Leveraging Go's type system for safe workflows
- [**State Management**](guides/STATE_MANAGEMENT.md) - Working with stores and managing workflow state
- [**Error Handling**](guides/ERROR_HANDLING.md) - Building resilient workflows with fallbacks
- [**Testing Workflows**](guides/TESTING.md) - Best practices for testing Pocket applications

### 🔧 Patterns
Common patterns and recipes for building sophisticated workflows.

- [**Concurrency Patterns**](patterns/CONCURRENCY.md) - Fan-out, fan-in, pipeline, and parallel execution
- [**Agent Patterns**](patterns/AGENT_PATTERNS.md) - Building autonomous agents with think-act loops
- [**Workflow Patterns**](patterns/WORKFLOW_PATTERNS.md) - Saga pattern, compensation, and complex flows
- [**Batch Processing**](patterns/BATCH_PROCESSING.md) - Processing large datasets efficiently

### 🚀 Advanced Topics
Deep dives into advanced features and customization.

- [**Plugin Development**](PLUGIN_DEVELOPMENT.md) - Guide to creating custom nodes for Pocket
- [**Middleware System**](advanced/MIDDLEWARE.md) - Hooks, middleware, and lifecycle customization
- [**YAML Integration**](advanced/YAML_INTEGRATION.md) - Building workflows from YAML definitions
- [**Performance Optimization**](advanced/PERFORMANCE.md) - Benchmarking and optimization techniques
- [**Custom Nodes**](advanced/CUSTOM_NODES.md) - Implementing custom node types

### 🧪 Experimental Features
Features under active development - APIs may change.

- [**CLI and YAML Workflows**](advanced/CLI.md) - Define and run workflows from YAML files (experimental)
- [**Plugin System**](plugins/) - Extend Pocket with custom nodes using built-in types, Lua scripts, or future WebAssembly plugins

### 📋 Reference
Detailed reference documentation.

- [**API Reference**](reference/API.md) - Complete API documentation
- [**Node Reference**](NODE_REFERENCE.md) - Comprehensive reference for all built-in nodes
- [**Configuration Options**](reference/CONFIGURATION.md) - All available configuration options
- [**Migration Guide**](reference/MIGRATION.md) - Migrating from other frameworks or versions

### 💡 Examples
Learn by example with our comprehensive demos.

- [**Example Projects**](examples/README.md) - Overview of all example implementations

## 🎓 Learning Path

### For Beginners
1. Start with [Getting Started](guides/GETTING_STARTED.md)
2. Understand [Core Architecture](concepts/ARCHITECTURE.md)
3. Learn the [Prep/Exec/Post Pattern](concepts/PREP_EXEC_POST.md)
4. Explore [Basic Examples](examples/README.md)

### For Intermediate Users
1. Master [Type Safety](guides/TYPE_SAFETY.md)
2. Learn [State Management](guides/STATE_MANAGEMENT.md)
3. Implement [Error Handling](guides/ERROR_HANDLING.md)
4. Study [Concurrency Patterns](patterns/CONCURRENCY.md)

### For Advanced Users
1. Explore [Graph Composition](concepts/GRAPH_COMPOSITION.md)
2. Build [Agent Systems](patterns/AGENT_PATTERNS.md)
3. Implement [Custom Middleware](advanced/MIDDLEWARE.md)
4. Optimize [Performance](advanced/PERFORMANCE.md)

## 🔍 Quick Links

- [Back to Main README](../README.md)
- [View on pkg.go.dev](https://pkg.go.dev/github.com/agentstation/pocket)
- [GitHub Repository](https://github.com/agentstation/pocket)
- [Report an Issue](https://github.com/agentstation/pocket/issues)

## 📝 Contributing to Documentation

We welcome documentation improvements! If you find errors or have suggestions:

1. Open an issue describing the improvement
2. Submit a PR with your changes
3. Ensure examples are tested and working

See our [Contributing Guide](../CONTRIBUTING.md) for more details.