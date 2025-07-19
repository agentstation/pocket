package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/agentstation/pocket"
)

// Timing adds execution timing to a node, storing metrics in the store.
func Timing() Middleware {
	return func(node pocket.Node) pocket.Node {
		return &middlewareNode{
			inner: node,
			name:  node.Name(),

			prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				td := getTimingData(ctx, store, node.Name())
				td.execStart = time.Now()

				result, err := node.Prep(ctx, store, input)
				if err == nil {
					return map[string]interface{}{
						"prepResult": result,
						"timingData": td,
					}, nil
				}
				return result, err
			},

			exec: func(ctx context.Context, input any) (any, error) {
				actualInput, execStart := extractTimingInput(input)

				result, err := node.Exec(ctx, actualInput)
				duration := time.Since(execStart)

				// Return result with timing
				return map[string]interface{}{
					"execResult":   result,
					"execDuration": duration,
					"execError":    err,
				}, err
			},

			post: func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				execResult, execDuration, td := extractTimingResults(prep, exec)

				// Update timing metrics
				td.totalDuration += execDuration
				td.execCount++

				saveTimingMetrics(ctx, store, node.Name(), execDuration, td)

				// Call original post with correct data
				actualPrep := prep
				if data, ok := prep.(map[string]interface{}); ok {
					if prepResult, ok := data["prepResult"]; ok {
						actualPrep = prepResult
					}
				}

				return node.Post(ctx, store, input, actualPrep, execResult)
			},
		}
	}
}

// timingData holds timing metrics for a node.
type timingData struct {
	totalDuration time.Duration
	execCount     int64
	execStart     time.Time
}

// getTimingData retrieves timing data from store.
func getTimingData(ctx context.Context, store pocket.StoreReader, nodeName string) timingData {
	key := fmt.Sprintf("node:%s:total_duration", nodeName)
	countKey := fmt.Sprintf("node:%s:execution_count", nodeName)

	total, _ := store.Get(ctx, key)
	count, _ := store.Get(ctx, countKey)

	td := timingData{}
	if d, ok := total.(time.Duration); ok {
		td.totalDuration = d
	}
	if c, ok := count.(int64); ok {
		td.execCount = c
	}
	return td
}

// extractTimingInput extracts the actual input and timing start from wrapped data.
func extractTimingInput(input any) (actualInput any, execStart time.Time) {
	actualInput = input
	execStart = time.Now()

	if data, ok := input.(map[string]interface{}); ok {
		if prepResult, ok := data["prepResult"]; ok {
			actualInput = prepResult
		}
		if td, ok := data["timingData"].(timingData); ok {
			execStart = td.execStart
		}
	}
	return
}

// extractTimingResults extracts timing info from exec and prep results.
func extractTimingResults(prep, exec any) (execResult any, execDuration time.Duration, td timingData) {
	execResult = exec

	// Extract from exec data
	if data, ok := exec.(map[string]interface{}); ok {
		if result, ok := data["execResult"]; ok {
			execResult = result
		}
		if duration, ok := data["execDuration"].(time.Duration); ok {
			execDuration = duration
		}
	}

	// Extract from prep data
	if data, ok := prep.(map[string]interface{}); ok {
		if timing, ok := data["timingData"].(timingData); ok {
			td = timing
		}
	}
	return
}

// saveTimingMetrics saves timing metrics to the store.
func saveTimingMetrics(ctx context.Context, store pocket.StoreWriter, nodeName string, execDuration time.Duration, td timingData) {
	_ = store.Set(ctx, fmt.Sprintf("node:%s:last_duration", nodeName), execDuration)
	_ = store.Set(ctx, fmt.Sprintf("node:%s:total_duration", nodeName), td.totalDuration)
	_ = store.Set(ctx, fmt.Sprintf("node:%s:execution_count", nodeName), td.execCount)

	if td.execCount > 0 {
		_ = store.Set(ctx, fmt.Sprintf("node:%s:avg_duration", nodeName), td.totalDuration/time.Duration(td.execCount))
	}
}
