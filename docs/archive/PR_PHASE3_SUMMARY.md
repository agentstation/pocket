# Phase 3: WebAssembly Plugin System

## Summary

This PR implements Phase 3 of the Pocket plugin system, adding WebAssembly (WASM) support to enable plugins written in any language that compiles to WASM. This builds on Phase 1 (built-in nodes) and Phase 2 (Lua scripting) to provide a complete, extensible plugin ecosystem.

## Major Features

### 1. WebAssembly Runtime Integration
- Integrated **wazero** as the WASM runtime (pure Go, no CGO dependencies)
- Implemented sandboxed execution with memory limits and timeouts
- Created plugin loader system for discovering and loading WASM plugins
- Added security features including filesystem restrictions and environment filtering

### 2. Plugin Architecture
- Designed comprehensive plugin interfaces (`Plugin`, `Metadata`, `NodeDefinition`)
- Implemented the three-phase lifecycle (prep/exec/post) for plugin nodes
- Created JSON-based communication protocol between host and plugins
- Added permission system for controlling plugin capabilities

### 3. TypeScript/JavaScript SDK
- Developed complete TypeScript SDK (`@pocket/plugin-sdk`)
- Base classes for plugin and node development
- Type definitions for all interfaces
- Memory management utilities for WASM
- Integrated Javy for JavaScript to WASM compilation

### 4. Example Plugins
Created three fully-functional example plugins demonstrating different languages:
- **sentiment-analyzer** (TypeScript): Text sentiment analysis with configurable thresholds
- **word-counter** (Rust): Word counting with stop word filtering
- **json-transformer** (Go): JSON data transformation with multiple operations

### 5. CLI Tool
Implemented `pocket-plugins` CLI with commands:
- `list` - List installed plugins
- `install` - Install a plugin from directory
- `remove` - Remove an installed plugin
- `info` - Show detailed plugin information
- `validate` - Validate plugin manifest and structure
- `run` - Execute a plugin function directly

### 6. Comprehensive Documentation
- Main plugin documentation (`docs/PLUGINS.md`)
- Complete API reference (`docs/PLUGIN_SDK_API.md`)
- Migration guide (`docs/PLUGIN_MIGRATION.md`)
- Updated README.md with plugin system information
- Added build scripts and examples for each language

## Technical Implementation

### Core Components

1. **Plugin Interface** (`/plugin/plugin.go`):
   ```go
   type Plugin interface {
       Metadata() Metadata
       Call(ctx context.Context, function string, input []byte) ([]byte, error)
       Close(ctx context.Context) error
   }
   ```

2. **WASM Plugin** (`/plugin/wasm/plugin.go`):
   - Uses wazero for WASM runtime
   - Implements memory management
   - Handles function calls and data marshaling

3. **Plugin Loader** (`/plugin/loader/loader.go`):
   - Discovers plugins in standard paths
   - Validates manifests
   - Manages plugin lifecycle

4. **TypeScript SDK** (`/plugin/sdk/typescript/`):
   - Abstract base classes
   - Type-safe interfaces
   - Build tooling integration

## Security Features

- **Sandboxed Execution**: Plugins run in isolated WASM environments
- **Memory Limits**: Configurable memory allocation limits
- **Execution Timeouts**: Prevent runaway plugins
- **No Network Access**: Plugins cannot make network requests
- **Filesystem Isolation**: No direct filesystem access
- **Environment Filtering**: Control which environment variables are accessible

## Testing

- Added comprehensive tests for WASM plugin system
- Fixed loader validation test expectations
- All tests passing with race detection enabled
- Fixed 40+ linting issues for production quality

## Usage Example

```yaml
# In a Pocket workflow
nodes:
  - name: analyze-sentiment
    type: sentiment
    config:
      threshold: 0.7
      
  - name: count-words
    type: word-count
    config:
      min_word_length: 3
      
connections:
  - from: analyze-sentiment
    to: count-words
    action: default
```

## Breaking Changes

None - Phase 3 is fully backward compatible with existing Pocket workflows.

## Future Enhancements

- Plugin marketplace/registry
- Hot reload support
- Network permissions for API access
- Persistent storage for plugins
- Plugin versioning and updates
- Performance optimizations

## Files Changed

### New Files
- `/plugin/plugin.go` - Core plugin interfaces
- `/plugin/wasm/plugin.go` - WASM runtime implementation
- `/plugin/loader/loader.go` - Plugin discovery and loading
- `/plugin/sdk/typescript/*` - TypeScript SDK
- `/plugin/examples/*` - Example plugins
- `/cmd/pocket-plugins/*` - CLI tool
- `/docs/PLUGINS.md` - Main documentation
- `/docs/PLUGIN_SDK_API.md` - API reference
- `/docs/PLUGIN_MIGRATION.md` - Migration guide

### Modified Files
- `/README.md` - Added plugin system information
- `/Makefile` - Added build-plugins target
- Various test files with fixed expectations

## Checklist

- [x] Code compiles and tests pass
- [x] Linting issues resolved
- [x] Documentation complete
- [x] Examples working
- [x] CLI tool functional
- [x] Security considerations addressed
- [x] Backward compatibility maintained

## How to Test

1. Build the plugin CLI:
   ```bash
   make build-plugins
   ```

2. Build and install an example plugin:
   ```bash
   cd plugin/examples/typescript/sentiment-analyzer
   npm install && npm run build
   pocket-plugins install .
   ```

3. Use in a workflow or test directly:
   ```bash
   pocket-plugins run sentiment-analyzer sentiment '{"text":"This is amazing!"}'
   ```

## Related Issues

- Continues from Phase 1 (Built-in Nodes) and Phase 2 (Lua Scripting)
- Completes the plugin system implementation

---

This PR delivers a production-ready WebAssembly plugin system that enables extending Pocket with custom functionality written in any language that compiles to WASM, while maintaining security and performance.