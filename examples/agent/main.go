package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
)

const (
	completeAction = "complete"
)

// Task represents a task for the agent.
type Task struct {
	Description string
	Steps       []string
}

// ThinkNode represents the agent's reasoning process.
type ThinkNode struct {
	store pocket.Store
}

// Process analyzes the current state and decides next action.
func (t *ThinkNode) Process(ctx context.Context, input any) (any, error) {
	// Load task from store
	taskData, _ := t.store.Get("task")
	task := taskData.(*Task)

	fmt.Printf("\n[THINK] Task: %s\n", task.Description)
	fmt.Printf("[THINK] Completed steps: %v\n", task.Steps)

	// Simple task decomposition logic
	switch {
	case len(task.Steps) == 0 && strings.Contains(task.Description, "write"):
		return "research", nil
	case len(task.Steps) == 1:
		return "draft", nil
	case len(task.Steps) == 2:
		return "review", nil
	default:
		return completeAction, nil
	}
}

// Route determines the next action based on thinking.
func (t *ThinkNode) Route(ctx context.Context, result any) (string, error) {
	action := result.(string)
	return action, nil
}

// ActionNode executes a specific action.
type ActionNode struct {
	actionType string
	store      pocket.Store
}

// Process performs the action.
func (a *ActionNode) Process(ctx context.Context, input any) (any, error) {
	// Get current task
	taskData, _ := a.store.Get("task")
	task := taskData.(*Task)

	var result string

	// Simulate action execution
	switch a.actionType {
	case "research":
		fmt.Printf("[ACT] Researching: %s\n", task.Description)
		result = "Research completed: Found best practices for Go concurrency"

	case "draft":
		fmt.Printf("[ACT] Drafting: %s\n", task.Description)
		result = "Draft created: Comprehensive guide on goroutines and channels"

	case "review":
		fmt.Printf("[ACT] Reviewing: %s\n", task.Description)
		result = "Review completed: Content polished and examples added"

	case completeAction:
		fmt.Printf("[ACT] Completing: %s\n", task.Description)
		result = fmt.Sprintf("Task completed: %s", task.Description)
		return result, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", a.actionType)
	}

	// Update task steps
	task.Steps = append(task.Steps, fmt.Sprintf("%s: %s", a.actionType, result))
	a.store.Set("task", task)

	return result, nil
}

// Route always goes back to think (except for complete).
func (a *ActionNode) Route(ctx context.Context, result any) (string, error) {
	if a.actionType == completeAction {
		return "done", nil
	}
	return "think", nil
}

func main() {
	// Create store and set initial task
	store := pocket.NewStore()
	task := &Task{
		Description: "write a blog post about Go concurrency patterns",
		Steps:       []string{},
	}
	store.Set("task", task)

	// Create think node
	think := pocket.NewNode("think", &ThinkNode{store: store})
	think.Router = &ThinkNode{store: store}

	// Create action nodes
	research := pocket.NewNode("research", &ActionNode{
		actionType: "research",
		store:      store,
	})
	research.Router = &ActionNode{actionType: "research", store: store}

	draft := pocket.NewNode("draft", &ActionNode{
		actionType: "draft",
		store:      store,
	})
	draft.Router = &ActionNode{actionType: "draft", store: store}

	review := pocket.NewNode("review", &ActionNode{
		actionType: "review",
		store:      store,
	})
	review.Router = &ActionNode{actionType: "review", store: store}

	complete := pocket.NewNode(completeAction, &ActionNode{
		actionType: completeAction,
		store:      store,
	})
	complete.Router = &ActionNode{actionType: completeAction, store: store}

	// Connect nodes - think decides which action
	think.Connect("research", research)
	think.Connect("draft", draft)
	think.Connect("review", review)
	think.Connect(completeAction, complete)

	// Actions loop back to think (except complete)
	research.Connect("think", think)
	draft.Connect("think", think)
	review.Connect("think", think)
	// complete has no connections (ends the flow)

	// Create and run the agent flow
	fmt.Println("=== Autonomous Agent Demo ===")
	flow := pocket.NewFlow(think, store)

	ctx := context.Background()
	result, err := flow.Run(ctx, nil)
	if err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	fmt.Printf("\n[FINAL] %v\n", result)

	// Show execution trace
	finalTask, _ := store.Get("task")
	if taskData, ok := finalTask.(*Task); ok {
		fmt.Println("\n=== Execution Trace ===")
		for i, step := range taskData.Steps {
			fmt.Printf("%d. %s\n", i+1, step)
		}
	}

	// Demonstrate agent with retry capability
	fmt.Println("\n=== Agent with Retry Demo ===")

	// Reset task
	task2 := &Task{
		Description: "analyze complex data and generate insights",
		Steps:       []string{},
	}
	store.Set("task", task2)

	// Create a flaky action that sometimes fails
	attempts := 0
	analyze := pocket.NewNode("analyze",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			attempts++
			if attempts < 2 {
				return nil, fmt.Errorf("temporary analysis failure")
			}
			return "Analysis completed successfully", nil
		}),
		pocket.WithRetry(3, 0), // Retry up to 3 times
	)

	// Simple flow: think -> analyze
	think2 := pocket.NewNode("think2",
		pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
			return "analyze", nil
		}),
	)
	think2.Router = pocket.RouterFunc(func(ctx context.Context, result any) (string, error) {
		return result.(string), nil
	})

	think2.Connect("analyze", analyze)

	flow2 := pocket.NewFlow(think2, store)
	result2, err := flow2.Run(ctx, nil)
	if err != nil {
		log.Printf("Retry demo error: %v", err)
	} else {
		fmt.Printf("Retry demo result: %v (after %d attempts)\n", result2, attempts)
	}
}
