// Package graph provides advanced graph composition utilities.
package graph

import (
	"context"
	"fmt"

	"github.com/agentstation/pocket"
)

// GraphNode wraps a Graph to make it usable as a Node within another Graph.
// This enables recursive composition where entire workflows can be treated
// as single nodes in larger workflows.
type GraphNode struct {
	name       string
	graph      *pocket.Graph
	inputKey   string
	outputKey  string
	successors map[string]pocket.Node
}

// NewGraphNode creates a new GraphNode that wraps the given graph.
// The inputKey and outputKey specify where in the store to read input
// and write output, enabling proper state isolation.
func NewGraphNode(name string, graph *pocket.Graph, inputKey, outputKey string) pocket.Node {
	fn := &GraphNode{
		name:       name,
		graph:      graph,
		inputKey:   inputKey,
		outputKey:  outputKey,
		successors: make(map[string]pocket.Node),
	}

	// Create a pocket.Node with our lifecycle implementation
	// Using [any, any] since GraphNode needs to be flexible
	node := pocket.NewNode[any, any](name, pocket.Steps{
		Prep: fn.prep,
		Exec: fn.exec,
		Post: fn.post,
	})

	return node
}

// prep reads the input from the store if specified.
func (fn *GraphNode) prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
	// If inputKey is specified, read from store instead of using passed input
	if fn.inputKey != "" {
		if val, exists := store.Get(ctx, fn.inputKey); exists {
			return val, nil
		}
		return nil, fmt.Errorf("input key %q not found in store", fn.inputKey)
	}
	return input, nil
}

// exec runs the wrapped graph with the prepared input.
func (fn *GraphNode) exec(ctx context.Context, input any) (any, error) {
	// Create isolated store for the graph if needed
	// For now, we'll use the same store but could scope it
	result, err := fn.graph.Run(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("graph %q failed: %w", fn.name, err)
	}
	return result, nil
}

// post writes the result to the store if specified and determines routing.
func (fn *GraphNode) post(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (output any, next string, err error) {
	// If outputKey is specified, write to store
	if fn.outputKey != "" {
		if err := store.Set(ctx, fn.outputKey, result); err != nil {
			return nil, "", fmt.Errorf("failed to store output: %w", err)
		}
	}

	// Default routing - could be enhanced to inspect result for action
	return result, "default", nil
}

// AsNodeWithStore converts a Graph into a Node with specific store keys for
// input and output, enabling better state isolation.
func AsNodeWithStore(graph *pocket.Graph, name, inputKey, outputKey string) pocket.Node {
	return NewGraphNode(name, graph, inputKey, outputKey)
}

// ComposeGraphs creates a new graph that executes multiple graphs in sequence.
// Each graph is wrapped as a node and connected in order.
func ComposeGraphs(name string, store pocket.Store, graphs ...*pocket.Graph) (*pocket.Graph, error) {
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

// ParallelGraphs creates a graph that executes multiple graphs in parallel.
// Results are collected in a map keyed by graph index.
func ParallelGraphs(ctx context.Context, store pocket.Store, graphs ...*pocket.Graph) ([]any, error) {
	if len(graphs) == 0 {
		return nil, fmt.Errorf("at least one graph must be provided")
	}

	// Create nodes from graphs
	nodes := make([]pocket.Node, len(graphs))
	inputs := make([]any, len(graphs))

	for i, graph := range graphs {
		nodes[i] = graph.AsNode(fmt.Sprintf("parallel-%d", i))
		inputs[i] = nil // Could accept input array
	}

	// Use pocket's RunConcurrent to execute all graphs in parallel
	results, err := pocket.RunConcurrent(ctx, nodes, store, inputs)
	if err != nil {
		return nil, fmt.Errorf("parallel execution failed: %w", err)
	}

	return results, nil
}

// NestedGraphBuilder provides a fluent API for building nested graph structures.
type NestedGraphBuilder struct {
	name       string
	store      pocket.Store
	nodes      []pocket.Node
	start      pocket.Node
	startIsSet bool
	errors     []error
}

// NewNestedGraphBuilder creates a new builder for nested graphs.
func NewNestedGraphBuilder(name string, store pocket.Store) *NestedGraphBuilder {
	return &NestedGraphBuilder{
		name:  name,
		store: store,
		nodes: []pocket.Node{},
	}
}

// AddGraph adds a graph as a node in the nested structure.
func (b *NestedGraphBuilder) AddGraph(name string, graph *pocket.Graph) *NestedGraphBuilder {
	node := graph.AsNode(name)
	b.nodes = append(b.nodes, node)
	if !b.startIsSet {
		b.start = node
		b.startIsSet = true
	}
	return b
}

// AddGraphWithStore adds a graph with specific store keys.
func (b *NestedGraphBuilder) AddGraphWithStore(name string, graph *pocket.Graph, inputKey, outputKey string) *NestedGraphBuilder {
	node := AsNodeWithStore(graph, name, inputKey, outputKey)
	b.nodes = append(b.nodes, node)
	if !b.startIsSet {
		b.start = node
		b.startIsSet = true
	}
	return b
}

// Connect connects two graphs by name with a specific action.
func (b *NestedGraphBuilder) Connect(from, action, to string) *NestedGraphBuilder {
	var fromNode, toNode pocket.Node
	var fromFound, toFound bool

	for _, node := range b.nodes {
		if node.Name() == from {
			fromNode = node
			fromFound = true
		}
		if node.Name() == to {
			toNode = node
			toFound = true
		}
	}

	if !fromFound {
		b.errors = append(b.errors, fmt.Errorf("node %q not found", from))
		return b
	}
	if !toFound {
		b.errors = append(b.errors, fmt.Errorf("node %q not found", to))
		return b
	}

	fromNode.Connect(action, toNode)
	return b
}

// Build creates the final nested graph.
func (b *NestedGraphBuilder) Build() (*pocket.Graph, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder errors: %v", b.errors)
	}
	if !b.startIsSet {
		return nil, fmt.Errorf("no nodes added to builder")
	}

	return pocket.NewGraph(b.start, b.store), nil
}
