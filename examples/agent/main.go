// Package main demonstrates an autonomous agent with a think-act loop pattern
// using Pocket's Prep/Exec/Post lifecycle. The agent cycles through thinking,
// researching, and drafting phases to produce a final report.
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
	doneRoute      = "done"
)

// Task represents a task for the agent.
type Task struct {
	Description string
	Steps       []string
}

func main() {
	// Create store and set initial task
	store := pocket.NewStore()
	ctx := context.Background()

	task := &Task{
		Description: "write a blog post about Go concurrency patterns",
		Steps:       []string{},
	}
	store.Set(ctx, "task", task)

	// Create think node that analyzes the task and decides next action
	think := pocket.NewNode[any, any]("think",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load current task state
			taskData, exists := store.Get(ctx, "task")
			if !exists {
				return nil, fmt.Errorf("no task found")
			}
			return taskData, nil
		}),
		pocket.WithExec(func(ctx context.Context, task any) (any, error) {
			// Analyze task and decide next action
			t := task.(*Task)

			fmt.Printf("\n[THINK] Task: %s\n", t.Description)
			fmt.Printf("[THINK] Completed steps: %v\n", t.Steps)

			// Simple task decomposition logic
			switch {
			case len(t.Steps) == 0 && strings.Contains(t.Description, "write"):
				return "research", nil
			case len(t.Steps) == 1:
				return "draft", nil
			case len(t.Steps) == 2:
				return "review", nil
			default:
				return completeAction, nil
			}
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, task, action any) (any, string, error) {
			// Route to the chosen action
			actionStr := action.(string)
			fmt.Printf("[THINK] Next action: %s\n", actionStr)
			return action, actionStr, nil
		}),
	)

	// Create action nodes using a helper function
	createActionNode := func(actionType string) pocket.Node {
		return pocket.NewNode[any, any](actionType,
			pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				// Prepare: get current task
				task, _ := store.Get(ctx, "task")
				return task, nil
			}),
			pocket.WithExec(func(ctx context.Context, task any) (any, error) {
				// Execute: perform the action
				t := task.(*Task)
				var result string

				fmt.Printf("[ACT] Performing: %s\n", actionType)

				switch actionType {
				case "research":
					result = "Research completed: Found best practices for Go concurrency"
				case "draft":
					result = "Draft created: Comprehensive guide on goroutines and channels"
				case "review":
					result = "Review completed: Content polished and examples added"
				case completeAction:
					result = fmt.Sprintf("Task completed: %s", t.Description)
				default:
					return nil, fmt.Errorf("unknown action: %s", actionType)
				}

				return result, nil
			}),
			pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, task, result any) (any, string, error) {
				// Post: update task steps and decide routing
				t := task.(*Task)
				resultStr := result.(string)

				// Update task steps (except for complete)
				if actionType != completeAction {
					t.Steps = append(t.Steps, fmt.Sprintf("%s: %s", actionType, resultStr))
					store.Set(ctx, "task", t)
				}

				// Decide next step
				if actionType == completeAction {
					return resultStr, doneRoute, nil // End the graph (no successor for "done")
				}
				return resultStr, "think", nil // Back to thinking
			}),
		)
	}

	// Create action nodes
	research := createActionNode("research")
	draft := createActionNode("draft")
	review := createActionNode("review")
	complete := createActionNode(completeAction)

	// Connect nodes - think decides which action
	think.Connect("research", research)
	think.Connect("draft", draft)
	think.Connect("review", review)
	think.Connect(completeAction, complete)

	// Actions loop back to think (except complete)
	research.Connect("think", think)
	draft.Connect("think", think)
	review.Connect("think", think)
	// complete has no connections (ends the graph)

	// Create and run the agent graph
	fmt.Println("=== Autonomous Agent Demo ===")
	graph := pocket.NewGraph(think, store)

	result, err := graph.Run(ctx, nil)
	if err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	if result == nil {
		// Get the final task state since result is nil
		finalTask, _ := store.Get(ctx, "task")
		if t, ok := finalTask.(*Task); ok && len(t.Steps) > 0 {
			fmt.Printf("\n[FINAL] Task completed with %d steps\n", len(t.Steps))
		}
	} else {
		fmt.Printf("\n[FINAL] %v\n", result)
	}

	// Show execution trace
	finalTask, _ := store.Get(ctx, "task")
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
	store.Set(ctx, "task", task2)

	// Create a flaky action that sometimes fails
	attempts := 0
	analyze := pocket.NewNode[any, any]("analyze",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			fmt.Printf("[PREP] Preparing analysis (attempt %d)\n", attempts+1)
			return input, nil
		}),
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			attempts++
			if attempts < 2 {
				return nil, fmt.Errorf("temporary analysis failure")
			}
			return "Analysis completed successfully", nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			fmt.Printf("[POST] Analysis result: %v\n", result)
			return result, doneRoute, nil
		}),
		pocket.WithRetry(3, 0), // Retry up to 3 times
	)

	// Simple graph: think -> analyze
	think2 := pocket.NewNode[any, any]("think2",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return "Starting analysis", nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			return exec, "analyze", nil
		}),
	)

	think2.Connect("analyze", analyze)

	graph2 := pocket.NewGraph(think2, store)
	result2, err := graph2.Run(ctx, nil)
	if err != nil {
		log.Printf("Retry demo error: %v", err)
	} else {
		fmt.Printf("Retry demo result: %v (after %d attempts)\n", result2, attempts)
	}
}
