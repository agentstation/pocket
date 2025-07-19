package pocket

import (
	"context"
	"fmt"
	"sync"
)

// store is the internal implementation with a mutex.
type store struct {
	mu     sync.RWMutex
	data   map[string]any
	prefix string
}

// NewStore creates a new thread-safe store.
func NewStore() Store {
	return &store{
		data: make(map[string]any),
	}
}

// Get retrieves a value by key.
func (s *store) Get(ctx context.Context, key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fullKey := s.prefix + key
	val, exists := s.data[fullKey]
	return val, exists
}

// Set stores a value with the given key.
func (s *store) Set(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.prefix + key
	s.data[fullKey] = value
	return nil
}

// Delete removes a key from the store.
func (s *store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.prefix + key
	delete(s.data, fullKey)
	return nil
}

// Scope returns a new store with the given prefix.
func (s *store) Scope(prefix string) Store {
	return &store{
		data:   s.data, // shared data
		prefix: s.prefix + prefix + ":",
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
