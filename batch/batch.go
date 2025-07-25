// Package batch provides generic batch processing capabilities for pocket workflows.
package batch

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/agentstation/pocket"
)

// Processor processes a batch of items of type T.
type Processor[T, R any] struct {
	// Extract retrieves items to process.
	Extract func(ctx context.Context, store pocket.StoreReader) ([]T, error)

	// Transform processes a single item.
	Transform func(ctx context.Context, item T) (R, error)

	// Reduce combines results into a final output.
	Reduce func(ctx context.Context, results []R) (any, error)

	// Options
	maxConcurrency int
	ordered        bool
}

// Option configures a batch processor.
type Option func(*options)

type options struct {
	maxConcurrency int
	ordered        bool
}

// WithConcurrency sets the maximum concurrent workers.
func WithConcurrency(n int) Option {
	return func(o *options) {
		o.maxConcurrency = n
	}
}

// WithOrdered ensures results maintain input order.
func WithOrdered() Option {
	return func(o *options) {
		o.ordered = true
	}
}

// NewProcessor creates a new batch processor.
func NewProcessor[T, R any](
	extract func(context.Context, pocket.StoreReader) ([]T, error),
	transform func(context.Context, T) (R, error),
	reduce func(context.Context, []R) (any, error),
	opts ...Option,
) *Processor[T, R] {
	p := &Processor[T, R]{
		Extract:        extract,
		Transform:      transform,
		Reduce:         reduce,
		maxConcurrency: 10,   // default
		ordered:        true, // default
	}

	options := &options{
		maxConcurrency: p.maxConcurrency,
		ordered:        p.ordered,
	}

	for _, opt := range opts {
		opt(options)
	}

	p.maxConcurrency = options.maxConcurrency
	p.ordered = options.ordered

	return p
}

// ToNode converts the batch processor into a pocket Node.
func (p *Processor[T, R]) ToNode(name string) pocket.Node {
	return pocket.NewNode[any, any](name,
		pocket.Steps{
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				// The store is passed as prep result for exec step
				return store, nil
			},
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				// Get store from prep result
				store := prepResult.(pocket.StoreReader)

				// Extract items
				items, err := p.Extract(ctx, store)
				if err != nil {
					return nil, fmt.Errorf("extract: %w", err)
				}

				if len(items) == 0 {
					return p.Reduce(ctx, []R{})
				}

				// Process items
				results, err := p.processItems(ctx, items)
				if err != nil {
					return nil, err
				}

				// Reduce results
				return p.Reduce(ctx, results)
			},
		},
	)
}

// processItems handles concurrent or sequential processing.
func (p *Processor[T, R]) processItems(ctx context.Context, items []T) ([]R, error) {
	if p.maxConcurrency <= 1 {
		return p.processSequential(ctx, items)
	}
	return p.processConcurrent(ctx, items)
}

// processSequential processes items one by one.
func (p *Processor[T, R]) processSequential(ctx context.Context, items []T) ([]R, error) {
	results := make([]R, len(items))

	for i, item := range items {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := p.Transform(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// processConcurrent processes items with worker pool.
func (p *Processor[T, R]) processConcurrent(ctx context.Context, items []T) ([]R, error) {
	g, ctx := errgroup.WithContext(ctx)

	// Results storage
	results := make([]R, len(items))
	var mu sync.Mutex

	// Work queue
	work := make(chan int, len(items))
	for i := range items {
		work <- i
	}
	close(work)

	// Start workers
	for w := 0; w < p.maxConcurrency && w < len(items); w++ {
		g.Go(func() error {
			for idx := range work {
				result, err := p.Transform(ctx, items[idx])
				if err != nil {
					return fmt.Errorf("item %d: %w", idx, err)
				}

				mu.Lock()
				results[idx] = result
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// MapReduce creates a map-reduce batch processor as a Node.
func MapReduce[T, R any](
	name string,
	extract func(context.Context, pocket.StoreReader) ([]T, error),
	mapper func(context.Context, T) (R, error),
	reducer func(context.Context, []R) (any, error),
	opts ...Option,
) pocket.Node {
	p := NewProcessor(extract, mapper, reducer, opts...)
	return p.ToNode(name)
}

// ForEach creates a batch processor that doesn't aggregate results.
func ForEach[T any](
	name string,
	extract func(context.Context, pocket.StoreReader) ([]T, error),
	process func(context.Context, T) error,
	opts ...Option,
) pocket.Node {
	transform := func(ctx context.Context, item T) (struct{}, error) {
		return struct{}{}, process(ctx, item)
	}

	reduce := func(ctx context.Context, results []struct{}) (any, error) {
		return len(results), nil
	}

	p := NewProcessor(extract, transform, reduce, opts...)
	return p.ToNode(name)
}

// Filter creates a batch processor that filters items.
func Filter[T any](
	name string,
	extract func(context.Context, pocket.StoreReader) ([]T, error),
	predicate func(context.Context, T) (bool, error),
	opts ...Option,
) pocket.Node {
	type result struct {
		item T
		keep bool
	}

	transform := func(ctx context.Context, item T) (result, error) {
		keep, err := predicate(ctx, item)
		return result{item: item, keep: keep}, err
	}

	reduce := func(ctx context.Context, results []result) (any, error) {
		var filtered []T
		for _, r := range results {
			if r.keep {
				filtered = append(filtered, r.item)
			}
		}
		return filtered, nil
	}

	p := NewProcessor(extract, transform, reduce, opts...)
	return p.ToNode(name)
}
