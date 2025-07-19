// Package main demonstrates stateful workflow processing where nodes
// maintain and share state through Pocket's Store using the Prep/Exec/Post
// lifecycle, showing workflows that accumulate data across multiple stages.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
)

func main() {
	// Create store for shared state
	store := pocket.NewStore()
	ctx := context.Background()

	// Initialize accumulator state
	store.Set(ctx, "accumulator:data", []string{})
	store.Set(ctx, "process:count", 0)

	// Create processor node that maintains state
	processor := pocket.NewNode[any, any]("processor",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load previous processing state
			count, _ := store.Get(ctx, "process:count")
			processCount := count.(int)
			
			// Validate input
			data, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("expected string input")
			}
			
			return map[string]interface{}{
				"data":         data,
				"processCount": processCount,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Process the data
			data := prepData.(map[string]interface{})
			text := data["data"].(string)
			count := data["processCount"].(int)
			
			// Transform data (uppercase and add count)
			processed := fmt.Sprintf("%s_%d", strings.ToUpper(text), count+1)
			
			// Return both the result and the history data to store in post
			return map[string]interface{}{
				"processed": processed,
				"history": map[string]interface{}{
					"input":  text,
					"output": processed,
					"index":  count,
				},
				"historyKey": fmt.Sprintf("history:%d", count),
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})
			processed := execResult["processed"].(string)
			history := execResult["history"]
			historyKey := execResult["historyKey"].(string)
			
			// Store processing history
			store.Set(ctx, historyKey, history)
			
			// Update process count
			data := prepData.(map[string]interface{})
			newCount := data["processCount"].(int) + 1
			store.Set(ctx, "process:count", newCount)
			
			// Store last processed item
			store.Set(ctx, "processor:last", processed)
			
			return processed, "accumulate", nil
		}),
	)

	// Create accumulator node that builds up results
	accumulator := pocket.NewNode[any, any]("accumulator",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load accumulated data
			accumulated, _ := store.Get(ctx, "accumulator:data")
			results := accumulated.([]string)
			
			return map[string]interface{}{
				"newItem":     input,
				"accumulated": results,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Add new item to accumulation
			data := prepData.(map[string]interface{})
			newItem := data["newItem"].(string)
			accumulated := data["accumulated"].([]string)
			
			// Append new item
			accumulated = append(accumulated, newItem)
			
			// Create summary
			summary := strings.Join(accumulated, ", ")
			
			return map[string]interface{}{
				"summary":     summary,
				"accumulated": accumulated,
				"count":       len(accumulated),
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			// Save accumulated state
			r := result.(map[string]interface{})
			store.Set(ctx, "accumulator:data", r["accumulated"])
			store.Set(ctx, "accumulator:count", r["count"])
			
			// Route based on accumulated count
			count := r["count"].(int)
			if count == 1 {
				return r["summary"], "single", nil
			} else if count < 5 {
				return r["summary"], "multiple", nil
			}
			return r["summary"], "many", nil
		}),
	)

	// Create validator node that checks accumulated state
	validator := pocket.NewNode[any, any]("validator",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load validation rules from store
			maxLength, exists := store.Get(ctx, "validator:maxLength")
			if !exists {
				maxLength = 50
				// Note: Cannot set in prep phase - will set default in post if needed
			}
			
			return map[string]interface{}{
				"data":      input,
				"maxLength": maxLength,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Validate accumulated data
			data := prepData.(map[string]interface{})
			summary := data["data"].(string)
			maxLen := data["maxLength"].(int)
			
			validation := map[string]interface{}{
				"data":        summary,
				"length":      len(summary),
				"isValid":     len(summary) <= maxLen,
				"hasMultiple": strings.Contains(summary, ","),
			}
			
			// Validation results will be stored in post phase
			
			return validation, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			v := result.(map[string]interface{})
			
			// Store validation results
			store.Set(ctx, "validator:lastResult", v)
			
			// Set default maxLength if it wasn't already set
			if _, exists := store.Get(ctx, "validator:maxLength"); !exists {
				store.Set(ctx, "validator:maxLength", 50)
			}
			
			// Update validation statistics
			stats, exists := store.Get(ctx, "validator:stats")
			if !exists {
				stats = map[string]int{"valid": 0, "invalid": 0}
			}
			
			s := stats.(map[string]int)
			if v["isValid"].(bool) {
				s["valid"]++
			} else {
				s["invalid"]++
			}
			store.Set(ctx, "validator:stats", s)
			
			return v["data"], "report", nil
		}),
	)

	// Create output handlers
	singleHandler := pocket.NewNode[any, any]("single",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Single item processed: %s", input), nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			return result, "validate", nil
		}),
	)

	multipleHandler := pocket.NewNode[any, any]("multiple",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Multiple items (%d) accumulated: %s", strings.Count(input.(string), ",")+1, input), nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			return result, "validate", nil
		}),
	)

	manyHandler := pocket.NewNode[any, any]("many",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return fmt.Sprintf("Many items accumulated! Summary: %s", input), nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			return result, "validate", nil
		}),
	)

	// Create final report node
	reporter := pocket.NewNode[any, any]("report",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Gather all state for final report
			processCount, _ := store.Get(ctx, "process:count")
			accumulatorData, _ := store.Get(ctx, "accumulator:data")
			validatorStats, _ := store.Get(ctx, "validator:stats")
			
			return map[string]interface{}{
				"finalData":      input,
				"processCount":   processCount,
				"accumulated":    accumulatorData,
				"validatorStats": validatorStats,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Generate final report
			data := prepData.(map[string]interface{})
			
			report := fmt.Sprintf("=== Processing Report ===\n")
			report += fmt.Sprintf("Final Data: %s\n", data["finalData"])
			report += fmt.Sprintf("Total Items Processed: %d\n", data["processCount"])
			
			if stats, ok := data["validatorStats"].(map[string]int); ok {
				report += fmt.Sprintf("Validation Stats - Valid: %d, Invalid: %d\n", stats["valid"], stats["invalid"])
			}
			
			return report, nil
		}),
	)

	// Connect nodes
	processor.Connect("accumulate", accumulator)
	accumulator.Connect("single", singleHandler)
	accumulator.Connect("multiple", multipleHandler)
	accumulator.Connect("many", manyHandler)
	singleHandler.Connect("validate", validator)
	multipleHandler.Connect("validate", validator)
	manyHandler.Connect("validate", validator)
	validator.Connect("report", reporter)

	// Create flow
	flow := pocket.NewFlow(processor, store)

	// Run multiple iterations to demonstrate state persistence
	inputs := []string{"hello", "world", "pocket", "framework", "state", "management"}

	fmt.Println("=== Stateful Workflow Demo ===")
	fmt.Println("Processing items one by one, accumulating state:")
	fmt.Println()

	for i, input := range inputs {
		fmt.Printf("\n--- Iteration %d - Input: %s ---\n", i+1, input)

		result, err := flow.Run(ctx, input)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(result)

		// Show current accumulated state
		if data, exists := store.Get(ctx, "accumulator:data"); exists {
			accumulated := data.([]string)
			fmt.Printf("Current accumulated items: %v\n", accumulated)
		}
	}

	// Show final state summary
	fmt.Println("\n=== Final State Summary ===")
	
	// Process count
	if count, exists := store.Get(ctx, "process:count"); exists {
		fmt.Printf("Total processing operations: %d\n", count)
	}

	// Show processing history
	fmt.Println("\nProcessing History:")
	for i := 0; i < len(inputs); i++ {
		if history, exists := store.Get(ctx, fmt.Sprintf("history:%d", i)); exists {
			h := history.(map[string]interface{})
			fmt.Printf("  %d. %s -> %s\n", h["index"], h["input"], h["output"])
		}
	}

	// Validation statistics
	if stats, exists := store.Get(ctx, "validator:stats"); exists {
		s := stats.(map[string]int)
		fmt.Printf("\nValidation Statistics:\n")
		fmt.Printf("  Valid: %d\n", s["valid"])
		fmt.Printf("  Invalid: %d\n", s["invalid"])
	}

	// Demonstrate store scoping for isolated state
	fmt.Println("\n=== Store Scoping Demo ===")
	
	userStore := store.Scope("user")
	sessionStore := store.Scope("session")
	
	// Set scoped values
	userStore.Set(ctx, "id", "user123")
	userStore.Set(ctx, "name", "Alice")
	sessionStore.Set(ctx, "id", "session456")
	sessionStore.Set(ctx, "active", true)
	
	// Retrieve scoped values
	if userId, exists := userStore.Get(ctx, "id"); exists {
		fmt.Printf("User ID: %s\n", userId)
	}
	if sessionId, exists := sessionStore.Get(ctx, "id"); exists {
		fmt.Printf("Session ID: %s\n", sessionId)
	}
	
	// Show that scoped stores are isolated
	if _, exists := userStore.Get(ctx, "active"); !exists {
		fmt.Println("User store correctly doesn't have 'active' key")
	}
	if _, exists := sessionStore.Get(ctx, "name"); !exists {
		fmt.Println("Session store correctly doesn't have 'name' key")
	}
}