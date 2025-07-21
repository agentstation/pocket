# Pocket Plugin System

The Pocket plugin system allows extending the workflow engine with custom nodes written in any language that can compile to WebAssembly.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Getting Started](#getting-started)
4. [Creating Plugins](#creating-plugins)
5. [Plugin SDK](#plugin-sdk)
6. [Security](#security)
7. [CLI Reference](#cli-reference)
8. [Examples](#examples)

## Overview

Pocket plugins are WebAssembly modules that extend the workflow engine with custom functionality. Plugins can:

- Add new node types for data processing
- Integrate with external services
- Implement custom business logic
- Transform and validate data

### Key Features

- **Language Agnostic**: Write plugins in TypeScript, Rust, Go, or any language that compiles to WASM
- **Sandboxed Execution**: Plugins run in isolated environments with controlled permissions
- **Type Safe**: Define schemas for inputs, outputs, and configuration
- **Lifecycle Integration**: Follows Pocket's Prep/Exec/Post pattern

## Architecture

### Plugin Structure

```
my-plugin/
├── manifest.yaml       # Plugin metadata and configuration
├── plugin.wasm        # Compiled WebAssembly binary
├── src/               # Source code (language-specific)
└── README.md          # Plugin documentation
```

### Manifest File

The manifest describes the plugin and its capabilities:

```yaml
name: my-plugin
version: 1.0.0
description: My custom plugin for Pocket
author: Your Name
license: MIT
runtime: wasm
binary: plugin.wasm

nodes:
  - type: process-data
    category: transform
    description: Process and transform data
    configSchema:
      type: object
      properties:
        mode:
          type: string
          enum: ["fast", "accurate"]
          default: "fast"
    inputSchema:
      type: object
      properties:
        data:
          type: array
      required: ["data"]
    outputSchema:
      type: object
      properties:
        result:
          type: array
        stats:
          type: object

permissions:
  memory: 10MB
  timeout: 5s

requirements:
  pocket: ">=1.0.0"
```

### Plugin Lifecycle

Plugins follow Pocket's three-phase lifecycle:

1. **Prep Phase**: Validate inputs, load configuration, prepare state
2. **Exec Phase**: Execute core business logic (pure function)
3. **Post Phase**: Process results, update state, determine routing

```typescript
class MyNode extends PluginNode {
  async prep(input: Input, config: Config, store: Store) {
    // Validate and prepare data
    return preparedData;
  }

  async exec(prepData: any, config: Config) {
    // Core logic - no side effects
    return result;
  }

  async post(input: Input, prepData: any, result: any, config: Config, store: Store) {
    // Post-processing and routing
    return { output: result, next: "success" };
  }
}
```

## Getting Started

### Prerequisites

1. Install the Pocket plugin CLI:
   ```bash
   go install github.com/agentstation/pocket/cmd/pocket-plugins@latest
   ```

2. Choose your development language and install its toolchain:
   - **TypeScript**: Node.js and Javy
   - **Rust**: Rust toolchain with wasm32-wasi target
   - **Go**: Go 1.21+ or TinyGo

### Quick Start

1. **Clone an example plugin**:
   ```bash
   cp -r $POCKET_ROOT/plugin/examples/typescript/sentiment-analyzer my-plugin
   cd my-plugin
   ```

2. **Modify the plugin**:
   - Edit `src/index.ts` (or equivalent)
   - Update `manifest.yaml`
   - Implement your logic

3. **Build the plugin**:
   ```bash
   make build
   ```

4. **Install locally**:
   ```bash
   pocket-plugins install .
   ```

5. **Use in a workflow**:
   ```yaml
   nodes:
     - name: my-processor
       type: my-node-type
       config:
         option: value
   ```

## Creating Plugins

### TypeScript/JavaScript

1. **Set up project**:
   ```bash
   npm init -y
   npm install @pocket/plugin-sdk
   npm install -D typescript @shopify/javy
   ```

2. **Create plugin**:
   ```typescript
   import { Plugin, PluginNode, initializePlugin } from '@pocket/plugin-sdk';

   class MyNode extends PluginNode<Input, Output, Config> {
     readonly type = 'my-node';
     readonly category = 'custom';
     readonly description = 'My custom node';

     async prep(input: Input, config: Config, store: Store) {
       // Preparation logic
       return { processedInput: input };
     }

     async exec(prepData: any, config: Config) {
       // Core processing
       return { result: process(prepData) };
     }

     async post(input: Input, prepData: any, result: Output, config: Config, store: Store) {
       // Post-processing
       return { output: result, next: 'done' };
     }
   }

   const plugin = new Plugin({
     name: 'my-plugin',
     version: '1.0.0',
     nodes: []
   });

   plugin.register(new MyNode());
   initializePlugin(plugin);
   ```

3. **Build to WASM**:
   ```bash
   npm run build
   javy compile dist/plugin.js -o plugin.wasm
   ```

### Rust

1. **Set up project**:
   ```bash
   cargo init --lib
   # Add to Cargo.toml:
   # [lib]
   # crate-type = ["cdylib"]
   ```

2. **Implement plugin**:
   ```rust
   use serde::{Deserialize, Serialize};

   #[no_mangle]
   pub extern "C" fn alloc(size: usize) -> *mut u8 {
       // Memory allocation
   }

   #[no_mangle]
   pub extern "C" fn call(ptr: *const u8, len: usize, out_ptr: *mut u8, out_len: usize) -> usize {
       // Handle plugin calls
   }
   ```

3. **Build**:
   ```bash
   cargo build --release --target wasm32-wasi
   ```

### Go

1. **Create plugin**:
   ```go
   //go:build wasm

   package main

   //export call
   func call(ptr uint32, size uint32, outPtr uint32, outSize uint32) uint32 {
       // Handle plugin calls
   }

   func main() {
       // Required for WASM
   }
   ```

2. **Build**:
   ```bash
   tinygo build -o plugin.wasm -target wasi main.go
   # or
   GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm main.go
   ```

## Plugin SDK

### TypeScript SDK

The TypeScript SDK provides:

- Base classes for plugins and nodes
- Type definitions for all interfaces
- Memory management utilities
- Helper functions

Key components:

```typescript
// Plugin class
class Plugin {
  constructor(metadata: Metadata);
  register(node: PluginNode): void;
}

// Base node class
abstract class PluginNode<TInput, TOutput, TConfig> {
  abstract prep(input: TInput, config: TConfig, store: Store): Promise<any>;
  abstract exec(prepData: any, config: TConfig): Promise<TOutput>;
  abstract post(...): Promise<{ output: TOutput; next: string }>;
}

// Store interface
interface Store {
  get(key: string): any;
  set(key: string, value: any): void;
  delete(key: string): boolean;
}
```

### Memory Management

WASM plugins must manage memory carefully:

```typescript
// Allocate memory for data transfer
export function __pocket_alloc(size: number): number {
  // Return pointer to allocated memory
}

// Free allocated memory
export function __pocket_free(ptr: number, size: number): void {
  // Free the memory
}
```

## Security

### Sandboxing

Plugins run in a sandboxed WebAssembly environment with:

- **Memory Isolation**: Each plugin has its own memory space
- **No Network Access**: Plugins cannot make network requests
- **Limited Filesystem**: No direct filesystem access
- **Resource Limits**: Configurable memory and execution time limits

### Permissions

Define permissions in the manifest:

```yaml
permissions:
  memory: 10MB        # Maximum memory allocation
  timeout: 5s         # Maximum execution time
  env: []            # Allowed environment variables
  filesystem: []     # Allowed filesystem paths (future)
```

### Best Practices

1. **Validate All Inputs**: Never trust external data
2. **Handle Errors Gracefully**: Return proper error responses
3. **Respect Resource Limits**: Design for constrained environments
4. **No Side Effects in Exec**: Keep exec phase pure
5. **Document Security Considerations**: Note any security implications

## CLI Reference

### Installation

```bash
go install github.com/agentstation/pocket/cmd/pocket-plugins@latest
```

### Commands

- `pocket-plugins list` - List installed plugins
- `pocket-plugins install <path>` - Install a plugin
- `pocket-plugins remove <name>` - Remove a plugin
- `pocket-plugins info <name>` - Show plugin details
- `pocket-plugins validate <path>` - Validate a plugin

See the [CLI documentation](../cmd/pocket-plugins/README.md) for detailed usage.

## Examples

### Sentiment Analysis (TypeScript)

Analyzes text sentiment with configurable thresholds:

```typescript
class SentimentAnalyzerNode extends PluginNode {
  async exec(prepData: any, config: Config) {
    const { words } = prepData;
    const positiveCount = words.filter(w => positiveWords.includes(w)).length;
    const negativeCount = words.filter(w => negativeWords.includes(w)).length;
    
    const score = (positiveCount - negativeCount) / words.length;
    const sentiment = score > config.threshold ? 'positive' : 
                     score < -config.threshold ? 'negative' : 'neutral';
    
    return { sentiment, score, confidence: Math.abs(score) };
  }
}
```

### Word Counter (Rust)

Counts words with stop word filtering:

```rust
fn handle_exec(request: &Request) -> Response {
    let words: Vec<String> = cleaned_text
        .split_whitespace()
        .filter(|w| !config.stop_words.contains(w))
        .collect();
    
    let output = WordCounterOutput {
        total_words: words.len(),
        unique_words: word_frequencies.len(),
        word_frequencies,
        average_word_length: total_length as f64 / words.len() as f64,
    };
    
    Response { success: true, output: serde_json::to_value(output).unwrap() }
}
```

### JSON Transformer (Go)

Transforms JSON data with various operations:

```go
func flattenJSON(data interface{}, params map[string]interface{}) (interface{}, int) {
    result := make(map[string]interface{})
    changes := 0
    flattenHelper(data, "", separator, result, &changes)
    return result, changes
}
```

## Troubleshooting

### Common Issues

1. **"Plugin not found"**
   - Check installation path: `~/.pocket/plugins/`
   - Verify manifest.yaml exists

2. **"Memory limit exceeded"**
   - Increase limit in manifest.yaml
   - Optimize memory usage

3. **"Timeout exceeded"**
   - Increase timeout in permissions
   - Optimize algorithm

4. **"Invalid WASM module"**
   - Verify compilation target (wasm32-wasi)
   - Check for missing exports

### Debug Tips

1. Add logging to each phase
2. Test with `pocket-plugins validate`
3. Start with simple logic, add complexity gradually
4. Use the example plugins as reference

## Contributing

To contribute plugins:

1. Follow the plugin structure
2. Include comprehensive tests
3. Document all configuration options
4. Add examples in manifest.yaml
5. Submit PR with plugin in `plugin/community/`

## Resources

- [Plugin SDK API Reference](./PLUGIN_SDK_API.md)
- [Example Plugins](../plugin/examples/)
- [Javy Documentation](https://github.com/Shopify/javy)
- [WebAssembly Specification](https://webassembly.org/)
- [Pocket Workflow Guide](./WORKFLOWS.md)