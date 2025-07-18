// Package main demonstrates a stateful workflow with Pocket.
// This example shows how to build a multi-step data processing pipeline
// that maintains state between operations.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
)

// DataProcessor demonstrates stateful processing
type DataProcessor struct {
	name string
}

func (p *DataProcessor) Process(ctx context.Context, input any) (any, error) {
	data := input.(string)
	return strings.ToUpper(data), nil
}

func (p *DataProcessor) LoadState(ctx context.Context, store pocket.Store) (any, error) {
	// Load previous results from store
	if prev, exists := store.Get(p.name + ":previous"); exists {
		return prev, nil
	}
	return "", nil
}

func (p *DataProcessor) SaveState(ctx context.Context, store pocket.Store, state any) error {
	// Save result for next iteration
	store.Set(p.name+":previous", state)
	store.Set(p.name+":processed", true)
	return nil
}

// AccumulatorNode accumulates results over multiple runs
type AccumulatorNode struct {
	results []string
}

func (a *AccumulatorNode) Process(ctx context.Context, input any) (any, error) {
	if str, ok := input.(string); ok && str != "" {
		a.results = append(a.results, str)
	}
	return strings.Join(a.results, ", "), nil
}

func (a *AccumulatorNode) LoadState(ctx context.Context, store pocket.Store) (any, error) {
	if data, exists := store.Get("accumulator:data"); exists {
		if results, ok := data.([]string); ok {
			a.results = results
		}
	}
	return nil, nil
}

func (a *AccumulatorNode) SaveState(ctx context.Context, store pocket.Store, state any) error {
	store.Set("accumulator:data", a.results)
	return nil
}

// ValidatorNode validates and routes data
type ValidatorNode struct{}

func (v *ValidatorNode) Process(ctx context.Context, input any) (any, error) {
	data := input.(string)
	if data == "" {
		return nil, fmt.Errorf("empty data")
	}
	return data, nil
}

func (v *ValidatorNode) Route(ctx context.Context, result any) (string, error) {
	data := result.(string)
	if len(data) > 10 {
		return "long", nil
	}
	if strings.Contains(data, ",") {
		return "multiple", nil
	}
	return "single", nil
}

func main() {
	// Create store for shared state
	store := pocket.NewStore()

	// Create stateful nodes
	processor := &DataProcessor{name: "processor"}
	accumulator := &AccumulatorNode{}
	validator := &ValidatorNode{}

	// Create nodes
	processNode := pocket.NewNode("process", processor)
	processNode.Stateful = processor

	accumulateNode := pocket.NewNode("accumulate", accumulator)
	accumulateNode.Stateful = accumulator

	validateNode := pocket.NewNode("validate", validator)
	validateNode.Router = validator

	// Create output nodes for different routes
	singleHandler := pocket.NewNode("single", pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		return fmt.Sprintf("Single item: %s", input), nil
	}))

	multipleHandler := pocket.NewNode("multiple", pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		return fmt.Sprintf("Multiple items: %s", input), nil
	}))

	longHandler := pocket.NewNode("long", pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		return fmt.Sprintf("Long data: %s (length: %d)", input, len(input.(string))), nil
	}))

	// Build the flow
	processNode.Default(accumulateNode)
	accumulateNode.Default(validateNode)
	validateNode.Connect("single", singleHandler)
	validateNode.Connect("multiple", multipleHandler)
	validateNode.Connect("long", longHandler)

	// Create flow
	flow := pocket.NewFlow(processNode, store)
	ctx := context.Background()

	// Run multiple iterations to demonstrate state persistence
	inputs := []string{"hello", "world", "pocket", "framework"}

	fmt.Println("=== Stateful Workflow Demo ===")
	fmt.Println("Processing items one by one, accumulating state:")
	fmt.Println()

	for i, input := range inputs {
		fmt.Printf("Iteration %d - Input: %s\n", i+1, input)

		result, err := flow.Run(ctx, input)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Result: %s\n", result)

		// Show accumulated state
		if data, exists := store.Get("accumulator:data"); exists {
			fmt.Printf("Accumulated data: %v\n", data)
		}

		fmt.Println()
	}

	// Show final state
	fmt.Println("=== Final State ===")
	store.Set("demo", "complete")

	// Iterate through store to show all state
	showStoreContents := func(key string, value any) {
		fmt.Printf("%s: %v\n", key, value)
	}

	// Note: In real implementation, you might want to add a method to iterate store
	// For demo, we'll check known keys
	keys := []string{
		"processor:previous",
		"processor:processed",
		"accumulator:data",
		"demo",
	}

	for _, key := range keys {
		if value, exists := store.Get(key); exists {
			showStoreContents(key, value)
		}
	}
}