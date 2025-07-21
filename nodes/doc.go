// Package nodes provides the built-in node types for Pocket workflows.
// These nodes are available out-of-the-box and provide core functionality
// for data manipulation, I/O operations, flow control, and more.
//
// The package is organized into categories:
//   - core: Essential nodes like echo, delay, router, conditional
//   - data: Data manipulation nodes like transform, template, jsonpath
//   - io: I/O operations like http, file
//   - flow: Flow control nodes like parallel, retry, cache
//
// Each node type is registered with metadata that describes its
// configuration schema, inputs, outputs, and usage examples.
package nodes
