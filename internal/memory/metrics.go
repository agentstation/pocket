package memory

import (
	"runtime"
	"sync"
	"time"
)

// Metrics tracks memory usage and performance metrics.
type Metrics struct {
	mu         sync.RWMutex
	samples    []Sample
	maxSamples int
	ticker     *time.Ticker
	stopCh     chan struct{}
}

// Sample represents a memory usage sample.
type Sample struct {
	Timestamp    time.Time
	Alloc        uint64 // bytes allocated and still in use
	TotalAlloc   uint64 // bytes allocated (even if freed)
	Sys          uint64 // bytes obtained from system
	NumGC        uint32 // number of completed GC cycles
	NumGoroutine int    // number of goroutines
	HeapAlloc    uint64 // bytes allocated on heap
	HeapSys      uint64 // heap obtained from system
	HeapInuse    uint64 // bytes in in-use spans
	HeapReleased uint64 // bytes released to OS
	StackInuse   uint64 // stack bytes in use
}

// NewMetrics creates a new metrics collector.
func NewMetrics(sampleInterval time.Duration, maxSamples int) *Metrics {
	m := &Metrics{
		samples:    make([]Sample, 0, maxSamples),
		maxSamples: maxSamples,
		stopCh:     make(chan struct{}),
	}

	// Start sampling
	m.ticker = time.NewTicker(sampleInterval)
	go m.collect()

	return m
}

// collect periodically samples memory metrics.
func (m *Metrics) collect() {
	for {
		select {
		case <-m.ticker.C:
			m.sample()
		case <-m.stopCh:
			return
		}
	}
}

// sample takes a memory usage sample.
func (m *Metrics) sample() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	sample := Sample{
		Timestamp:    time.Now(),
		Alloc:        stats.Alloc,
		TotalAlloc:   stats.TotalAlloc,
		Sys:          stats.Sys,
		NumGC:        stats.NumGC,
		NumGoroutine: runtime.NumGoroutine(),
		HeapAlloc:    stats.HeapAlloc,
		HeapSys:      stats.HeapSys,
		HeapInuse:    stats.HeapInuse,
		HeapReleased: stats.HeapReleased,
		StackInuse:   stats.StackInuse,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.samples = append(m.samples, sample)

	// Remove old samples if needed
	if len(m.samples) > m.maxSamples {
		m.samples = m.samples[len(m.samples)-m.maxSamples:]
	}
}

// GetSamples returns recent memory samples.
func (m *Metrics) GetSamples() []Sample {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Sample, len(m.samples))
	copy(result, m.samples)
	return result
}

// GetLatest returns the most recent sample.
func (m *Metrics) GetLatest() *Sample {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.samples) == 0 {
		return nil
	}

	latest := m.samples[len(m.samples)-1]
	return &latest
}

// GetStats calculates statistics over the sample window.
func (m *Metrics) GetStats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.samples) == 0 {
		return nil
	}

	stats := &Stats{
		SampleCount: len(m.samples),
		TimeSpan:    m.samples[len(m.samples)-1].Timestamp.Sub(m.samples[0].Timestamp),
	}

	// Calculate averages and peaks
	var totalAlloc, totalHeap, totalGoroutines uint64
	stats.PeakAlloc = m.samples[0].Alloc
	stats.PeakGoroutines = m.samples[0].NumGoroutine

	for _, s := range m.samples {
		totalAlloc += s.Alloc
		totalHeap += s.HeapAlloc
		if s.NumGoroutine >= 0 && s.NumGoroutine <= int(^uint64(0)>>1) {
			totalGoroutines += uint64(s.NumGoroutine)
		}

		if s.Alloc > stats.PeakAlloc {
			stats.PeakAlloc = s.Alloc
		}
		if s.NumGoroutine > stats.PeakGoroutines {
			stats.PeakGoroutines = s.NumGoroutine
		}
	}

	stats.AvgAlloc = totalAlloc / uint64(len(m.samples))
	stats.AvgHeapAlloc = totalHeap / uint64(len(m.samples))
	if len(m.samples) > 0 {
		avgGoroutines := totalGoroutines / uint64(len(m.samples))
		const maxInt = int(^uint(0) >> 1)
		if avgGoroutines <= uint64(maxInt) {
			stats.AvgGoroutines = int(avgGoroutines)
		} else {
			stats.AvgGoroutines = maxInt // Max int value
		}
	}

	// Calculate allocation rate
	if len(m.samples) > 1 {
		first := m.samples[0]
		last := m.samples[len(m.samples)-1]
		duration := last.Timestamp.Sub(first.Timestamp).Seconds()
		if duration > 0 {
			allocDiff := last.TotalAlloc - first.TotalAlloc
			stats.AllocRate = float64(allocDiff) / duration
		}

		// GC rate
		gcDiff := last.NumGC - first.NumGC
		stats.GCRate = float64(gcDiff) / duration
	}

	return stats
}

// Stats represents aggregated memory statistics.
type Stats struct {
	SampleCount    int
	TimeSpan       time.Duration
	AvgAlloc       uint64
	AvgHeapAlloc   uint64
	AvgGoroutines  int
	PeakAlloc      uint64
	PeakGoroutines int
	AllocRate      float64 // bytes per second
	GCRate         float64 // GCs per second
}

// Stop stops the metrics collector.
func (m *Metrics) Stop() {
	m.ticker.Stop()
	close(m.stopCh)
}

// Tracker provides memory tracking for specific operations.
type Tracker struct {
	name      string
	startMem  runtime.MemStats
	startTime time.Time
	endMem    runtime.MemStats
	endTime   time.Time
	mu        sync.Mutex
}

// NewTracker creates a new memory tracker.
func NewTracker(name string) *Tracker {
	t := &Tracker{
		name:      name,
		startTime: time.Now(),
	}
	runtime.ReadMemStats(&t.startMem)
	return t
}

// Stop stops tracking and captures final metrics.
func (t *Tracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.endTime = time.Now()
	runtime.ReadMemStats(&t.endMem)
}

// safeUint64ToInt64Diff safely calculates the difference between two uint64 values as int64.
func safeUint64ToInt64Diff(end, start uint64) int64 {
	const maxInt64 = int64(^uint64(0) >> 1) // Max positive int64 value

	if end >= start {
		diff := end - start
		if diff > uint64(maxInt64) {
			return maxInt64
		}
		return int64(diff)
	}
	// If end < start, we have a negative difference
	diff := start - end
	if diff > uint64(maxInt64) {
		return -maxInt64
	}
	return -int64(diff)
}

// Report returns a tracking report.
func (t *Tracker) Report() *TrackingReport {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If not stopped yet, use current stats
	if t.endTime.IsZero() {
		t.endTime = time.Now()
		runtime.ReadMemStats(&t.endMem)
	}

	return &TrackingReport{
		Name:            t.name,
		Duration:        t.endTime.Sub(t.startTime),
		AllocDelta:      safeUint64ToInt64Diff(t.endMem.Alloc, t.startMem.Alloc),
		TotalAllocDelta: t.endMem.TotalAlloc - t.startMem.TotalAlloc,
		NumGCDelta:      t.endMem.NumGC - t.startMem.NumGC,
		HeapAllocDelta:  safeUint64ToInt64Diff(t.endMem.HeapAlloc, t.startMem.HeapAlloc),
	}
}

// TrackingReport contains memory tracking results.
type TrackingReport struct {
	Name            string
	Duration        time.Duration
	AllocDelta      int64  // can be negative if memory was freed
	TotalAllocDelta uint64 // always increases
	NumGCDelta      uint32
	HeapAllocDelta  int64
}

// MemoryPressure monitors system memory pressure.
type MemoryPressure struct {
	mu         sync.RWMutex
	thresholds []Threshold
	handlers   map[Level][]func()
}

// Level represents memory pressure level.
type Level int

const (
	LevelNormal Level = iota
	LevelModerate
	LevelHigh
	LevelCritical
)

// Threshold defines a memory pressure threshold.
type Threshold struct {
	Level      Level
	MemoryUsed float64 // percentage of system memory
	HeapSize   uint64  // absolute heap size in bytes
}

// NewMemoryPressure creates a memory pressure monitor.
func NewMemoryPressure() *MemoryPressure {
	return &MemoryPressure{
		thresholds: []Threshold{
			{Level: LevelModerate, MemoryUsed: 70, HeapSize: 500 * 1024 * 1024},  // 500MB
			{Level: LevelHigh, MemoryUsed: 85, HeapSize: 1024 * 1024 * 1024},     // 1GB
			{Level: LevelCritical, MemoryUsed: 95, HeapSize: 2048 * 1024 * 1024}, // 2GB
		},
		handlers: make(map[Level][]func()),
	}
}

// RegisterHandler registers a handler for a pressure level.
func (p *MemoryPressure) RegisterHandler(level Level, handler func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.handlers[level] = append(p.handlers[level], handler)
}

// Check evaluates current memory pressure.
func (p *MemoryPressure) Check() Level {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	// Calculate memory usage percentage
	memUsed := float64(stats.Sys) / float64(stats.Sys+stats.HeapReleased) * 100

	p.mu.RLock()
	defer p.mu.RUnlock()

	currentLevel := LevelNormal

	for _, threshold := range p.thresholds {
		if memUsed >= threshold.MemoryUsed || stats.HeapAlloc >= threshold.HeapSize {
			currentLevel = threshold.Level
		}
	}

	// Trigger handlers for the current level
	if handlers, ok := p.handlers[currentLevel]; ok {
		for _, handler := range handlers {
			go handler()
		}
	}

	return currentLevel
}

// Monitor starts continuous monitoring.
func (p *MemoryPressure) Monitor(interval time.Duration, stopCh chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.Check()
		case <-stopCh:
			return
		}
	}
}
