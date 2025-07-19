package yaml

import (
	"context"
	"fmt"
	"time"

	"github.com/agentstation/pocket"
)

const (
	defaultAction = "default"
)

// NodeFactory creates nodes from definitions.
type NodeFactory interface {
	CreateNode(def *NodeDefinition) (*pocket.Node, error)
}

// defaultNodeFactory provides basic node creation.
type defaultNodeFactory struct {
	registry map[string]NodeBuilder
}

// NodeBuilder is a function that builds a node from a definition.
type NodeBuilder func(def *NodeDefinition) (*pocket.Node, error)

// Loader loads flow definitions and creates executable flows.
type Loader struct {
	parser  *Parser
	factory NodeFactory
}

// NewLoader creates a new YAML flow loader.
func NewLoader() *Loader {
	return &Loader{
		parser: NewParser(),
		factory: &defaultNodeFactory{
			registry: make(map[string]NodeBuilder),
		},
	}
}

// WithNodeFactory sets a custom node factory.
func (l *Loader) WithNodeFactory(factory NodeFactory) *Loader {
	l.factory = factory
	return l
}

// RegisterNodeType registers a builder for a node type.
func (l *Loader) RegisterNodeType(nodeType string, builder NodeBuilder) {
	if df, ok := l.factory.(*defaultNodeFactory); ok {
		df.registry[nodeType] = builder
	}
}

// LoadFile loads a flow from a YAML file.
func (l *Loader) LoadFile(filename string, store pocket.Store) (*pocket.Flow, error) {
	def, err := l.parser.ParseFile(filename)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	return l.LoadDefinition(def, store)
}

// LoadString loads a flow from a YAML string.
func (l *Loader) LoadString(yamlStr string, store pocket.Store) (*pocket.Flow, error) {
	def, err := l.parser.ParseString(yamlStr)
	if err != nil {
		return nil, fmt.Errorf("parse string: %w", err)
	}

	return l.LoadDefinition(def, store)
}

// LoadDefinition creates a flow from a parsed definition.
func (l *Loader) LoadDefinition(def *FlowDefinition, store pocket.Store) (*pocket.Flow, error) {
	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("invalid flow definition: %w", err)
	}

	// Create all nodes
	nodes := make(map[string]*pocket.Node)
	for _, nodeDef := range def.Nodes {
		node, err := l.factory.CreateNode(&nodeDef)
		if err != nil {
			return nil, fmt.Errorf("create node %s: %w", nodeDef.Name, err)
		}
		nodes[nodeDef.Name] = node
	}

	// Connect nodes
	for _, conn := range def.Connections {
		fromNode := nodes[conn.From]
		toNode := nodes[conn.To]

		action := conn.Action
		if action == "" {
			action = defaultAction
		}

		fromNode.Connect(action, toNode)
	}

	// Create flow with start node
	startNode := nodes[def.Start]
	if startNode == nil {
		return nil, fmt.Errorf("start node %s not found", def.Start)
	}

	// Store metadata in the store
	if def.Metadata != nil {
		for k, v := range def.Metadata {
			_ = store.Set(context.Background(), fmt.Sprintf("flow:metadata:%s", k), v)
		}
	}

	return pocket.NewFlow(startNode, store), nil
}

// CreateNode implements NodeFactory for defaultNodeFactory.
func (f *defaultNodeFactory) CreateNode(def *NodeDefinition) (*pocket.Node, error) {
	builder, exists := f.registry[def.Type]
	if !exists {
		// Fall back to generic node creation
		return f.createGenericNode(def)
	}

	return builder(def)
}

// createGenericNode creates a basic node with config stored.
func (f *defaultNodeFactory) createGenericNode(def *NodeDefinition) (*pocket.Node, error) {
	// Set up basic exec that stores config
	execFunc := func(ctx context.Context, input any) (any, error) {
		// Return input with config data to be stored in post
		return map[string]interface{}{
			"input":      input,
			"nodeConfig": def.Config,
			"nodeType":   def.Type,
			"nodeName":   def.Name,
		}, nil
	}

	// Wrap with retry if configured
	if def.Retry != nil {
		delay, err := def.Retry.GetRetryDelay()
		if err != nil {
			return nil, fmt.Errorf("parse retry delay: %w", err)
		}
		originalExec := execFunc
		execFunc = func(ctx context.Context, input any) (any, error) {
			var lastErr error
			for attempt := 0; attempt < def.Retry.MaxAttempts; attempt++ {
				if attempt > 0 {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(delay * time.Duration(attempt)):
						// Exponential backoff
					}
				}
				result, err := originalExec(ctx, input)
				if err == nil {
					return result, nil
				}
				lastErr = err
			}
			return nil, fmt.Errorf("failed after %d attempts: %w", def.Retry.MaxAttempts, lastErr)
		}
	}

	// Wrap with timeout if configured
	if def.Timeout != "" {
		timeout, err := def.GetTimeout()
		if err != nil {
			return nil, fmt.Errorf("parse timeout: %w", err)
		}
		originalExec := execFunc
		execFunc = func(ctx context.Context, input any) (any, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan struct{})
			var result any
			var err error

			go func() {
				result, err = originalExec(timeoutCtx, input)
				close(done)
			}()

			select {
			case <-done:
				return result, err
			case <-timeoutCtx.Done():
				return nil, fmt.Errorf("node %s timed out after %v", def.Name, timeout)
			}
		}
	}

	// Add post function to store config
	postFunc := func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
		// Extract data from exec result
		if data, ok := exec.(map[string]interface{}); ok {
			nodeName := data["nodeName"].(string)
			nodeConfig := data["nodeConfig"]
			nodeType := data["nodeType"].(string)
			actualInput := data["input"]

			// Store node config
			_ = store.Set(ctx, fmt.Sprintf("node:%s:config", nodeName), nodeConfig)
			_ = store.Set(ctx, fmt.Sprintf("node:%s:type", nodeName), nodeType)

			// Return the actual input
			return actualInput, "default", nil
		}
		// If not wrapped data, just pass through
		return exec, "default", nil
	}

	node := pocket.NewNode[any, any](def.Name,
		pocket.WithExec(execFunc),
		pocket.WithPost(postFunc),
	)

	// Note: Cannot set type information directly on nodes in the new API

	return node, nil
}

// // getTypeByName returns a reflect.Type for common type names.
// func getTypeByName(typeName string) reflect.Type {
// 	switch typeName {
// 	case "string":
// 		return reflect.TypeOf("")
// 	case "int":
// 		return reflect.TypeOf(0)
// 	case "float64":
// 		return reflect.TypeOf(0.0)
// 	case "bool":
// 		return reflect.TypeOf(false)
// 	case "map":
// 		return reflect.TypeOf(map[string]interface{}{})
// 	case "slice":
// 		return reflect.TypeOf([]interface{}{})
// 	default:
// 		// For unknown types, use interface{}
// 		return reflect.TypeOf((*interface{})(nil)).Elem()
// 	}
// }

// Example builders for common node types

// LLMNodeBuilder creates an LLM node from a definition.
func LLMNodeBuilder(def *NodeDefinition) (*pocket.Node, error) {
	model, _ := def.Config["model"].(string)
	prompt, _ := def.Config["prompt"].(string)

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// This would integrate with an actual LLM
			return fmt.Sprintf("LLM response from %s for prompt: %s", model, prompt), nil
		}),
	), nil
}

// ValidatorNodeBuilder creates a validator node.
func ValidatorNodeBuilder(def *NodeDefinition) (*pocket.Node, error) {
	requiredFields, _ := def.Config["required_fields"].([]interface{})

	return pocket.NewNode[any, any](def.Name,
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			// Simple validation logic
			inputMap, ok := input.(map[string]interface{})
			if !ok {
				return nil, "invalid", fmt.Errorf("input must be a map")
			}

			for _, field := range requiredFields {
				if _, exists := inputMap[field.(string)]; !exists {
					return nil, "invalid", fmt.Errorf("missing required field: %s", field)
				}
			}

			return input, "valid", nil
		}),
	), nil
}
