package main

import (
	"context"
	"fmt"
	"log"
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
func extractDocuments(ctx context.Context, store pocket.Store) ([]Document, error) {
	// In a real app, might load from database
	return []Document{
		{ID: 1, Title: "Go Concurrency", Content: "Goroutines and channels are the foundation of Go's concurrency model"},
		{ID: 2, Title: "Error Handling", Content: "Go uses explicit error returns instead of exceptions for error handling"},
		{ID: 3, Title: "Interfaces", Content: "Go interfaces are satisfied implicitly, enabling flexible and composable designs"},
		{ID: 4, Title: "Testing", Content: "Go has built-in testing support with the testing package and go test command"},
		{ID: 5, Title: "Modules", Content: "Go modules provide dependency management with semantic versioning"},
	}, nil
}

// processDocument simulates document processing (e.g., calling an LLM).
func processDocument(ctx context.Context, doc Document) (ProcessedDoc, error) {
	start := time.Now()

	// Simulate processing time based on content length
	processingTime := time.Duration(50+len(doc.Content)) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate processing
	summary := fmt.Sprintf("Summary of %s: %s...", doc.Title, doc.Content[:30])
	keywords := []string{doc.Title, "Go", "Programming"}

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
		"results":            results,
	}

	return report, nil
}

func main() {
	store := pocket.NewStore()
	ctx := context.Background()

	fmt.Println("=== Parallel Document Processing Demo ===")

	// Create a parallel batch processor
	parallelProcessor := batch.MapReduce(
		extractDocuments,
		processDocument,
		aggregateResults,
		batch.WithConcurrency(3), // Process up to 3 documents concurrently
	)

	// Create a sequential processor for comparison
	sequentialProcessor := batch.MapReduce(
		extractDocuments,
		processDocument,
		aggregateResults,
		batch.WithConcurrency(1), // Sequential processing
	)

	// Run parallel processing
	fmt.Println("Running parallel processing (3 workers)...")
	parallelStart := time.Now()

	parallelNode := pocket.NewNode("parallel", parallelProcessor)
	parallelFlow := pocket.NewFlow(parallelNode, store)

	parallelResult, err := parallelFlow.Run(ctx, store)
	if err != nil {
		log.Fatalf("Parallel processing failed: %v", err)
	}
	parallelDuration := time.Since(parallelStart)

	// Run sequential processing
	fmt.Println("\nRunning sequential processing...")
	sequentialStart := time.Now()

	sequentialNode := pocket.NewNode("sequential", sequentialProcessor)
	sequentialFlow := pocket.NewFlow(sequentialNode, store)

	_, err = sequentialFlow.Run(ctx, store)
	if err != nil {
		log.Fatalf("Sequential processing failed: %v", err)
	}
	sequentialDuration := time.Since(sequentialStart)

	// Compare results
	fmt.Println("\n=== Performance Comparison ===")
	fmt.Printf("Parallel execution time: %v\n", parallelDuration)
	fmt.Printf("Sequential execution time: %v\n", sequentialDuration)
	fmt.Printf("Speedup: %.2fx\n", float64(sequentialDuration)/float64(parallelDuration))

	// Show results
	if report, ok := parallelResult.(map[string]any); ok {
		fmt.Printf("\nDocuments processed: %v\n", report["documentsProcessed"])
		fmt.Printf("Average processing time: %v\n", report["avgDuration"])
		fmt.Printf("Unique keywords found: %v\n", report["uniqueKeywords"])
	}

	// Demonstrate FanOut pattern
	fmt.Println("\n=== FanOut Pattern Demo ===")

	// Create a simple processor node
	summarizer := pocket.NewNode("summarize",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			doc := input.(Document)
			time.Sleep(100 * time.Millisecond) // Simulate work
			return fmt.Sprintf("Doc %d: %s", doc.ID, doc.Title), nil
		}),
	)

	// Get documents
	docs, _ := extractDocuments(ctx, store)

	// Process all documents concurrently
	fanOutStart := time.Now()
	summaries, err := pocket.FanOut(ctx, summarizer, store, docs)
	if err != nil {
		log.Fatalf("FanOut failed: %v", err)
	}
	fanOutDuration := time.Since(fanOutStart)

	fmt.Printf("Processed %d documents in %v using FanOut\n", len(docs), fanOutDuration)
	for i, summary := range summaries {
		fmt.Printf("  %d. %v\n", i+1, summary)
	}

	// Demonstrate Pipeline pattern
	fmt.Println("\n=== Pipeline Pattern Demo ===")

	// Create pipeline stages
	stage1 := pocket.NewNode("extract",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			doc := input.(Document)
			return doc.Content, nil
		}),
	)

	stage2 := pocket.NewNode("analyze",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			content := input.(string)
			wordCount := len(content) / 5 // Rough estimate
			return wordCount, nil
		}),
	)

	stage3 := pocket.NewNode("format",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			count := input.(int)
			return fmt.Sprintf("Word count: ~%d", count), nil
		}),
	)

	// Run pipeline on first document
	pipelineResult, err := pocket.Pipeline(ctx, []*pocket.Node{stage1, stage2, stage3}, store, docs[0])
	if err != nil {
		log.Fatalf("Pipeline failed: %v", err)
	}

	fmt.Printf("Pipeline result for '%s': %v\n", docs[0].Title, pipelineResult)
}
