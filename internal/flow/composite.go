// Package flow provides advanced flow composition utilities.
package flow

import (
	"context"
	"fmt"

	"github.com/agentstation/pocket"
)

// FlowNode wraps a Flow to make it usable as a Node within another Flow.
// This enables recursive composition where entire workflows can be treated
// as single nodes in larger workflows.
type FlowNode struct {
	name       string
	flow       *pocket.Flow
	inputKey   string
	outputKey  string
	successors map[string]*pocket.Node
}

// NewFlowNode creates a new FlowNode that wraps the given flow.
// The inputKey and outputKey specify where in the store to read input
// and write output, enabling proper state isolation.
func NewFlowNode(name string, flow *pocket.Flow, inputKey, outputKey string) *pocket.Node {
	fn := &FlowNode{
		name:       name,
		flow:       flow,
		inputKey:   inputKey,
		outputKey:  outputKey,
		successors: make(map[string]*pocket.Node),
	}

	// Create a pocket.Node with our lifecycle implementation
	// Using [any, any] since FlowNode needs to be flexible
	node := pocket.NewNode[any, any](name,
		pocket.WithPrep(fn.prep),
		pocket.WithExec(fn.exec),
		pocket.WithPost(fn.post),
	)

	return node
}

// prep reads the input from the store if specified.
func (fn *FlowNode) prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
	// If inputKey is specified, read from store instead of using passed input
	if fn.inputKey != "" {
		if val, exists := store.Get(ctx, fn.inputKey); exists {
			return val, nil
		}
		return nil, fmt.Errorf("input key %q not found in store", fn.inputKey)
	}
	return input, nil
}

// exec runs the wrapped flow with the prepared input.
func (fn *FlowNode) exec(ctx context.Context, input any) (any, error) {
	// Create isolated store for the flow if needed
	// For now, we'll use the same store but could scope it
	result, err := fn.flow.Run(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("flow %q failed: %w", fn.name, err)
	}
	return result, nil
}

// post writes the result to the store if specified and determines routing.
func (fn *FlowNode) post(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (output any, next string, err error) {
	// If outputKey is specified, write to store
	if fn.outputKey != "" {
		if err := store.Set(ctx, fn.outputKey, result); err != nil {
			return nil, "", fmt.Errorf("failed to store output: %w", err)
		}
	}

	// Default routing - could be enhanced to inspect result for action
	return result, "default", nil
}

// AsNode converts a Flow into a Node that can be used within another Flow.
// This is a convenience function that creates a FlowNode with default settings.
func AsNode(flow *pocket.Flow, name string) *pocket.Node {
	return NewFlowNode(name, flow, "", "")
}

// AsNodeWithStore converts a Flow into a Node with specific store keys for
// input and output, enabling better state isolation.
func AsNodeWithStore(flow *pocket.Flow, name, inputKey, outputKey string) *pocket.Node {
	return NewFlowNode(name, flow, inputKey, outputKey)
}

// ComposeFlows creates a new flow that executes multiple flows in sequence.
// Each flow is wrapped as a node and connected in order.
func ComposeFlows(name string, store pocket.Store, flows ...*pocket.Flow) (*pocket.Flow, error) {
	if len(flows) == 0 {
		return nil, fmt.Errorf("at least one flow must be provided")
	}

	// Create nodes from flows
	nodes := make([]*pocket.Node, len(flows))
	for i, flow := range flows {
		nodeName := fmt.Sprintf("%s-%d", name, i)
		nodes[i] = AsNode(flow, nodeName)
	}

	// Connect nodes in sequence
	for i := 0; i < len(nodes)-1; i++ {
		nodes[i].Connect("default", nodes[i+1])
	}

	// Create composite flow starting from first node
	return pocket.NewFlow(nodes[0], store), nil
}

// ParallelFlows creates a flow that executes multiple flows in parallel.
// Results are collected in a map keyed by flow index.
func ParallelFlows(ctx context.Context, store pocket.Store, flows ...*pocket.Flow) ([]any, error) {
	if len(flows) == 0 {
		return nil, fmt.Errorf("at least one flow must be provided")
	}

	// Create nodes from flows
	nodes := make([]*pocket.Node, len(flows))
	inputs := make([]any, len(flows))

	for i, flow := range flows {
		nodes[i] = AsNode(flow, fmt.Sprintf("parallel-%d", i))
		inputs[i] = nil // Could accept input array
	}

	// Use pocket's RunConcurrent to execute all flows in parallel
	results, err := pocket.RunConcurrent(ctx, nodes, store, inputs)
	if err != nil {
		return nil, fmt.Errorf("parallel execution failed: %w", err)
	}

	return results, nil
}

// NestedFlowBuilder provides a fluent API for building nested flow structures.
type NestedFlowBuilder struct {
	name   string
	store  pocket.Store
	nodes  []*pocket.Node
	start  *pocket.Node
	errors []error
}

// NewNestedFlowBuilder creates a new builder for nested flows.
func NewNestedFlowBuilder(name string, store pocket.Store) *NestedFlowBuilder {
	return &NestedFlowBuilder{
		name:  name,
		store: store,
		nodes: []*pocket.Node{},
	}
}

// AddFlow adds a flow as a node in the nested structure.
func (b *NestedFlowBuilder) AddFlow(name string, flow *pocket.Flow) *NestedFlowBuilder {
	node := AsNode(flow, name)
	b.nodes = append(b.nodes, node)
	if b.start == nil {
		b.start = node
	}
	return b
}

// AddFlowWithStore adds a flow with specific store keys.
func (b *NestedFlowBuilder) AddFlowWithStore(name string, flow *pocket.Flow, inputKey, outputKey string) *NestedFlowBuilder {
	node := AsNodeWithStore(flow, name, inputKey, outputKey)
	b.nodes = append(b.nodes, node)
	if b.start == nil {
		b.start = node
	}
	return b
}

// Connect connects two flows by name with a specific action.
func (b *NestedFlowBuilder) Connect(from, action, to string) *NestedFlowBuilder {
	var fromNode, toNode *pocket.Node

	for _, node := range b.nodes {
		if node.Name == from {
			fromNode = node
		}
		if node.Name == to {
			toNode = node
		}
	}

	if fromNode == nil {
		b.errors = append(b.errors, fmt.Errorf("node %q not found", from))
		return b
	}
	if toNode == nil {
		b.errors = append(b.errors, fmt.Errorf("node %q not found", to))
		return b
	}

	fromNode.Connect(action, toNode)
	return b
}

// Build creates the final nested flow.
func (b *NestedFlowBuilder) Build() (*pocket.Flow, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder errors: %v", b.errors)
	}
	if b.start == nil {
		return nil, fmt.Errorf("no nodes added to builder")
	}

	return pocket.NewFlow(b.start, b.store), nil
}
