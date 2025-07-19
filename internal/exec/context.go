package exec

import (
	"context"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Context provides enhanced context for graph execution.
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

// GraphContext provides graph-specific context.
type GraphContext struct {
	*Context
	graphID   string
	startTime time.Time
	nodeStack []string
	mu        sync.Mutex
}

// NewGraphContext creates a graph execution context.
func NewGraphContext(parent context.Context, graphID string) *GraphContext {
	return &GraphContext{
		Context:   NewContext(parent),
		graphID:   graphID,
		startTime: time.Now(),
		nodeStack: []string{},
	}
}

// GraphID returns the graph ID.
func (gc *GraphContext) GraphID() string {
	return gc.graphID
}

// Duration returns the graph execution duration.
func (gc *GraphContext) Duration() time.Duration {
	return time.Since(gc.startTime)
}

// EnterNode records entering a node.
func (gc *GraphContext) EnterNode(nodeName string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.nodeStack = append(gc.nodeStack, nodeName)
	gc.Log("debug", "entering node", "node", nodeName, "depth", len(gc.nodeStack))
}

// ExitNode records exiting a node.
func (gc *GraphContext) ExitNode(nodeName string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if len(gc.nodeStack) > 0 {
		gc.nodeStack = gc.nodeStack[:len(gc.nodeStack)-1]
	}
	gc.Log("debug", "exiting node", "node", nodeName, "depth", len(gc.nodeStack))
}

// CurrentNode returns the current node name.
func (gc *GraphContext) CurrentNode() string {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if len(gc.nodeStack) > 0 {
		return gc.nodeStack[len(gc.nodeStack)-1]
	}
	return ""
}

// NodePath returns the current node execution path.
func (gc *GraphContext) NodePath() []string {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	path := make([]string, len(gc.nodeStack))
	copy(path, gc.nodeStack)
	return path
}

// Deadline wraps context deadline with graph awareness.
func (gc *GraphContext) Deadline() (deadline time.Time, ok bool) {
	deadline, ok = gc.Context.Deadline()
	if ok && gc.logger != nil {
		remaining := time.Until(deadline)
		gc.Log("debug", "graph deadline check",
			"graph", gc.graphID,
			"remaining", remaining,
			"elapsed", gc.Duration())
	}
	return
}

// ContextKey type for context values.
type ContextKey string

const (
	// GraphIDKey is the context key for graph ID.
	GraphIDKey ContextKey = "graph_id"
	// NodeNameKey is the context key for current node.
	NodeNameKey ContextKey = "node_name"
	// StoreKey is the context key for the store.
	StoreKey ContextKey = "store"
	// TracerKey is the context key for tracer.
	TracerKey ContextKey = "tracer"
)

// WithGraphID adds graph ID to context.
func WithGraphID(ctx context.Context, graphID string) context.Context {
	return context.WithValue(ctx, GraphIDKey, graphID)
}

// GetGraphID retrieves graph ID from context.
func GetGraphID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(GraphIDKey).(string)
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
