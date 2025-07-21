# Pocket Plugin Examples

This directory contains example plugins demonstrating how to create Pocket plugins in different languages.

## Languages

### TypeScript/JavaScript
- **[sentiment-analyzer](./typescript/sentiment-analyzer/)**: Sentiment analysis plugin that analyzes text and determines positive/negative/neutral sentiment

### Rust
- **[word-counter](./rust/word-counter/)**: Word counting and text analysis plugin with configurable stop words and minimum word length

### Go
- **[json-transformer](./go/json-transformer/)**: JSON transformation plugin supporting flatten, nest, filter, map, and reduce operations

## Building Plugins

Each plugin directory contains:
- Source code in the respective language
- `manifest.yaml` - Plugin metadata and configuration
- `README.md` - Plugin-specific documentation
- Build instructions for compiling to WebAssembly

## Common Patterns

All plugins follow the same patterns:

1. **Memory Management**: Export `alloc` and `dealloc` functions for memory allocation
2. **Metadata Export**: Export plugin metadata through the `metadata` function
3. **Call Function**: Main `call` function that handles prep/exec/post lifecycle
4. **JSON Communication**: All data exchange uses JSON encoding
5. **Error Handling**: Proper error responses with descriptive messages

## Plugin Lifecycle

Each node in a plugin follows the Pocket lifecycle:

1. **Prep**: Validate input, prepare data for processing
2. **Exec**: Core business logic execution
3. **Post**: Process results, determine routing, update state

## Getting Started

To create a new plugin:

1. Choose your language and copy the relevant example
2. Modify the node types and logic for your use case
3. Update the manifest.yaml with your plugin details
4. Build the plugin to WebAssembly
5. Install with `pocket plugins install ./`

## Testing Plugins

Each example includes test cases in the manifest that demonstrate:
- Expected inputs and outputs
- Configuration options
- Routing behavior

You can test plugins using:
```bash
pocket plugins test <plugin-name>
```