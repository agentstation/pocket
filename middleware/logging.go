package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/agentstation/pocket"
)

// Logging adds structured logging to a node's lifecycle.
func Logging(logger pocket.Logger) Middleware {
	return func(node pocket.Node) pocket.Node {
		return &middlewareNode{
			inner: node,
			name:  node.Name(),
			prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				logger.Debug(ctx, "node prep starting", "node", node.Name(), "input_type", fmt.Sprintf("%T", input))
				start := time.Now()

				result, err := node.Prep(ctx, store, input)

				logger.Debug(ctx, "node prep completed",
					"node", node.Name(),
					"duration", time.Since(start),
					"error", err)

				return result, err
			},
			exec: func(ctx context.Context, input any) (any, error) {
				logger.Info(ctx, "node exec starting", "node", node.Name())
				start := time.Now()

				result, err := node.Exec(ctx, input)

				if err != nil {
					logger.Error(ctx, "node exec failed",
						"node", node.Name(),
						"duration", time.Since(start),
						"error", err)
				} else {
					logger.Info(ctx, "node exec completed",
						"node", node.Name(),
						"duration", time.Since(start),
						"result_type", fmt.Sprintf("%T", result))
				}

				return result, err
			},
			post: func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				logger.Debug(ctx, "node post starting", "node", node.Name())

				output, next, err := node.Post(ctx, store, input, prep, exec)

				logger.Debug(ctx, "node post completed",
					"node", node.Name(),
					"next", next,
					"error", err)

				return output, next, err
			},
		}
	}
}
