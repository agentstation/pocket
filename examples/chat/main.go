// Package main demonstrates a multi-agent chat system with intelligent routing
// between different specialized agents (assistant, expert, creative) based on
// the type of user query using the Prep/Exec/Post lifecycle.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
)

const (
	doneRoute = "done"
)

func main() {
	// Create shared store
	store := pocket.NewStore()
	ctx := context.Background()

	// Initialize chat history
	_ = store.Set(ctx, "history", []string{})

	// Create router node that analyzes messages and routes to appropriate bot
	router := pocket.NewNode[any, any]("router",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate input is a string message
			message, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("expected string message, got %T", input)
			}
			if message == "" {
				return nil, fmt.Errorf("empty message")
			}
			return message, nil
		}),
		pocket.WithExec(func(ctx context.Context, message any) (any, error) {
			// Analyze message (in real world, this could use LLM)
			msg := message.(string)

			// Simple analysis based on content
			analysis := map[string]interface{}{
				"message":       msg,
				"length":        len(msg),
				"hasQuestion":   strings.Contains(msg, "?"),
				"isComplex":     len(msg) > 50 || strings.Contains(strings.ToLower(msg), "explain"),
				"needsCreative": strings.ContainsAny(strings.ToLower(msg), "story poem joke creative"),
			}

			return analysis, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, message, analysis any) (any, string, error) {
			// Route based on analysis
			a := analysis.(map[string]interface{})

			if a["needsCreative"].(bool) {
				return input, "creative", nil
			} else if a["isComplex"].(bool) {
				return input, "expert", nil
			}
			return input, "simple", nil
		}),
	)

	// Create simple bot for basic responses
	simpleBot := pocket.NewNode[any, any]("simple",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load chat context
			history, _ := store.Get(ctx, "history")
			return map[string]interface{}{
				"message": input,
				"history": history,
				"bot":     "SimpleBot",
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Generate response
			d := data.(map[string]interface{})
			message := d["message"].(string)
			bot := d["bot"].(string)

			// Simulate simple response
			response := fmt.Sprintf("[%s] I can help with that! %s", bot, message)
			if strings.Contains(message, "?") {
				response = fmt.Sprintf("[%s] Good question! The answer is: it depends.", bot)
			}

			return response, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, response any) (any, string, error) {
			// Update history
			d := data.(map[string]interface{})
			history := d["history"].([]string)
			message := d["message"].(string)

			history = append(history,
				fmt.Sprintf("User: %s", message),
				response.(string))
			_ = store.Set(ctx, "history", history)

			return response, doneRoute, nil
		}),
	)

	// Create expert bot for complex queries
	expertBot := pocket.NewNode[any, any]("expert",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load chat context and prepare analysis
			history, _ := store.Get(ctx, "history")
			return map[string]interface{}{
				"message": input,
				"history": history,
				"bot":     "ExpertBot",
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Generate expert response
			d := data.(map[string]interface{})
			message := d["message"].(string)
			bot := d["bot"].(string)

			// Simulate expert analysis
			response := fmt.Sprintf("[%s] Let me provide a detailed analysis: %s", bot, message)
			response += "\n  • First, I'll break down your query"
			response += "\n  • Then, I'll provide comprehensive insights"
			response += "\n  • Finally, I'll offer actionable recommendations"

			return response, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, response any) (any, string, error) {
			// Update history with expert response
			d := data.(map[string]interface{})
			history := d["history"].([]string)
			message := d["message"].(string)

			history = append(history,
				fmt.Sprintf("User: %s", message),
				response.(string))
			_ = store.Set(ctx, "history", history)

			return response, doneRoute, nil
		}),
	)

	// Create creative bot for creative requests
	creativeBot := pocket.NewNode[any, any]("creative",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Prepare creative context
			history, _ := store.Get(ctx, "history")
			return map[string]interface{}{
				"message": input,
				"history": history,
				"bot":     "CreativeBot",
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Generate creative response
			d := data.(map[string]interface{})
			message := d["message"].(string)
			bot := d["bot"].(string)

			// Simulate creative response
			response := fmt.Sprintf("[%s] 🎨 How delightful! Here's something creative: ", bot)

			lowerMessage := strings.ToLower(message)
			switch {
			case strings.Contains(lowerMessage, "story"):
				response += "Once upon a time in a digital realm..."
			case strings.Contains(lowerMessage, "poem"):
				response += "\n  Roses are red,\n  Violets are blue,\n  AI writes poems,\n  Just for you!"
			case strings.Contains(lowerMessage, "joke"):
				response += "Why did the AI go to therapy? It had too many deep issues! 😄"
			default:
				response += "Let me paint you a word picture of possibilities..."
			}

			return response, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, response any) (any, string, error) {
			// Update history
			d := data.(map[string]interface{})
			history := d["history"].([]string)
			message := d["message"].(string)

			history = append(history,
				fmt.Sprintf("User: %s", message),
				response.(string))
			_ = store.Set(ctx, "history", history)

			return response, doneRoute, nil
		}),
	)

	// Connect nodes
	router.Connect("simple", simpleBot)
	router.Connect("expert", expertBot)
	router.Connect("creative", creativeBot)

	// Create graph
	graph := pocket.NewGraph(router, store)

	// Example conversations
	messages := []string{
		"Hello!",
		"What's the weather?",
		"Explain quantum computing in detail",
		"Tell me a joke",
		"Write a short poem about AI",
		"How are you?",
		"This is a very long message that requires expert handling because it contains complex information about machine learning algorithms and needs detailed analysis",
	}

	fmt.Println("=== Chat Session ===")
	for _, msg := range messages {
		fmt.Printf("\nUser: %s\n", msg)

		result, err := graph.Run(ctx, msg)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(result)
	}

	// Print chat history
	if history, ok := store.Get(ctx, "history"); ok {
		fmt.Println("\n=== Chat History ===")
		for _, line := range history.([]string) {
			fmt.Println(line)
		}
	}

	// Demonstrate builder pattern with validation
	fmt.Println("\n=== Using Builder Pattern ===")

	// Clear history
	_ = store.Set(ctx, "history", []string{})

	// Build a more complex graph with input validation
	inputValidator := pocket.NewNode[any, any]("input",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate and normalize input
			msg, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("input must be a string")
			}
			msg = strings.TrimSpace(msg)
			if msg == "" {
				return nil, fmt.Errorf("empty message not allowed")
			}
			if len(msg) > 1000 {
				return nil, fmt.Errorf("message too long (max 1000 chars)")
			}
			return msg, nil
		}),
		pocket.WithExec(func(ctx context.Context, msg any) (any, error) {
			// Add metadata
			return map[string]interface{}{
				"message":   msg,
				"timestamp": "2024-01-01T12:00:00Z", // In real app, use time.Now()
				"validated": true,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, msg, data any) (any, string, error) {
			// Extract message for routing
			d := data.(map[string]interface{})
			return d["message"], "route", nil
		}),
	)

	graph2, err := pocket.NewBuilder(store).
		Add(inputValidator).
		Add(router).
		Add(simpleBot).
		Add(expertBot).
		Add(creativeBot).
		Connect("input", "route", "router").
		Connect("router", "simple", "simple").
		Connect("router", "expert", "expert").
		Connect("router", "creative", "creative").
		Start("input").
		Build()

	if err != nil {
		log.Fatal(err)
	}

	// Test the built graph
	testMessages := []string{
		"Builder pattern test",
		"Tell me a story about builder patterns",
		"", // This should fail validation
	}

	for _, msg := range testMessages {
		fmt.Printf("\nTesting: %q\n", msg)
		result, err := graph2.Run(ctx, msg)
		if err != nil {
			fmt.Printf("Validation error: %v\n", err)
			continue
		}
		fmt.Printf("Result: %v\n", result)
	}
}
