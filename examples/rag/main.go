// Package main demonstrates a Retrieval-Augmented Generation (RAG) pattern
// using Pocket's Prep/Exec/Post lifecycle. This example shows how to build a
// document processing pipeline where Prep validates queries, Exec performs
// retrieval/augmentation/generation, and Post handles caching and routing.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
)

// Document represents a knowledge base document.
type Document struct {
	ID      string
	Title   string
	Content string
	Tags    []string
}

// Query represents a user query with metadata.
type Query struct {
	Text    string
	UserID  string
	Context []string
}

// RetrievedContext contains documents relevant to a query.
type RetrievedContext struct {
	Query     Query
	Documents []Document
	Relevance map[string]float64
}

// AugmentedQuery contains the original query enhanced with context.
type AugmentedQuery struct {
	Original Query
	Context  RetrievedContext
	Prompt   string
}

// GeneratedResponse is the final RAG output.
type GeneratedResponse struct {
	Answer     string
	Sources    []string
	Confidence float64
}

func main() {
	// Initialize knowledge base
	documents := []Document{
		{
			ID:      "1",
			Title:   "Go Concurrency Patterns",
			Content: "Go provides goroutines and channels for concurrent programming. Goroutines are lightweight threads managed by the Go runtime.",
			Tags:    []string{"go", "concurrency", "programming"},
		},
		{
			ID:      "2",
			Title:   "Error Handling in Go",
			Content: "Go handles errors explicitly using error values. The idiomatic way is to check errors immediately after operations that might fail.",
			Tags:    []string{"go", "errors", "best-practices"},
		},
		{
			ID:      "3",
			Title:   "Go Interfaces",
			Content: "Interfaces in Go provide a way to specify behavior. A type implements an interface by implementing its methods.",
			Tags:    []string{"go", "interfaces", "design"},
		},
	}

	// Create store for caching
	store := pocket.NewStore()
	ctx := context.Background()

	// Create retriever node with lifecycle
	retriever := pocket.NewNode[Query, RetrievedContext]("retrieve",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, query Query) (any, error) {
			// Validate query
			if query.Text == "" {
				return nil, fmt.Errorf("query text cannot be empty")
			}

			// Check cache for previous results
			cacheKey := fmt.Sprintf("query_cache:%s", query.Text)
			if cached, exists := store.Get(ctx, cacheKey); exists {
				fmt.Println("  📾 Using cached retrieval results")
				return cached, nil
			}

			// Prepare query for retrieval
			return map[string]interface{}{
				"query":     query,
				"keywords":  strings.Fields(strings.ToLower(query.Text)),
				"documents": documents,
			}, nil
		}),
		pocket.WithExec[any, RetrievedContext](func(ctx context.Context, data any) (RetrievedContext, error) {
			// If we got cached data, return it
			if retrieved, ok := data.(RetrievedContext); ok {
				return retrieved, nil
			}

			// Otherwise, perform retrieval
			d := data.(map[string]interface{})
			query := d["query"].(Query)
			keywords := d["keywords"].([]string)
			docs := d["documents"].([]Document)

			// Simple keyword-based retrieval (in real world, use vector embeddings)
			var relevant []Document
			relevance := make(map[string]float64)

			for _, doc := range docs {
				score := 0.0
				docLower := strings.ToLower(doc.Content + " " + doc.Title)

				for _, keyword := range keywords {
					if strings.Contains(docLower, keyword) {
						score += 1.0
					}
				}

				if score > 0 {
					relevant = append(relevant, doc)
					relevance[doc.ID] = score / float64(len(keywords))
				}
			}

			fmt.Printf("  🔍 Retrieved %d relevant documents\n", len(relevant))

			return RetrievedContext{
				Query:     query,
				Documents: relevant,
				Relevance: relevance,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, query Query, data any, retrieved RetrievedContext) (RetrievedContext, string, error) {
			// Cache the results
			cacheKey := fmt.Sprintf("query_cache:%s", query.Text)
			store.Set(ctx, cacheKey, retrieved)

			// Route based on retrieval results
			if len(retrieved.Documents) == 0 {
				return retrieved, "no_results", nil
			}
			return retrieved, "augment", nil
		}),
	)

	// Create augmenter node with lifecycle
	augmenter := pocket.NewNode[RetrievedContext, AugmentedQuery]("augment",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, retrieved RetrievedContext) (any, error) {
			// Prepare context building
			return map[string]interface{}{
				"retrieved":  retrieved,
				"maxContext": 3, // Limit context size
			}, nil
		}),
		pocket.WithExec[any, AugmentedQuery](func(ctx context.Context, data any) (AugmentedQuery, error) {
			d := data.(map[string]interface{})
			retrieved := d["retrieved"].(RetrievedContext)
			maxContext := d["maxContext"].(int)

			// Build context from top retrieved documents
			contextParts := make([]string, 0, len(retrieved.Documents))
			count := 0
			for _, doc := range retrieved.Documents {
				if count >= maxContext {
					break
				}
				contextParts = append(contextParts, fmt.Sprintf("[%s]: %s", doc.Title, doc.Content))
				count++
			}

			// Create augmented prompt
			prompt := fmt.Sprintf(
				"Based on the following context, answer the question.\n\nContext:\n%s\n\nQuestion: %s\n\nAnswer:",
				strings.Join(contextParts, "\n"),
				retrieved.Query.Text,
			)

			fmt.Printf("  📝 Augmented query with %d context documents\n", len(contextParts))

			return AugmentedQuery{
				Original: retrieved.Query,
				Context:  retrieved,
				Prompt:   prompt,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, retrieved RetrievedContext, data any, augmented AugmentedQuery) (AugmentedQuery, string, error) {
			// Always proceed to generation
			return augmented, "generate", nil
		}),
	)

	// Create generator node with lifecycle
	generator := pocket.NewNode[AugmentedQuery, GeneratedResponse]("generate",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, augmented AugmentedQuery) (any, error) {
			// Check response cache
			cacheKey := fmt.Sprintf("response_cache:%s", augmented.Original.Text)
			if cached, exists := store.Get(ctx, cacheKey); exists {
				fmt.Println("  📾 Using cached response")
				return cached, nil
			}

			// Prepare generation context
			return augmented, nil
		}),
		pocket.WithExec[any, GeneratedResponse](func(ctx context.Context, data any) (GeneratedResponse, error) {
			// If we got cached response, return it
			if response, ok := data.(GeneratedResponse); ok {
				return response, nil
			}

			// Otherwise, generate response
			augmented := data.(AugmentedQuery)

			// Simulate LLM response generation
			// In real implementation, this would call an actual LLM API
			answer := ""
			if len(augmented.Context.Documents) > 0 {
				// Generate specific answer based on context
				firstDoc := augmented.Context.Documents[0]
				answer = fmt.Sprintf(
					"Based on the documentation, %s %s",
					strings.ToLower(augmented.Original.Text),
					strings.Split(firstDoc.Content, ".")[0],
				)
			} else {
				answer = "I couldn't find relevant information to answer your question."
			}

			// Extract source references
			sources := make([]string, 0, len(augmented.Context.Documents))
			for _, doc := range augmented.Context.Documents {
				sources = append(sources, doc.Title)
			}

			// Calculate confidence based on relevance scores
			confidence := 0.0
			if len(augmented.Context.Documents) > 0 {
				for _, score := range augmented.Context.Relevance {
					confidence += score
				}
				confidence /= float64(len(augmented.Context.Documents))
			}

			fmt.Printf("  🤖 Generated response with %.2f confidence\n", confidence)

			return GeneratedResponse{
				Answer:     answer,
				Sources:    sources,
				Confidence: confidence,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, augmented AugmentedQuery, data any, response GeneratedResponse) (GeneratedResponse, string, error) {
			// Cache the response
			cacheKey := fmt.Sprintf("response_cache:%s", augmented.Original.Text)
			store.Set(ctx, cacheKey, response)

			// Update user history
			userHistory, _ := store.Get(ctx, fmt.Sprintf("user:%s:history", augmented.Original.UserID))
			if userHistory == nil {
				userHistory = []string{}
			}
			history := userHistory.([]string)
			history = append(history, augmented.Original.Text)
			store.Set(ctx, fmt.Sprintf("user:%s:history", augmented.Original.UserID), history)

			return response, "done", nil
		}),
	)

	// Create no results handler
	noResultsHandler := pocket.NewNode[any, any]("no_results",
		pocket.WithExec[any, any](func(ctx context.Context, input any) (any, error) {
			retrieved := input.(RetrievedContext)
			return GeneratedResponse{
				Answer:     fmt.Sprintf("I couldn't find any relevant documents for '%s'. Please try rephrasing your question.", retrieved.Query.Text),
				Sources:    []string{},
				Confidence: 0.0,
			}, nil
		}),
	)

	// Connect the pipeline
	retriever.Connect("augment", augmenter)
	retriever.Connect("no_results", noResultsHandler)
	augmenter.Connect("generate", generator)

	// Validate the flow
	if err := pocket.ValidateFlow(retriever); err != nil {
		log.Fatalf("Flow validation failed: %v", err)
	}

	// Test queries
	queries := []Query{
		{
			Text:   "How does Go handle concurrency?",
			UserID: "user1",
		},
		{
			Text:   "What are Go interfaces?",
			UserID: "user2",
		},
		{
			Text:   "Tell me about error handling",
			UserID: "user3",
		},
	}

	fmt.Println("=== RAG Pipeline Demo with Prep/Exec/Post ===")
	fmt.Println()

	for i, query := range queries {
		fmt.Printf("Query %d: %s\n", i+1, query.Text)
		fmt.Println("Processing:")

		// Create and run the RAG flow
		flow := pocket.NewFlow(retriever, store)
		result, err := flow.Run(ctx, query)
		if err != nil {
			log.Printf("Error processing query: %v", err)
			continue
		}

		response := result.(GeneratedResponse)
		fmt.Printf("\nAnswer: %s\n", response.Answer)
		fmt.Printf("Sources: %v\n", response.Sources)
		fmt.Printf("Confidence: %.2f\n", response.Confidence)
		fmt.Println(strings.Repeat("-", 60))
	}

	// Demonstrate caching by re-running a query
	fmt.Println("\nDemonstrating caching (re-running first query):")
	flow := pocket.NewFlow(retriever, store)
	result, err := flow.Run(ctx, queries[0])
	if err == nil {
		response := result.(GeneratedResponse)
		fmt.Printf("\nAnswer: %s\n", response.Answer)
		fmt.Printf("(Response was served from cache)\n")
	}

	// Show user history
	fmt.Println("\n=== User Query History ===")
	for _, query := range queries {
		if history, exists := store.Get(ctx, fmt.Sprintf("user:%s:history", query.UserID)); exists {
			fmt.Printf("User %s: %v\n", query.UserID, history)
		}
	}

	// Demonstrate builder pattern for RAG
	fmt.Println("\n=== RAG Builder Pattern ===")

	// Create a more complex RAG pipeline with quality filter
	qualityFilter := pocket.NewNode[any, any]("quality_filter",
		pocket.WithPrep[any](func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			response := input.(GeneratedResponse)
			return response, nil
		}),
		pocket.WithExec[any, any](func(ctx context.Context, response any) (any, error) {
			resp := response.(GeneratedResponse)

			// Filter based on confidence
			if resp.Confidence < 0.5 {
				resp.Answer = "Low confidence response: " + resp.Answer
			}

			return resp, nil
		}),
	)

	_, err = pocket.NewBuilder(store).
		Add(retriever).
		Add(augmenter).
		Add(generator).
		Add(qualityFilter).
		Add(noResultsHandler).
		Connect("retrieve", "augment", "augment").
		Connect("retrieve", "no_results", "no_results").
		Connect("augment", "generate", "generate").
		Connect("generate", "done", "quality_filter").
		Start("retrieve").
		Build()

	if err != nil {
		log.Printf("Failed to build RAG pipeline: %v", err)
	} else {
		fmt.Println("Successfully built complex RAG pipeline with quality filtering")
	}
}
