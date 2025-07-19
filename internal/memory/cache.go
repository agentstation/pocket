package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Cache provides result caching for nodes.
type Cache interface {
	Get(key string) (value any, exists bool)
	Set(key string, value any, ttl time.Duration)
	Delete(key string)
	Clear()
	Stats() CacheStats
}

// CacheStats contains cache statistics.
type CacheStats struct {
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
	Size      int
	MaxSize   int
}

// LRUCache implements an LRU cache with TTL support.
type LRUCache struct {
	mu      sync.RWMutex
	maxSize int
	entries map[string]*cacheEntry
	head    *cacheEntry
	tail    *cacheEntry
	stats   CacheStats
}

type cacheEntry struct {
	key        string
	value      any
	expiry     time.Time
	prev, next *cacheEntry
}

// NewLRUCache creates a new LRU cache.
func NewLRUCache(maxSize int) *LRUCache {
	c := &LRUCache{
		maxSize: maxSize,
		entries: make(map[string]*cacheEntry),
		stats:   CacheStats{MaxSize: maxSize},
	}

	// Create sentinel nodes
	c.head = &cacheEntry{}
	c.tail = &cacheEntry{}
	c.head.next = c.tail
	c.tail.prev = c.head

	return c
}

// Get retrieves a value from the cache.
func (c *LRUCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check expiry
	if time.Now().After(entry.expiry) {
		c.removeEntry(entry)
		c.stats.Misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.moveToFront(entry)
	c.stats.Hits++
	return entry.value, true
}

// Set stores a value in the cache.
func (c *LRUCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.Sets++

	// Check if key exists
	if entry, exists := c.entries[key]; exists {
		entry.value = value
		entry.expiry = time.Now().Add(ttl)
		c.moveToFront(entry)
		return
	}

	// Create new entry
	entry := &cacheEntry{
		key:    key,
		value:  value,
		expiry: time.Now().Add(ttl),
	}

	c.entries[key] = entry
	c.addToFront(entry)
	c.stats.Size++

	// Evict if necessary
	if c.stats.Size > c.maxSize {
		c.evictOldest()
	}
}

// Delete removes a key from the cache.
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.entries[key]; exists {
		c.removeEntry(entry)
		c.stats.Deletes++
	}
}

// Clear removes all entries from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
	c.head.next = c.tail
	c.tail.prev = c.head
	c.stats.Size = 0
}

// Stats returns cache statistics.
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *LRUCache) removeEntry(entry *cacheEntry) {
	delete(c.entries, entry.key)
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
	c.stats.Size--
}

func (c *LRUCache) moveToFront(entry *cacheEntry) {
	// Remove from current position
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
	// Add to front
	c.addToFront(entry)
}

func (c *LRUCache) addToFront(entry *cacheEntry) {
	entry.next = c.head.next
	entry.prev = c.head
	c.head.next.prev = entry
	c.head.next = entry
}

func (c *LRUCache) evictOldest() {
	oldest := c.tail.prev
	if oldest != c.head {
		c.removeEntry(oldest)
		c.stats.Evictions++
	}
}

// CachedNode wraps a node with caching.
type CachedNode struct {
	*pocket.Node
	cache   Cache
	keyFunc func(input any) string
	ttl     time.Duration
}

// NewCachedNode creates a node with caching.
func NewCachedNode(node *pocket.Node, cache Cache, keyFunc func(any) string, ttl time.Duration) *CachedNode {
	return &CachedNode{
		Node:    node,
		cache:   cache,
		keyFunc: keyFunc,
		ttl:     ttl,
	}
}

// Execute runs the node with caching.
func (n *CachedNode) Execute(ctx context.Context, store pocket.Store, input any) (any, error) {
	key := n.keyFunc(input)

	// Check cache
	if cached, exists := n.cache.Get(key); exists {
		// Store cache hit metadata
		_ = store.Set(ctx, fmt.Sprintf("node:%s:cache_hit", n.Name), true)
		return cached, nil
	}

	// Execute node
	graph := pocket.NewGraph(n.Node, store)
	result, err := graph.Run(ctx, input)
	if err != nil {
		return nil, err
	}

	// Cache result
	n.cache.Set(key, result, n.ttl)
	_ = store.Set(ctx, fmt.Sprintf("node:%s:cache_miss", n.Name), true)

	return result, nil
}

// CacheMiddleware creates a caching middleware for nodes.
func CacheMiddleware(cache Cache, keyFunc func(any) string, ttl time.Duration) func(*pocket.Node) *pocket.Node {
	return func(node *pocket.Node) *pocket.Node {
		originalExec := node.Exec

		node.Exec = func(ctx context.Context, input any) (any, error) {
			key := keyFunc(input)

			// Check cache
			if cached, exists := cache.Get(key); exists {
				return cached, nil
			}

			// Execute original
			result, err := originalExec(ctx, input)
			if err == nil {
				cache.Set(key, result, ttl)
			}

			return result, err
		}

		return node
	}
}

// HashKeyFunc creates a key function that hashes the input.
func HashKeyFunc(prefix string) func(any) string {
	return func(input any) string {
		h := sha256.New()
		_, _ = fmt.Fprintf(h, "%v", input)
		return prefix + ":" + hex.EncodeToString(h.Sum(nil))
	}
}

// CompositeKeyFunc combines multiple key functions.
func CompositeKeyFunc(funcs ...func(any) string) func(any) string {
	return func(input any) string {
		parts := make([]string, len(funcs))
		for i, fn := range funcs {
			parts[i] = fn(input)
		}
		return fmt.Sprintf("%s", parts)
	}
}

// TTLCache implements a simple TTL-only cache.
type TTLCache struct {
	mu      sync.RWMutex
	entries map[string]*ttlEntry
	stats   CacheStats
}

type ttlEntry struct {
	value  any
	expiry time.Time
}

// NewTTLCache creates a new TTL cache.
func NewTTLCache() *TTLCache {
	c := &TTLCache{
		entries: make(map[string]*ttlEntry),
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

// Get retrieves a value from the cache.
func (c *TTLCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	if time.Now().After(entry.expiry) {
		c.stats.Misses++
		return nil, false
	}

	c.stats.Hits++
	return entry.value, true
}

// Set stores a value in the cache.
func (c *TTLCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &ttlEntry{
		value:  value,
		expiry: time.Now().Add(ttl),
	}
	c.stats.Sets++
	c.stats.Size = len(c.entries)
}

// Delete removes a key from the cache.
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		delete(c.entries, key)
		c.stats.Deletes++
		c.stats.Size = len(c.entries)
	}
}

// Clear removes all entries.
func (c *TTLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*ttlEntry)
	c.stats.Size = 0
}

// Stats returns cache statistics.
func (c *TTLCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// cleanup periodically removes expired entries.
func (c *TTLCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.After(entry.expiry) {
				delete(c.entries, key)
				c.stats.Evictions++
			}
		}
		c.stats.Size = len(c.entries)
		c.mu.Unlock()
	}
}
