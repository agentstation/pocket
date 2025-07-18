// Package pocket provides a minimalist framework for building LLM workflows
// using composable nodes in a directed graph structure.
package pocket

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Common errors.
var (
	// ErrNoStartNode is returned when a flow has no start node defined.
	ErrNoStartNode = errors.New("pocket: no start node defined")

	// ErrNodeNotFound is returned when a referenced node doesn't exist.
	ErrNodeNotFound = errors.New("pocket: node not found")

	// ErrInvalidInput is returned when input type doesn't match expected type.
	ErrInvalidInput = errors.New("pocket: invalid input type")
)

// Processor handles the main execution logic of a node.
type Processor interface {
	// Process executes the node's main logic.
	Process(ctx context.Context, input any) (output any, err error)
}

// ProcessorFunc is an adapter to allow ordinary functions to be used as Processors.
type ProcessorFunc func(ctx context.Context, input any) (any, error)

// Process calls f(ctx, input).
func (f ProcessorFunc) Process(ctx context.Context, input any) (any, error) {
	return f(ctx, input)
}

// Stateful manages state interaction with a Store.
type Stateful interface {
	// LoadState retrieves state from the store before processing.
	LoadState(ctx context.Context, store Store) (state any, err error)

	// SaveState persists state to the store after processing.
	SaveState(ctx context.Context, store Store, state any) error
}

// Router determines the next node based on processing results.
type Router interface {
	// Route returns the name of the next node to execute.
	Route(ctx context.Context, result any) (next string, err error)
}

// RouterFunc is an adapter for router functions.
type RouterFunc func(ctx context.Context, result any) (string, error)

// Route calls f(ctx, result).
func (f RouterFunc) Route(ctx context.Context, result any) (string, error) {
	return f(ctx, result)
}

// Store provides thread-safe storage for shared state.
type Store interface {
	// Get retrieves a value by key.
	Get(key string) (value any, exists bool)

	// Set stores a value with the given key.
	Set(key string, value any)

	// Delete removes a key from the store.
	Delete(key string)
}

// Node represents a processing unit in a workflow.
// It combines processing, state management, and routing.
type Node struct {
	// Name identifies the node in the flow.
	Name string

	// Processor handles the main logic.
	Processor

	// Optional state management.
	Stateful

	// Optional routing logic.
	Router

	// Successors maps action names to next nodes.
	successors map[string]*Node

	// Options
	opts nodeOptions
}

// nodeOptions holds configuration for a Node.
type nodeOptions struct {
	maxRetries int
	retryDelay time.Duration
	timeout    time.Duration
	onError    func(error)
}

// Option configures a Node.
type Option func(*nodeOptions)

// WithRetry configures retry behavior.
func WithRetry(maxRetries int, delay time.Duration) Option {
	return func(o *nodeOptions) {
		o.maxRetries = maxRetries
		o.retryDelay = delay
	}
}

// WithTimeout sets execution timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *nodeOptions) {
		o.timeout = timeout
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(handler func(error)) Option {
	return func(o *nodeOptions) {
		o.onError = handler
	}
}

// NewNode creates a new node with the given processor.
func NewNode(name string, processor Processor, opts ...Option) *Node {
	n := &Node{
		Name:       name,
		Processor:  processor,
		successors: make(map[string]*Node),
	}

	// Apply options
	for _, opt := range opts {
		opt(&n.opts)
	}

	// Set defaults
	if n.opts.retryDelay == 0 {
		n.opts.retryDelay = 100 * time.Millisecond
	}

	return n
}

// Connect adds a successor node for the given action.
func (n *Node) Connect(action string, next *Node) *Node {
	n.successors[action] = next
	return n
}

// Default connects to the default next node.
func (n *Node) Default(next *Node) *Node {
	return n.Connect("default", next)
}

// Flow orchestrates the execution of connected nodes.
type Flow struct {
	start *Node
	store Store
	opts  flowOptions
}

// flowOptions holds configuration for a Flow.
type flowOptions struct {
	logger Logger
	tracer Tracer
}

// FlowOption configures a Flow.
type FlowOption func(*flowOptions)

// WithLogger adds logging to the flow.
func WithLogger(logger Logger) FlowOption {
	return func(o *flowOptions) {
		o.logger = logger
	}
}

// WithTracer adds distributed tracing.
func WithTracer(tracer Tracer) FlowOption {
	return func(o *flowOptions) {
		o.tracer = tracer
	}
}

// NewFlow creates a new flow starting from the given node.
func NewFlow(start *Node, store Store, opts ...FlowOption) *Flow {
	f := &Flow{
		start: start,
		store: store,
	}

	for _, opt := range opts {
		opt(&f.opts)
	}

	return f
}

// Run executes the flow with the given input.
func (f *Flow) Run(ctx context.Context, input any) (output any, err error) {
	if f.start == nil {
		return nil, ErrNoStartNode
	}

	current := f.start
	currentInput := input

	for current != nil {
		// Log node execution
		if f.opts.logger != nil {
			f.opts.logger.Debug(ctx, "executing node", "name", current.Name)
		}

		// Execute node
		output, err = f.executeNode(ctx, current, currentInput)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", current.Name, err)
		}

		// Determine next node
		next := "default"
		if current.Router != nil {
			next, err = current.Route(ctx, output)
			if err != nil {
				return nil, fmt.Errorf("routing failed in node %s: %w", current.Name, err)
			}
		}

		// Move to next node
		current = current.successors[next]
		currentInput = output
	}

	return output, nil
}

// executeNode runs a single node with all its lifecycle phases.
func (f *Flow) executeNode(ctx context.Context, node *Node, input any) (any, error) {
	// Apply timeout if configured
	if node.opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, node.opts.timeout)
		defer cancel()
	}

	// Load state if stateful
	var state any
	if node.Stateful != nil {
		var err error
		state, err = node.LoadState(ctx, f.store)
		if err != nil {
			return nil, fmt.Errorf("load state: %w", err)
		}

		// Use state as input if no input provided
		if input == nil {
			input = state
		}
	}

	// Execute with retry
	output, err := f.executeWithRetry(ctx, node, input)
	if err != nil {
		if node.opts.onError != nil {
			node.opts.onError(err)
		}
		return nil, err
	}

	// Save state if stateful
	if node.Stateful != nil {
		if err := node.SaveState(ctx, f.store, output); err != nil {
			return nil, fmt.Errorf("save state: %w", err)
		}
	}

	return output, nil
}

// executeWithRetry handles retry logic for node execution.
func (f *Flow) executeWithRetry(ctx context.Context, node *Node, input any) (any, error) {
	attempts := 0
	maxAttempts := node.opts.maxRetries + 1

	for attempts < maxAttempts {
		if attempts > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(node.opts.retryDelay):
			}
		}

		output, err := node.Process(ctx, input)
		if err == nil {
			return output, nil
		}

		attempts++
		if attempts < maxAttempts {
			if f.opts.logger != nil {
				f.opts.logger.Debug(ctx, "retrying node",
					"name", node.Name,
					"attempt", attempts,
					"error", err)
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts", attempts)
}

// Logger provides structured logging.
type Logger interface {
	Debug(ctx context.Context, msg string, keysAndValues ...any)
	Info(ctx context.Context, msg string, keysAndValues ...any)
	Error(ctx context.Context, msg string, keysAndValues ...any)
}

// Tracer provides distributed tracing capabilities.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, func())
}
