package exec

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Event represents an execution event.
type Event struct {
	ID        string
	Type      EventType
	Timestamp time.Time
	FlowID    string
	NodeName  string
	Data      map[string]any
	Error     error
}

// EventType represents the type of execution event.
type EventType string

const (
	// Flow events.
	EventFlowStart    EventType = "flow.start"
	EventFlowComplete EventType = "flow.complete"
	EventFlowError    EventType = "flow.error"

	// Node events.
	EventNodeEnter   EventType = "node.enter"
	EventNodeExit    EventType = "node.exit"
	EventNodeError   EventType = "node.error"
	EventNodeRetry   EventType = "node.retry"
	EventNodeTimeout EventType = "node.timeout"

	// Step events.
	EventPrepStart    EventType = "prep.start"
	EventPrepComplete EventType = "prep.complete"
	EventExecStart    EventType = "exec.start"
	EventExecComplete EventType = "exec.complete"
	EventPostStart    EventType = "post.start"
	EventPostComplete EventType = "post.complete"

	// Routing events.
	EventRoute      EventType = "route"
	EventRouteError EventType = "route.error"

	// State events.
	EventStateChange EventType = "state.change"
	EventStoreUpdate EventType = "store.update"
)

// EventHandler processes events.
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

// EventBus distributes events to handlers.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]EventHandler
	filters  []EventFilter
	buffer   chan Event
	workers  int
	stopCh   chan struct{}
}

// EventFilter filters events before processing.
type EventFilter func(Event) bool

// NewEventBus creates a new event bus.
func NewEventBus(bufferSize, workers int) *EventBus {
	bus := &EventBus{
		handlers: make(map[EventType][]EventHandler),
		filters:  []EventFilter{},
		buffer:   make(chan Event, bufferSize),
		workers:  workers,
		stopCh:   make(chan struct{}),
	}

	// Start workers
	for i := 0; i < workers; i++ {
		go bus.worker()
	}

	return bus
}

// Subscribe registers a handler for event types.
func (b *EventBus) Subscribe(handler EventHandler, types ...EventType) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, eventType := range types {
		b.handlers[eventType] = append(b.handlers[eventType], handler)
	}
}

// Unsubscribe removes a handler.
func (b *EventBus) Unsubscribe(handler EventHandler, types ...EventType) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, eventType := range types {
		handlers := b.handlers[eventType]
		filtered := handlers[:0]

		for _, h := range handlers {
			if h != handler {
				filtered = append(filtered, h)
			}
		}

		b.handlers[eventType] = filtered
	}
}

// AddFilter adds an event filter.
func (b *EventBus) AddFilter(filter EventFilter) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.filters = append(b.filters, filter)
}

// Publish publishes an event.
func (b *EventBus) Publish(event Event) {
	// Apply filters
	b.mu.RLock()
	for _, filter := range b.filters {
		if !filter(event) {
			b.mu.RUnlock()
			return
		}
	}
	b.mu.RUnlock()

	// Send to buffer
	select {
	case b.buffer <- event:
	default:
		// Buffer full, drop event
		// In production, might want to handle this differently
	}
}

// Stop stops the event bus.
func (b *EventBus) Stop() {
	close(b.stopCh)
}

// worker processes events.
func (b *EventBus) worker() {
	for {
		select {
		case event := <-b.buffer:
			b.processEvent(event)
		case <-b.stopCh:
			return
		}
	}
}

// processEvent handles a single event.
func (b *EventBus) processEvent(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	ctx := context.Background()
	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			// Log error but continue processing
			fmt.Printf("Event handler error: %v\n", err)
		}
	}
}

// Common event handlers

// LoggingHandler logs events.
type LoggingHandler struct {
	logger pocket.Logger
	level  string
}

// NewLoggingHandler creates a logging event handler.
func NewLoggingHandler(logger pocket.Logger, level string) *LoggingHandler {
	return &LoggingHandler{
		logger: logger,
		level:  level,
	}
}

// Handle logs the event.
func (h *LoggingHandler) Handle(ctx context.Context, event Event) error {
	fields := []any{
		"event_type", event.Type,
		"flow_id", event.FlowID,
		"node", event.NodeName,
		"timestamp", event.Timestamp,
	}

	if event.Error != nil {
		fields = append(fields, "error", event.Error)
	}

	for k, v := range event.Data {
		fields = append(fields, k, v)
	}

	switch h.level {
	case "debug":
		h.logger.Debug(ctx, "flow event", fields...)
	case "info":
		h.logger.Info(ctx, "flow event", fields...)
	case "error":
		if event.Error != nil {
			h.logger.Error(ctx, "flow event", fields...)
		} else {
			h.logger.Info(ctx, "flow event", fields...)
		}
	}

	return nil
}

// MetricsHandler collects metrics from events.
type MetricsHandler struct {
	mu      sync.RWMutex
	metrics map[string]*EventMetrics
}

// EventMetrics tracks event statistics.
type EventMetrics struct {
	Count       int64
	ErrorCount  int64
	LastSeen    time.Time
	AvgDuration time.Duration
}

// NewMetricsHandler creates a metrics event handler.
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		metrics: make(map[string]*EventMetrics),
	}
}

// Handle updates metrics based on event.
func (h *MetricsHandler) Handle(ctx context.Context, event Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := fmt.Sprintf("%s:%s", event.FlowID, event.NodeName)
	metrics, exists := h.metrics[key]
	if !exists {
		metrics = &EventMetrics{}
		h.metrics[key] = metrics
	}

	metrics.Count++
	metrics.LastSeen = event.Timestamp

	if event.Error != nil {
		metrics.ErrorCount++
	}

	// Update duration for completion events
	if duration, ok := event.Data["duration"].(time.Duration); ok {
		if metrics.AvgDuration == 0 {
			metrics.AvgDuration = duration
		} else {
			// Simple moving average
			metrics.AvgDuration = (metrics.AvgDuration + duration) / 2
		}
	}

	return nil
}

// GetMetrics returns collected metrics.
func (h *MetricsHandler) GetMetrics() map[string]*EventMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*EventMetrics)
	for k, v := range h.metrics {
		result[k] = v
	}
	return result
}

// StorageHandler persists events.
type StorageHandler struct {
	store pocket.Store
}

// NewStorageHandler creates a storage event handler.
func NewStorageHandler(store pocket.Store) *StorageHandler {
	return &StorageHandler{store: store}
}

// Handle stores the event.
func (h *StorageHandler) Handle(ctx context.Context, event Event) error {
	key := fmt.Sprintf("event:%s:%s", event.Type, event.ID)
	return h.store.Set(ctx, key, event)
}

// ChainHandler chains multiple handlers.
type ChainHandler struct {
	handlers []EventHandler
}

// NewChainHandler creates a chain of handlers.
func NewChainHandler(handlers ...EventHandler) *ChainHandler {
	return &ChainHandler{handlers: handlers}
}

// Handle calls all handlers in sequence.
func (h *ChainHandler) Handle(ctx context.Context, event Event) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// ConditionalHandler conditionally handles events.
type ConditionalHandler struct {
	condition func(Event) bool
	handler   EventHandler
}

// NewConditionalHandler creates a conditional handler.
func NewConditionalHandler(condition func(Event) bool, handler EventHandler) *ConditionalHandler {
	return &ConditionalHandler{
		condition: condition,
		handler:   handler,
	}
}

// Handle processes event if condition is met.
func (h *ConditionalHandler) Handle(ctx context.Context, event Event) error {
	if h.condition(event) {
		return h.handler.Handle(ctx, event)
	}
	return nil
}

// EventRecorder records events for replay.
type EventRecorder struct {
	mu      sync.RWMutex
	events  []Event
	maxSize int
}

// NewEventRecorder creates an event recorder.
func NewEventRecorder(maxSize int) *EventRecorder {
	return &EventRecorder{
		events:  make([]Event, 0, maxSize),
		maxSize: maxSize,
	}
}

// Handle records the event.
func (r *EventRecorder) Handle(ctx context.Context, event Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)

	// Trim if needed
	if len(r.events) > r.maxSize {
		r.events = r.events[len(r.events)-r.maxSize:]
	}

	return nil
}

// GetEvents returns recorded events.
func (r *EventRecorder) GetEvents() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Event, len(r.events))
	copy(result, r.events)
	return result
}

// Replay replays events to a handler.
func (r *EventRecorder) Replay(ctx context.Context, handler EventHandler, filter EventFilter) error {
	events := r.GetEvents()

	for _, event := range events {
		if filter != nil && !filter(event) {
			continue
		}

		if err := handler.Handle(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
