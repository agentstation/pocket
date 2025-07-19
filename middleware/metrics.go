package middleware

import (
	"context"

	"github.com/agentstation/pocket"
)

// MetricsCollector collects node execution metrics.
type MetricsCollector interface {
	RecordPhaseStart(nodeName, phase string)
	RecordPhaseEnd(nodeName, phase string, err error)
	RecordRouting(nodeName, next string)
}

// Metrics adds comprehensive metrics collection to a node.
func Metrics(collector MetricsCollector) Middleware {
	return func(node pocket.Node) pocket.Node {
		return &middlewareNode{
			inner: node,
			name:  node.Name(),
			prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				collector.RecordPhaseStart(node.Name(), "prep")
				result, err := node.Prep(ctx, store, input)
				collector.RecordPhaseEnd(node.Name(), "prep", err)
				return result, err
			},
			exec: func(ctx context.Context, input any) (any, error) {
				collector.RecordPhaseStart(node.Name(), "exec")
				result, err := node.Exec(ctx, input)
				collector.RecordPhaseEnd(node.Name(), "exec", err)
				return result, err
			},
			post: func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				collector.RecordPhaseStart(node.Name(), "post")
				output, next, err := node.Post(ctx, store, input, prep, exec)
				collector.RecordPhaseEnd(node.Name(), "post", err)
				collector.RecordRouting(node.Name(), next)
				return output, next, err
			},
		}
	}
}
