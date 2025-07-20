package builtin

import (
	"fmt"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/yaml"
)

// NodeBuilder creates nodes and provides metadata.
type NodeBuilder interface {
	Metadata() NodeMetadata
	Build(def *yaml.NodeDefinition) (pocket.Node, error)
}

// Registry manages all built-in nodes.
type Registry struct {
	builders map[string]NodeBuilder
}

// NewRegistry creates a new node registry.
func NewRegistry() *Registry {
	return &Registry{
		builders: make(map[string]NodeBuilder),
	}
}

// Register adds a node builder.
func (r *Registry) Register(builder NodeBuilder) {
	meta := builder.Metadata()
	r.builders[meta.Type] = builder
}

// Get returns a builder by type.
func (r *Registry) Get(nodeType string) (NodeBuilder, bool) {
	builder, exists := r.builders[nodeType]
	return builder, exists
}

// All returns all registered builders.
func (r *Registry) All() map[string]NodeBuilder {
	return r.builders
}

// RegisterAll registers all built-in nodes with a YAML loader.
func RegisterAll(loader *yaml.Loader, verbose bool) *Registry {
	registry := NewRegistry()

	// Register core nodes
	registry.Register(&EchoNodeBuilder{Verbose: verbose})
	registry.Register(&DelayNodeBuilder{Verbose: verbose})
	registry.Register(&RouterNodeBuilder{Verbose: verbose})
	registry.Register(&ConditionalNodeBuilder{Verbose: verbose})

	// Register data nodes
	registry.Register(&TransformNodeBuilder{Verbose: verbose})
	registry.Register(&TemplateNodeBuilder{Verbose: verbose})
	registry.Register(&JSONPathNodeBuilder{Verbose: verbose})
	registry.Register(&ValidateNodeBuilder{Verbose: verbose})
	registry.Register(&AggregateNodeBuilder{Verbose: verbose})

	// Register I/O nodes
	registry.Register(&HTTPNodeBuilder{Verbose: verbose})
	registry.Register(&FileNodeBuilder{Verbose: verbose})
	registry.Register(&ExecNodeBuilder{Verbose: verbose})

	// Register flow nodes
	registry.Register(&ParallelNodeBuilder{Verbose: verbose})

	// Register all with YAML loader with validation
	for _, builder := range registry.All() {
		meta := builder.Metadata()
		// Wrap the builder with validation
		wrappedBuilder := createValidatingBuilder(builder)
		loader.RegisterNodeType(meta.Type, wrappedBuilder)
	}

	return registry
}

// createValidatingBuilder wraps a builder with config validation.
func createValidatingBuilder(builder NodeBuilder) func(def *yaml.NodeDefinition) (pocket.Node, error) {
	return func(def *yaml.NodeDefinition) (pocket.Node, error) {
		// Validate config against schema
		meta := builder.Metadata()
		if err := ValidateNodeConfig(&meta, def.Config); err != nil {
			return nil, fmt.Errorf("config validation failed for node '%s': %w", def.Name, err)
		}

		// Build the node
		return builder.Build(def)
	}
}
