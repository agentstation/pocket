// Package plugin provides the core interfaces and types for the Pocket plugin system.
package plugin

import (
	"context"
	"encoding/json"
	"time"
)

// Plugin represents a loaded plugin instance.
type Plugin interface {
	// Metadata returns the plugin's metadata
	Metadata() Metadata

	// Call invokes a function exported by the plugin
	Call(ctx context.Context, function string, input []byte) ([]byte, error)

	// Close releases plugin resources
	Close(ctx context.Context) error
}

// Metadata contains plugin information.
type Metadata struct {
	// Basic info
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Author      string `json:"author" yaml:"author"`
	License     string `json:"license,omitempty" yaml:"license,omitempty"`

	// Runtime requirements
	Runtime    string `json:"runtime" yaml:"runtime"`                           // "wasm" for now
	Binary     string `json:"binary" yaml:"binary"`                             // Path to .wasm file
	EntryPoint string `json:"entryPoint,omitempty" yaml:"entryPoint,omitempty"` // Main function

	// Node definitions
	Nodes []NodeDefinition `json:"nodes" yaml:"nodes"`

	// Security permissions
	Permissions Permissions `json:"permissions,omitempty" yaml:"permissions,omitempty"`

	// Requirements
	Requirements Requirements `json:"requirements,omitempty" yaml:"requirements,omitempty"`
}

// NodeDefinition describes a node exported by the plugin.
type NodeDefinition struct {
	Type         string                 `json:"type" yaml:"type"`
	Category     string                 `json:"category" yaml:"category"`
	Description  string                 `json:"description" yaml:"description"`
	ConfigSchema map[string]interface{} `json:"configSchema,omitempty" yaml:"configSchema,omitempty"`
	InputSchema  map[string]interface{} `json:"inputSchema,omitempty" yaml:"inputSchema,omitempty"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty" yaml:"outputSchema,omitempty"`
	Examples     []Example              `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// Example shows how to use a node.
type Example struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Input       interface{}            `json:"input,omitempty" yaml:"input,omitempty"`
	Output      interface{}            `json:"output,omitempty" yaml:"output,omitempty"`
}

// Permissions defines what the plugin is allowed to access.
type Permissions struct {
	// Network access
	Network []string `json:"network,omitempty" yaml:"network,omitempty"` // Allowed domains/patterns

	// Environment variables
	Env []string `json:"env,omitempty" yaml:"env,omitempty"` // Allowed env var names

	// File system access
	Filesystem []string `json:"filesystem,omitempty" yaml:"filesystem,omitempty"` // Allowed paths

	// Resource limits
	Memory  string        `json:"memory,omitempty" yaml:"memory,omitempty"`   // Max memory (e.g., "100MB")
	CPU     string        `json:"cpu,omitempty" yaml:"cpu,omitempty"`         // Max CPU time per call
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"` // Max execution time
}

// Requirements specifies plugin dependencies.
type Requirements struct {
	Pocket string `json:"pocket,omitempty" yaml:"pocket,omitempty"` // Min Pocket version
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"` // Required memory
}

// Request is sent to plugin functions.
type Request struct {
	// Node type being invoked
	Node string `json:"node"`

	// Function to call (prep, exec, post)
	Function string `json:"function"`

	// Node configuration
	Config map[string]interface{} `json:"config,omitempty"`

	// Input data
	Input json.RawMessage `json:"input,omitempty"`

	// Additional data for post function
	PrepResult json.RawMessage `json:"prepResult,omitempty"`
	ExecResult json.RawMessage `json:"execResult,omitempty"`
}

// Response is returned from plugin functions.
type Response struct {
	// Success/error status
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`

	// Function results
	Output json.RawMessage `json:"output,omitempty"`

	// For post function - routing decision
	Next string `json:"next,omitempty"`
}

// Loader discovers and loads plugins.
type Loader interface {
	// Discover finds all plugins in the given paths
	Discover(paths ...string) ([]Metadata, error)

	// Load loads a plugin from the given path
	Load(ctx context.Context, path string) (Plugin, error)

	// LoadFromMetadata loads a plugin using its metadata
	LoadFromMetadata(ctx context.Context, metadata Metadata) (Plugin, error)
}

// Registry manages loaded plugins.
type Registry interface {
	// Register adds a plugin to the registry
	Register(plugin Plugin) error

	// Get retrieves a plugin by name
	Get(name string) (Plugin, bool)

	// List returns all registered plugins
	List() []Plugin

	// Close releases all plugin resources
	Close(ctx context.Context) error
}
