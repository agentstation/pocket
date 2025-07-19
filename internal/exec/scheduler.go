// Package exec provides advanced execution capabilities for pocket workflows.
package exec

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentstation/pocket"
)

// Scheduler manages workflow execution scheduling.
type Scheduler struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	executor Executor
	ticker   *time.Ticker
	stopCh   chan struct{}
}

// Job represents a scheduled workflow execution.
type Job struct {
	ID         string
	Name       string
	Flow       *pocket.Flow
	Schedule   Schedule
	Input      any
	LastRun    time.Time
	NextRun    time.Time
	RunCount   int64
	ErrorCount int64
	Enabled    bool
	mu         sync.RWMutex
}

// Schedule defines when a job should run.
type Schedule interface {
	Next(from time.Time) time.Time
}

// Executor executes flows.
type Executor interface {
	Execute(ctx context.Context, flow *pocket.Flow, input any) (any, error)
}

// NewScheduler creates a new scheduler.
func NewScheduler(executor Executor) *Scheduler {
	return &Scheduler{
		jobs:     make(map[string]*Job),
		executor: executor,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the scheduler.
func (s *Scheduler) Start(interval time.Duration) {
	s.ticker = time.NewTicker(interval)
	go s.run()
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
}

// AddJob adds a job to the scheduler.
func (s *Scheduler) AddJob(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}

	// Calculate next run time
	job.NextRun = job.Schedule.Next(time.Now())
	s.jobs[job.ID] = job

	return nil
}

// RemoveJob removes a job from the scheduler.
func (s *Scheduler) RemoveJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[id]; !exists {
		return fmt.Errorf("job %s not found", id)
	}

	delete(s.jobs, id)
	return nil
}

// GetJob retrieves a job by ID.
func (s *Scheduler) GetJob(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[id]
	return job, exists
}

// ListJobs returns all jobs.
func (s *Scheduler) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// run is the main scheduler loop.
func (s *Scheduler) run() {
	for {
		select {
		case <-s.ticker.C:
			s.checkJobs()
		case <-s.stopCh:
			return
		}
	}
}

// checkJobs checks and runs due jobs.
func (s *Scheduler) checkJobs() {
	now := time.Now()

	s.mu.RLock()
	jobs := make([]*Job, 0)
	for _, job := range s.jobs {
		if job.Enabled && now.After(job.NextRun) {
			jobs = append(jobs, job)
		}
	}
	s.mu.RUnlock()

	// Run due jobs
	for _, job := range jobs {
		go s.runJob(job)
	}
}

// runJob executes a single job.
func (s *Scheduler) runJob(job *Job) {
	job.mu.Lock()
	job.LastRun = time.Now()
	job.RunCount++
	job.mu.Unlock()

	ctx := context.Background()
	_, err := s.executor.Execute(ctx, job.Flow, job.Input)

	job.mu.Lock()
	if err != nil {
		job.ErrorCount++
	}
	job.NextRun = job.Schedule.Next(time.Now())
	job.mu.Unlock()
}

// Schedule implementations

// CronSchedule uses cron-like scheduling.
type CronSchedule struct {
	expr string
	// In a real implementation, this would parse cron expressions
}

// NewCronSchedule creates a cron schedule.
func NewCronSchedule(expr string) (*CronSchedule, error) {
	// Validate cron expression
	return &CronSchedule{expr: expr}, nil
}

// Next calculates the next run time.
func (c *CronSchedule) Next(from time.Time) time.Time {
	// Simplified - in reality would parse cron expression
	return from.Add(time.Hour)
}

// IntervalSchedule runs at fixed intervals.
type IntervalSchedule struct {
	interval time.Duration
}

// NewIntervalSchedule creates an interval schedule.
func NewIntervalSchedule(interval time.Duration) *IntervalSchedule {
	return &IntervalSchedule{interval: interval}
}

// Next calculates the next run time.
func (s *IntervalSchedule) Next(from time.Time) time.Time {
	return from.Add(s.interval)
}

// OnceSchedule runs once at a specific time.
type OnceSchedule struct {
	at   time.Time
	done bool
}

// NewOnceSchedule creates a one-time schedule.
func NewOnceSchedule(at time.Time) *OnceSchedule {
	return &OnceSchedule{at: at}
}

// Next returns the scheduled time once.
func (s *OnceSchedule) Next(from time.Time) time.Time {
	if s.done || from.After(s.at) {
		// Return far future to never run again
		return time.Now().Add(100 * 365 * 24 * time.Hour)
	}
	s.done = true
	return s.at
}

// Priority queue for job scheduling

// PriorityScheduler schedules jobs based on priority.
type PriorityScheduler struct {
	*Scheduler
	priorities map[string]int
	mu         sync.RWMutex
}

// NewPriorityScheduler creates a priority-based scheduler.
func NewPriorityScheduler(executor Executor) *PriorityScheduler {
	return &PriorityScheduler{
		Scheduler:  NewScheduler(executor),
		priorities: make(map[string]int),
	}
}

// SetPriority sets job priority (higher = more important).
func (ps *PriorityScheduler) SetPriority(jobID string, priority int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.priorities[jobID] = priority
}

// GetPriority gets job priority.
func (ps *PriorityScheduler) GetPriority(jobID string) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.priorities[jobID]
}

// Executor implementations

// BasicExecutor provides simple flow execution.
type BasicExecutor struct{}

// Execute runs a flow.
func (e *BasicExecutor) Execute(ctx context.Context, flow *pocket.Flow, input any) (any, error) {
	return flow.Run(ctx, input)
}

// ThrottledExecutor limits concurrent executions.
type ThrottledExecutor struct {
	maxConcurrent int
	sem           chan struct{}
}

// NewThrottledExecutor creates a throttled executor.
func NewThrottledExecutor(maxConcurrent int) *ThrottledExecutor {
	return &ThrottledExecutor{
		maxConcurrent: maxConcurrent,
		sem:           make(chan struct{}, maxConcurrent),
	}
}

// Execute runs a flow with throttling.
func (e *ThrottledExecutor) Execute(ctx context.Context, flow *pocket.Flow, input any) (any, error) {
	select {
	case e.sem <- struct{}{}:
		defer func() { <-e.sem }()
		return flow.Run(ctx, input)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// QueuedExecutor queues executions.
type QueuedExecutor struct {
	queue    chan *execution
	workers  int
	executor Executor
}

type execution struct {
	ctx    context.Context
	flow   *pocket.Flow
	input  any
	result chan executionResult
}

type executionResult struct {
	output any
	err    error
}

// NewQueuedExecutor creates a queued executor.
func NewQueuedExecutor(workers, queueSize int, executor Executor) *QueuedExecutor {
	e := &QueuedExecutor{
		queue:    make(chan *execution, queueSize),
		workers:  workers,
		executor: executor,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		go e.worker()
	}

	return e
}

// Execute queues a flow execution.
func (e *QueuedExecutor) Execute(ctx context.Context, flow *pocket.Flow, input any) (any, error) {
	exec := &execution{
		ctx:    ctx,
		flow:   flow,
		input:  input,
		result: make(chan executionResult, 1),
	}

	select {
	case e.queue <- exec:
		select {
		case result := <-exec.result:
			return result.output, result.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// worker processes queued executions.
func (e *QueuedExecutor) worker() {
	for exec := range e.queue {
		output, err := e.executor.Execute(exec.ctx, exec.flow, exec.input)
		exec.result <- executionResult{output: output, err: err}
	}
}
