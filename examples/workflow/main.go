// Package main demonstrates a complex multi-stage workflow with the
// Prep/Exec/Post lifecycle pattern. This example shows how to build
// workflows where Prep validates preconditions, Exec performs business
// logic, and Post handles routing decisions and state updates.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/agentstation/pocket"
)

// WorkflowData represents data flowing through the workflow.
type WorkflowData struct {
	OrderID    string
	CustomerID string
	Items      []Item
	Status     string
	Timestamp  time.Time
}

// Item represents a product in an order.
type Item struct {
	SKU      string
	Quantity int
	Price    float64
}

// ValidationResult contains validation outcome and reasons.
type ValidationResult struct {
	Data   WorkflowData
	Valid  bool
	Errors []string
}

func main() {
	// Create workflow store
	store := pocket.NewStore()
	ctx := context.Background()

	// Create order validator node with lifecycle
	validator := pocket.NewNode[any, any]("validate",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Ensure we have WorkflowData
			data, ok := input.(WorkflowData)
			if !ok {
				return nil, fmt.Errorf("expected WorkflowData, got %T", input)
			}

			// Initialize status
			data.Status = "validating"
			data.Timestamp = time.Now()

			return data, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Perform validation
			order := data.(WorkflowData)
			var errs []string

			if order.OrderID == "" {
				errs = append(errs, "order ID is required")
			}
			if order.CustomerID == "" {
				errs = append(errs, "customer ID is required")
			}
			if len(order.Items) == 0 {
				errs = append(errs, "order must contain at least one item")
			}

			for i, item := range order.Items {
				if item.Quantity <= 0 {
					errs = append(errs, fmt.Sprintf("item %d: invalid quantity", i))
				}
				if item.Price < 0 {
					errs = append(errs, fmt.Sprintf("item %d: invalid price", i))
				}
			}

			fmt.Printf("[Validator] Validated order %s: %d errors\n", order.OrderID, len(errs))

			return ValidationResult{
				Data:   order,
				Valid:  len(errs) == 0,
				Errors: errs,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, result any) (any, string, error) {
			// Route based on validation result
			valResult := result.(ValidationResult)

			// Store validation result
			store.Set(ctx, fmt.Sprintf("order:%s:validation", valResult.Data.OrderID), valResult)

			if valResult.Valid {
				valResult.Data.Status = "validated"
				return valResult, "inventory", nil
			}
			return valResult, "error", nil
		}),
	)

	// Create inventory checker node
	inventoryCheck := pocket.NewNode[any, any]("inventory",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load inventory data from store
			result := input.(ValidationResult)

			// Check if inventory exists
			inventory, exists := store.Get(ctx, "inventory")
			if !exists {
				// Will initialize in post step
				inventory = map[string]int{
					"WIDGET-1":     100,
					"GADGET-2":     50,
					"PREMIUM-1":    200,
					"OUT-OF-STOCK": 0,
				}
			}

			return map[string]interface{}{
				"result":    result,
				"inventory": inventory,
				"needsInit": !exists,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Check inventory availability
			data := prepData.(map[string]interface{})
			result := data["result"].(ValidationResult)
			stock := data["inventory"].(map[string]int)

			for _, item := range result.Data.Items {
				if available, ok := stock[item.SKU]; !ok || available < item.Quantity {
					return nil, fmt.Errorf("item %s is out of stock (requested: %d, available: %d)",
						item.SKU, item.Quantity, available)
				}
			}

			fmt.Printf("[Inventory] All items available for order %s\n", result.Data.OrderID)
			result.Data.Status = "inventory_checked"

			return map[string]interface{}{
				"result":    result,
				"inventory": data["inventory"],
				"needsInit": data["needsInit"],
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})
			valResult := execResult["result"].(ValidationResult)

			// Initialize inventory if needed
			if execResult["needsInit"].(bool) {
				store.Set(ctx, "inventory", execResult["inventory"])
			}

			// Reserve inventory
			inventory, _ := store.Get(ctx, "inventory")
			stock := inventory.(map[string]int)

			for _, item := range valResult.Data.Items {
				stock[item.SKU] -= item.Quantity
			}
			store.Set(ctx, "inventory", stock)

			// Store reservation
			store.Set(ctx, fmt.Sprintf("order:%s:reserved_items", valResult.Data.OrderID), valResult.Data.Items)

			return valResult, "payment", nil
		}),
	)

	// Create payment processor node
	payment := pocket.NewNode[any, any]("payment",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Calculate order total
			result := input.(ValidationResult)
			total := 0.0

			for _, item := range result.Data.Items {
				total += float64(item.Quantity) * item.Price
			}

			return map[string]interface{}{
				"result": result,
				"total":  total,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Process payment
			d := data.(map[string]interface{})
			result := d["result"].(ValidationResult)
			total := d["total"].(float64)

			// Simulate payment processing
			if total > 10000 {
				return nil, errors.New("payment declined: amount exceeds limit")
			}

			fmt.Printf("[Payment] Processed payment for order %s: $%.2f\n", result.Data.OrderID, total)
			result.Data.Status = "payment_processed"

			return result, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, result any) (any, string, error) {
			// Record payment
			valResult := result.(ValidationResult)
			d := data.(map[string]interface{})
			total := d["total"].(float64)

			store.Set(ctx, fmt.Sprintf("order:%s:payment", valResult.Data.OrderID), total)
			store.Set(ctx, fmt.Sprintf("order:%s:payment_time", valResult.Data.OrderID), time.Now())

			return valResult, "fulfillment", nil
		}),
	)

	// Create fulfillment service node
	fulfillment := pocket.NewNode[any, any]("fulfillment",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Check shipping address (simulate)
			result := input.(ValidationResult)

			// Load customer shipping info
			shipInfo, exists := store.Get(ctx, fmt.Sprintf("customer:%s:shipping", result.Data.CustomerID))
			if !exists {
				// Create default shipping info
				shipInfo = map[string]string{
					"address": "123 Main St",
					"city":    "Anytown",
					"state":   "CA",
					"zip":     "12345",
				}
			}

			return map[string]interface{}{
				"needsShipInfo": !exists,
				"shipInfoKey":   fmt.Sprintf("customer:%s:shipping", result.Data.CustomerID),
				"result":        result,
				"shipping":      shipInfo,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Create fulfillment order
			d := data.(map[string]interface{})
			result := d["result"].(ValidationResult)

			fmt.Printf("[Fulfillment] Order %s sent to warehouse\n", result.Data.OrderID)
			result.Data.Status = "fulfilled"

			// Generate tracking number
			trackingNum := fmt.Sprintf("TRACK-%s-%d", result.Data.OrderID, time.Now().Unix())

			return map[string]interface{}{
				"result":   result,
				"tracking": trackingNum,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, output any) (any, string, error) {
			// Store fulfillment info
			out := output.(map[string]interface{})
			result := out["result"].(ValidationResult)
			tracking := out["tracking"].(string)

			// Store shipping info if it was created
			prep := prepData.(map[string]interface{})
			if prep["needsShipInfo"].(bool) {
				store.Set(ctx, prep["shipInfoKey"].(string), prep["shipping"])
			}

			store.Set(ctx, fmt.Sprintf("order:%s:tracking", result.Data.OrderID), tracking)
			store.Set(ctx, fmt.Sprintf("order:%s:status", result.Data.OrderID), "fulfilled")

			return result, "notify", nil
		}),
	)

	// Create notification service node
	notification := pocket.NewNode[any, any]("notify",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Load customer preferences
			var result ValidationResult
			switch v := input.(type) {
			case ValidationResult:
				result = v
			default:
				return nil, fmt.Errorf("unexpected input type: %T", input)
			}

			prefs, exists := store.Get(ctx, fmt.Sprintf("customer:%s:notification_prefs", result.Data.CustomerID))
			if !exists {
				prefs = map[string]bool{
					"email": true,
					"sms":   false,
				}
			}

			return map[string]interface{}{
				"result": result,
				"prefs":  prefs,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			// Send notifications
			d := data.(map[string]interface{})
			result := d["result"].(ValidationResult)
			prefs := d["prefs"].(map[string]bool)

			message := fmt.Sprintf("Order %s status: %s", result.Data.OrderID, result.Data.Status)

			if prefs["email"] {
				fmt.Printf("[Notification] Email sent to customer %s: %s\n", result.Data.CustomerID, message)
			}
			if prefs["sms"] {
				fmt.Printf("[Notification] SMS sent to customer %s: %s\n", result.Data.CustomerID, message)
			}

			return result, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, result any) (any, string, error) {
			// Log notification sent
			valResult := result.(ValidationResult)
			notificationLog, _ := store.Get(ctx, fmt.Sprintf("customer:%s:notifications", valResult.Data.CustomerID))
			if notificationLog == nil {
				notificationLog = []string{}
			}
			log := notificationLog.([]string)
			log = append(log, fmt.Sprintf("%s: Order %s - %s", time.Now().Format(time.RFC3339), valResult.Data.OrderID, valResult.Data.Status))
			store.Set(ctx, fmt.Sprintf("customer:%s:notifications", valResult.Data.CustomerID), log)

			return valResult, "done", nil
		}),
	)

	// Create error handler node
	errorHandler := pocket.NewNode[any, any]("error",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Prepare error context
			return input, nil
		}),
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Handle various error types
			switch v := input.(type) {
			case ValidationResult:
				if !v.Valid {
					fmt.Printf("[Error] Order %s validation failed: %v\n", v.Data.OrderID, v.Errors)
					v.Data.Status = "validation_failed"
				}
				return v, nil
			default:
				fmt.Printf("[Error] Processing failed: %v\n", input)
				return input, nil
			}
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, result any) (any, string, error) {
			// Log error and send notification
			if valResult, ok := result.(ValidationResult); ok {
				store.Set(ctx, fmt.Sprintf("order:%s:error", valResult.Data.OrderID), valResult.Errors)
				return valResult, "notify", nil
			}
			return result, "done", nil
		}),
	)

	// Connect workflow nodes
	validator.Connect("inventory", inventoryCheck)
	validator.Connect("error", errorHandler)
	inventoryCheck.Connect("payment", payment)
	payment.Connect("fulfillment", fulfillment)
	fulfillment.Connect("notify", notification)
	errorHandler.Connect("notify", notification)

	// Test orders
	orders := []WorkflowData{
		{
			OrderID:    "ORD-001",
			CustomerID: "CUST-123",
			Items: []Item{
				{SKU: "WIDGET-1", Quantity: 2, Price: 29.99},
				{SKU: "GADGET-2", Quantity: 1, Price: 99.99},
			},
			Timestamp: time.Now(),
		},
		{
			OrderID:    "ORD-002",
			CustomerID: "CUST-456",
			Items: []Item{
				{SKU: "OUT-OF-STOCK", Quantity: 1, Price: 49.99},
			},
			Timestamp: time.Now(),
		},
		{
			OrderID:    "ORD-003",
			CustomerID: "",
			Items:      []Item{},
			Timestamp:  time.Now(),
		},
		{
			OrderID:    "ORD-004",
			CustomerID: "CUST-789",
			Items: []Item{
				{SKU: "PREMIUM-1", Quantity: 100, Price: 150.00}, // Will exceed payment limit
			},
			Timestamp: time.Now(),
		},
	}

	fmt.Println("=== E-commerce Workflow Demo with Prep/Exec/Post ===")
	fmt.Println()

	for _, order := range orders {
		fmt.Printf("--- Processing Order %s ---\n", order.OrderID)

		// Create new flow for each order
		flow := pocket.NewFlow(validator, store)

		result, err := flow.Run(ctx, order)
		if err != nil {
			fmt.Printf("Workflow error for order %s: %v\n", order.OrderID, err)

			// Attempt error recovery flow
			errorFlow := pocket.NewFlow(errorHandler, store)
			_, _ = errorFlow.Run(ctx, order)
		} else {
			fmt.Printf("\n✅ Order %s completed successfully\n", order.OrderID)
			if valResult, ok := result.(ValidationResult); ok {
				fmt.Printf("Final status: %s\n", valResult.Data.Status)
			}
		}

		fmt.Println()
	}

	// Show workflow statistics
	fmt.Println("=== Workflow Statistics ===")

	// Show inventory status
	if inventory, exists := store.Get(ctx, "inventory"); exists {
		stock := inventory.(map[string]int)
		fmt.Println("\nCurrent Inventory:")
		for sku, qty := range stock {
			fmt.Printf("  %s: %d units\n", sku, qty)
		}
	}

	// Show order summaries
	fmt.Println("\nOrder Summaries:")
	for _, order := range orders {
		if status, exists := store.Get(ctx, fmt.Sprintf("order:%s:status", order.OrderID)); exists {
			payment, _ := store.Get(ctx, fmt.Sprintf("order:%s:payment", order.OrderID))
			tracking, _ := store.Get(ctx, fmt.Sprintf("order:%s:tracking", order.OrderID))

			fmt.Printf("  Order %s: Status=%s", order.OrderID, status)
			if payment != nil {
				fmt.Printf(", Amount=$%.2f", payment.(float64))
			}
			if tracking != nil {
				fmt.Printf(", Tracking=%s", tracking.(string))
			}
			fmt.Println()
		} else if valResult, exists := store.Get(ctx, fmt.Sprintf("order:%s:validation", order.OrderID)); exists {
			val := valResult.(ValidationResult)
			if !val.Valid {
				fmt.Printf("  Order %s: Failed validation - %v\n", order.OrderID, val.Errors)
			}
		}
	}

	// Demonstrate builder pattern for complex workflow
	fmt.Println("\n=== Workflow Builder Pattern ===")

	// Create a monitoring wrapper node
	monitor := func(name string, node *pocket.Node) *pocket.Node {
		return pocket.NewNode[any, any](name+"-monitor",
			pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
				fmt.Printf("[Monitor] Entering stage: %s\n", name)
				return input, nil
			}),
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				// Pass through to wrapped node
				flow := pocket.NewFlow(node, store)
				return flow.Run(ctx, input)
			}),
			pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, result any) (any, string, error) {
				fmt.Printf("[Monitor] Completed stage: %s\n", name)
				// Forward to same route as wrapped node would
				return result, "next", nil
			}),
		)
	}

	// Build workflow with monitoring
	_, err := pocket.NewBuilder(store).
		Add(validator).
		Add(monitor("inventory", inventoryCheck)).
		Add(monitor("payment", payment)).
		Add(monitor("fulfillment", fulfillment)).
		Add(notification).
		Add(errorHandler).
		Connect("validate", "inventory", "inventory-monitor").
		Connect("validate", "error", "error").
		Connect("inventory-monitor", "next", "payment-monitor").
		Connect("payment-monitor", "next", "fulfillment-monitor").
		Connect("fulfillment-monitor", "next", "notify").
		Connect("error", "notify", "notify").
		Start("validate").
		Build()

	if err != nil {
		fmt.Printf("Failed to build workflow: %v\n", err)
	} else {
		fmt.Println("Successfully built monitored workflow with builder pattern")
	}
}
