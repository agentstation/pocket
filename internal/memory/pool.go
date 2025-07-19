// Package memory provides memory management utilities for pocket workflows.
package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// NodePool provides object pooling for nodes to reduce allocations.
type NodePool struct {
	pools map[string]*sync.Pool
	mu    sync.RWMutex
	stats *PoolStats
}

// PoolStats tracks pool usage statistics.
type PoolStats struct {
	mu    sync.RWMutex
	gets  map[string]int64
	puts  map[string]int64
	news  map[string]int64
	inUse map[string]int64
}

// NewNodePool creates a new node pool.
func NewNodePool() *NodePool {
	return &NodePool{
		pools: make(map[string]*sync.Pool),
		stats: &PoolStats{
			gets:  make(map[string]int64),
			puts:  make(map[string]int64),
			news:  make(map[string]int64),
			inUse: make(map[string]int64),
		},
	}
}

// Register registers a node type with a factory function.
func (p *NodePool) Register(nodeType string, factory func() *pocket.Node) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pools[nodeType] = &sync.Pool{
		New: func() interface{} {
			p.stats.mu.Lock()
			p.stats.news[nodeType]++
			p.stats.mu.Unlock()
			return factory()
		},
	}
}

// Get retrieves a node from the pool.
func (p *NodePool) Get(nodeType string) (*pocket.Node, bool) {
	p.mu.RLock()
	pool, exists := p.pools[nodeType]
	p.mu.RUnlock()

	if !exists {
		return nil, false
	}

	p.stats.mu.Lock()
	p.stats.gets[nodeType]++
	p.stats.inUse[nodeType]++
	p.stats.mu.Unlock()

	node := pool.Get().(*pocket.Node)
	return node, true
}

// Put returns a node to the pool.
func (p *NodePool) Put(nodeType string, node *pocket.Node) {
	p.mu.RLock()
	pool, exists := p.pools[nodeType]
	p.mu.RUnlock()

	if !exists {
		return
	}

	// Reset node state before returning to pool
	resetNode(node)

	p.stats.mu.Lock()
	p.stats.puts[nodeType]++
	p.stats.inUse[nodeType]--
	p.stats.mu.Unlock()

	pool.Put(node)
}

// GetStats returns current pool statistics.
func (p *NodePool) GetStats() map[string]NodePoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	stats := make(map[string]NodePoolStats)
	for nodeType := range p.pools {
		stats[nodeType] = NodePoolStats{
			Gets:  p.stats.gets[nodeType],
			Puts:  p.stats.puts[nodeType],
			News:  p.stats.news[nodeType],
			InUse: p.stats.inUse[nodeType],
		}
	}
	return stats
}

// NodePoolStats represents statistics for a single node type.
type NodePoolStats struct {
	Gets  int64
	Puts  int64
	News  int64
	InUse int64
}

// resetNode clears node state for reuse.
func resetNode(node *pocket.Node) {
	// In the new API, we cannot access internal fields of nodes
	// Node reuse is not recommended as nodes are lightweight
	// and should be created fresh for each use
}

// BufferPool manages byte buffer pooling.
type BufferPool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
}

// NewBufferPool creates a new buffer pool.
func NewBufferPool() *BufferPool {
	bp := &BufferPool{
		pools: make(map[int]*sync.Pool),
	}

	// Pre-create pools for common sizes
	sizes := []int{512, 1024, 4096, 8192, 16384}
	for _, size := range sizes {
		s := size // capture for closure
		bp.pools[size] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, s)
			},
		}
	}

	return bp
}

// Get retrieves a buffer of at least the requested size.
func (b *BufferPool) Get(size int) []byte {
	// Find the smallest pool that fits
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, poolSize := range []int{512, 1024, 4096, 8192, 16384} {
		if poolSize >= size {
			if pool, ok := b.pools[poolSize]; ok {
				buf := pool.Get().([]byte)
				return buf[:size]
			}
		}
	}

	// No suitable pool, allocate new
	return make([]byte, size)
}

// Put returns a buffer to the pool.
func (b *BufferPool) Put(buf []byte) {
	size := cap(buf)

	b.mu.RLock()
	pool, ok := b.pools[size]
	b.mu.RUnlock()

	if ok {
		// Clear buffer before returning
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(&buf)
	}
}

// ResultPool pools result objects to reduce allocations.
type ResultPool[T any] struct {
	pool  *sync.Pool
	reset func(*T)
}

// NewResultPool creates a typed result pool.
func NewResultPool[T any](factory func() *T, reset func(*T)) *ResultPool[T] {
	return &ResultPool[T]{
		pool: &sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
		reset: reset,
	}
}

// Get retrieves a result from the pool.
func (p *ResultPool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put returns a result to the pool.
func (p *ResultPool[T]) Put(result *T) {
	if p.reset != nil {
		p.reset(result)
	}
	p.pool.Put(result)
}

// MemoryManager provides centralized memory management.
type MemoryManager struct {
	nodePool    *NodePool
	bufferPool  *BufferPool
	resultPools map[string]interface{}
	mu          sync.RWMutex
	gcInterval  time.Duration
	stopCh      chan struct{}
}

// NewMemoryManager creates a new memory manager.
func NewMemoryManager() *MemoryManager {
	m := &MemoryManager{
		nodePool:    NewNodePool(),
		bufferPool:  NewBufferPool(),
		resultPools: make(map[string]interface{}),
		gcInterval:  5 * time.Minute,
		stopCh:      make(chan struct{}),
	}

	// Start periodic GC
	go m.gcLoop()

	return m
}

// RegisterResultPool registers a typed result pool.
func (m *MemoryManager) RegisterResultPool(name string, pool interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resultPools[name] = pool
}

// GetResultPool retrieves a result pool by name.
func (m *MemoryManager) GetResultPool(name string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pool, ok := m.resultPools[name]
	return pool, ok
}

// NodePool returns the node pool.
func (m *MemoryManager) NodePool() *NodePool {
	return m.nodePool
}

// BufferPool returns the buffer pool.
func (m *MemoryManager) BufferPool() *BufferPool {
	return m.bufferPool
}

// gcLoop runs periodic garbage collection.
func (m *MemoryManager) gcLoop() {
	ticker := time.NewTicker(m.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Force GC to return memory to OS
			m.gc()
		case <-m.stopCh:
			return
		}
	}
}

// gc performs garbage collection tasks.
func (m *MemoryManager) gc() {
	// This is where we could implement more sophisticated
	// memory management strategies, such as:
	// - Clearing pools that haven't been used recently
	// - Compacting memory
	// - Adjusting pool sizes based on usage patterns
}

// Stop stops the memory manager.
func (m *MemoryManager) Stop() {
	close(m.stopCh)
}

// PooledNode wraps a node with automatic pooling.
type PooledNode struct {
	*pocket.Node
	pool     *NodePool
	nodeType string
}

// NewPooledNode creates a node that returns to the pool when done.
func NewPooledNode(pool *NodePool, nodeType string) (*PooledNode, error) {
	node, ok := pool.Get(nodeType)
	if !ok {
		return nil, fmt.Errorf("node type %s not registered", nodeType)
	}

	return &PooledNode{
		Node:     node,
		pool:     pool,
		nodeType: nodeType,
	}, nil
}

// Release returns the node to the pool.
func (n *PooledNode) Release() {
	n.pool.Put(n.nodeType, n.Node)
}

// WithPooling creates a node wrapper that uses pooling.
func WithPooling(pool *NodePool, nodeType string, fn func(*pocket.Node) error) error {
	node, err := NewPooledNode(pool, nodeType)
	if err != nil {
		return err
	}
	defer node.Release()

	return fn(node.Node)
}
