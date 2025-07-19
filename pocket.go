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
	// ErrNoStartNode is returned when a flow has no start node defined.
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

// StoreReader provides read-only access to the store.
// Used in the Prep phase to enforce read-only semantics.
type StoreReader interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) (value any, exists bool)

	// Scope returns a new store with the given prefix.
	Scope(prefix string) Store
}

// StoreWriter provides full read-write access to the store.
// Used in the Post phase for state mutations.
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

// Node represents a processing unit in a workflow with Prep/Exec/Post lifecycle.
type Node struct {
	// Name identifies the node in the flow.
	Name string

	// Lifecycle methods (never nil - have defaults).
	Prep PrepFunc
	Exec ExecFunc
	Post PostFunc

	// Type information for validation (optional).
	InputType  reflect.Type
	OutputType reflect.Type

	// Successors maps action names to next nodes.
	successors map[string]*Node

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
	onError    func(error)
	fallback   func(ctx context.Context, input any, err error) (any, error)
	
	// Cleanup hooks
	onSuccess func(ctx context.Context, store StoreWriter, output any)
	onFailure func(ctx context.Context, store StoreWriter, err error)
	onComplete func(ctx context.Context, store StoreWriter)
}

// Option configures a Node.
type Option func(*nodeOptions)

// WithPrep sets the preparation function with type safety.
// The input type In should match the node's input type when used with NewNode[In, Out].
// For dynamic typing, use WithPrep[any].
// The store parameter provides read-only access to enforce Prep phase semantics.
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
// Post functions have access to all phase results and determine routing.
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

// WithFallback adds a fallback function that runs if the exec phase fails.
// The types In and Out should match the node's types when used with NewNode[In, Out].
// For dynamic typing, use WithFallback[any, any].
// Like exec functions, fallback functions do not have store access.
func WithFallback[In, Out any](fn func(ctx context.Context, input In, err error) (Out, error)) Option {
	return func(o *nodeOptions) {
		o.fallback = func(ctx context.Context, input any, err error) (any, error) {
			// Type assertion for input
			typedInput, ok := input.(In)
			if !ok {
				return nil, fmt.Errorf("%w: fallback expected %T, got %T", ErrInvalidInput, *new(In), input)
			}
			
			// Execute typed fallback
			result, fallbackErr := fn(ctx, typedInput, err)
			if fallbackErr != nil {
				return nil, fallbackErr
			}
			return result, nil
		}
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

// Default implementations for lifecycle methods.
func defaultPrep(ctx context.Context, store StoreReader, input any) (any, error) {
	return input, nil // pass through
}

func defaultExec(ctx context.Context, prepResult any) (any, error) {
	return prepResult, nil // pass through
}

func defaultPost(ctx context.Context, store StoreWriter, input, prepResult, execResult any) (any, string, error) {
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
func newNodeBase(name string, opts ...Option) *Node {
	// Get global defaults
	defaultPrep, defaultExec, defaultPost, defaultOpts := getDefaults()
	
	// Create node with defaults
	node := &Node{
		Name:       name,
		Prep:       defaultPrep,
		Exec:       defaultExec,
		Post:       defaultPost,
		successors: make(map[string]*Node),
		opts:       defaultOpts,
	}
	
	// Apply lifecycle functions from defaults if set
	if defaultOpts.prep != nil {
		node.Prep = defaultOpts.prep
	}
	if defaultOpts.exec != nil {
		node.Exec = defaultOpts.exec
	}
	if defaultOpts.post != nil {
		node.Post = defaultOpts.post
	}
	
	// Apply provided options (override defaults)
	for _, opt := range opts {
		opt(&node.opts)
	}
	
	// Apply lifecycle functions from options
	if node.opts.prep != nil {
		node.Prep = node.opts.prep
	}
	if node.opts.exec != nil {
		node.Exec = node.opts.exec
	}
	if node.opts.post != nil {
		node.Post = node.opts.post
	}
	
	return node
}

// NewNode creates a new node with optional compile-time type safety.
//
// Type parameters:
//   - In: The expected input type for this node (use 'any' for dynamic typing)
//   - Out: The output type this node produces (use 'any' for dynamic typing)
//
// Type safety mechanism:
//
//   1. Compile-time: When In/Out are not 'any', the node stores type information
//      for validation. Using generic options like WithExec, WithPrep ensures function
//      signatures match the declared types at compile time.
//
//   2. Initialization-time: Call ValidateFlow on your start node to verify the
//      entire workflow graph has compatible types between connected nodes.
//
//   3. Runtime: When using regular options (WithExec, WithPrep) with typed nodes,
//      the framework automatically wraps functions to ensure type safety.
//
// Examples:
//
//   // Typed node - enables full type checking across the workflow
//   validator := NewNode[User, ValidationResult]("validator",
//       WithExec(func(ctx context.Context, store Store, user User) (ValidationResult, error) {
//           // Compile-time type safety - no casting needed
//           return ValidationResult{Valid: true}, nil
//       }),
//   )
//
//   // Untyped node - no compile-time checks (explicit [any, any] encourages adding types)
//   processor := NewNode[any, any]("processor",
//       WithExec(func(ctx context.Context, store Store, input any) (any, error) {
//           return processData(input), nil
//       }),
//   )
func NewNode[In, Out any](name string, opts ...Option) *Node {
	// Create base node using existing logic
	node := newNodeBase(name, opts...)
	
	// Determine if types are specified (not 'any')
	inType := reflect.TypeOf((*In)(nil)).Elem()
	outType := reflect.TypeOf((*Out)(nil)).Elem()
	
	// Set type information on node if types are not 'any'
	// This enables ValidateFlow to check type compatibility between nodes
	if !isAnyType(inType) {
		node.InputType = inType
	}
	if !isAnyType(outType) {
		node.OutputType = outType
	}
	
	return node
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

// ValidateFlow provides initialization-time type safety by validating the entire workflow graph.
//
// Type validation process:
//   1. Traverses the graph starting from the given node using depth-first search
//   2. For each connection between nodes, verifies type compatibility:
//      - Source node's OutputType must be assignable to target node's InputType
//      - Interface satisfaction is checked (e.g., concrete type implements interface)
//      - Untyped nodes (InputType/OutputType = nil) are skipped but successors are validated
//   3. Returns detailed error messages identifying the exact type mismatch location
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
//   // Build your workflow
//   validator := NewNode[User, ValidationResult]("validator", ...)
//   processor := NewNode[ValidationResult, Response]("processor", ...)
//   validator.Connect("valid", processor)
//   
//   // Validate before execution - catches type mismatches early
//   if err := ValidateFlow(validator); err != nil {
//       // Error: "type mismatch: node 'validator' outputs ValidationResult
//       //         but node 'wrongNode' expects User (via action 'valid')"
//       log.Fatal(err)
//   }
//   
//   // Safe to execute - types are verified
//   flow := NewFlow(validator, store)
//   result, err := flow.Run(ctx, user)
func ValidateFlow(start *Node) error {
	visited := make(map[string]bool)
	return validateNode(start, visited)
}

func validateNode(node *Node, visited map[string]bool) error {
	if node == nil || visited[node.Name] {
		return nil
	}
	visited[node.Name] = true

	// If this node has no type information, skip validation
	if node.OutputType == nil {
		// Still validate successors
		for _, successor := range node.successors {
			if err := validateNode(successor, visited); err != nil {
				return err
			}
		}
		return nil
	}

	// Check each successor
	for action, successor := range node.successors {
		if successor.InputType != nil {
			// Both types are specified, check compatibility
			if !isTypeCompatible(node.OutputType, successor.InputType) {
				return fmt.Errorf("type mismatch: node %q outputs %v but node %q expects %v (via action %q)",
					node.Name, node.OutputType, successor.Name, successor.InputType, action)
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

// Run executes the flow with the given input.
func (f *Flow) Run(ctx context.Context, input any) (output any, err error) {
	if f.start == nil {
		return nil, ErrNoStartNode
	}

	current := f.start
	currentInput := input
	var lastOutput any

	for current != nil {
		// Log node execution
		if f.opts.logger != nil {
			f.opts.logger.Debug(ctx, "executing node", "name", current.Name)
		}

		// Execute node with lifecycle
		output, next, err := f.executeNode(ctx, current, currentInput)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", current.Name, err)
		}

		// Save the output
		lastOutput = output

		// Move to next node
		current = current.successors[next]
		currentInput = output
	}

	return lastOutput, nil
}

// executeNode runs a single node with runtime type safety checks at each lifecycle phase.
//
// Runtime type safety:
//   - Input validation: If node has InputType set, verifies input matches before execution
//   - Wrapped functions: Type assertions in wrapped lifecycle functions ensure correctness
//   - Error propagation: Type mismatches result in clear error messages with actual vs expected types
//
// This is where runtime type checking occurs, complementing compile-time and init-time checks.
// For typed nodes using generic options like WithExec, type assertions are handled
// automatically through Go's type inference.
func (f *Flow) executeNode(ctx context.Context, node *Node, input any) (output any, next string, err error) {
	// Runtime type check: Validate input matches node's expected type
	// This catches any type mismatches that slipped through earlier checks
	if node.InputType != nil && input != nil {
		inputType := reflect.TypeOf(input)
		if !isTypeCompatible(inputType, node.InputType) {
			return nil, "", fmt.Errorf("%w: node %q expects %v but got %v", 
				ErrInvalidInput, node.Name, node.InputType, inputType)
		}
	}
	// Apply timeout to entire lifecycle if configured
	if node.opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, node.opts.timeout)
		defer cancel()
	}

	// Execute lifecycle with retry support for each phase
	output, next, err = f.executeLifecycle(ctx, node, input)
	if err != nil {
		if node.opts.onError != nil {
			node.opts.onError(err)
		}
		return nil, "", err
	}

	return output, next, nil
}

// executeLifecycle runs the Prep/Exec/Post phases.
func (f *Flow) executeLifecycle(ctx context.Context, node *Node, input any) (output any, next string, err error) {
	// Ensure cleanup hooks run
	defer func() {
		// Run success or failure hook based on error state first
		if err != nil {
			if node.opts.onFailure != nil {
				node.opts.onFailure(ctx, f.store, err)
			}
		} else {
			if node.opts.onSuccess != nil {
				node.opts.onSuccess(ctx, f.store, output)
			}
		}
		
		// Always run onComplete last
		if node.opts.onComplete != nil {
			node.opts.onComplete(ctx, f.store)
		}
	}()

	// Prep phase with retry
	prepResult, err := f.executeWithRetry(ctx, node, func() (any, error) {
		return node.Prep(ctx, f.store, input)
	})
	if err != nil {
		return nil, "", fmt.Errorf("prep failed: %w", err)
	}

	// Exec phase with retry
	execResult, err := f.executeWithRetry(ctx, node, func() (any, error) {
		return node.Exec(ctx, prepResult)
	})
	if err != nil {
		// Check if node has a fallback
		if node.opts.fallback != nil {
			if f.opts.logger != nil {
				f.opts.logger.Debug(ctx, "executing fallback", "name", node.Name, "error", err)
			}
			
			// Execute fallback with original input
			fallbackResult, fallbackErr := node.opts.fallback(ctx, input, err)
			if fallbackErr != nil {
				return nil, "", fmt.Errorf("exec failed and fallback failed: primary=%w, fallback=%v", err, fallbackErr)
			}
			
			// Continue with fallback result
			execResult = fallbackResult
		} else {
			return nil, "", fmt.Errorf("exec failed: %w", err)
		}
	}

	// Post phase (no retry for routing decisions)
	output, next, err = node.Post(ctx, f.store, input, prepResult, execResult)
	if err != nil {
		return nil, "", fmt.Errorf("post failed: %w", err)
	}

	return output, next, nil
}

// executeWithRetry handles retry logic for lifecycle phases.
func (f *Flow) executeWithRetry(ctx context.Context, node *Node, fn func() (any, error)) (any, error) {
	attempts := 0
	maxAttempts := node.opts.maxRetries + 1
	var lastErr error

	for attempts < maxAttempts {
		if attempts > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(node.opts.retryDelay):
			}
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		attempts++
		if attempts < maxAttempts {
			if f.opts.logger != nil {
				f.opts.logger.Debug(ctx, "retrying node phase",
					"name", node.Name,
					"attempt", attempts,
					"error", err)
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", attempts, lastErr)
}

// AsNode converts this Flow into a Node that can be used within another Flow.
// This enables flow composition where entire workflows become single nodes.
func (f *Flow) AsNode(name string) *Node {
	return NewNode[any, any](name,
		WithExec(func(ctx context.Context, input any) (any, error) {
			// Run the flow with the provided input
			// The flow uses its own store, maintaining isolation
			return f.Run(ctx, input)
		}),
	)
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

