# Word Counter Plugin

This is an example Rust plugin for Pocket that performs word counting and text analysis.

## Features

- Count total and unique words
- Calculate word frequencies
- Find longest and shortest words
- Calculate average word length
- Filter by minimum word length
- Exclude stop words
- Case-sensitive/insensitive analysis

## Building

1. Install Rust and `wasm32-wasi` target:
```bash
rustup target add wasm32-wasi
```

2. Build the plugin:
```bash
cargo build --release --target wasm32-wasi
cp target/wasm32-wasi/release/word_counter.wasm plugin.wasm
```

## Installation

```bash
pocket plugins install ./
```

## Usage

```yaml
nodes:
  - name: analyze-document
    type: word-count
    config:
      min_word_length: 3
      stop_words: ["the", "a", "an", "and", "or", "but"]
    
  - name: handle-short
    type: echo
    config:
      message: "Short document processed"
    
  - name: handle-long
    type: echo
    config:
      message: "Long document processed"

connections:
  - from: analyze-document
    condition: short
    to: handle-short
  - from: analyze-document
    condition: long
    to: handle-long
```

## Development

The plugin demonstrates:
1. **WASM exports**: `alloc`, `dealloc`, `metadata`, and `call` functions
2. **Memory management**: Safe memory allocation and deallocation
3. **JSON communication**: Parsing requests and generating responses
4. **Error handling**: Graceful error responses
5. **Configuration**: Using config schemas with defaults