# Javy Integration Guide

This guide explains how to compile TypeScript/JavaScript plugins to WebAssembly using Javy.

## Prerequisites

1. **Install Javy**:
   ```bash
   # Using npm (recommended)
   npm install -g @shopify/javy

   # Or download binary from GitHub
   # https://github.com/Shopify/javy/releases
   ```

2. **Install Node.js dependencies**:
   ```bash
   npm install
   ```

## Building Plugins

### Quick Build

```bash
# Build everything (TypeScript → JavaScript → WASM)
make build

# Or use npm scripts
npm run build:all
```

### Step-by-Step Build

1. **Compile TypeScript to JavaScript**:
   ```bash
   npm run build
   ```

2. **Bundle for Javy** (creates a single JS file):
   ```bash
   npm run bundle
   ```

3. **Compile to WebAssembly**:
   ```bash
   javy compile dist/plugin.js -o plugin.wasm
   ```

## How It Works

### 1. TypeScript Plugin Structure

Your plugin extends the SDK's base classes:

```typescript
import { Plugin, PluginNode } from '@pocket/plugin-sdk';

class MyNode extends PluginNode<Input, Output, Config> {
  async prep(input: Input, config: Config, store: Store) {
    // Validate and prepare data
  }

  async exec(prepResult: any, config: Config) {
    // Core logic
  }

  async post(input: Input, prepResult: any, execResult: Output, config: Config, store: Store) {
    // Post-processing and routing
  }
}
```

### 2. Bundling Process

The build script:
1. Compiles TypeScript to JavaScript
2. Bundles all dependencies into a single file
3. Adds Javy-specific wrappers for WASM compatibility

### 3. WASM Execution

When running in WASM:
1. Javy provides a minimal JavaScript runtime
2. The plugin communicates via JSON serialization
3. Memory is managed by the WASM runtime

## Limitations

### Async Operations
Javy has limited async support. The SDK provides sync wrappers, but complex async operations may need restructuring.

### API Access
- No network access
- No filesystem access (except through host)
- Limited to computation and data transformation

### Memory
- Default memory limit: 10MB
- Can be configured in manifest.yaml

## Best Practices

1. **Keep It Simple**: Focus on pure data transformation
2. **Minimize Dependencies**: Large libraries increase WASM size
3. **Test Locally**: Test JS version before compiling to WASM
4. **Handle Errors**: Always return proper error responses

## Troubleshooting

### "Javy not found"
```bash
npm install -g @shopify/javy
```

### "Memory limit exceeded"
Increase limit in manifest.yaml:
```yaml
permissions:
  memory: 20MB
```

### "Async operation failed"
Restructure to use synchronous operations or callbacks.

## Example Build Output

```
$ make build
Installing dependencies...
Building JavaScript bundle...
Bundling for Javy...
Compiling to WebAssembly with Javy...
WASM plugin built successfully!
```

The final `plugin.wasm` file is ready to use with Pocket!