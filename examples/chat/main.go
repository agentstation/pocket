package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/agentstation/pocket"
)

// ChatBot simulates an LLM chat response.
type ChatBot struct {
	name  string
	store pocket.Store
}

// Process handles the chat interaction.
func (c *ChatBot) Process(ctx context.Context, input any) (any, error) {
	message, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string message, got %T", input)
	}
	
	// Get chat history from store
	history, _ := c.store.Get("history")
	if history == nil {
		history = []string{}
	}
	
	// Simulate LLM response
	response := fmt.Sprintf("[%s] Received: %s", c.name, message)
	
	// Update history
	historySlice := history.([]string)
	historySlice = append(historySlice, fmt.Sprintf("User: %s", message))
	historySlice = append(historySlice, response)
	c.store.Set("history", historySlice)
	
	return response, nil
}

// RouterBot routes messages to appropriate chat bot.
type RouterBot struct{}

// Process analyzes the message.
func (r *RouterBot) Process(ctx context.Context, input any) (any, error) {
	// Pass through the message
	return input, nil
}

// Route determines which bot should handle the message.
func (r *RouterBot) Route(ctx context.Context, result any) (string, error) {
	message := result.(string)
	
	// Simple routing based on message length
	if len(message) > 50 {
		return "expert", nil
	}
	return "simple", nil
}

func main() {
	// Create shared store
	store := pocket.NewStore()
	
	// Create router node
	router := pocket.NewNode("router", &RouterBot{})
	router.Router = &RouterBot{} // Set router interface
	
	// Create chat bots
	simpleBot := pocket.NewNode("simple", &ChatBot{
		name:  "SimpleBot",
		store: store,
	})
	
	expertBot := pocket.NewNode("expert", &ChatBot{
		name:  "ExpertBot",
		store: store,
	})
	
	// Connect nodes
	router.Connect("simple", simpleBot)
	router.Connect("expert", expertBot)
	
	// Create flow
	flow := pocket.NewFlow(router, store)
	
	// Example conversations
	messages := []string{
		"Hello!",
		"This is a very long message that requires expert handling because it contains complex information and needs detailed analysis",
		"How are you?",
		"What's the weather?",
	}
	
	ctx := context.Background()
	
	fmt.Println("=== Chat Session ===")
	for _, msg := range messages {
		fmt.Printf("\nUser: %s\n", msg)
		
		result, err := flow.Run(ctx, msg)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}
		
		fmt.Println(result)
	}
	
	// Print chat history
	if history, ok := store.Get("history"); ok {
		fmt.Println("\n=== Chat History ===")
		for _, line := range history.([]string) {
			fmt.Println(line)
		}
	}
	
	// Demonstrate builder pattern
	fmt.Println("\n=== Using Builder Pattern ===")
	
	// Clear history
	store.Set("history", []string{})
	
	// Build a more complex flow
	flow2, err := pocket.NewBuilder(store).
		Add(pocket.NewNode("input", pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			// Validate input
			msg := input.(string)
			if msg == "" {
				return nil, fmt.Errorf("empty message")
			}
			return msg, nil
		}))).
		Add(router).
		Add(simpleBot).
		Add(expertBot).
		Connect("input", "default", "router").
		Connect("router", "simple", "simple").
		Connect("router", "expert", "expert").
		Start("input").
		Build()
	
	if err != nil {
		log.Fatal(err)
	}
	
	// Test the built flow
	result, err := flow2.Run(ctx, "Builder pattern test")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("\nBuilder result: %v\n", result)
}