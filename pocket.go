// Package pocket provides a minimalist framework for building LLM workflows
// using composable nodes in a directed graph structure with Prep/Exec/Post lifecycle.
//
// Type Safety:
// The framework provides three levels of type safety for workflow validation:
//
//   - Compile-time: Generic node creation with NewNode[In, Out] enforces type
//     consistency within nodes. The Go compiler checks function signatures when
//     using the generic WithExec, WithPrep, etc. with typed nodes.
//
//   - Initialization-time: ValidateFlow checks type compatibility across the
//     entire workflow graph before execution begins. This catches type mismatches
//     between connected nodes.
//
//   - Runtime: Type assertions in lifecycle functions ensure data integrity
//     during execution. These are minimized when using typed nodes.
//
// The goal is to verify type safety of the workflow graph as early as possible,
// catching errors before any workflow execution begins.
package pocket

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// Common errors.
var (
	// ErrNoStartNode is returned when a graph has no start node defined.
	ErrNoStartNode = errors.New("pocket: no start node defined")

	// ErrNodeNotFound is returned when a referenced node doesn't exist.
	ErrNodeNotFound = errors.New("pocket: node not found")

	// ErrInvalidInput is returned when input type doesn't match expected type.
	ErrInvalidInput = errors.New("pocket: invalid input type")
)

// PrepFunc prepares data before execution with read-only store access.
type PrepFunc func(ctx context.Context, store StoreReader, input any) (prepResult any, err error)

// ExecFunc performs the main processing logic without store access.
type ExecFunc func(ctx context.Context, prepResult any) (execResult any, err error)

// PostFunc processes results and determines routing with full store access.
type PostFunc func(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error)

// FallbackFunc handles errors from the Exec step using the prepared data.
// It receives the prepResult (like Exec) and the error from the failed Exec.
// Like ExecFunc, it has no store access to maintain purity.
type FallbackFunc func(ctx context.Context, prepResult any, execErr error) (fallbackResult any, err error)

// Steps groups the lifecycle functions for a node.
// All fields are optional - if not provided, default implementations will be used.
type Steps struct {
	// Prep prepares data before execution with read-only store access.
	Prep PrepFunc

	// Exec performs the main processing logic without store access.
	Exec ExecFunc

	// Fallback handles Exec errors with the prepared data.
	// Like Exec, it receives prepResult and has no store access.
	Fallback FallbackFunc

	// Post processes results and determines routing with full store access.
	Post PostFunc
}

// StoreReader provides read-only access to the store.
// Used in the Prep step to enforce read-only semantics.
type StoreReader interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) (value any, exists bool)

	// Scope returns a new store with the given prefix.
	Scope(prefix string) Store
}

// StoreWriter provides full read-write access to the store.
// Used in the Post step for state mutations.
type StoreWriter interface {
	Store
}

// Store provides thread-safe storage for shared state.
type Store interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) (value any, exists bool)

	// Set stores a value with the given key.
	Set(ctx context.Context, key string, value any) error

	// Delete removes a key from the store.
	Delete(ctx context.Context, key string) error

	// Scope returns a new store with the given prefix.
	Scope(prefix string) Store
}

// Node is the core interface for all execution units in a workflow.
// Both simple nodes and graphs implement this interface.
type Node interface {
	// Name returns the node's identifier.
	Name() string

	// Lifecycle methods for the Prep/Exec/Post pattern.
	Prep(ctx context.Context, store StoreReader, input any) (prepResult any, err error)
	Exec(ctx context.Context, prepResult any) (execResult any, err error)
	Post(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error)

	// Connect adds a successor node for the given action.
	Connect(action string, next Node) Node

	// Successors returns all connected nodes.
	Successors() map[string]Node

	// Type information for validation (optional).
	InputType() reflect.Type
	OutputType() reflect.Type
}

// node is the private implementation of Node for simple execution units.
type node struct {
	// Name identifies the node in the graph.
	name string

	// Lifecycle methods (never nil - have defaults).
	prep PrepFunc
	exec ExecFunc
	post PostFunc

	// Type information for validation (optional).
	inputType  reflect.Type
	outputType reflect.Type

	// Successors maps action names to next nodes.
	successors map[string]Node

	// Options
	opts nodeOptions
}

// nodeOptions holds configuration for a Node.
type nodeOptions struct {
	// Lifecycle functions
	prep PrepFunc
	exec ExecFunc
	post PostFunc

	// Retry and timeout
	maxRetries int
	retryDelay time.Duration
	timeout    time.Duration

	// Error handling
	onError  func(error)
	fallback func(ctx context.Context, prepResult any, err error) (any, error)

	// Cleanup hooks
	onSuccess  func(ctx context.Context, store StoreWriter, output any)
	onFailure  func(ctx context.Context, store StoreWriter, err error)
	onComplete func(ctx context.Context, store StoreWriter)
}

// Option configures a Node.
type Option func(*nodeOptions)

// WithPrep sets the preparation function with type safety.
// The input type In should match the node's input type when used with NewNode[In, Out].
// For dynamic typing, use WithPrep[any].
// The store parameter provides read-only access to enforce Prep step semantics.
func WithPrep[In any](fn func(ctx context.Context, store StoreReader, input In) (any, error)) Option {
	return func(o *nodeOptions) {
		o.prep = func(ctx context.Context, store StoreReader, input any) (any, error) {
			// Handle nil input specially
			if input == nil {
				// Call with zero value of In
				return fn(ctx, store, *new(In))
			}

			// Type assertion with safety check
			typedInput, ok := input.(In)
			if !ok {
				return nil, fmt.Errorf("%w: prep expected %T, got %T", ErrInvalidInput, *new(In), input)
			}
			return fn(ctx, store, typedInput)
		}
	}
}

// WithExec sets the execution function with type safety.
// The types In and Out should match the node's types when used with NewNode[In, Out].
// For dynamic typing, use WithExec[any, any].
// Exec functions do not have store access to enforce pure business logic.
func WithExec[In, Out any](fn func(ctx context.Context, input In) (Out, error)) Option {
	return func(o *nodeOptions) {
		o.exec = func(ctx context.Context, prepResult any) (any, error) {
			// Handle nil input specially
			if prepResult == nil {
				// Call with zero value of In
				result, err := fn(ctx, *new(In))
				if err != nil {
					return nil, err
				}
				return result, nil
			}

			// Convert prep result to expected input type
			typedInput, ok := prepResult.(In)
			if !ok {
				return nil, fmt.Errorf("%w: exec expected %T, got %T", ErrInvalidInput, *new(In), prepResult)
			}
			// Execute with type safety
			result, err := fn(ctx, typedInput)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}
}

// WithPost sets the post-processing function with type safety.
// The types In and Out should match the node's types when used with NewNode[In, Out].
// Post functions have access to all step results and determine routing.
// For dynamic typing, use WithPost[any, any].
// The store parameter provides full read-write access for state mutations.
func WithPost[In, Out any](fn func(ctx context.Context, store StoreWriter, input In, prepResult any, execResult Out) (Out, string, error)) Option {
	return func(o *nodeOptions) {
		o.post = func(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (any, string, error) {
			// Handle nil inputs specially
			var typedInput In
			if input == nil {
				typedInput = *new(In)
			} else {
				var ok bool
				typedInput, ok = input.(In)
				if !ok {
					return nil, "", fmt.Errorf("%w: post expected input %T, got %T", ErrInvalidInput, *new(In), input)
				}
			}

			var typedExecResult Out
			if execResult == nil {
				typedExecResult = *new(Out)
			} else {
				var ok bool
				typedExecResult, ok = execResult.(Out)
				if !ok {
					return nil, "", fmt.Errorf("%w: post expected exec result %T, got %T", ErrInvalidInput, *new(Out), execResult)
				}
			}

			// Execute with type safety
			output, next, err := fn(ctx, store, typedInput, prepResult, typedExecResult)
			if err != nil {
				return nil, "", err
			}
			return output, next, nil
		}
	}
}

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

// WithOnSuccess sets a cleanup hook that runs after successful execution.
// The type Out should match the node's output type when used with NewNode[In, Out].
// For dynamic typing, use WithOnSuccess[any].
// The store parameter provides full read-write access for cleanup operations.
func WithOnSuccess[Out any](fn func(ctx context.Context, store StoreWriter, output Out)) Option {
	return func(o *nodeOptions) {
		o.onSuccess = func(ctx context.Context, store StoreWriter, output any) {
			// Type assertion for output
			typedOutput, ok := output.(Out)
			if !ok {
				// Log error but don't fail - this is a cleanup hook
				return
			}
			fn(ctx, store, typedOutput)
		}
	}
}

// WithOnFailure sets a cleanup hook that runs after failed execution.
// The store parameter provides full read-write access for cleanup operations.
func WithOnFailure(fn func(ctx context.Context, store StoreWriter, err error)) Option {
	return func(o *nodeOptions) {
		o.onFailure = fn
	}
}

// WithOnComplete sets a cleanup hook that always runs after execution.
// The store parameter provides full read-write access for cleanup operations.
func WithOnComplete(fn func(ctx context.Context, store StoreWriter)) Option {
	return func(o *nodeOptions) {
		o.onComplete = fn
	}
}

// Implementation of Node interface for node struct

// Name returns the node's identifier.
func (n *node) Name() string {
	return n.name
}

// Prep implements the preparation phase of the node lifecycle.
func (n *node) Prep(ctx context.Context, store StoreReader, input any) (any, error) {
	if n.prep != nil {
		return n.prep(ctx, store, input)
	}
	return defaultPrep(ctx, store, input)
}

// Exec implements the execution phase of the node lifecycle.
func (n *node) Exec(ctx context.Context, prepResult any) (any, error) {
	if n.exec != nil {
		return n.exec(ctx, prepResult)
	}
	return defaultExec(ctx, prepResult)
}

// Post implements the post-processing phase of the node lifecycle.
func (n *node) Post(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error) {
	if n.post != nil {
		return n.post(ctx, store, input, prepResult, execResult)
	}
	return defaultPost(ctx, store, input, prepResult, execResult)
}

// Connect adds a successor node for the given action.
func (n *node) Connect(action string, next Node) Node {
	n.successors[action] = next
	return n
}

// Successors returns all connected nodes.
func (n *node) Successors() map[string]Node {
	return n.successors
}

// InputType returns the expected input type for validation.
func (n *node) InputType() reflect.Type {
	return n.inputType
}

// OutputType returns the expected output type for validation.
func (n *node) OutputType() reflect.Type {
	return n.outputType
}

// Default implementations for lifecycle methods.
func defaultPrep(ctx context.Context, store StoreReader, input any) (any, error) {
	return input, nil // pass through
}

func defaultExec(ctx context.Context, prepResult any) (any, error) {
	return prepResult, nil // pass through
}

func defaultPost(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error) {
	return execResult, "default", nil
}

// isAnyType checks if a reflect.Type represents the 'any' interface.
// This is used to determine whether a node should enforce type checking.
// Returns true if the type is interface{} (any), false otherwise.
func isAnyType(t reflect.Type) bool {
	if t == nil {
		return true
	}
	// Check if it's an interface with no methods (i.e., 'any' or interface{})
	return t.Kind() == reflect.Interface && t.NumMethod() == 0
}

// newNodeBase creates a basic node without type parameters.
// This is an internal helper used by the new generic NewNode function.
func newNodeBase(name string, opts ...Option) *node {
	// Get global defaults
	defaultPrep, defaultExec, defaultPost, defaultOpts := getDefaults()

	// Create node with defaults
	n := &node{
		name:       name,
		prep:       defaultPrep,
		exec:       defaultExec,
		post:       defaultPost,
		successors: make(map[string]Node),
		opts:       defaultOpts,
	}

	// Apply lifecycle functions from defaults if set
	if defaultOpts.prep != nil {
		n.prep = defaultOpts.prep
	}
	if defaultOpts.exec != nil {
		n.exec = defaultOpts.exec
	}
	if defaultOpts.post != nil {
		n.post = defaultOpts.post
	}

	// Apply provided options (override defaults)
	for _, opt := range opts {
		opt(&n.opts)
	}

	// Apply lifecycle functions from options
	if n.opts.prep != nil {
		n.prep = n.opts.prep
	}
	if n.opts.exec != nil {
		n.exec = n.opts.exec
	}
	if n.opts.post != nil {
		n.post = n.opts.post
	}

	return n
}

// NewNode creates a new node with optional compile-time type safety.
//
// Type parameters:
//   - In: The expected input type for this node (use 'any' for dynamic typing)
//   - Out: The output type this node produces (use 'any' for dynamic typing)
//
// Parameters:
//   - name: The node's identifier
//   - steps: The lifecycle functions (Prep, Exec, Post) - all fields are optional
//   - opts: Additional options like retry, timeout, error handlers, etc.
//
// Type safety mechanism:
//
//  1. Compile-time: When In/Out are not 'any', the node stores type information
//     for validation. Using generic options like WithExec, WithPrep ensures function
//     signatures match the declared types at compile time.
//
//  2. Initialization-time: Call ValidateFlow on your start node to verify the
//     entire workflow graph has compatible types between connected nodes.
//
//  3. Runtime: When using regular options (WithExec, WithPrep) with typed nodes,
//     the framework automatically wraps functions to ensure type safety.
//
// Examples:
//
//	// Typed node - enables full type checking across the workflow
//	validator := NewNode[User, ValidationResult]("validator",
//	    Steps{
//	        Exec: func(ctx context.Context, user any) (any, error) {
//	            // Type assertions handled by the framework
//	            return ValidationResult{Valid: true}, nil
//	        },
//	    },
//	    WithRetry(3, time.Second),
//	)
//
//	// Untyped node - no compile-time checks (explicit [any, any] encourages adding types)
//	processor := NewNode[any, any]("processor",
//	    Steps{
//	        Prep: prepFunc,
//	        Exec: execFunc,
//	        Post: postFunc,
//	    },
//	)
func NewNode[In, Out any](name string, steps Steps, opts ...Option) Node {
	// Apply lifecycle functions from Steps as options first
	allOpts := make([]Option, 0, len(opts)+3)

	// Add Steps functions as options if they're provided
	if steps.Prep != nil {
		allOpts = append(allOpts, func(o *nodeOptions) {
			o.prep = steps.Prep
		})
	}
	if steps.Exec != nil {
		allOpts = append(allOpts, func(o *nodeOptions) {
			o.exec = steps.Exec
		})
	}
	if steps.Post != nil {
		allOpts = append(allOpts, func(o *nodeOptions) {
			o.post = steps.Post
		})
	}
	if steps.Fallback != nil {
		allOpts = append(allOpts, func(o *nodeOptions) {
			o.fallback = func(ctx context.Context, prepResult any, err error) (any, error) {
				return steps.Fallback(ctx, prepResult, err)
			}
		})
	}

	// Add any additional options
	allOpts = append(allOpts, opts...)

	// Create base node using existing logic
	n := newNodeBase(name, allOpts...)

	// Determine if types are specified (not 'any')
	inType := reflect.TypeOf((*In)(nil)).Elem()
	outType := reflect.TypeOf((*Out)(nil)).Elem()

	// Set type information on node if types are not 'any'
	// This enables ValidateFlow to check type compatibility between nodes
	if !isAnyType(inType) {
		n.inputType = inType
	}
	if !isAnyType(outType) {
		n.outputType = outType
	}

	return n
}

// Default is a helper function to connect to the default next node.
func Default(n, next Node) Node {
	return n.Connect("default", next)
}

// graph is the private implementation of Node for composite execution.
type graph struct {
	name       string
	start      Node
	store      Store
	successors map[string]Node
	opts       graphOptions
}

// Graph is the public handle to a graph for backward compatibility.
type Graph struct {
	*graph // embed private graph
}

// graphOptions holds configuration for a Graph.
type graphOptions struct {
	logger Logger
	tracer Tracer
}

// GraphOption configures a Graph.
type GraphOption func(*graphOptions)

// WithLogger adds logging to the graph.
func WithLogger(logger Logger) GraphOption {
	return func(o *graphOptions) {
		o.logger = logger
	}
}

// WithTracer adds distributed tracing.
func WithTracer(tracer Tracer) GraphOption {
	return func(o *graphOptions) {
		o.tracer = tracer
	}
}

// Implementation of Node interface for graph struct

// Name returns the graph's identifier.
func (g *graph) Name() string {
	return g.name
}

// Prep for a graph doesn't need to do anything special.
func (g *graph) Prep(ctx context.Context, store StoreReader, input any) (any, error) {
	// Graphs use their own internal store, so just pass through the input
	return input, nil
}

// Exec runs the graph workflow.
func (g *graph) Exec(ctx context.Context, input any) (any, error) {
	// Create a new Graph wrapper to use existing Run logic
	wrapper := &Graph{graph: g}
	return wrapper.Run(ctx, input)
}

// Post handles the graph execution results.
func (g *graph) Post(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (output any, next string, err error) {
	// Return the result and default routing
	return execResult, "default", nil
}

// Connect adds a successor node for when the graph is used as a node.
func (g *graph) Connect(action string, next Node) Node {
	if g.successors == nil {
		g.successors = make(map[string]Node)
	}
	g.successors[action] = next
	return g
}

// Successors returns all connected nodes.
func (g *graph) Successors() map[string]Node {
	return g.successors
}

// InputType returns nil for graphs (dynamic typing).
func (g *graph) InputType() reflect.Type {
	return nil
}

// OutputType returns nil for graphs (dynamic typing).
func (g *graph) OutputType() reflect.Type {
	return nil
}

// NewGraph creates a new graph starting from the given node.
func NewGraph(start Node, store Store, opts ...GraphOption) *Graph {
	name := "graph"
	if start != nil {
		name = "graph-" + start.Name()
	}

	g := &graph{
		name:       name,
		start:      start,
		store:      store,
		successors: make(map[string]Node),
	}

	for _, opt := range opts {
		opt(&g.opts)
	}

	return &Graph{graph: g}
}

// ValidateGraph provides initialization-time type safety by validating the entire workflow graph.
//
// Type validation process:
//  1. Traverses the graph starting from the given node using depth-first search
//  2. For each connection between nodes, verifies type compatibility:
//     - Source node's OutputType must be assignable to target node's InputType
//     - Interface satisfaction is checked (e.g., concrete type implements interface)
//     - Untyped nodes (InputType/OutputType = nil) are skipped but successors are validated
//  3. Returns detailed error messages identifying the exact type mismatch location
//
// This is a critical part of the type safety system, catching errors before any
// workflow execution begins. It complements compile-time checks by validating
// the connections between nodes.
//
// Type compatibility rules:
//   - Exact type match: string -> string ✓
//   - Interface satisfaction: ConcreteType -> Interface (if implements) ✓
//   - Any type: any -> ConcreteType ✓ (but loses compile-time safety)
//   - Assignability: Uses Go's reflect.Type.AssignableTo for compatibility
//
// Example:
//
//	// Build your workflow
//	validator := NewNode[User, ValidationResult]("validator", ...)
//	processor := NewNode[ValidationResult, Response]("processor", ...)
//	validator.Connect("valid", processor)
//
//	// Validate before execution - catches type mismatches early
//	if err := ValidateGraph(validator); err != nil {
//	    // Error: "type mismatch: node 'validator' outputs ValidationResult
//	    //         but node 'wrongNode' expects User (via action 'valid')"
//	    log.Fatal(err)
//	}
//
//	// Safe to execute - types are verified
//	graph := NewGraph(validator, store)
//	result, err := graph.Run(ctx, user)
func ValidateGraph(start Node) error {
	visited := make(map[string]bool)
	return validateNode(start, visited)
}

func validateNode(node Node, visited map[string]bool) error {
	if node == nil || visited[node.Name()] {
		return nil
	}
	visited[node.Name()] = true

	// If this node has no type information, skip validation
	if node.OutputType() == nil {
		// Still validate successors
		for _, successor := range node.Successors() {
			if err := validateNode(successor, visited); err != nil {
				return err
			}
		}
		return nil
	}

	// Check each successor
	for action, successor := range node.Successors() {
		if successor.InputType() != nil {
			// Both types are specified, check compatibility
			if !isTypeCompatible(node.OutputType(), successor.InputType()) {
				return fmt.Errorf("type mismatch: node %q outputs %v but node %q expects %v (via action %q)",
					node.Name(), node.OutputType(), successor.Name(), successor.InputType(), action)
			}
		}

		// Recursively validate successor
		if err := validateNode(successor, visited); err != nil {
			return err
		}
	}

	return nil
}

// isTypeCompatible checks if output type can be used as input type.
// This handles interface satisfaction and type identity.
func isTypeCompatible(outputType, inputType reflect.Type) bool {
	// Exact match
	if outputType == inputType {
		return true
	}

	// Check if output implements input (if input is an interface)
	if inputType.Kind() == reflect.Interface {
		return outputType.Implements(inputType)
	}

	// Check if both are interfaces and output is broader
	if outputType.Kind() == reflect.Interface && inputType.Kind() == reflect.Interface {
		// This is a simplified check - in practice you might want more sophisticated logic
		return true
	}

	// Check if output is assignable to input
	return outputType.AssignableTo(inputType)
}

// Run executes the graph with the given input.
func (g *Graph) Run(ctx context.Context, input any) (output any, err error) {
	if g.start == nil {
		return nil, ErrNoStartNode
	}

	current := g.start
	currentInput := input
	var lastOutput any

	for current != nil {
		// Log node execution
		if g.opts.logger != nil {
			g.opts.logger.Debug(ctx, "executing node", "name", current.Name())
		}

		// Execute node with lifecycle
		output, next, err := g.executeNode(ctx, current, currentInput)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", current.Name(), err)
		}

		// Save the output
		lastOutput = output

		// Move to next node
		successors := current.Successors()
		current = successors[next]
		currentInput = output
	}

	return lastOutput, nil
}

// executeNode runs a single node with runtime type safety checks at each lifecycle step.
//
// Runtime type safety:
//   - Input validation: If node has InputType set, verifies input matches before execution
//   - Wrapped functions: Type assertions in wrapped lifecycle functions ensure correctness
//   - Error propagation: Type mismatches result in clear error messages with actual vs expected types
//
// This is where runtime type checking occurs, complementing compile-time and init-time checks.
// For typed nodes using generic options like WithExec, type assertions are handled
// automatically through Go's type inference.
func (g *Graph) executeNode(ctx context.Context, n Node, input any) (output any, next string, err error) {
	// Runtime type check: Validate input matches node's expected type
	// This catches any type mismatches that slipped through earlier checks
	if n.InputType() != nil && input != nil {
		inputType := reflect.TypeOf(input)
		if !isTypeCompatible(inputType, n.InputType()) {
			return nil, "", fmt.Errorf("%w: node %q expects %v but got %v",
				ErrInvalidInput, n.Name(), n.InputType(), inputType)
		}
	}

	// Check if this is a simple node with options
	if simpleNode, ok := n.(*node); ok && simpleNode.opts.timeout > 0 {
		// Apply timeout to entire lifecycle if configured
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, simpleNode.opts.timeout)
		defer cancel()
	}

	// Execute lifecycle with retry support for each step
	output, next, err = g.executeLifecycle(ctx, n, input)
	if err != nil {
		// Check if this is a simple node with error handler
		if simpleNode, ok := n.(*node); ok && simpleNode.opts.onError != nil {
			simpleNode.opts.onError(err)
		}
		return nil, "", err
	}

	return output, next, nil
}

// executeLifecycle runs the Prep/Exec/Post steps.
func (g *Graph) executeLifecycle(ctx context.Context, n Node, input any) (output any, next string, err error) {
	// Check if this is a simple node with hooks
	var simpleNode *node
	if sn, ok := n.(*node); ok {
		simpleNode = sn
	}

	// Ensure cleanup hooks run
	defer func() {
		if simpleNode == nil {
			return
		}
		// Run success or failure hook based on error state first
		if err != nil {
			if simpleNode.opts.onFailure != nil {
				simpleNode.opts.onFailure(ctx, g.store, err)
			}
		} else {
			if simpleNode.opts.onSuccess != nil {
				simpleNode.opts.onSuccess(ctx, g.store, output)
			}
		}

		// Always run onComplete last
		if simpleNode.opts.onComplete != nil {
			simpleNode.opts.onComplete(ctx, g.store)
		}
	}()

	// Prep step with retry
	prepResult, err := g.executeWithRetry(ctx, n, func() (any, error) {
		return n.Prep(ctx, g.store, input)
	})
	if err != nil {
		return nil, "", fmt.Errorf("prep failed: %w", err)
	}

	// Exec step with retry
	execResult, err := g.executeWithRetry(ctx, n, func() (any, error) {
		return n.Exec(ctx, prepResult)
	})
	if err != nil {
		// Check if node has a fallback
		if simpleNode != nil && simpleNode.opts.fallback != nil {
			if g.opts.logger != nil {
				g.opts.logger.Debug(ctx, "executing fallback", "name", n.Name(), "error", err)
			}

			// Execute fallback with prepResult
			fallbackResult, fallbackErr := simpleNode.opts.fallback(ctx, prepResult, err)
			if fallbackErr != nil {
				return nil, "", fmt.Errorf("exec failed and fallback failed: primary=%w, fallback=%v", err, fallbackErr)
			}

			// Continue with fallback result
			execResult = fallbackResult
		} else {
			return nil, "", fmt.Errorf("exec failed: %w", err)
		}
	}

	// Post step (no retry for routing decisions)
	output, next, err = n.Post(ctx, g.store, input, prepResult, execResult)
	if err != nil {
		return nil, "", fmt.Errorf("post failed: %w", err)
	}

	return output, next, nil
}

// executeWithRetry handles retry logic for lifecycle steps.
func (g *Graph) executeWithRetry(ctx context.Context, n Node, fn func() (any, error)) (any, error) {
	attempts := 0
	maxAttempts := 1 // default no retry
	var retryDelay time.Duration

	// Check if this is a simple node with retry options
	if simpleNode, ok := n.(*node); ok {
		maxAttempts = simpleNode.opts.maxRetries + 1
		retryDelay = simpleNode.opts.retryDelay
	}

	var lastErr error

	for attempts < maxAttempts {
		if attempts > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err
		attempts++
		if attempts < maxAttempts {
			if g.opts.logger != nil {
				g.opts.logger.Debug(ctx, "retrying node step",
					"name", n.Name(),
					"attempt", attempts,
					"error", err)
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", attempts, lastErr)
}

// AsNode returns the graph as a Node interface.
// Since graph already implements Node, we just return it.
// This method exists for backward compatibility.
func (g *Graph) AsNode(name string) Node {
	// Update the graph's name if provided
	if name != "" {
		g.name = name
	}
	return g.graph
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
