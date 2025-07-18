package pocket

import (
	"context"
	"sync"
)

// NewStore creates a new thread-safe store.
func NewStore() Store {
	return &SyncStore{}
}

// SyncStore is a thread-safe implementation of Store using sync.Map.
type SyncStore struct {
	data sync.Map
}

// Get retrieves a value by key.
func (s *SyncStore) Get(key string) (any, bool) {
	return s.data.Load(key)
}

// Set stores a value with the given key.
func (s *SyncStore) Set(key string, value any) {
	s.data.Store(key, value)
}

// Delete removes a key from the store.
func (s *SyncStore) Delete(key string) {
	s.data.Delete(key)
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

func (t *typedStore[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var zero T
	value, exists := t.store.Get(key)
	if !exists {
		return zero, false, nil
	}

	typed, ok := value.(T)
	if !ok {
		return zero, false, ErrInvalidInput
	}

	return typed, true, nil
}

func (t *typedStore[T]) Set(ctx context.Context, key string, value T) error {
	t.store.Set(key, value)
	return nil
}

func (t *typedStore[T]) Delete(ctx context.Context, key string) error {
	t.store.Delete(key)
	return nil
}

// ScopedStore provides isolated storage with a prefix.
type ScopedStore struct {
	store  Store
	prefix string
}

// NewScopedStore creates a store with key prefixing.
func NewScopedStore(store Store, prefix string) *ScopedStore {
	return &ScopedStore{
		store:  store,
		prefix: prefix,
	}
}

func (s *ScopedStore) key(k string) string {
	return s.prefix + ":" + k
}

func (s *ScopedStore) Get(key string) (any, bool) {
	return s.store.Get(s.key(key))
}

func (s *ScopedStore) Set(key string, value any) {
	s.store.Set(s.key(key), value)
}

func (s *ScopedStore) Delete(key string) {
	s.store.Delete(s.key(key))
}
