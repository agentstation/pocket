package node

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Registry manages node type registration and creation.
type Registry struct {
	mu        sync.RWMutex
	builders  map[string]Builder
	metadata  map[string]Metadata
	validator TypeValidator
}

// Builder creates nodes of a specific type.
type Builder func(config map[string]any) (pocket.Node, error)

// Metadata contains information about a node type.
type Metadata struct {
	Name         string
	Description  string
	Version      string
	InputType    reflect.Type
	OutputType   reflect.Type
	ConfigSchema map[string]ConfigField
	Tags         []string
}

// ConfigField describes a configuration field.
type ConfigField struct {
	Name        string
	Type        string
	Required    bool
	Default     any
	Description string
	Validator   func(any) error
}

// TypeValidator validates node type compatibility.
type TypeValidator interface {
	ValidateConnection(from, to *Metadata) error
}

// NewRegistry creates a new node registry.
func NewRegistry() *Registry {
	return &Registry{
		builders:  make(map[string]Builder),
		metadata:  make(map[string]Metadata),
		validator: &defaultTypeValidator{},
	}
}

// Register registers a node type.
func (r *Registry) Register(nodeType string, builder Builder, metadata *Metadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.builders[nodeType]; exists {
		return fmt.Errorf("node type %q already registered", nodeType)
	}

	metadata.Name = nodeType
	r.builders[nodeType] = builder
	r.metadata[nodeType] = *metadata

	return nil
}

// Create creates a node instance.
func (r *Registry) Create(nodeType string, config map[string]any) (pocket.Node, error) {
	r.mu.RLock()
	builder, exists := r.builders[nodeType]
	metadata := r.metadata[nodeType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown node type: %q", nodeType)
	}

	// Validate config
	if err := r.validateConfig(&metadata, config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Apply defaults
	config = r.applyDefaults(&metadata, config)

	// Create node
	node, err := builder(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	// Note: In the new API, we cannot set type information directly on nodes
	// This could be handled through middleware or wrapper functions

	return node, nil
}

// GetMetadata returns metadata for a node type.
func (r *Registry) GetMetadata(nodeType string) (Metadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[nodeType]
	return metadata, exists
}

// List returns all registered node types.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.builders))
	for nodeType := range r.builders {
		types = append(types, nodeType)
	}
	return types
}

// ListByTag returns node types with a specific tag.
func (r *Registry) ListByTag(tag string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var types []string
	for nodeType, metadata := range r.metadata {
		for _, t := range metadata.Tags {
			if t == tag {
				types = append(types, nodeType)
				break
			}
		}
	}
	return types
}

// ValidateConnection validates that two node types can be connected.
func (r *Registry) ValidateConnection(fromType, toType string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fromMeta, fromExists := r.metadata[fromType]
	toMeta, toExists := r.metadata[toType]

	if !fromExists {
		return fmt.Errorf("unknown node type: %q", fromType)
	}
	if !toExists {
		return fmt.Errorf("unknown node type: %q", toType)
	}

	return r.validator.ValidateConnection(&fromMeta, &toMeta)
}

// validateConfig validates node configuration.
func (r *Registry) validateConfig(metadata *Metadata, config map[string]any) error {
	for fieldName, field := range metadata.ConfigSchema {
		value, exists := config[fieldName]

		if field.Required && !exists {
			return fmt.Errorf("required field %q is missing", fieldName)
		}

		if exists && field.Validator != nil {
			if err := field.Validator(value); err != nil {
				return fmt.Errorf("field %q validation failed: %w", fieldName, err)
			}
		}
	}

	// Check for unknown fields
	for key := range config {
		if _, ok := metadata.ConfigSchema[key]; !ok {
			return fmt.Errorf("unknown configuration field: %q", key)
		}
	}

	return nil
}

// applyDefaults applies default values to config.
func (r *Registry) applyDefaults(metadata *Metadata, config map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy existing config
	for k, v := range config {
		result[k] = v
	}

	// Apply defaults for missing fields
	for fieldName, field := range metadata.ConfigSchema {
		if _, exists := result[fieldName]; !exists && field.Default != nil {
			result[fieldName] = field.Default
		}
	}

	return result
}

// defaultTypeValidator provides basic type validation.
type defaultTypeValidator struct{}

func (v *defaultTypeValidator) ValidateConnection(from, to *Metadata) error {
	if from.OutputType == nil || to.InputType == nil {
		// No type information, allow connection
		return nil
	}

	if !isTypeCompatible(from.OutputType, to.InputType) {
		return fmt.Errorf("type mismatch: %s outputs %v but %s expects %v",
			from.Name, from.OutputType, to.Name, to.InputType)
	}

	return nil
}

// isTypeCompatible checks if output type is compatible with input type.
func isTypeCompatible(outputType, inputType reflect.Type) bool {
	// Same type is always compatible
	if outputType == inputType {
		return true
	}

	// Check if output type is assignable to input type
	if outputType.AssignableTo(inputType) {
		return true
	}

	// Check if output type implements input interface
	if inputType.Kind() == reflect.Interface && outputType.Implements(inputType) {
		return true
	}

	// Check if both are interface{} (any)
	if outputType.Kind() == reflect.Interface && outputType.NumMethod() == 0 &&
		inputType.Kind() == reflect.Interface && inputType.NumMethod() == 0 {
		return true
	}

	return false
}

// Common node type builders

// RegisterCommonTypes registers commonly used node types.
func RegisterCommonTypes(registry *Registry) {
	// Transform node
	_ = registry.Register("transform",
		func(config map[string]any) (pocket.Node, error) {
			transformFn, ok := config["function"].(func(any) (any, error))
			if !ok {
				return nil, fmt.Errorf("transform function required")
			}

			return pocket.NewNode[any, any]("transform",
				pocket.WithExec(func(ctx context.Context, input any) (any, error) {
					return transformFn(input)
				}),
			), nil
		},
		&Metadata{
			Description: "Transforms input data",
			Tags:        []string{"utility", "data"},
			ConfigSchema: map[string]ConfigField{
				"function": {
					Name:        "function",
					Type:        "func(any) (any, error)",
					Required:    true,
					Description: "Transformation function",
				},
			},
		},
	)

	// Filter node
	_ = registry.Register("filter",
		func(config map[string]any) (pocket.Node, error) {
			predicateFn, ok := config["predicate"].(func(any) bool)
			if !ok {
				return nil, fmt.Errorf("predicate function required")
			}

			return pocket.NewNode[any, any]("filter",
				pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
					if predicateFn(exec) {
						return exec, "pass", nil
					}
					return nil, "fail", nil
				}),
			), nil
		},
		&Metadata{
			Description: "Filters data based on predicate",
			Tags:        []string{"utility", "control"},
			ConfigSchema: map[string]ConfigField{
				"predicate": {
					Name:        "predicate",
					Type:        "func(any) bool",
					Required:    true,
					Description: "Filter predicate function",
				},
			},
		},
	)

	// Delay node
	_ = registry.Register("delay",
		func(config map[string]any) (pocket.Node, error) {
			duration, ok := config["duration"].(time.Duration)
			if !ok {
				return nil, fmt.Errorf("duration required")
			}

			return pocket.NewNode[any, any]("delay",
				pocket.WithExec(func(ctx context.Context, input any) (any, error) {
					select {
					case <-time.After(duration):
						return input, nil
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}),
			), nil
		},
		&Metadata{
			Description: "Delays execution",
			Tags:        []string{"utility", "timing"},
			ConfigSchema: map[string]ConfigField{
				"duration": {
					Name:        "duration",
					Type:        "time.Duration",
					Required:    true,
					Description: "Delay duration",
				},
			},
		},
	)

	// Logger node
	_ = registry.Register("logger",
		func(config map[string]any) (pocket.Node, error) {
			level, _ := config["level"].(string)
			if level == "" {
				level = "info"
			}

			return pocket.NewNode[any, any]("logger",
				pocket.WithExec(func(ctx context.Context, input any) (any, error) {
					// This would use actual logger
					fmt.Printf("[%s] %v\n", level, input)
					return input, nil
				}),
			), nil
		},
		&Metadata{
			Description: "Logs data",
			Tags:        []string{"utility", "debugging"},
			ConfigSchema: map[string]ConfigField{
				"level": {
					Name:        "level",
					Type:        "string",
					Default:     "info",
					Description: "Log level",
				},
			},
		},
	)
}

// Factory provides convenient node creation.
type Factory struct {
	registry *Registry
	defaults map[string]map[string]any
}

// NewFactory creates a new node factory.
func NewFactory(registry *Registry) *Factory {
	return &Factory{
		registry: registry,
		defaults: make(map[string]map[string]any),
	}
}

// SetDefaults sets default config for a node type.
func (f *Factory) SetDefaults(nodeType string, defaults map[string]any) {
	f.defaults[nodeType] = defaults
}

// Create creates a node with factory defaults applied.
func (f *Factory) Create(nodeType string, config map[string]any) (pocket.Node, error) {
	// Merge with factory defaults
	mergedConfig := make(map[string]any)

	if defaults, ok := f.defaults[nodeType]; ok {
		for k, v := range defaults {
			mergedConfig[k] = v
		}
	}

	for k, v := range config {
		mergedConfig[k] = v
	}

	return f.registry.Create(nodeType, mergedConfig)
}
