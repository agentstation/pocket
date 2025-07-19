// Package store provides enhanced store implementations for pocket workflows.
package store

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// EvictionPolicy defines how entries are evicted from a bounded store.
type EvictionPolicy string

const (
	// LRU evicts the least recently used entry.
	LRU EvictionPolicy = "lru"
	// LFU evicts the least frequently used entry.
	LFU EvictionPolicy = "lfu"
	// FIFO evicts the oldest entry.
	FIFO EvictionPolicy = "fifo"
	// TTL evicts entries based on time-to-live.
	TTL EvictionPolicy = "ttl"
)

// BoundedStore implements a store with size limits and eviction policies.
type BoundedStore struct {
	mu          sync.RWMutex
	data        map[string]*entry
	evictList   *list.List
	maxEntries  int
	maxSize     int64
	currentSize int64
	policy      EvictionPolicy
	ttl         time.Duration
	onEvict     func(key string, value any)
	prefix      string
}

type entry struct {
	key        string
	value      any
	size       int64
	element    *list.Element
	accessTime time.Time
	createTime time.Time
	accessCount int64
}

// BoundedStoreOption configures a BoundedStore.
type BoundedStoreOption func(*BoundedStore)

// WithMaxEntries sets the maximum number of entries.
func WithMaxEntries(maxEntries int) BoundedStoreOption {
	return func(s *BoundedStore) {
		s.maxEntries = maxEntries
	}
}

// WithMaxSize sets the maximum total size in bytes.
func WithMaxSize(maxSize int64) BoundedStoreOption {
	return func(s *BoundedStore) {
		s.maxSize = maxSize
	}
}

// WithEvictionPolicy sets the eviction policy.
func WithEvictionPolicy(policy EvictionPolicy) BoundedStoreOption {
	return func(s *BoundedStore) {
		s.policy = policy
	}
}

// WithTTL sets the time-to-live for entries.
func WithTTL(ttl time.Duration) BoundedStoreOption {
	return func(s *BoundedStore) {
		s.ttl = ttl
	}
}

// WithEvictionCallback sets a callback for evicted entries.
func WithEvictionCallback(fn func(key string, value any)) BoundedStoreOption {
	return func(s *BoundedStore) {
		s.onEvict = fn
	}
}

// NewBoundedStore creates a new bounded store.
func NewBoundedStore(opts ...BoundedStoreOption) *BoundedStore {
	s := &BoundedStore{
		data:       make(map[string]*entry),
		evictList:  list.New(),
		maxEntries: 1000,
		maxSize:    0, // 0 means no size limit
		policy:     LRU,
		ttl:        0, // 0 means no TTL
	}

	for _, opt := range opts {
		opt(s)
	}

	// Start TTL cleaner if TTL is set
	if s.ttl > 0 {
		go s.ttlCleaner()
	}

	return s
}

// Get retrieves a value by key.
func (s *BoundedStore) Get(ctx context.Context, key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.fullKey(key)
	ent, exists := s.data[fullKey]
	if !exists {
		return nil, false
	}

	// Check TTL
	if s.ttl > 0 && time.Since(ent.createTime) > s.ttl {
		s.removeEntry(fullKey)
		return nil, false
	}

	// Update access info
	ent.accessTime = time.Now()
	ent.accessCount++

	// Move to front for LRU
	if s.policy == LRU {
		s.evictList.MoveToFront(ent.element)
	}

	return ent.value, true
}

// Set stores a value with the given key.
func (s *BoundedStore) Set(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.fullKey(key)
	size := s.estimateSize(value)

	// Check if we need to make room
	if s.maxSize > 0 && size > s.maxSize {
		return errors.New("value too large for store")
	}

	// Update existing entry
	if ent, exists := s.data[fullKey]; exists {
		oldSize := ent.size
		ent.value = value
		ent.size = size
		ent.accessTime = time.Now()
		ent.accessCount++
		s.currentSize += size - oldSize

		if s.policy == LRU {
			s.evictList.MoveToFront(ent.element)
		}

		s.enforceLimit()
		return nil
	}

	// Create new entry
	ent := &entry{
		key:         fullKey,
		value:       value,
		size:        size,
		accessTime:  time.Now(),
		createTime:  time.Now(),
		accessCount: 1,
	}

	// Add to eviction list
	switch s.policy {
	case LRU, FIFO, TTL:
		ent.element = s.evictList.PushFront(ent)
	case LFU:
		// For LFU, we maintain order by access count
		s.insertLFU(ent)
	}

	s.data[fullKey] = ent
	s.currentSize += size

	// Enforce limits
	s.enforceLimit()

	return nil
}

// Delete removes a key from the store.
func (s *BoundedStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullKey := s.fullKey(key)
	s.removeEntry(fullKey)
	return nil
}

// Scope returns a new store with the given prefix.
func (s *BoundedStore) Scope(prefix string) pocket.Store {
	return &BoundedStore{
		data:       s.data,
		evictList:  s.evictList,
		maxEntries: s.maxEntries,
		maxSize:    s.maxSize,
		policy:     s.policy,
		ttl:        s.ttl,
		onEvict:    s.onEvict,
		prefix:     s.fullKey(prefix),
	}
}

// fullKey returns the full key with prefix.
func (s *BoundedStore) fullKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + ":" + key
}

// enforceLimit ensures the store stays within configured limits.
func (s *BoundedStore) enforceLimit() {
	// Enforce entry count limit
	for s.maxEntries > 0 && len(s.data) > s.maxEntries {
		s.evictOne()
	}

	// Enforce size limit
	for s.maxSize > 0 && s.currentSize > s.maxSize {
		s.evictOne()
	}
}

// evictOne evicts a single entry based on the policy.
func (s *BoundedStore) evictOne() {
	if s.evictList.Len() == 0 {
		return
	}

	var toEvict *entry

	switch s.policy {
	case LRU, FIFO, TTL:
		// Evict from back of list
		elem := s.evictList.Back()
		if elem != nil {
			toEvict = elem.Value.(*entry)
		}
	case LFU:
		// Find least frequently used
		toEvict = s.findLFU()
	}

	if toEvict != nil {
		s.removeEntry(toEvict.key)
	}
}

// removeEntry removes an entry from the store.
func (s *BoundedStore) removeEntry(key string) {
	ent, exists := s.data[key]
	if !exists {
		return
	}

	delete(s.data, key)
	s.currentSize -= ent.size

	if ent.element != nil {
		s.evictList.Remove(ent.element)
	}

	if s.onEvict != nil {
		s.onEvict(key, ent.value)
	}
}

// insertLFU inserts an entry maintaining LFU order.
func (s *BoundedStore) insertLFU(ent *entry) {
	// Find the correct position based on access count
	for e := s.evictList.Back(); e != nil; e = e.Prev() {
		if e.Value.(*entry).accessCount <= ent.accessCount {
			ent.element = s.evictList.InsertAfter(ent, e)
			return
		}
	}
	// If we get here, this is the least frequently used
	ent.element = s.evictList.PushFront(ent)
}

// findLFU finds the least frequently used entry.
func (s *BoundedStore) findLFU() *entry {
	var lfu *entry
	var minCount int64 = -1

	for _, ent := range s.data {
		if minCount == -1 || ent.accessCount < minCount {
			lfu = ent
			minCount = ent.accessCount
		}
	}

	return lfu
}

// estimateSize estimates the size of a value in bytes.
func (s *BoundedStore) estimateSize(value any) int64 {
	// This is a simplified estimation
	// In production, you might want more sophisticated size calculation
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case int, int32, int64, uint, uint32, uint64:
		return 8
	case bool:
		return 1
	case float32:
		return 4
	case float64:
		return 8
	default:
		// For complex types, use a default size
		return 64
	}
}

// ttlCleaner periodically removes expired entries.
func (s *BoundedStore) ttlCleaner() {
	ticker := time.NewTicker(s.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanExpired()
	}
}

// cleanExpired removes all expired entries.
func (s *BoundedStore) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for key, ent := range s.data {
		if now.Sub(ent.createTime) > s.ttl {
			toRemove = append(toRemove, key)
		}
	}

	for _, key := range toRemove {
		s.removeEntry(key)
	}
}

// GetStats returns store statistics.
func (s *BoundedStore) GetStats() StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := StoreStats{
		Entries:     len(s.data),
		CurrentSize: s.currentSize,
		MaxEntries:  s.maxEntries,
		MaxSize:     s.maxSize,
		Policy:      string(s.policy),
	}

	// Calculate access statistics
	for _, ent := range s.data {
		stats.TotalAccesses += ent.accessCount
		if stats.OldestEntry.IsZero() || ent.createTime.Before(stats.OldestEntry) {
			stats.OldestEntry = ent.createTime
		}
		if ent.accessTime.After(stats.LastAccess) {
			stats.LastAccess = ent.accessTime
		}
	}

	return stats
}

// StoreStats contains store statistics.
type StoreStats struct {
	Entries       int
	CurrentSize   int64
	MaxEntries    int
	MaxSize       int64
	Policy        string
	TotalAccesses int64
	OldestEntry   time.Time
	LastAccess    time.Time
}

// MultiTieredStore implements a multi-tiered storage system.
type MultiTieredStore struct {
	tiers      []pocket.Store
	// tierNames  []string // TODO: Remove if not needed
	promotions map[string]int // tracks which tier each key is in
	mu         sync.RWMutex
}

// NewMultiTieredStore creates a store with multiple tiers.
func NewMultiTieredStore(tiers ...pocket.Store) *MultiTieredStore {
	return &MultiTieredStore{
		tiers:      tiers,
		promotions: make(map[string]int),
	}
}

// Get retrieves from the highest tier containing the key.
func (m *MultiTieredStore) Get(ctx context.Context, key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check each tier in order
	for i, tier := range m.tiers {
		if value, exists := tier.Get(ctx, key); exists {
			// Promote to higher tier if not already in the highest
			if i > 0 {
				go m.promote(ctx, key, value, i)
			}
			return value, true
		}
	}

	return nil, false
}

// Set stores in the first tier.
func (m *MultiTieredStore) Set(ctx context.Context, key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Always write to the first tier
	if err := m.tiers[0].Set(ctx, key, value); err != nil {
		return err
	}

	m.promotions[key] = 0
	return nil
}

// Delete removes from all tiers.
func (m *MultiTieredStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for _, tier := range m.tiers {
		if err := tier.Delete(ctx, key); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	delete(m.promotions, key)
	return firstErr
}

// Scope returns a scoped multi-tiered store.
func (m *MultiTieredStore) Scope(prefix string) pocket.Store {
	scopedTiers := make([]pocket.Store, len(m.tiers))
	for i, tier := range m.tiers {
		scopedTiers[i] = tier.Scope(prefix)
	}
	return NewMultiTieredStore(scopedTiers...)
}

// promote moves a value to a higher tier.
func (m *MultiTieredStore) promote(ctx context.Context, key string, value any, currentTier int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Promote to the tier above
	targetTier := currentTier - 1
	if targetTier >= 0 {
		_ = m.tiers[targetTier].Set(ctx, key, value)
		m.promotions[key] = targetTier
	}
}

// ShardedStore distributes data across multiple stores.
type ShardedStore struct {
	shards    []pocket.Store
	numShards int
	hashFunc  func(string) int
	// mu        sync.RWMutex // TODO: Remove if not needed
}

// NewShardedStore creates a store that shards data.
func NewShardedStore(numShards int) *ShardedStore {
	shards := make([]pocket.Store, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = pocket.NewStore()
	}

	return &ShardedStore{
		shards:    shards,
		numShards: numShards,
		hashFunc:  defaultHashFunc,
	}
}

// Get retrieves from the appropriate shard.
func (s *ShardedStore) Get(ctx context.Context, key string) (any, bool) {
	shard := s.getShard(key)
	return shard.Get(ctx, key)
}

// Set stores in the appropriate shard.
func (s *ShardedStore) Set(ctx context.Context, key string, value any) error {
	shard := s.getShard(key)
	return shard.Set(ctx, key, value)
}

// Delete removes from the appropriate shard.
func (s *ShardedStore) Delete(ctx context.Context, key string) error {
	shard := s.getShard(key)
	return shard.Delete(ctx, key)
}

// Scope returns a scoped sharded store.
func (s *ShardedStore) Scope(prefix string) pocket.Store {
	return &scopedShardedStore{
		base:   s,
		prefix: prefix,
	}
}

// getShard determines which shard to use for a key.
func (s *ShardedStore) getShard(key string) pocket.Store {
	index := s.hashFunc(key) % s.numShards
	return s.shards[index]
}

// defaultHashFunc is a simple hash function for sharding.
func defaultHashFunc(key string) int {
	hash := 0
	for _, c := range key {
		hash = (hash * 31) + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// scopedShardedStore wraps a sharded store with a scope.
type scopedShardedStore struct {
	base   *ShardedStore
	prefix string
}

func (s *scopedShardedStore) Get(ctx context.Context, key string) (any, bool) {
	return s.base.Get(ctx, s.fullKey(key))
}

func (s *scopedShardedStore) Set(ctx context.Context, key string, value any) error {
	return s.base.Set(ctx, s.fullKey(key), value)
}

func (s *scopedShardedStore) Delete(ctx context.Context, key string) error {
	return s.base.Delete(ctx, s.fullKey(key))
}

func (s *scopedShardedStore) Scope(prefix string) pocket.Store {
	return &scopedShardedStore{
		base:   s.base,
		prefix: s.fullKey(prefix),
	}
}

func (s *scopedShardedStore) fullKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", s.prefix, key)
}