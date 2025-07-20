// Package main demonstrates advanced Pocket features including graph composition,
// YAML support, fallback mechanisms, circuit breakers, and memory management.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/fallback"
)

const (
	defaultRoute = "default"
)

// LLMRequest represents a request to an LLM service.
type LLMRequest struct {
	Prompt      string  `yaml:"prompt"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
	Model       string  `yaml:"model"`
}

// LLMResponse represents a response from an LLM service.
type LLMResponse struct {
	Text       string    `yaml:"text"`
	Model      string    `yaml:"model"`
	TokensUsed int       `yaml:"tokens_used"`
	Timestamp  time.Time `yaml:"timestamp"`
}

func main() {
	ctx := context.Background()

	// Create a bounded store with LRU eviction
	boundedStore := pocket.NewStore(
		pocket.WithMaxEntries(100),
		pocket.WithTTL(5*time.Minute),
		pocket.WithEvictionCallback(func(key string, value any) {
			log.Printf("Evicted: %s", key)
		}),
	)

	fmt.Println("=== Advanced Pocket Features Demo ===")
	fmt.Println()

	// Demo 1: Circuit Breaker with Fallback
	fmt.Println("1. Circuit Breaker Pattern")
	fmt.Println("--------------------------")

	// Simulate an unreliable LLM service
	callCount := 0
	unreliableLLM := pocket.NewNode[any, any]("unreliable-llm",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			callCount++
			// Fail the first 3 calls to trigger circuit breaker
			if callCount <= 3 {
				return nil, errors.New("LLM service unavailable")
			}

			req := input.(LLMRequest)
			return LLMResponse{
				Text:       fmt.Sprintf("Generated response for: %s", req.Prompt),
				Model:      req.Model,
				TokensUsed: 150,
				Timestamp:  time.Now(),
			}, nil
		}),
	)

	// Create a fallback handler
	fallbackHandler := func(_ context.Context, _ pocket.StoreWriter, input any, err error) (any, error) {
		// Fallback to a simpler model
		log.Printf("Circuit breaker triggered, using fallback: %v", err)
		req := input.(LLMRequest)
		return LLMResponse{
			Text:       fmt.Sprintf("Fallback response for: %s", req.Prompt),
			Model:      "fallback-model",
			TokensUsed: 50,
			Timestamp:  time.Now(),
		}, nil
	}

	// Create a circuit breaker policy
	circuitBreaker := pocket.NewNode[any, any]("protected-llm",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Use a simple circuit breaker pattern inline

			result, err := unreliableLLM.Exec(ctx, input)
			if err != nil {
				// Use the fallback handler
				return fallbackHandler(ctx, boundedStore, input, err)
			}
			return result, nil
		}),
	)

	// Test the circuit breaker
	for i := 0; i < 5; i++ {
		req := LLMRequest{
			Prompt:      fmt.Sprintf("Test prompt %d", i+1),
			MaxTokens:   100,
			Temperature: 0.7,
			Model:       "gpt-4",
		}

		graph := pocket.NewGraph(circuitBreaker, boundedStore)
		result, err := graph.Run(ctx, req)
		if err != nil {
			log.Printf("Request %d failed: %v", i+1, err)
		} else {
			resp := result.(LLMResponse)
			fmt.Printf("Request %d: Model=%s, Response=%s\n", i+1, resp.Model, resp.Text)
		}
	}

	// Demo 2: Graph Composition with YAML Output
	fmt.Println("\n2. Graph Composition with YAML Output")
	fmt.Println("------------------------------------")

	// Create a sub-graph for data extraction
	extractionGraph := createExtractionGraph()

	// Convert the graph to a node
	extractionNode := extractionGraph.AsNode("extraction-subgraph")

	// Create a main graph that uses the extraction subgraph
	mainGraph := pocket.NewNode[any, any]("main-pipeline",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Prepare the input for extraction
			text := input.(string)
			return map[string]interface{}{
				"original_text": text,
				"text":          text,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			data := prepData.(map[string]interface{})
			return data["text"], nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			data := prepData.(map[string]interface{})
			if err := store.Set(ctx, "original_text", data["original_text"]); err != nil {
				return nil, "", fmt.Errorf("failed to store original text: %w", err)
			}
			return result, defaultRoute, nil
		}),
	)

	// Connect to extraction node
	mainGraph.Connect("default", extractionNode)

	// Add YAML formatting node
	yamlFormatter := pocket.NewNode[any, any]("yaml-formatter",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Format the extraction result as YAML
			yamlBytes, err := yaml.Marshal(input)
			if err != nil {
				return nil, err
			}
			return string(yamlBytes), nil
		}),
	)

	extractionNode.Connect("default", yamlFormatter)

	// Run the composed graph
	composedGraph := pocket.NewGraph(mainGraph, boundedStore)
	result, err := composedGraph.Run(ctx, "Analyze this text for sentiment and extract key entities.")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Extracted and formatted as YAML:")
	fmt.Println(result)

	// Demo 3: Advanced Fallback Chain
	fmt.Println("\n3. Advanced Fallback Chain")
	fmt.Println("--------------------------")

	// Create a chain of LLM providers
	chain := fallback.NewChain("llm-chain")

	// Add primary provider
	chain.AddLink(fallback.Link{
		Name:   "primary-gpt4",
		Weight: 1.0,
		Handler: func(ctx context.Context, input any) (any, error) {
			// Simulate primary provider
			if time.Now().Unix()%3 == 0 { // Fail 1/3 of the time
				return nil, errors.New("primary provider timeout")
			}
			return map[string]string{"provider": "GPT-4", "response": "Primary response"}, nil
		},
	})

	// Add secondary provider
	chain.AddLink(fallback.Link{
		Name:   "secondary-claude",
		Weight: 0.8,
		Handler: func(ctx context.Context, input any) (any, error) {
			// Simulate secondary provider
			return map[string]string{"provider": "Claude", "response": "Secondary response"}, nil
		},
	})

	// Add tertiary provider with transformation
	chain.AddLink(fallback.Link{
		Name:   "tertiary-local",
		Weight: 0.5,
		Handler: func(ctx context.Context, input any) (any, error) {
			return map[string]string{"provider": "Local Model", "response": "Tertiary response"}, nil
		},
		Transform: func(input any) any {
			// Transform input for local model
			return fmt.Sprintf("Simplified: %v", input)
		},
	})

	// Create a node with the fallback chain
	chainNode := pocket.NewNode[any, any]("llm-chain-node",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Pass input and store to exec step
			return map[string]interface{}{
				"input": input,
				"store": store,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Extract store and input
			data := prepData.(map[string]interface{})
			store := data["store"].(pocket.Store)
			input := data["input"]

			return chain.Execute(ctx, store, input)
		}),
	)

	// Test the chain
	chainGraph := pocket.NewGraph(chainNode, boundedStore)
	for i := 0; i < 5; i++ {
		result, err := chainGraph.Run(ctx, fmt.Sprintf("Query %d", i+1))
		if err != nil {
			log.Printf("Chain failed: %v", err)
		} else {
			resp := result.(map[string]string)
			fmt.Printf("Chain attempt %d: Provider=%s, Response=%s\n", i+1, resp["provider"], resp["response"])
		}
	}

	// Show chain metrics
	metrics := chain.GetMetrics()
	fmt.Printf("\nChain Metrics: Total executions=%d\n", metrics.TotalExecutions)
	for name, stats := range metrics.LinkStats {
		fmt.Printf("  %s: Executions=%d, Successes=%d, Failures=%d, AvgLatency=%v\n",
			name, stats.Executions, stats.Successes, stats.Failures, stats.AvgLatency)
	}

	// Demo 4: Cleanup Hooks and Resource Management
	fmt.Println("\n4. Cleanup Hooks and Resource Management")
	fmt.Println("---------------------------------------")

	// Create a node with cleanup hooks
	resourceNode := pocket.NewNode[any, any]("resource-manager",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Prepare resource allocation data
			return map[string]interface{}{
				"resource_id":  "res-12345",
				"allocated_at": time.Now(),
				"input":        input,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Use resources
			data := prepData.(map[string]interface{})
			resourceID := data["resource_id"]
			fmt.Printf("Using resource: %v\n", resourceID)

			// Simulate work
			time.Sleep(100 * time.Millisecond)

			return map[string]any{
				"result":      "processed",
				"resource_id": resourceID,
			}, nil
		}),
		pocket.WithOnSuccess(func(ctx context.Context, store pocket.StoreWriter, output any) {
			fmt.Println("Success hook: Marking resources as successfully used")
			if err := store.Set(ctx, "cleanup_status", "success"); err != nil {
				log.Printf("Failed to set cleanup status: %v", err)
			}
		}),
		pocket.WithOnFailure(func(ctx context.Context, store pocket.StoreWriter, err error) {
			fmt.Printf("Failure hook: Cleaning up after error: %v\n", err)
			if setErr := store.Set(ctx, "cleanup_status", "failed"); setErr != nil {
				log.Printf("Failed to set cleanup status: %v", setErr)
			}
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, execResult any) (any, string, error) {
			// Store resource info and mark as allocated
			data := prepData.(map[string]interface{})
			if err := store.Set(ctx, "resource_id", data["resource_id"]); err != nil {
				return nil, "", fmt.Errorf("failed to store resource_id: %w", err)
			}
			if err := store.Set(ctx, "allocated_at", data["allocated_at"]); err != nil {
				return nil, "", fmt.Errorf("failed to store allocated_at: %w", err)
			}
			fmt.Println("Resources allocated")
			return execResult, "done", nil
		}),
		pocket.WithOnComplete(func(ctx context.Context, store pocket.StoreWriter) {
			// Always runs - cleanup
			fmt.Println("Complete hook: Cleaning up resources")
			if err := store.Set(ctx, "cleanup_completed", time.Now()); err != nil {
				log.Printf("Failed to set cleanup_completed: %v", err)
			}
		}),
	)

	// Test cleanup hooks
	cleanupGraph := pocket.NewGraph(resourceNode, boundedStore)
	_, err = cleanupGraph.Run(ctx, "test input")
	if err != nil {
		log.Printf("Cleanup test error: %v", err)
	}

	// Demo 5: Store Configuration
	fmt.Println("\n5. Store Configuration")
	fmt.Println("------------------")

	fmt.Println("Store configured with:")
	fmt.Println("  Max Entries: 100")
	fmt.Println("  TTL: 5 minutes")
	fmt.Println("  Eviction: LRU (built-in)")
	fmt.Println("  Callback: Logs evicted entries")

	fmt.Println("\n=== Demo Complete ===")
}

// createExtractionGraph creates a sub-graph for data extraction.
func createExtractionGraph() *pocket.Graph {
	graphStore := pocket.NewStore()

	// Entity extraction node
	entityExtractor := pocket.NewNode[any, any]("entity-extractor",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Simulate entity extraction
			entities := []string{"sentiment", "entities", "text"}
			return map[string]any{
				"entities": entities,
				"count":    len(entities),
			}, nil
		}),
	)

	// Sentiment analysis node
	sentimentAnalyzer := pocket.NewNode[any, any]("sentiment-analyzer",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Simulate sentiment analysis
			return map[string]any{
				"sentiment": "positive",
				"score":     0.85,
			}, nil
		}),
	)

	// Combine results
	combiner := pocket.NewNode[any, any]("result-combiner",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Get results from store in prep step
			entities, _ := store.Get(ctx, "entities")
			sentiment, _ := store.Get(ctx, "sentiment")

			return map[string]interface{}{
				"entities":  entities,
				"sentiment": sentiment,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			data := prepData.(map[string]interface{})
			return map[string]any{
				"extraction_results": map[string]any{
					"entities":  data["entities"],
					"sentiment": data["sentiment"],
					"timestamp": time.Now(),
				},
			}, nil
		}),
	)

	// Connect nodes
	entityExtractor.Connect("default", sentimentAnalyzer)
	sentimentAnalyzer.Connect("default", combiner)

	// Store intermediate results
	wrappedEntityExtractor := pocket.NewNode[any, any]("wrapped-entity",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			result, err := entityExtractor.Exec(ctx, input)
			return result, err
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			if err := store.Set(ctx, "entities", result); err != nil {
				return nil, "", fmt.Errorf("failed to store entities: %w", err)
			}
			return result, defaultRoute, nil
		}),
	)

	wrappedSentimentAnalyzer := pocket.NewNode[any, any]("wrapped-sentiment",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			result, err := sentimentAnalyzer.Exec(ctx, input)
			return result, err
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			if err := store.Set(ctx, "sentiment", result); err != nil {
				return nil, "", fmt.Errorf("failed to store sentiment: %w", err)
			}
			return result, defaultRoute, nil
		}),
	)

	// Connect wrapped nodes
	wrappedEntityExtractor.Connect("default", wrappedSentimentAnalyzer)
	wrappedSentimentAnalyzer.Connect("default", combiner)

	return pocket.NewGraph(wrappedEntityExtractor, graphStore)
}
