# Go Style Guide for Pocket

This document outlines the Go coding standards and conventions used in the Pocket project. These standards are automatically enforced through our formatting tools and Claude Code hooks.

## Automatic Formatting

The Pocket project uses automatic Go formatting that runs whenever files are created or modified through Claude Code. This ensures consistent code style without manual intervention.

### What Gets Formatted Automatically

1. **Code formatting** - Standard Go formatting via `gofmt`
2. **Import organization** - Imports are grouped and sorted via `goimports`
3. **Comment punctuation** - Single-line comments starting with a capital letter automatically get periods added
4. **Linting fixes** - Various auto-fixable issues are resolved via `golangci-lint`

### How It Works

When you save or modify a Go file:
1. `gofmt` formats the code structure
2. `goimports` organizes imports with local packages grouped separately
3. Comments are checked and periods added where appropriate
4. Auto-fixable linting issues are resolved

## Code Organization

### Package Structure

```go
package builtin

import (
	// Standard library imports
	"context"
	"fmt"
	"time"

	// Third-party imports
	"github.com/some/external/package"

	// Local imports (automatically grouped)
	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/yaml"
)
```

### File Organization

1. Package declaration
2. Import statements (automatically organized)
3. Constants
4. Types
5. Variables
6. Functions (exported first, then unexported)

## Naming Conventions

### General Rules

- Use meaningful, descriptive names
- Avoid abbreviations unless widely understood (e.g., `ctx` for context)
- Prefer clarity over brevity

### Specific Conventions

```go
// Interfaces - "er" suffix for single-method interfaces
type NodeBuilder interface { ... }

// Structs - Noun or noun phrase
type HTTPNodeBuilder struct { ... }

// Functions - Verb or verb phrase
func ValidateNodeConfig() error { ... }

// Constants - MixedCaps or mixedCaps
const DefaultTimeout = 30 * time.Second
const maxRetries = 3
```

## Comments and Documentation

### Package Comments

Every package should have a package comment:

```go
// Package builtin provides the built-in node implementations for Pocket.
package builtin
```

### Exported Types and Functions

All exported types and functions must have comments starting with the name:

```go
// NodeBuilder creates nodes and provides metadata.
type NodeBuilder interface { ... }

// Build creates a node from a definition.
func (b *HTTPNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) { ... }
```

### Comment Formatting

- Comments are automatically formatted to end with periods
- Use complete sentences for doc comments
- For inline comments, be concise but clear

```go
// ValidateNodeConfig validates a node configuration against its schema.
func ValidateNodeConfig(meta *NodeMetadata, config map[string]interface{}) error {
	// No schema defined, skip validation
	if len(meta.ConfigSchema) == 0 {
		return nil
	}
	// ... rest of function
}
```

## Error Handling

### Error Messages

- Error messages should be lowercase
- Don't end with punctuation
- Include context when wrapping errors

```go
if err != nil {
	return fmt.Errorf("failed to parse template: %w", err)
}
```

### Error Checking

Always check errors immediately:

```go
data, err := os.ReadFile(path)
if err != nil {
	return nil, fmt.Errorf("read file: %w", err)
}
```

## Testing

### Test File Naming

Test files should be named `*_test.go` and placed in the same package:

```go
// builders_test.go
package builtin

func TestHTTPNodeBuilder(t *testing.T) { ... }
```

### Test Organization

1. Test the happy path first
2. Test edge cases
3. Test error conditions
4. Use table-driven tests for multiple scenarios

```go
func TestValidateNodeConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		// Happy path test
	})

	t.Run("missing required field", func(t *testing.T) {
		// Error condition test
	})
}
```

## Concurrency

### Context Usage

Always pass context as the first parameter:

```go
func (b *HTTPNodeBuilder) Build(ctx context.Context, def *yaml.NodeDefinition) (pocket.Node, error) {
	// Use context for cancellation and timeouts
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	// ...
}
```

### Goroutine Management

- Always ensure goroutines can be cancelled
- Use sync.WaitGroup or channels for coordination
- Handle panics in goroutines

## Performance Considerations

### Preallocate Slices

When the size is known, preallocate slices:

```go
// Good
conditions := make([]condition, 0, len(conditionsRaw))

// Avoid
var conditions []condition
```

### Reuse Buffers

Use `bytes.Buffer` or `strings.Builder` for string concatenation:

```go
var sb strings.Builder
sb.WriteString("prefix")
sb.WriteString(value)
result := sb.String()
```

## Disabling Formatting

In rare cases where automatic formatting needs to be disabled:

### For Linting Issues

Use `//nolint` directives sparingly and with explanations:

```go
//nolint:gocyclo // Complex configuration parsing requires many conditions
func ComplexFunction() { ... }
```

### For Formatting

There's no way to disable gofmt, but you can use build tags to exclude files:

```go
//go:build ignore

package main
```

## Running Formatters Manually

While formatting happens automatically, you can also run it manually:

```bash
# Basic formatting
make fmt

# Comprehensive formatting with all tools
make fmt-all

# Check formatting without making changes
make fmt-check
```

## Continuous Integration

Our CI pipeline enforces these standards by:
1. Running `make fmt-check` to ensure code is formatted
2. Running `make lint` to check for linting issues
3. Failing the build if formatting is needed

## Contributing

When contributing to Pocket:
1. Let the automatic formatting handle style
2. Focus on writing clear, idiomatic Go code
3. Add meaningful comments and documentation
4. Write comprehensive tests
5. Follow the patterns established in the codebase

The automatic formatting ensures consistency, so you can focus on the logic and functionality of your code rather than formatting details.