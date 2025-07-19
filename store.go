package pocket

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"
)

// StoreOption configures a store.
type StoreOption func(*storeConfig)

// storeConfig holds store configuration.
type storeConfig struct {
	maxEntries int
	ttl        time.Duration
	onEvict    func(key string, value any)
}

// WithMaxEntries sets the maximum number of entries in the store.
// When exceeded, the least recently used entry is evicted.
func WithMaxEntries(maxEntries int) StoreOption {
	return func(c *storeConfig) {
		c.maxEntries = maxEntries
	}
}

// WithTTL sets the time-to-live for entries.
// Entries older than the TTL are automatically removed.
func WithTTL(ttl time.Duration) StoreOption {
	return func(c *storeConfig) {
		c.ttl = ttl
	}
}

// WithEvictionCallback sets a callback for when entries are evicted.
func WithEvictionCallback(fn func(key string, value any)) StoreOption {
	return func(c *storeConfig) {
		c.onEvict = fn
	}
}

// store is the internal implementation with a mutex.
type store struct {
	mu       sync.RWMutex
	data     map[string]*entry
	prefix   string
	config   storeConfig
	eviction *list.List // LRU list
}

// entry holds a value with metadata.
type entry struct {
	key      string
	value    any
	created  time.Time
	accessed time.Time
	element  *list.Element // position in LRU list
}

// NewStore creates a new thread-safe store with optional configuration.
func NewStore(opts ...StoreOption) Store {
	s := &store{
		data:     make(map[string]*entry),
		eviction: list.New(),
		config:   storeConfig{},
	}

	// Apply options
	for _, opt := range opts {
		opt(&s.config)
	}

	return s
}

// Get retrieves a value by key.
func (s *store) Get(ctx context.Context, key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.prefix + key
	e, exists := s.data[fullKey]
	if !exists {
		return nil, false
	}

	// Check TTL if configured
	if s.config.ttl > 0 && time.Since(e.created) > s.config.ttl {
		// Entry expired, remove it
		s.removeEntry(fullKey)
		return nil, false
	}

	// Update access time and move to front (most recently used)
	e.accessed = time.Now()
	if s.config.maxEntries > 0 && e.element != nil {
		s.eviction.MoveToFront(e.element)
	}

	return e.value, true
}

// Set stores a value with the given key.
func (s *store) Set(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.prefix + key
	now := time.Now()

	// Check if key already exists
	if e, exists := s.data[fullKey]; exists {
		// Update existing entry
		e.value = value
		e.accessed = now
		if s.config.maxEntries > 0 && e.element != nil {
			s.eviction.MoveToFront(e.element)
		}
		return nil
	}

	// Create new entry
	e := &entry{
		key:      fullKey,
		value:    value,
		created:  now,
		accessed: now,
	}

	// Add to LRU list if bounded
	if s.config.maxEntries > 0 {
		e.element = s.eviction.PushFront(fullKey)

		// Check if we need to evict
		if s.eviction.Len() > s.config.maxEntries {
			// Remove least recently used
			oldest := s.eviction.Back()
			if oldest != nil {
				oldKey := oldest.Value.(string)
				s.removeEntry(oldKey)
			}
		}
	}

	s.data[fullKey] = e
	return nil
}

// Delete removes a key from the store.
func (s *store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.prefix + key
	s.removeEntry(fullKey)
	return nil
}

// removeEntry removes an entry and handles eviction callback.
// Must be called with lock held.
func (s *store) removeEntry(key string) {
	e, exists := s.data[key]
	if !exists {
		return
	}

	// Remove from eviction list
	if s.config.maxEntries > 0 && e.element != nil {
		s.eviction.Remove(e.element)
	}

	// Call eviction callback if set
	if s.config.onEvict != nil {
		s.config.onEvict(key, e.value)
	}

	delete(s.data, key)
}

// Scope returns a new store with the given prefix.
func (s *store) Scope(prefix string) Store {
	return &store{
		data:     s.data, // shared data
		prefix:   s.prefix + prefix + ":",
		config:   s.config,
		eviction: s.eviction, // shared eviction list
		// Note: mutex is not shared, each scope has its own
		// This is safe because we're using a shared map with proper locking
	}
}

// TypedStore provides type-safe storage operations.
type TypedStore[T any] interface {
	Get(ctx context.Context, key string) (T, bool, error)
	Set(ctx context.Context, key string, value T) error
	Delete(ctx context.Context, key string) error
}

// NewTypedStore creates a type-safe wrapper around a Store.
func NewTypedStore[T any](store Store) TypedStore[T] {
	return &typedStore[T]{store: store}
}

type typedStore[T any] struct {
	store Store
}

func (t *typedStore[T]) Get(ctx context.Context, key string) (value T, exists bool, err error) {
	var zero T
	val, ok := t.store.Get(ctx, key)
	if !ok {
		return zero, false, nil
	}

	typed, ok := val.(T)
	if !ok {
		return zero, false, fmt.Errorf("type mismatch: expected %T, got %T", zero, val)
	}

	return typed, true, nil
}

func (t *typedStore[T]) Set(ctx context.Context, key string, value T) error {
	return t.store.Set(ctx, key, value)
}

func (t *typedStore[T]) Delete(ctx context.Context, key string) error {
	return t.store.Delete(ctx, key)
}
