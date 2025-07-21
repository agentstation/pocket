# Sentiment Analyzer Plugin

This is an example TypeScript plugin for Pocket that performs sentiment analysis on text.

## Features

- Analyzes text sentiment (positive, negative, neutral)
- Calculates confidence scores
- Extracts sentiment keywords
- Supports configurable thresholds
- Routes based on sentiment results

## Building

1. Install dependencies:
```bash
npm install
```

2. Build the plugin:
```bash
npm run build:all
```

This will:
- Compile TypeScript to JavaScript
- Bundle all dependencies
- Compile to WebAssembly using Javy

## Installation

```bash
pocket plugins install ./
```

## Usage

```yaml
nodes:
  - name: analyze-feedback
    type: sentiment
    config:
      threshold: 0.15
      languages: ["en", "es"]
    
  - name: handle-positive
    type: echo
    config:
      message: "Thank you for your positive feedback!"
    
  - name: handle-negative
    type: echo
    config:
      message: "We're sorry to hear that. Let us help."

connections:
  - from: analyze-feedback
    condition: high-positive
    to: handle-positive
  - from: analyze-feedback
    condition: high-negative
    to: handle-negative
```

## Development

The plugin is built using the Pocket Plugin SDK for TypeScript. See the source code for implementation details.

### Project Structure

```
sentiment-analyzer/
├── src/
│   └── index.ts      # Plugin implementation
├── manifest.yaml     # Plugin metadata
├── package.json      # Node dependencies
├── tsconfig.json     # TypeScript config
└── plugin.wasm       # Compiled plugin (after build)
```

### Key Concepts

1. **PluginNode Class**: Extends the SDK's base class
2. **Lifecycle Methods**: Implements prep, exec, and post phases
3. **Type Safety**: Uses TypeScript interfaces for input/output
4. **Schema Validation**: Defines JSON schemas for configuration
5. **Routing**: Returns next node based on analysis results