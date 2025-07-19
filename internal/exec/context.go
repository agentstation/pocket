package exec

import (
	"context"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Context provides enhanced context for flow execution.
type Context struct {
	context.Context
	mu       sync.RWMutex
	values   map[string]any
	metadata map[string]any
	tracer   Tracer
	logger   pocket.Logger
}

// Tracer provides execution tracing.
type Tracer interface {
	StartSpan(name string) Span
}

// Span represents a trace span.
type Span interface {
	End()
	SetTag(key string, value any)
	LogEvent(event string, fields map[string]any)
}

// NewContext creates an enhanced execution context.
func NewContext(parent context.Context) *Context {
	ctx := &Context{
		Context:  parent,
		values:   make(map[string]any),
		metadata: make(map[string]any),
	}
	return ctx
}

// WithTracer adds a tracer to the context.
func (c *Context) WithTracer(tracer Tracer) *Context {
	c.tracer = tracer
	return c
}

// WithLogger adds a logger to the context.
func (c *Context) WithLogger(logger pocket.Logger) *Context {
	c.logger = logger
	return c
}

// Set stores a value in the context.
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// Get retrieves a value from the context.
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.values[key]
	return val, ok
}

// SetMetadata stores metadata.
func (c *Context) SetMetadata(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metadata[key] = value
}

// GetMetadata retrieves metadata.
func (c *Context) GetMetadata(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.metadata[key]
	return val, ok
}

// StartSpan starts a trace span if tracer is available.
func (c *Context) StartSpan(name string) Span {
	if c.tracer != nil {
		return c.tracer.StartSpan(name)
	}
	return &noopSpan{}
}

// Log logs a message if logger is available.
func (c *Context) Log(level, msg string, keysAndValues ...any) {
	if c.logger == nil {
		return
	}

	switch level {
	case "debug":
		c.logger.Debug(c, msg, keysAndValues...)
	case "info":
		c.logger.Info(c, msg, keysAndValues...)
	case "error":
		c.logger.Error(c, msg, keysAndValues...)
	}
}

// Fork creates a child context with isolated values.
func (c *Context) Fork() *Context {
	child := NewContext(c)

	// Copy parent metadata
	c.mu.RLock()
	for k, v := range c.metadata {
		child.metadata[k] = v
	}
	c.mu.RUnlock()

	// Inherit tracer and logger
	child.tracer = c.tracer
	child.logger = c.logger

	return child
}

// noopSpan is a no-op trace span.
type noopSpan struct{}

func (s *noopSpan) End()                                         {}
func (s *noopSpan) SetTag(key string, value any)                 {}
func (s *noopSpan) LogEvent(event string, fields map[string]any) {}

// ContextStore wraps a store with context awareness.
type ContextStore struct {
	pocket.Store
	ctx *Context
}

// NewContextStore creates a context-aware store.
func NewContextStore(store pocket.Store, ctx *Context) *ContextStore {
	return &ContextStore{
		Store: store,
		ctx:   ctx,
	}
}

// Get retrieves a value, checking context first.
func (cs *ContextStore) Get(ctx context.Context, key string) (any, bool) {
	// Check context values first
	if val, ok := cs.ctx.Get(key); ok {
		return val, true
	}
	// Fall back to underlying store
	return cs.Store.Get(ctx, key)
}

// Set stores a value in both context and store.
func (cs *ContextStore) Set(ctx context.Context, key string, value any) error {
	// Set in context for fast access
	cs.ctx.Set(key, value)
	// Also persist to store
	return cs.Store.Set(ctx, key, value)
}

// FlowContext provides flow-specific context.
type FlowContext struct {
	*Context
	flowID    string
	startTime time.Time
	nodeStack []string
	mu        sync.Mutex
}

// NewFlowContext creates a flow execution context.
func NewFlowContext(parent context.Context, flowID string) *FlowContext {
	return &FlowContext{
		Context:   NewContext(parent),
		flowID:    flowID,
		startTime: time.Now(),
		nodeStack: []string{},
	}
}

// FlowID returns the flow ID.
func (fc *FlowContext) FlowID() string {
	return fc.flowID
}

// Duration returns the flow execution duration.
func (fc *FlowContext) Duration() time.Duration {
	return time.Since(fc.startTime)
}

// EnterNode records entering a node.
func (fc *FlowContext) EnterNode(nodeName string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.nodeStack = append(fc.nodeStack, nodeName)
	fc.Log("debug", "entering node", "node", nodeName, "depth", len(fc.nodeStack))
}

// ExitNode records exiting a node.
func (fc *FlowContext) ExitNode(nodeName string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if len(fc.nodeStack) > 0 {
		fc.nodeStack = fc.nodeStack[:len(fc.nodeStack)-1]
	}
	fc.Log("debug", "exiting node", "node", nodeName, "depth", len(fc.nodeStack))
}

// CurrentNode returns the current node name.
func (fc *FlowContext) CurrentNode() string {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if len(fc.nodeStack) > 0 {
		return fc.nodeStack[len(fc.nodeStack)-1]
	}
	return ""
}

// NodePath returns the current node execution path.
func (fc *FlowContext) NodePath() []string {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	path := make([]string, len(fc.nodeStack))
	copy(path, fc.nodeStack)
	return path
}

// Deadline wraps context deadline with flow awareness.
func (fc *FlowContext) Deadline() (deadline time.Time, ok bool) {
	deadline, ok = fc.Context.Deadline()
	if ok && fc.logger != nil {
		remaining := time.Until(deadline)
		fc.Log("debug", "flow deadline check",
			"flow", fc.flowID,
			"remaining", remaining,
			"elapsed", fc.Duration())
	}
	return
}

// ContextKey type for context values.
type ContextKey string

const (
	// FlowIDKey is the context key for flow ID.
	FlowIDKey ContextKey = "flow_id"
	// NodeNameKey is the context key for current node.
	NodeNameKey ContextKey = "node_name"
	// StoreKey is the context key for the store.
	StoreKey ContextKey = "store"
	// TracerKey is the context key for tracer.
	TracerKey ContextKey = "tracer"
)

// WithFlowID adds flow ID to context.
func WithFlowID(ctx context.Context, flowID string) context.Context {
	return context.WithValue(ctx, FlowIDKey, flowID)
}

// GetFlowID retrieves flow ID from context.
func GetFlowID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(FlowIDKey).(string)
	return id, ok
}

// WithNodeName adds node name to context.
func WithNodeName(ctx context.Context, nodeName string) context.Context {
	return context.WithValue(ctx, NodeNameKey, nodeName)
}

// GetNodeName retrieves node name from context.
func GetNodeName(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(NodeNameKey).(string)
	return name, ok
}

// WithStore adds store to context.
func WithStore(ctx context.Context, store pocket.Store) context.Context {
	return context.WithValue(ctx, StoreKey, store)
}

// GetStore retrieves store from context.
func GetStore(ctx context.Context) (pocket.Store, bool) {
	store, ok := ctx.Value(StoreKey).(pocket.Store)
	return store, ok
}
