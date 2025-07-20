// Package main demonstrates parallel document processing using Pocket's
// concurrency patterns and the Prep/Exec/Post lifecycle for efficient
// batch processing with FanOut and Pipeline patterns.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/batch"
)

// Document represents a document to process.
type Document struct {
	ID      int
	Title   string
	Content string
}

// ProcessedDoc represents the result of processing.
type ProcessedDoc struct {
	ID       int
	Title    string
	Summary  string
	Keywords []string
	Duration time.Duration
}

// extractDocuments retrieves documents from the store or creates test data.
func extractDocuments(ctx context.Context, store pocket.StoreReader) ([]Document, error) {
	// In a real app, might load from database
	return []Document{
		{ID: 1, Title: "Go Concurrency", Content: "Goroutines and channels are the foundation of Go's concurrency model. They enable efficient parallel execution."},
		{ID: 2, Title: "Error Handling", Content: "Go uses explicit error returns instead of exceptions for error handling. This makes error paths clear."},
		{ID: 3, Title: "Interfaces", Content: "Go interfaces are satisfied implicitly, enabling flexible and composable designs without explicit declarations."},
		{ID: 4, Title: "Testing", Content: "Go has built-in testing support with the testing package and go test command for comprehensive testing."},
		{ID: 5, Title: "Modules", Content: "Go modules provide dependency management with semantic versioning for reproducible builds."},
	}, nil
}

// processDocument simulates document processing (e.g., calling an LLM).
func processDocument(ctx context.Context, doc Document) (ProcessedDoc, error) {
	start := time.Now()

	// Simulate processing time based on content length
	processingTime := time.Duration(50+len(doc.Content)) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate processing - extract keywords
	words := strings.Fields(strings.ToLower(doc.Content))
	keywords := []string{doc.Title}
	keywordMap := make(map[string]bool)

	for _, word := range words {
		if len(word) > 5 && !keywordMap[word] {
			keywords = append(keywords, word)
			keywordMap[word] = true
			if len(keywords) >= 5 {
				break
			}
		}
	}

	// Create summary
	summary := fmt.Sprintf("Summary of %s: %s", doc.Title, doc.Content[:40])
	if len(doc.Content) > 40 {
		summary += "..."
	}

	return ProcessedDoc{
		ID:       doc.ID,
		Title:    doc.Title,
		Summary:  summary,
		Keywords: keywords,
		Duration: time.Since(start),
	}, nil
}

// aggregateResults combines all processed documents into a report.
func aggregateResults(ctx context.Context, results []ProcessedDoc) (any, error) {
	var totalDuration time.Duration
	allKeywords := make(map[string]int)

	for _, doc := range results {
		totalDuration += doc.Duration
		for _, keyword := range doc.Keywords {
			allKeywords[keyword]++
		}
	}

	report := map[string]any{
		"documentsProcessed": len(results),
		"totalDuration":      totalDuration,
		"avgDuration":        totalDuration / time.Duration(len(results)),
		"uniqueKeywords":     len(allKeywords),
		"topKeywords":        getTopKeywords(allKeywords, 5),
		"results":            results,
	}

	return report, nil
}

func getTopKeywords(keywords map[string]int, n int) []string {
	// Simple top-N selection (in real app, use a heap)
	top := make([]string, 0, 3)
	for k := range keywords {
		top = append(top, k)
		if len(top) >= n {
			break
		}
	}
	return top
}

func main() {
	store := pocket.NewStore()
	ctx := context.Background()

	fmt.Println("=== Parallel Document Processing Demo ===")

	// Create a parallel batch processor using lifecycle pattern
	parallelProcessor := pocket.NewNode[any, any]("parallel-batch",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Extract documents in prep step
			docs, err := extractDocuments(ctx, store)
			if err != nil {
				return nil, fmt.Errorf("failed to extract documents: %w", err)
			}

			// Return docs along with metadata to be stored in post
			return map[string]interface{}{
				"docs":       docs,
				"doc_count":  len(docs),
				"start_time": time.Now(),
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Extract documents from prep result
			data := prepData.(map[string]interface{})
			documents := data["docs"].([]Document)
			results := make([]ProcessedDoc, len(documents))

			// Use errgroup for parallel processing
			var processed int32
			ch := make(chan struct{}, 3) // Limit concurrency to 3

			for i, doc := range documents {
				i, doc := i, doc // Capture loop variables
				ch <- struct{}{} // Acquire semaphore

				go func() {
					defer func() { <-ch }() // Release semaphore

					result, err := processDocument(ctx, doc)
					if err != nil {
						log.Printf("Error processing doc %d: %v", doc.ID, err)
						return
					}

					results[i] = result
					current := atomic.AddInt32(&processed, 1)
					fmt.Printf("Processed document %d/%d: %s\n", current, len(documents), doc.Title)
				}()
			}

			// Wait for all goroutines
			for i := 0; i < cap(ch); i++ {
				ch <- struct{}{}
			}

			return results, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, results any) (any, string, error) {
			// Store metadata from prep step
			data := prepData.(map[string]interface{})
			if err := store.Set(ctx, "doc_count", data["doc_count"]); err != nil {
				return nil, "", fmt.Errorf("failed to store doc_count: %w", err)
			}
			if err := store.Set(ctx, "start_time", data["start_time"]); err != nil {
				return nil, "", fmt.Errorf("failed to store start_time: %w", err)
			}

			// Aggregate results in post step
			processedDocs := results.([]ProcessedDoc)
			report, err := aggregateResults(ctx, processedDocs)
			if err != nil {
				return nil, "", err
			}

			// Calculate total time
			totalTime := time.Since(data["start_time"].(time.Time))

			// Add total time to report
			if r, ok := report.(map[string]any); ok {
				r["totalExecutionTime"] = totalTime
			}

			return report, "done", nil
		}),
	)

	// Create graph and run
	graph := pocket.NewGraph(parallelProcessor, store)

	result, err := graph.Run(ctx, nil)
	if err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	// Display results
	if report, ok := result.(map[string]any); ok {
		fmt.Println("\n=== Processing Report ===")
		fmt.Printf("Documents processed: %v\n", report["documentsProcessed"])
		fmt.Printf("Total execution time: %v\n", report["totalExecutionTime"])
		fmt.Printf("Average processing time per doc: %v\n", report["avgDuration"])
		fmt.Printf("Unique keywords found: %v\n", report["uniqueKeywords"])
		fmt.Printf("Top keywords: %v\n", report["topKeywords"])
	}

	// Demonstrate FanOut pattern with lifecycle
	fmt.Println("\n=== FanOut Pattern with Lifecycle ===")

	// Create a document processor node
	docProcessor := pocket.NewNode[any, any]("doc-processor",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate input is a document
			doc, ok := input.(Document)
			if !ok {
				return nil, fmt.Errorf("expected Document, got %T", input)
			}

			// Prepare processing context
			return map[string]interface{}{
				"document":  doc,
				"startTime": time.Now(),
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Extract and process document
			d := data.(map[string]interface{})
			doc := d["document"].(Document)

			// Simulate processing
			time.Sleep(100 * time.Millisecond)

			// Extract key information
			wordCount := len(strings.Fields(doc.Content))

			return map[string]interface{}{
				"id":        doc.ID,
				"title":     doc.Title,
				"wordCount": wordCount,
				"summary":   fmt.Sprintf("Doc %d: %s (%d words)", doc.ID, doc.Title, wordCount),
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			// Calculate processing time
			d := prep.(map[string]interface{})
			startTime := d["startTime"].(time.Time)
			duration := time.Since(startTime)

			r := result.(map[string]interface{})
			r["duration"] = duration

			return r["summary"], "done", nil
		}),
	)

	// Get documents
	docs, _ := extractDocuments(ctx, store)

	// Process all documents concurrently
	fanOutStart := time.Now()
	summaries, err := pocket.FanOut(ctx, docProcessor, store, docs)
	if err != nil {
		log.Fatalf("FanOut failed: %v", err)
	}
	fanOutDuration := time.Since(fanOutStart)

	fmt.Printf("Processed %d documents in %v using FanOut\n", len(docs), fanOutDuration)
	for i, summary := range summaries {
		fmt.Printf("  %d. %v\n", i+1, summary)
	}

	// Demonstrate Pipeline pattern with lifecycle
	fmt.Println("\n=== Pipeline Pattern with Lifecycle ===")

	// Create pipeline stages using lifecycle
	extract := pocket.NewNode[any, any]("extract",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate document
			doc, ok := input.(Document)
			if !ok {
				return nil, fmt.Errorf("expected Document")
			}
			return doc, nil
		}),
		pocket.WithExec(func(ctx context.Context, doc any) (any, error) {
			// Extract content
			d := doc.(Document)
			return map[string]string{
				"title":   d.Title,
				"content": d.Content,
			}, nil
		}),
	)

	analyze := pocket.NewNode[any, any]("analyze",
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Analyze content
			d := data.(map[string]string)
			content := d["content"]

			words := strings.Fields(content)
			sentences := strings.Count(content, ".") + strings.Count(content, "!") + strings.Count(content, "?")

			return map[string]interface{}{
				"title":     d["title"],
				"wordCount": len(words),
				"sentences": sentences,
				"avgWords":  float64(len(words)) / float64(sentences),
			}, nil
		}),
	)

	format := pocket.NewNode[any, any]("format",
		pocket.WithExec(func(ctx context.Context, analysis any) (any, error) {
			// Format results
			a := analysis.(map[string]interface{})
			return fmt.Sprintf("%s: %d words, %d sentences, %.1f words/sentence",
				a["title"], a["wordCount"], a["sentences"], a["avgWords"]), nil
		}),
	)

	// Run pipeline on first document
	pipelineResult, err := pocket.Pipeline(ctx, []pocket.Node{extract, analyze, format}, store, docs[0])
	if err != nil {
		log.Fatalf("Pipeline failed: %v", err)
	}

	fmt.Printf("\nPipeline analysis: %v\n", pipelineResult)

	// Demonstrate MapReduce pattern from batch package
	fmt.Println("\n=== MapReduce Pattern Demo ===")

	mapReduceNode := batch.MapReduce(
		"mapreduce",
		extractDocuments,
		processDocument,
		aggregateResults,
		batch.WithConcurrency(5),
	)

	mrGraph := pocket.NewGraph(mapReduceNode, store)
	mrResult, err := mrGraph.Run(ctx, nil)
	if err != nil {
		log.Fatalf("MapReduce failed: %v", err)
	}

	if report, ok := mrResult.(map[string]any); ok {
		fmt.Printf("MapReduce processed %v documents with %v unique keywords\n",
			report["documentsProcessed"], report["uniqueKeywords"])
	}
}
