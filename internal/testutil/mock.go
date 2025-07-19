// Package testutil provides testing utilities for pocket.
package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

const (
	defaultRoute = "default"
)

// MockStore provides a mock implementation of pocket.Store for testing.
type MockStore struct {
	mu       sync.RWMutex
	data     map[string]any
	calls    []StoreCall
	errors   map[string]error
	behavior StoreBehavior
}

// StoreCall records a store method call.
type StoreCall struct {
	Method string
	Key    string
	Value  any
}

// StoreBehavior defines mock behavior.
type StoreBehavior struct {
	FailGet    bool
	FailSet    bool
	FailDelete bool
	GetDelay   time.Duration
	SetDelay   time.Duration
}

// NewMockStore creates a new mock store.
func NewMockStore() *MockStore {
	return &MockStore{
		data:   make(map[string]any),
		calls:  []StoreCall{},
		errors: make(map[string]error),
	}
}

// Get retrieves a value by key.
func (s *MockStore) Get(ctx context.Context, key string) (any, bool) {
	s.mu.Lock()
	s.calls = append(s.calls, StoreCall{Method: "Get", Key: key})
	s.mu.Unlock()

	if s.behavior.GetDelay > 0 {
		time.Sleep(s.behavior.GetDelay)
	}

	if s.behavior.FailGet {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	val, exists := s.data[key]
	return val, exists
}

// Set stores a value with the given key.
func (s *MockStore) Set(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, StoreCall{Method: "Set", Key: key, Value: value})

	if s.behavior.SetDelay > 0 {
		time.Sleep(s.behavior.SetDelay)
	}

	if s.behavior.FailSet {
		return fmt.Errorf("mock set error")
	}

	if err, hasError := s.errors[key]; hasError {
		return err
	}

	s.data[key] = value
	return nil
}

// Delete removes a key from the store.
func (s *MockStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, StoreCall{Method: "Delete", Key: key})

	if s.behavior.FailDelete {
		return fmt.Errorf("mock delete error")
	}

	delete(s.data, key)
	return nil
}

// Scope returns a new store with the given prefix.
func (s *MockStore) Scope(prefix string) pocket.Store {
	return &scopedMockStore{
		parent: s,
		prefix: prefix,
	}
}

// GetCalls returns all recorded calls.
func (s *MockStore) GetCalls() []StoreCall {
	s.mu.RLock()
	defer s.mu.RUnlock()

	calls := make([]StoreCall, len(s.calls))
	copy(calls, s.calls)
	return calls
}

// SetError sets an error for a specific key.
func (s *MockStore) SetError(key string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors[key] = err
}

// SetBehavior sets the mock behavior.
func (s *MockStore) SetBehavior(behavior StoreBehavior) {
	s.behavior = behavior
}

// Reset clears all data and calls.
func (s *MockStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]any)
	s.calls = []StoreCall{}
	s.errors = make(map[string]error)
	s.behavior = StoreBehavior{}
}

// scopedMockStore implements scoped store.
type scopedMockStore struct {
	parent *MockStore
	prefix string
}

func (s *scopedMockStore) Get(ctx context.Context, key string) (any, bool) {
	return s.parent.Get(ctx, s.prefix+":"+key)
}

func (s *scopedMockStore) Set(ctx context.Context, key string, value any) error {
	return s.parent.Set(ctx, s.prefix+":"+key, value)
}

func (s *scopedMockStore) Delete(ctx context.Context, key string) error {
	return s.parent.Delete(ctx, s.prefix+":"+key)
}

func (s *scopedMockStore) Scope(prefix string) pocket.Store {
	return &scopedMockStore{
		parent: s.parent,
		prefix: s.prefix + ":" + prefix,
	}
}

// MockNode creates a mock node for testing.
type MockNode struct {
	Name      string
	PrepFunc  pocket.PrepFunc
	ExecFunc  pocket.ExecFunc
	PostFunc  pocket.PostFunc
	CallCount map[string]int
	mu        sync.Mutex
}

// NewMockNode creates a new mock node.
func NewMockNode(name string) *MockNode {
	m := &MockNode{
		Name:      name,
		CallCount: make(map[string]int),
	}

	// Set default functions that track calls
	m.PrepFunc = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
		m.recordCall("prep")
		return input, nil
	}

	m.ExecFunc = func(ctx context.Context, input any) (any, error) {
		m.recordCall("exec")
		return input, nil
	}

	m.PostFunc = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
		m.recordCall("post")
		return exec, defaultRoute, nil
	}

	return m
}

// Build creates a pocket.Node from the mock.
func (m *MockNode) Build() *pocket.Node {
	var opts []pocket.Option
	
	if m.PrepFunc != nil {
		opts = append(opts, pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			return m.PrepFunc(ctx, store, input)
		}))
	}
	
	if m.ExecFunc != nil {
		opts = append(opts, pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return m.ExecFunc(ctx, input)
		}))
	}
	
	if m.PostFunc != nil {
		opts = append(opts, pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			return m.PostFunc(ctx, store, input, prep, exec)
		}))
	}
	
	return pocket.NewNode[any, any](m.Name, opts...)
}

// WithPrep sets a custom prep function.
func (m *MockNode) WithPrep(fn pocket.PrepFunc) *MockNode {
	m.PrepFunc = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
		m.recordCall("prep")
		return fn(ctx, store, input)
	}
	return m
}

// WithExec sets a custom exec function.
func (m *MockNode) WithExec(fn pocket.ExecFunc) *MockNode {
	m.ExecFunc = func(ctx context.Context, input any) (any, error) {
		m.recordCall("exec")
		return fn(ctx, input)
	}
	return m
}

// WithPost sets a custom post function.
func (m *MockNode) WithPost(fn pocket.PostFunc) *MockNode {
	m.PostFunc = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
		m.recordCall("post")
		return fn(ctx, store, input, prep, exec)
	}
	return m
}

// WithError makes the node return an error.
func (m *MockNode) WithError(phase string, err error) *MockNode {
	switch phase {
	case "prep":
		m.PrepFunc = func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			m.recordCall("prep")
			return nil, err
		}
	case "exec":
		m.ExecFunc = func(ctx context.Context, input any) (any, error) {
			m.recordCall("exec")
			return nil, err
		}
	case "post":
		m.PostFunc = func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			m.recordCall("post")
			return nil, "", err
		}
	}
	return m
}

// GetCallCount returns the call count for a phase.
func (m *MockNode) GetCallCount(phase string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CallCount[phase]
}

// Reset resets call counts.
func (m *MockNode) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount = make(map[string]int)
}

func (m *MockNode) recordCall(phase string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount[phase]++
}

// MockLogger provides a mock logger for testing.
type MockLogger struct {
	mu      sync.Mutex
	entries []LogEntry
}

// LogEntry represents a log entry.
type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]any
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		entries: []LogEntry{},
	}
}

// Debug logs a debug message.
func (l *MockLogger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	l.log("debug", msg, keysAndValues...)
}

// Info logs an info message.
func (l *MockLogger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	l.log("info", msg, keysAndValues...)
}

// Error logs an error message.
func (l *MockLogger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	l.log("error", msg, keysAndValues...)
}

func (l *MockLogger) log(level, msg string, keysAndValues ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fields := make(map[string]any)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields[key] = keysAndValues[i+1]
		}
	}

	l.entries = append(l.entries, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  fields,
	})
}

// GetEntries returns all log entries.
func (l *MockLogger) GetEntries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries := make([]LogEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// HasEntry checks if a log entry exists.
func (l *MockLogger) HasEntry(level, msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, entry := range l.entries {
		if entry.Level == level && entry.Message == msg {
			return true
		}
	}
	return false
}

// Reset clears all log entries.
func (l *MockLogger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = []LogEntry{}
}

// MockTracer provides a mock tracer for testing.
type MockTracer struct {
	mu    sync.Mutex
	spans []SpanInfo
}

// SpanInfo represents span information.
type SpanInfo struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Tags      map[string]any
}

// NewMockTracer creates a new mock tracer.
func NewMockTracer() *MockTracer {
	return &MockTracer{
		spans: []SpanInfo{},
	}
}

// StartSpan starts a new span.
func (t *MockTracer) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	span := SpanInfo{
		Name:      name,
		StartTime: time.Now(),
		Tags:      make(map[string]any),
	}

	idx := len(t.spans)
	t.spans = append(t.spans, span)

	finish := func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.spans[idx].EndTime = time.Now()
	}

	return ctx, finish
}

// GetSpans returns all recorded spans.
func (t *MockTracer) GetSpans() []SpanInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	spans := make([]SpanInfo, len(t.spans))
	copy(spans, t.spans)
	return spans
}

// Reset clears all spans.
func (t *MockTracer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = []SpanInfo{}
}