// Package compose provides advanced graph composition utilities for building
// complex workflows from simpler sub-graphs.
package compose

import (
	"context"
	"fmt"

	"github.com/agentstation/pocket"
)

// AsNodeWithStore converts a Graph into a Node with specific store keys for
// input and output, enabling proper state isolation between composed graphs.
func AsNodeWithStore(graph *pocket.Graph, name, inputKey, outputKey string) pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// If inputKey is specified, read from store instead of using passed input
			if inputKey != "" {
				if val, exists := store.Get(ctx, inputKey); exists {
					return val, nil
				}
				return nil, fmt.Errorf("input key %q not found in store", inputKey)
			}
			return input, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepResult any) (any, error) {
			// Run the graph with the prepared input
			result, err := graph.Run(ctx, prepResult)
			if err != nil {
				return nil, fmt.Errorf("graph %q failed: %w", name, err)
			}
			return result, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			// If outputKey is specified, write to store
			if outputKey != "" {
				if err := store.Set(ctx, outputKey, result); err != nil {
					return nil, "", fmt.Errorf("failed to store output: %w", err)
				}
			}
			return result, "default", nil
		}),
	)
}

// SequentialGraphs creates a new graph that executes multiple graphs in sequence.
// Each graph is wrapped as a node and connected in order.
func SequentialGraphs(name string, store pocket.Store, graphs ...*pocket.Graph) (*pocket.Graph, error) {
	if len(graphs) == 0 {
		return nil, fmt.Errorf("at least one graph must be provided")
	}

	// Create nodes from graphs
	nodes := make([]pocket.Node, len(graphs))
	for i, graph := range graphs {
		nodeName := fmt.Sprintf("%s-%d", name, i)
		nodes[i] = graph.AsNode(nodeName)
	}

	// Connect nodes in sequence
	for i := 0; i < len(nodes)-1; i++ {
		nodes[i].Connect("default", nodes[i+1])
	}

	// Create composite graph starting from first node
	return pocket.NewGraph(nodes[0], store), nil
}

// ParallelGraphs executes multiple graphs in parallel and returns their results.
// Results are returned in the same order as the input graphs.
func ParallelGraphs(ctx context.Context, store pocket.Store, graphs ...*pocket.Graph) ([]any, error) {
	if len(graphs) == 0 {
		return nil, fmt.Errorf("at least one graph must be provided")
	}

	// Create nodes from graphs
	nodes := make([]pocket.Node, len(graphs))
	inputs := make([]any, len(graphs))

	for i, graph := range graphs {
		nodes[i] = graph.AsNode(fmt.Sprintf("parallel-%d", i))
		inputs[i] = nil // Could accept input array in future
	}

	// Use pocket's RunConcurrent to execute all graphs in parallel
	results, err := pocket.RunConcurrent(ctx, nodes, store, inputs)
	if err != nil {
		return nil, fmt.Errorf("parallel execution failed: %w", err)
	}

	return results, nil
}

// Builder provides a fluent API for building nested graph structures.
type Builder struct {
	name   string
	store  pocket.Store
	nodes  []pocket.Node
	start  pocket.Node
	errors []error
}

// NewBuilder creates a new builder for composing graphs.
func NewBuilder(name string, store pocket.Store) *Builder {
	return &Builder{
		name:  name,
		store: store,
		nodes: []pocket.Node{},
	}
}

// AddGraph adds a graph as a node in the composition.
func (b *Builder) AddGraph(name string, graph *pocket.Graph) *Builder {
	node := graph.AsNode(name)
	b.nodes = append(b.nodes, node)
	if b.start == nil {
		b.start = node
	}
	return b
}

// AddGraphWithStore adds a graph with specific store keys for input/output isolation.
func (b *Builder) AddGraphWithStore(name string, graph *pocket.Graph, inputKey, outputKey string) *Builder {
	node := AsNodeWithStore(graph, name, inputKey, outputKey)
	b.nodes = append(b.nodes, node)
	if b.start == nil {
		b.start = node
	}
	return b
}

// Connect connects two graphs by name with a specific action.
func (b *Builder) Connect(from, action, to string) *Builder {
	var fromNode, toNode pocket.Node

	for _, node := range b.nodes {
		if node.Name() == from {
			fromNode = node
		}
		if node.Name() == to {
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

// Build creates the final composed graph.
func (b *Builder) Build() (*pocket.Graph, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder errors: %v", b.errors)
	}
	if b.start == nil {
		return nil, fmt.Errorf("no nodes added to builder")
	}

	return pocket.NewGraph(b.start, b.store), nil
}
