# JSON Transformer Plugin

This is an example Go plugin for Pocket that performs various JSON transformations.

## Features

- **Flatten**: Convert nested JSON to flat key-value pairs
- **Nest**: Convert flat JSON back to nested structure
- **Filter**: Include or exclude specific fields
- **Map**: Transform values (example: uppercase strings)
- **Reduce**: Aggregate data (example: count elements)

## Building

1. Install Go and TinyGo:
```bash
# Install TinyGo from https://tinygo.org/getting-started/
brew install tinygo
```

2. Build the plugin:
```bash
tinygo build -o plugin.wasm -target wasi main.go
```

Alternative with standard Go (requires Go 1.21+):
```bash
GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm main.go
```

## Installation

```bash
pocket plugins install ./
```

## Usage

```yaml
nodes:
  - name: flatten-data
    type: json-transform
    config:
      transforms:
        my-flatten:
          type: flatten
          parameters:
            separator: "."
    
  - name: filter-sensitive
    type: json-transform
    config:
      transforms:
        remove-private:
          type: filter
          parameters:
            fields: ["password", "api_key", "secret"]
            exclude: true

connections:
  - from: flatten-data
    to: filter-sensitive
```

### Transform Types

1. **Flatten**: Converts nested objects to dot-notation keys
   ```yaml
   input: {a: {b: {c: 1}}}
   output: {"a.b.c": 1}
   ```

2. **Nest**: Reverse of flatten
   ```yaml
   input: {"a.b.c": 1}
   output: {a: {b: {c: 1}}}
   ```

3. **Filter**: Include/exclude fields
   ```yaml
   parameters:
     fields: ["name", "email"]
     exclude: false  # false = include only these, true = exclude these
   ```

4. **Map**: Transform all string values (example: uppercase)
   ```yaml
   input: {name: "john"}
   output: {name: "JOHN"}
   ```

5. **Reduce**: Count all elements in the structure
   ```yaml
   output: {total_elements: 15, type: "map[string]interface{}"}
   ```

## Development

The plugin demonstrates:
1. **WASM exports**: Using Go's `//export` directive
2. **Memory management**: Safe memory operations with Go slices
3. **JSON processing**: Working with dynamic JSON structures
4. **Error handling**: Proper error propagation
5. **Routing**: Decision-based routing in post phase