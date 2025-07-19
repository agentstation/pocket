package node

import (
	"context"
	"fmt"
	"sync"

	"github.com/agentstation/pocket"
)

// Hook represents a lifecycle hook.
type Hook interface {
	Name() string
	Execute(ctx context.Context, event Event) error
}

// Event represents a lifecycle event.
type Event struct {
	Type      EventType
	NodeName  string
	Phase     string
	Input     any
	Output    any
	Error     error
	Metadata  map[string]any
}

// EventType represents the type of lifecycle event.
type EventType string

const (
	EventPrep    EventType = "prep"
	EventExec    EventType = "exec"
	EventPost    EventType = "post"
	EventSuccess EventType = "success"
	EventError   EventType = "error"
	EventRoute   EventType = "route"
)

// HookManager manages lifecycle hooks for nodes.
type HookManager struct {
	mu     sync.RWMutex
	hooks  map[EventType][]Hook
	global []Hook
}

// NewHookManager creates a new hook manager.
func NewHookManager() *HookManager {
	return &HookManager{
		hooks:  make(map[EventType][]Hook),
		global: []Hook{},
	}
}

// Register registers a hook for specific event types.
func (m *HookManager) Register(hook Hook, events ...EventType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(events) == 0 {
		// Register as global hook
		m.global = append(m.global, hook)
		return
	}

	for _, event := range events {
		m.hooks[event] = append(m.hooks[event], hook)
	}
}

// Trigger executes hooks for an event.
func (m *HookManager) Trigger(ctx context.Context, event Event) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Execute type-specific hooks
	if hooks, ok := m.hooks[event.Type]; ok {
		for _, hook := range hooks {
			if err := hook.Execute(ctx, event); err != nil {
				return fmt.Errorf("hook %s failed: %w", hook.Name(), err)
			}
		}
	}

	// Execute global hooks
	for _, hook := range m.global {
		if err := hook.Execute(ctx, event); err != nil {
			return fmt.Errorf("global hook %s failed: %w", hook.Name(), err)
		}
	}

	return nil
}

// WithHooks adds hook support to a node.
func WithHooks(manager *HookManager) Middleware {
	return func(node *pocket.Node) *pocket.Node {
		originalPrep := node.Prep
		originalExec := node.Exec
		originalPost := node.Post

		node.Prep = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Trigger pre-prep event
			_ = manager.Trigger(ctx, Event{
				Type:     EventPrep,
				NodeName: node.Name,
				Phase:    "before",
				Input:    input,
				Metadata: map[string]any{"stage": "prep"},
			})

			result, err := originalPrep(ctx, store, input)

			// Trigger post-prep event
			event := Event{
				Type:     EventPrep,
				NodeName: node.Name,
				Phase:    "after",
				Input:    input,
				Output:   result,
				Error:    err,
				Metadata: map[string]any{"stage": "prep"},
			}
			
			if err != nil {
				event.Type = EventError
			}
			
			_ = manager.Trigger(ctx, event)
			return result, err
		}

		node.Exec = func(ctx context.Context, input any) (any, error) {
			// Trigger pre-exec event
			_ = manager.Trigger(ctx, Event{
				Type:     EventExec,
				NodeName: node.Name,
				Phase:    "before",
				Input:    input,
				Metadata: map[string]any{"stage": "exec"},
			})

			result, err := originalExec(ctx, input)

			// Trigger post-exec event
			event := Event{
				Type:     EventExec,
				NodeName: node.Name,
				Phase:    "after",
				Input:    input,
				Output:   result,
				Error:    err,
				Metadata: map[string]any{"stage": "exec"},
			}
			
			if err != nil {
				event.Type = EventError
			} else {
				_ = manager.Trigger(ctx, Event{
					Type:     EventSuccess,
					NodeName: node.Name,
					Phase:    "exec",
					Output:   result,
				})
			}
			
			_ = manager.Trigger(ctx, event)
			return result, err
		}

		node.Post = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			// Trigger pre-post event
			_ = manager.Trigger(ctx, Event{
				Type:     EventPost,
				NodeName: node.Name,
				Phase:    "before",
				Input:    input,
				Metadata: map[string]any{
					"stage": "post",
					"prep":  prep,
					"exec":  exec,
				},
			})

			output, next, err := originalPost(ctx, store, input, prep, exec)

			// Trigger post-post event
			event := Event{
				Type:     EventPost,
				NodeName: node.Name,
				Phase:    "after",
				Input:    input,
				Output:   output,
				Error:    err,
				Metadata: map[string]any{
					"stage": "post",
					"next":  next,
				},
			}
			
			if err == nil {
				// Trigger routing event
				_ = manager.Trigger(ctx, Event{
					Type:     EventRoute,
					NodeName: node.Name,
					Metadata: map[string]any{
						"next": next,
						"output": output,
					},
				})
			}
			
			_ = manager.Trigger(ctx, event)
			return output, next, err
		}

		return node
	}
}

// Common hook implementations

// LoggingHook logs lifecycle events.
type LoggingHook struct {
	name   string
	logger pocket.Logger
}

// NewLoggingHook creates a logging hook.
func NewLoggingHook(logger pocket.Logger) *LoggingHook {
	return &LoggingHook{
		name:   "logging",
		logger: logger,
	}
}

func (h *LoggingHook) Name() string {
	return h.name
}

func (h *LoggingHook) Execute(ctx context.Context, event Event) error {
	fields := []any{
		"event", event.Type,
		"node", event.NodeName,
		"phase", event.Phase,
	}

	if event.Error != nil {
		fields = append(fields, "error", event.Error)
		h.logger.Error(ctx, "node lifecycle event", fields...)
	} else {
		h.logger.Debug(ctx, "node lifecycle event", fields...)
	}

	return nil
}

// MetricsHook collects metrics from events.
type MetricsHook struct {
	name      string
	collector MetricsCollector
}

// NewMetricsHook creates a metrics collection hook.
func NewMetricsHook(collector MetricsCollector) *MetricsHook {
	return &MetricsHook{
		name:      "metrics",
		collector: collector,
	}
}

func (h *MetricsHook) Name() string {
	return h.name
}

func (h *MetricsHook) Execute(ctx context.Context, event Event) error {
	switch event.Type {
	case EventPrep, EventExec, EventPost:
		if event.Phase == "before" {
			h.collector.RecordPhaseStart(event.NodeName, string(event.Type))
		} else {
			h.collector.RecordPhaseEnd(event.NodeName, string(event.Type), event.Error)
		}
	case EventRoute:
		if next, ok := event.Metadata["next"].(string); ok {
			h.collector.RecordRouting(event.NodeName, next)
		}
	}
	return nil
}

// TracingHook adds distributed tracing.
type TracingHook struct {
	name   string
	tracer pocket.Tracer
	spans  map[string]func()
	mu     sync.Mutex
}

// NewTracingHook creates a tracing hook.
func NewTracingHook(tracer pocket.Tracer) *TracingHook {
	return &TracingHook{
		name:   "tracing",
		tracer: tracer,
		spans:  make(map[string]func()),
	}
}

func (h *TracingHook) Name() string {
	return h.name
}

func (h *TracingHook) Execute(ctx context.Context, event Event) error {
	spanKey := fmt.Sprintf("%s-%s-%s", event.NodeName, event.Type, event.Phase)

	h.mu.Lock()
	defer h.mu.Unlock()

	switch event.Phase {
	case "before":
		// Start span
		_, finish := h.tracer.StartSpan(ctx, fmt.Sprintf("%s.%s", event.NodeName, event.Type))
		h.spans[spanKey] = finish
	case "after":
		// End span
		if finish, ok := h.spans[spanKey]; ok {
			finish()
			delete(h.spans, spanKey)
		}
	}

	return nil
}

// StoreHook stores event data in the store.
type StoreHook struct {
	name   string
	prefix string
}

// NewStoreHook creates a hook that stores events.
func NewStoreHook(prefix string) *StoreHook {
	return &StoreHook{
		name:   "store",
		prefix: prefix,
	}
}

func (h *StoreHook) Name() string {
	return h.name
}

func (h *StoreHook) Execute(ctx context.Context, event Event) error {
	// This would need access to the store
	// In practice, you'd pass the store in the context or hook constructor
	return nil
}

// ConditionalHook executes only when condition is met.
type ConditionalHook struct {
	name      string
	condition func(Event) bool
	wrapped   Hook
}

// NewConditionalHook creates a conditional hook.
func NewConditionalHook(condition func(Event) bool, wrapped Hook) *ConditionalHook {
	return &ConditionalHook{
		name:      fmt.Sprintf("conditional(%s)", wrapped.Name()),
		condition: condition,
		wrapped:   wrapped,
	}
}

func (h *ConditionalHook) Name() string {
	return h.name
}

func (h *ConditionalHook) Execute(ctx context.Context, event Event) error {
	if h.condition(event) {
		return h.wrapped.Execute(ctx, event)
	}
	return nil
}

// ChainHook executes multiple hooks in sequence.
type ChainHook struct {
	name  string
	hooks []Hook
}

// NewChainHook creates a hook that chains multiple hooks.
func NewChainHook(name string, hooks ...Hook) *ChainHook {
	return &ChainHook{
		name:  name,
		hooks: hooks,
	}
}

func (h *ChainHook) Name() string {
	return h.name
}

func (h *ChainHook) Execute(ctx context.Context, event Event) error {
	for _, hook := range h.hooks {
		if err := hook.Execute(ctx, event); err != nil {
			return err
		}
	}
	return nil
}