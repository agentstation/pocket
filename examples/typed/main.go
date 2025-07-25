// Package main demonstrates type-safe workflow construction using the new Steps
// struct format with comprehensive type safety at compile-time, init-time,
// and runtime through the Prep/Exec/Post lifecycle pattern.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/agentstation/pocket"
)

// Domain types for our workflow
type User struct {
	ID    string
	Name  string
	Email string
}

type ValidationResult struct {
	User   User
	Valid  bool
	Errors []string
}

type EnrichmentResult struct {
	User       User
	Department string
	Manager    string
	Role       string
}

type NotificationResult struct {
	MessageID string
	Status    string
	Recipient string
}

func main() {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create typed validator node using the new Steps struct format
	// NewNode[User, ValidationResult] provides compile-time type safety
	validator := pocket.NewNode[User, ValidationResult]("validate",
		pocket.Steps{
			// Prep prepares data with read-only store access
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				// Type assertion needed for untyped prep function
				user := input.(User)
				user.Email = strings.ToLower(strings.TrimSpace(user.Email))
				user.Name = strings.TrimSpace(user.Name)
				return user, nil
			},
			// Exec performs the main validation logic
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				user := prepResult.(User)
				errors := []string{}

				if user.ID == "" {
					errors = append(errors, "ID is required")
				}
				if user.Name == "" {
					errors = append(errors, "name is required")
				}
				if user.Email == "" {
					errors = append(errors, "email is required")
				} else if !strings.Contains(user.Email, "@") {
					errors = append(errors, "email must contain @")
				}

				return ValidationResult{
					User:   user,
					Valid:  len(errors) == 0,
					Errors: errors,
				}, nil
			},
			// Post processes results and determines routing
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prepResult, execResult any) (any, string, error) {
				user := input.(User)
				result := execResult.(ValidationResult)
				// Post: route based on validation
				if result.Valid {
					// Store validated user for audit
					store.Set(ctx, fmt.Sprintf("user:%s:validated", user.ID), true)
					return result, "valid", nil
				}
				return result, "invalid", nil
			},
		},
	)

	// Create typed enricher node using new Steps struct format
	enricher := pocket.NewNode[ValidationResult, EnrichmentResult]("enrich",
		pocket.Steps{
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				result := input.(ValidationResult)
				// Check if we have cached enrichment data
				cacheKey := fmt.Sprintf("user:%s:enrichment", result.User.ID)
				if cached, exists := store.Get(ctx, cacheKey); exists {
					return cached, nil
				}
				return result, nil
			},
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				// If we got cached data, return it
				if enriched, ok := prepResult.(EnrichmentResult); ok {
					return enriched, nil
				}

				// Otherwise, enrich the validated user
				result := prepResult.(ValidationResult)

				// Simulate looking up additional data (in real app, call external service)
				enriched := EnrichmentResult{
					User:       result.User,
					Department: "Engineering",
					Manager:    "Jane Smith",
					Role:       "Software Engineer",
				}

				// Determine department based on email domain
				if strings.HasSuffix(result.User.Email, "@sales.example.com") {
					enriched.Department = "Sales"
					enriched.Manager = "Bob Johnson"
					enriched.Role = "Sales Representative"
				} else if strings.HasSuffix(result.User.Email, "@hr.example.com") {
					enriched.Department = "Human Resources"
					enriched.Manager = "Alice Brown"
					enriched.Role = "HR Specialist"
				}

				return enriched, nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prepResult, execResult any) (any, string, error) {
				validationResult := input.(ValidationResult)
				result := execResult.(EnrichmentResult)
				// Cache enrichment data
				cacheKey := fmt.Sprintf("user:%s:enrichment", validationResult.User.ID)
				store.Set(ctx, cacheKey, result)

				// Always proceed to notification
				return result, "notify", nil
			},
		},
	)

	// Create typed notifier node
	notifier := pocket.NewNode[EnrichmentResult, NotificationResult]("notify",
		pocket.Steps{
			Prep: func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				enriched := input.(EnrichmentResult)
				// Prepare notification context
				return map[string]interface{}{
					"user":       enriched.User,
					"department": enriched.Department,
					"manager":    enriched.Manager,
					"role":       enriched.Role,
				}, nil
			},
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				// With typed nodes, the framework ensures type safety
				d := prepResult.(map[string]interface{})
				user := d["user"].(User)
				dept := d["department"].(string)
				manager := d["manager"].(string)

				// Simulate sending notification
				messageID := fmt.Sprintf("msg-%s-%d", user.ID, len(user.Name))

				// Log the notification
				fmt.Printf("  📧 Sending welcome email to %s (%s)\n", user.Email, dept)
				fmt.Printf("     Manager: %s\n", manager)

				return NotificationResult{
					MessageID: messageID,
					Status:    "sent",
					Recipient: user.Email,
				}, nil
			},
			Post: func(ctx context.Context, store pocket.StoreWriter, input, prepResult, execResult any) (any, string, error) {
				enriched := input.(EnrichmentResult)
				result := execResult.(NotificationResult)
				// Record notification sent
				store.Set(ctx, fmt.Sprintf("user:%s:notified", enriched.User.ID), result.MessageID)
				return result, "done", nil
			},
		},
	)

	// Create error handler for invalid users - using untyped node for flexibility
	errorHandler := pocket.NewNode[any, any]("error",
		pocket.Steps{
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				// prepResult is the input since we don't have a Prep function
				result := prepResult.(ValidationResult)

				// Log validation errors
				fmt.Printf("  ❌ Validation failed for user %s:\n", result.User.ID)
				for _, err := range result.Errors {
					fmt.Printf("     - %s\n", err)
				}

				return fmt.Sprintf("Validation failed with %d errors", len(result.Errors)), nil
			},
		},
	)

	// Connect nodes
	validator.Connect("valid", enricher)
	validator.Connect("invalid", errorHandler)
	enricher.Connect("notify", notifier)

	// Validate the graph before running
	fmt.Println("=== Type-Safe Workflow Demo ===")
	fmt.Println("\nValidating graph types...")
	if err := pocket.ValidateGraph(validator); err != nil {
		log.Fatalf("Graph validation failed: %v", err)
	}
	fmt.Println("✅ Graph validation passed!")

	// Test cases
	testUsers := []User{
		{
			ID:    "123",
			Name:  "John Doe",
			Email: "john@example.com",
		},
		{
			ID:    "456",
			Name:  "Jane Sales",
			Email: "jane@sales.example.com",
		},
		{
			ID:    "789",
			Name:  "",
			Email: "missing-name@example.com",
		},
		{
			ID:    "",
			Name:  "No ID",
			Email: "invalid-email",
		},
	}

	// Process each user
	for _, user := range testUsers {
		fmt.Printf("\n--- Processing User: %s ---\n", user.ID)

		graph := pocket.NewGraph(validator, store)
		result, err := graph.Run(ctx, user)
		if err != nil {
			fmt.Printf("Graph error: %v\n", err)
			continue
		}

		if notification, ok := result.(NotificationResult); ok {
			fmt.Printf("✅ Success! Message ID: %s\n", notification.MessageID)
		} else {
			fmt.Printf("Result: %v\n", result)
		}
	}

	// Demonstrate type mismatch detection
	fmt.Println("\n=== Type Mismatch Detection ===")

	// Create a node with incompatible input type
	wrongTypedNode := pocket.NewNode[string, NotificationResult]("wrong-type",
		pocket.Steps{
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				// This would never execute due to type mismatch
				return NotificationResult{}, nil
			},
		},
	)

	// Try to connect incompatible nodes
	enricher.Connect("error", wrongTypedNode)

	// This should fail validation
	fmt.Println("Testing incompatible node connection...")
	if err := pocket.ValidateGraph(validator); err != nil {
		fmt.Printf("✅ Correctly caught type mismatch: %v\n", err)
	} else {
		fmt.Println("❌ Type mismatch should have been detected!")
	}

	// Demonstrate regular options with typed nodes
	fmt.Println("\n=== Using Regular Options with Typed Nodes ===")

	// Create a typed node with retry and timeout using regular options
	reliableProcessor := pocket.NewNode[string, User]("reliable-processor",
		pocket.Steps{
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				// prepResult is the input string
				input := prepResult.(string)
				// Parse user from string with possible transient failures
				parts := strings.Split(input, ",")
				if len(parts) != 3 {
					return User{}, fmt.Errorf("invalid input format")
				}

				return User{
					ID:    strings.TrimSpace(parts[0]),
					Name:  strings.TrimSpace(parts[1]),
					Email: strings.TrimSpace(parts[2]),
				}, nil
			},
		},
		// Regular options work directly!
		pocket.WithRetry(3, 100*time.Millisecond),
		pocket.WithTimeout(5*time.Second),
		pocket.WithErrorHandler(func(err error) {
			fmt.Printf("Error handler: %v\n", err)
		}),
	)

	// Test the reliable processor
	fmt.Println("Testing reliable processor with retry and timeout...")
	reliableGraph := pocket.NewGraph(reliableProcessor, store)
	_, _ = reliableGraph.Run(ctx, "888,Test User,test@example.com")

	fmt.Println("\n=== Builder Pattern with Typed Nodes ===")

	// Create a more complex typed workflow - demonstrating the Steps struct format
	preprocessor := pocket.NewNode[string, User]("preprocess",
		pocket.Steps{
			Exec: func(ctx context.Context, prepResult any) (any, error) {
				input := prepResult.(string)
				// Parse user from string (e.g., CSV line)
				parts := strings.Split(input, ",")
				if len(parts) != 3 {
					return User{}, fmt.Errorf("invalid input format")
				}

				return User{
					ID:    strings.TrimSpace(parts[0]),
					Name:  strings.TrimSpace(parts[1]),
					Email: strings.TrimSpace(parts[2]),
				}, nil
			},
		},
	)

	// Build complete graph
	graph, err := pocket.NewBuilder(store).
		Add(preprocessor).
		Add(validator).
		Add(enricher).
		Add(notifier).
		Add(errorHandler).
		Connect("preprocess", "default", "validate").
		Connect("validate", "valid", "enrich").
		Connect("validate", "invalid", "error").
		Connect("enrich", "notify", "notify").
		Start("preprocess").
		Build()

	if err != nil {
		log.Fatalf("Failed to build graph: %v", err)
	}

	// Test with CSV input
	csvInput := "999,Alice Developer,alice@example.com"
	fmt.Printf("\nProcessing CSV input: %s\n", csvInput)

	result, err := graph.Run(ctx, csvInput)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if notification, ok := result.(NotificationResult); ok {
		fmt.Printf("✅ Successfully processed! Message ID: %s\n", notification.MessageID)
	}
}
