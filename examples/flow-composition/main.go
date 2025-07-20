// Package main demonstrates graph-as-node composition where entire workflows
// can be treated as single nodes within larger workflows, enabling modular
// and reusable workflow design with the Prep/Exec/Post lifecycle.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/compose"
)

// TextProcessor represents a reusable text processing workflow.
func createTextProcessorGraph(store pocket.Store) *pocket.Graph {
	// Node 1: Clean text
	cleaner := pocket.NewNode[any, any]("clean",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			text, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("expected string input")
			}
			return text, nil
		}),
		pocket.WithExec(func(ctx context.Context, text any) (any, error) {
			// Remove extra spaces and trim
			cleaned := strings.TrimSpace(text.(string))
			cleaned = strings.Join(strings.Fields(cleaned), " ")
			return cleaned, nil
		}),
	)

	// Node 2: Analyze text
	analyzer := pocket.NewNode[any, any]("analyze",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			text := input.(string)
			analysis := map[string]interface{}{
				"text":      text,
				"length":    len(text),
				"wordCount": len(strings.Fields(text)),
				"hasDigits": strings.ContainsAny(text, "0123456789"),
			}
			return analysis, nil
		}),
	)

	// Connect nodes
	cleaner.Connect("default", analyzer)

	return pocket.NewGraph(cleaner, store)
}

// TranslationGraph simulates a translation workflow.
func createTranslationGraph(store pocket.Store) *pocket.Graph {
	translator := pocket.NewNode[any, any]("translate",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Extract text from analysis if provided
			switch v := input.(type) {
			case string:
				return v, nil
			case map[string]interface{}:
				if text, ok := v["text"].(string); ok {
					return text, nil
				}
			}
			return nil, fmt.Errorf("cannot extract text from input")
		}),
		pocket.WithExec(func(ctx context.Context, text any) (any, error) {
			// Simulate translation
			original := text.(string)
			// Simple mock translation: reverse words
			words := strings.Fields(original)
			for i := 0; i < len(words)/2; i++ {
				words[i], words[len(words)-1-i] = words[len(words)-1-i], words[i]
			}

			return map[string]interface{}{
				"original":   original,
				"translated": strings.Join(words, " "),
				"language":   "mock-lang",
			}, nil
		}),
	)

	return pocket.NewGraph(translator, store)
}

// QualityCheckGraph performs quality checks on translations.
func createQualityCheckGraph(store pocket.Store) *pocket.Graph {
	checker := pocket.NewNode[any, any]("quality-check",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			data, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map input")
			}

			original, _ := data["original"].(string)
			translated, _ := data["translated"].(string)

			// Simple quality metrics
			quality := map[string]interface{}{
				"original":       original,
				"translated":     translated,
				"lengthRatio":    float64(len(translated)) / float64(len(original)),
				"preservedWords": 0,
				"qualityScore":   0.0,
			}

			// Count preserved words
			origWords := strings.Fields(strings.ToLower(original))
			transWords := strings.Fields(strings.ToLower(translated))
			preserved := 0
			for _, word := range origWords {
				for _, tword := range transWords {
					if word == tword {
						preserved++
						break
					}
				}
			}
			quality["preservedWords"] = preserved

			// Calculate quality score
			lengthScore := 1.0 - abs(1.0-quality["lengthRatio"].(float64))
			preserveScore := float64(preserved) / float64(len(origWords))
			quality["qualityScore"] = (lengthScore + preserveScore) / 2.0

			return quality, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			quality := result.(map[string]interface{})
			score := quality["qualityScore"].(float64)

			// Route based on quality
			if score > 0.7 {
				return quality, "approved", nil
			} else if score > 0.4 {
				return quality, "review", nil
			}
			return quality, "rejected", nil
		}),
	)

	return pocket.NewGraph(checker, store)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	store := pocket.NewStore()
	ctx := context.Background()

	fmt.Println("=== Graph-as-Node Composition Demo ===")
	fmt.Println()

	// Create individual graphs
	textProcessor := createTextProcessorGraph(store)
	translator := createTranslationGraph(store)
	qualityChecker := createQualityCheckGraph(store)

	// Method 1: Using Graph.AsNode() method
	fmt.Println("Method 1: Using Graph.AsNode()")
	fmt.Println("-----------------------------")

	// Convert graphs to nodes
	processNode := textProcessor.AsNode("text-processor")
	translateNode := translator.AsNode("translator")
	qualityNode := qualityChecker.AsNode("quality-checker")

	// Create approval/rejection handlers
	approveHandler := pocket.NewNode[any, any]("approve",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			quality := input.(map[string]interface{})
			fmt.Printf("✅ Translation APPROVED (score: %.2f)\n", quality["qualityScore"])
			return quality, nil
		}),
	)

	reviewHandler := pocket.NewNode[any, any]("review",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			quality := input.(map[string]interface{})
			fmt.Printf("🔍 Translation needs REVIEW (score: %.2f)\n", quality["qualityScore"])
			return quality, nil
		}),
	)

	rejectHandler := pocket.NewNode[any, any]("reject",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			quality := input.(map[string]interface{})
			fmt.Printf("❌ Translation REJECTED (score: %.2f)\n", quality["qualityScore"])
			return quality, nil
		}),
	)

	// Connect the graph nodes
	processNode.Connect("default", translateNode)
	translateNode.Connect("default", qualityNode)
	qualityNode.Connect("approved", approveHandler)
	qualityNode.Connect("review", reviewHandler)
	qualityNode.Connect("rejected", rejectHandler)

	// Create the composite graph
	pipeline := pocket.NewGraph(processNode, store)

	// Test with different inputs
	testInputs := []string{
		"  Hello   world   from   Pocket  ",
		"The quick brown fox jumps over the lazy dog",
		"Simple test",
	}

	for i, input := range testInputs {
		fmt.Printf("\nTest %d: %q\n", i+1, input)
		result, err := pipeline.Run(ctx, input)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		if quality, ok := result.(map[string]interface{}); ok {
			fmt.Printf("Original: %s\n", quality["original"])
			fmt.Printf("Translated: %s\n", quality["translated"])
		}
	}

	// Method 2: Using NestedGraphBuilder
	fmt.Println("\n\nMethod 2: Using NestedGraphBuilder")
	fmt.Println("---------------------------------")

	// Create a nested graph with store isolation
	nestedGraph, err := compose.NewBuilder("translation-pipeline", store).
		AddGraph("process", textProcessor).
		AddGraph("translate", translator).
		AddGraph("quality", qualityChecker).
		Connect("process", "default", "translate").
		Connect("translate", "default", "quality").
		Build()

	if err != nil {
		log.Fatalf("Failed to build nested graph: %v", err)
	}

	// Run the nested graph
	result, err := nestedGraph.Run(ctx, "  Nested   graph   test   with   spaces  ")
	if err != nil {
		log.Fatalf("Nested graph error: %v", err)
	}

	if quality, ok := result.(map[string]interface{}); ok {
		fmt.Printf("\nNested Graph Result:\n")
		fmt.Printf("Original: %s\n", quality["original"])
		fmt.Printf("Translated: %s\n", quality["translated"])
		fmt.Printf("Quality Score: %.2f\n", quality["qualityScore"])
	}

	// Method 3: Parallel graph composition
	fmt.Println("\n\nMethod 3: Parallel Graph Composition")
	fmt.Println("-----------------------------------")

	// Create multiple translation graphs for different "languages"
	translationGraphs := []*pocket.Graph{
		createTranslationGraph(store),
		createTranslationGraph(store),
		createTranslationGraph(store),
	}

	// Process the same text through multiple translators in parallel
	parallelResults, err := compose.ParallelGraphs(ctx, store, translationGraphs...)
	if err != nil {
		log.Printf("Parallel execution error: %v", err)
	} else {
		fmt.Println("\nParallel Translation Results:")
		for i, result := range parallelResults {
			if trans, ok := result.(map[string]interface{}); ok {
				fmt.Printf("Translator %d: %s\n", i+1, trans["translated"])
			}
		}
	}

	// Method 4: Graph composition with state sharing
	fmt.Println("\n\nMethod 4: Graph Composition with Shared State")
	fmt.Println("-------------------------------------------")

	// Store configuration in shared store
	if err := store.Set(ctx, "config:maxLength", 100); err != nil {
		log.Fatalf("Failed to set maxLength: %v", err)
	}
	if err := store.Set(ctx, "config:targetLanguage", "es"); err != nil {
		log.Fatalf("Failed to set targetLanguage: %v", err)
	}

	// Create a graph that uses shared configuration
	configAwareGraph := pocket.NewGraph(
		pocket.NewNode[any, any]("config-processor",
			pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				maxLen, _ := store.Get(ctx, "config:maxLength")
				targetLang, _ := store.Get(ctx, "config:targetLanguage")

				return map[string]interface{}{
					"text":       input,
					"maxLength":  maxLen,
					"targetLang": targetLang,
				}, nil
			}),
			pocket.WithExec(func(ctx context.Context, config any) (any, error) {
				cfg := config.(map[string]interface{})
				text := cfg["text"].(string)
				maxLen := cfg["maxLength"].(int)

				if len(text) > maxLen {
					text = text[:maxLen] + "..."
				}

				return fmt.Sprintf("Processed for %s: %s", cfg["targetLang"], text), nil
			}),
		),
		store,
	)

	// Use as node in larger graph
	configNode := configAwareGraph.AsNode("config-aware")

	result, err = pocket.NewGraph(configNode, store).Run(ctx, "This is a very long text that might need to be truncated based on configuration settings")
	if err != nil {
		log.Printf("Config graph error: %v", err)
	} else {
		fmt.Printf("\nConfig-aware result: %s\n", result)
	}

	fmt.Println("\n=== Demo Complete ===")
}
