package pocket

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Builder provides a fluent API for constructing flows.
type Builder struct {
	nodes map[string]*Node
	start *Node
	store Store
	opts  []FlowOption
}

// NewBuilder creates a new flow builder.
func NewBuilder(store Store) *Builder {
	return &Builder{
		nodes: make(map[string]*Node),
		store: store,
	}
}

// Add registers a node in the flow.
func (b *Builder) Add(node *Node) *Builder {
	b.nodes[node.Name] = node
	if b.start == nil {
		b.start = node
	}
	return b
}

// Start sets the starting node.
func (b *Builder) Start(name string) *Builder {
	if node, ok := b.nodes[name]; ok {
		b.start = node
	}
	return b
}

// Connect creates a connection between nodes.
func (b *Builder) Connect(from, action, to string) *Builder {
	fromNode, ok := b.nodes[from]
	if !ok {
		return b
	}

	toNode, ok := b.nodes[to]
	if !ok {
		return b
	}

	fromNode.Connect(action, toNode)
	return b
}

// WithOptions adds flow options.
func (b *Builder) WithOptions(opts ...FlowOption) *Builder {
	b.opts = append(b.opts, opts...)
	return b
}

// Build creates the flow.
func (b *Builder) Build() (*Flow, error) {
	if b.start == nil {
		return nil, ErrNoStartNode
	}

	return NewFlow(b.start, b.store, b.opts...), nil
}

// RunConcurrent executes multiple nodes concurrently.
func RunConcurrent(ctx context.Context, nodes []*Node, store Store, inputs []any) ([]any, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	// If inputs is nil or empty, create nil inputs for each node
	if len(inputs) == 0 {
		inputs = make([]any, len(nodes))
	}

	if len(inputs) != len(nodes) {
		return nil, fmt.Errorf("input count (%d) must match node count (%d)", len(inputs), len(nodes))
	}

	g, ctx := errgroup.WithContext(ctx)
	results := make([]any, len(nodes))
	mu := &sync.Mutex{}

	for i, node := range nodes {
		i, node := i, node // capture loop variables
		input := inputs[i]
		g.Go(func() error {
			// Each concurrent execution gets its own scoped store
			scopedStore := store.Scope(fmt.Sprintf("concurrent-%d", i))
			flow := NewFlow(node, scopedStore)
			result, err := flow.Run(ctx, input)
			if err != nil {
				return fmt.Errorf("node %s: %w", node.Name, err)
			}

			mu.Lock()
			results[i] = result
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// Pipeline executes nodes sequentially, passing output to input.
func Pipeline(ctx context.Context, nodes []*Node, store Store, input any) (any, error) {
	current := input

	for _, node := range nodes {
		flow := NewFlow(node, store)
		output, err := flow.Run(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("pipeline failed at %s: %w", node.Name, err)
		}
		current = output
	}

	return current, nil
}

// FanOut executes a node for each input item concurrently.
func FanOut[T any](ctx context.Context, node *Node, store Store, items []T) ([]any, error) {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]any, len(items))
	mu := &sync.Mutex{}

	for i, item := range items {
		i, item := i, item
		g.Go(func() error {
			// Each item gets its own scoped store
			scopedStore := store.Scope(fmt.Sprintf("item-%d", i))
			flow := NewFlow(node, scopedStore)
			result, err := flow.Run(ctx, item)
			if err != nil {
				return err
			}

			mu.Lock()
			results[i] = result
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// FanIn collects results from multiple sources.
type FanIn struct {
	sources []*Node
	combine func([]any) (any, error)
}

// NewFanIn creates a fan-in pattern.
func NewFanIn(combine func([]any) (any, error), sources ...*Node) *FanIn {
	return &FanIn{
		sources: sources,
		combine: combine,
	}
}

// Run executes the fan-in pattern.
func (f *FanIn) Run(ctx context.Context, store Store) (any, error) {
	// Create nil inputs for all sources
	inputs := make([]any, len(f.sources))
	results, err := RunConcurrent(ctx, f.sources, store, inputs)
	if err != nil {
		return nil, err
	}

	return f.combine(results)
}