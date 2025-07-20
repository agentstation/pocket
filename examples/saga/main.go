// Package main demonstrates the Saga pattern with Pocket using the
// Prep/Exec/Post lifecycle. This example shows how to implement distributed
// transactions with compensating actions for rollback, where Prep validates
// preconditions, Exec performs the transaction, and Post handles routing
// or compensation setup.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/agentstation/pocket"
)

// Order represents our domain object.
type Order struct {
	ID         string
	CustomerID string
	Amount     float64
	Items      []string
	Status     string
}

// TransactionState tracks saga progress.
type TransactionState struct {
	CompletedSteps []string
	FailedStep     string
	RollbackMode   bool
}

func main() {
	// Initialize random number generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create store
	store := pocket.NewStore()
	ctx := context.Background()

	// Create inventory reservation node with lifecycle
	reserveInventory := pocket.NewNode[any, any]("reserve_inventory",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate order
			order, ok := input.(*Order)
			if !ok {
				return nil, fmt.Errorf("expected *Order, got %T", input)
			}
			if len(order.Items) == 0 {
				return nil, fmt.Errorf("order has no items")
			}

			// Check current inventory state
			inventory, exists := store.Get(ctx, "inventory")
			if !exists {
				// Will initialize inventory in post step
				inventory = map[string]int{
					"item-1": 10,
					"item-2": 5,
					"item-3": 3,
					"item-4": 8,
					"item-5": 2,
					"item-6": 4,
				}
			}

			return map[string]interface{}{
				"order":     order,
				"inventory": inventory,
				"needsInit": !exists,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			data := prepData.(map[string]interface{})
			o := data["order"].(*Order)
			fmt.Printf("Reserving inventory for order %s...\n", o.ID)

			// Simulate potential failure
			if rng.Float32() > 0.7 {
				return nil, fmt.Errorf("insufficient inventory")
			}

			// Prepare reservation data for post step
			return map[string]interface{}{
				"order":          o,
				"reservationKey": fmt.Sprintf("reservation:%s", o.ID),
				"items":          o.Items,
				"inventory":      data["inventory"],
				"needsInit":      data["needsInit"],
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})
			o := execResult["order"].(*Order)

			// Initialize inventory if needed
			if execResult["needsInit"].(bool) {
				if err := store.Set(ctx, "inventory", execResult["inventory"]); err != nil {
					return nil, "", fmt.Errorf("failed to store inventory: %w", err)
				}
			}

			// Store reservation
			if err := store.Set(ctx, execResult["reservationKey"].(string), execResult["items"]); err != nil {
				return nil, "", fmt.Errorf("failed to store reservation: %w", err)
			}

			// Record successful step
			state, _ := store.Get(ctx, "transaction_state")
			if state == nil {
				state = &TransactionState{}
			}
			txState := state.(*TransactionState)
			txState.CompletedSteps = append(txState.CompletedSteps, "reserve_inventory")
			if err := store.Set(ctx, "transaction_state", txState); err != nil {
				return nil, "", fmt.Errorf("failed to update transaction state: %w", err)
			}

			return o, "charge_payment", nil
		}),
	)

	// Create payment processing node
	chargePayment := pocket.NewNode[any, any]("charge_payment",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate payment details
			order := input.(*Order)
			if order.Amount <= 0 {
				return nil, fmt.Errorf("invalid payment amount")
			}

			// Check payment method availability
			_, exists := store.Get(ctx, "payment_service_available")

			return map[string]interface{}{
				"order":     order,
				"needsInit": !exists,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			data := prepData.(map[string]interface{})
			o := data["order"].(*Order)
			fmt.Printf("Charging payment of $%.2f for order %s...\n", o.Amount, o.ID)

			// Simulate potential failure
			if rng.Float32() > 0.8 {
				return nil, fmt.Errorf("payment declined")
			}

			// Prepare payment data for post step
			return map[string]interface{}{
				"order":      o,
				"paymentKey": fmt.Sprintf("payment:%s", o.ID),
				"amount":     o.Amount,
				"needsInit":  data["needsInit"],
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})
			o := execResult["order"].(*Order)

			// Initialize payment service if needed
			if execResult["needsInit"].(bool) {
				if err := store.Set(ctx, "payment_service_available", true); err != nil {
					return nil, "", fmt.Errorf("failed to initialize payment service: %w", err)
				}
			}

			// Record payment
			if err := store.Set(ctx, execResult["paymentKey"].(string), execResult["amount"]); err != nil {
				return nil, "", fmt.Errorf("failed to record payment: %w", err)
			}

			// Record successful step
			state, _ := store.Get(ctx, "transaction_state")
			txState := state.(*TransactionState)
			txState.CompletedSteps = append(txState.CompletedSteps, "charge_payment")
			if err := store.Set(ctx, "transaction_state", txState); err != nil {
				return nil, "", fmt.Errorf("failed to update transaction state: %w", err)
			}

			return o, "create_shipment", nil
		}),
	)

	// Create shipment node
	createShipment := pocket.NewNode[any, any]("create_shipment",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Validate shipping address
			order := input.(*Order)

			// In real app, would validate shipping address
			// For demo, just check order status
			if order.Status == "cancelled" {
				return nil, fmt.Errorf("cannot ship cancelled order")
			}

			return order, nil
		}),
		pocket.WithExec(func(ctx context.Context, order any) (any, error) {
			o := order.(*Order)
			fmt.Printf("Creating shipment for order %s...\n", o.ID)

			// Simulate potential failure
			if rng.Float32() > 0.9 {
				return nil, fmt.Errorf("shipping service unavailable")
			}

			// Create shipment
			shipmentID := fmt.Sprintf("SHIP-%s-%d", o.ID, time.Now().Unix())

			return map[string]interface{}{
				"order":       o,
				"shipmentKey": fmt.Sprintf("shipment:%s", o.ID),
				"shipmentID":  shipmentID,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, order, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})
			o := execResult["order"].(*Order)

			// Store shipment
			if err := store.Set(ctx, execResult["shipmentKey"].(string), execResult["shipmentID"]); err != nil {
				return nil, "", fmt.Errorf("failed to store shipment: %w", err)
			}

			// Record successful step
			state, _ := store.Get(ctx, "transaction_state")
			txState := state.(*TransactionState)
			txState.CompletedSteps = append(txState.CompletedSteps, "create_shipment")
			if err := store.Set(ctx, "transaction_state", txState); err != nil {
				return nil, "", fmt.Errorf("failed to update transaction state: %w", err)
			}

			return o, "send_confirmation", nil
		}),
	)

	// Create confirmation node
	sendConfirmation := pocket.NewNode[any, any]("send_confirmation",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Prepare email context
			order := input.(*Order)
			shipmentID, _ := store.Get(ctx, fmt.Sprintf("shipment:%s", order.ID))

			return map[string]interface{}{
				"order":      order,
				"shipmentID": shipmentID,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			d := data.(map[string]interface{})
			order := d["order"].(*Order)

			fmt.Printf("Sending confirmation email for order %s...\n", order.ID)

			// Prepare confirmation data for post step
			return map[string]interface{}{
				"order":            order,
				"confirmationKey":  fmt.Sprintf("confirmation:%s", order.ID),
				"confirmationTime": time.Now(),
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})
			o := execResult["order"].(*Order)
			o.Status = "completed"

			// Record confirmation
			if err := store.Set(ctx, execResult["confirmationKey"].(string), execResult["confirmationTime"]); err != nil {
				return nil, "", fmt.Errorf("failed to record confirmation: %w", err)
			}

			// Mark saga as complete
			state, _ := store.Get(ctx, "transaction_state")
			txState := state.(*TransactionState)
			txState.CompletedSteps = append(txState.CompletedSteps, "send_confirmation")
			if err := store.Set(ctx, "transaction_state", txState); err != nil {
				return nil, "", fmt.Errorf("failed to update transaction state: %w", err)
			}

			return o, "success", nil
		}),
	)

	// Create success node
	success := pocket.NewNode[any, any]("success",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			order := input.(*Order)
			return fmt.Sprintf("Order %s completed successfully!", order.ID), nil
		}),
	)

	// Create compensation node with lifecycle
	compensate := pocket.NewNode[any, any]("compensate",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Get transaction state to know what to compensate
			state, _ := store.Get(ctx, "transaction_state")
			if state == nil {
				return nil, fmt.Errorf("no transaction state found")
			}

			order := input.(*Order)
			txState := state.(*TransactionState)

			return map[string]interface{}{
				"order":   order,
				"txState": txState,
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, data any) (any, error) {
			d := data.(map[string]interface{})
			order := d["order"].(*Order)
			txState := d["txState"].(*TransactionState)

			fmt.Println("\n🔄 Starting compensation...")

			// Prepare keys to delete in post step
			keysToDelete := []string{}

			// Compensate in reverse order
			for i := len(txState.CompletedSteps) - 1; i >= 0; i-- {
				step := txState.CompletedSteps[i]
				fmt.Printf("Compensating: %s\n", step)

				switch step {
				case "send_confirmation":
					fmt.Printf("Sending cancellation email for order %s\n", order.ID)
					keysToDelete = append(keysToDelete, fmt.Sprintf("confirmation:%s", order.ID))

				case "create_shipment":
					fmt.Printf("Cancelling shipment for order %s\n", order.ID)
					keysToDelete = append(keysToDelete, fmt.Sprintf("shipment:%s", order.ID))

				case "charge_payment":
					fmt.Printf("Refunding payment of $%.2f for order %s\n", order.Amount, order.ID)
					keysToDelete = append(keysToDelete, fmt.Sprintf("payment:%s", order.ID))

				case "reserve_inventory":
					fmt.Printf("Releasing inventory reservation for order %s\n", order.ID)
					keysToDelete = append(keysToDelete, fmt.Sprintf("reservation:%s", order.ID))
				}
			}

			return map[string]interface{}{
				"message":      "Saga rolled back successfully",
				"keysToDelete": keysToDelete,
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, data, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})

			// Delete all compensation keys
			for _, key := range execResult["keysToDelete"].([]string) {
				_ = store.Delete(ctx, key)
			}

			// Clear transaction state
			_ = store.Delete(ctx, "transaction_state")

			return execResult["message"], "done", nil
		}),
	)

	// Connect nodes
	reserveInventory.Connect("charge_payment", chargePayment)
	chargePayment.Connect("create_shipment", createShipment)
	createShipment.Connect("send_confirmation", sendConfirmation)
	sendConfirmation.Connect("success", success)

	// Create error handler that triggers compensation
	errorHandler := pocket.NewNode[any, any]("error_handler",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Extract error and order from input
			errData := input.(map[string]interface{})
			return errData, nil
		}),
		pocket.WithExec(func(ctx context.Context, errData any) (any, error) {
			data := errData.(map[string]interface{})
			err := data["error"].(error)
			order := data["order"].(*Order)

			fmt.Printf("\n❌ Error: %v\n", err)

			// Trigger compensation
			compensateGraph := pocket.NewGraph(compensate, store)
			result, compErr := compensateGraph.Run(ctx, order)

			if compErr != nil {
				return nil, fmt.Errorf("saga failed and compensation failed: %w", compErr)
			}

			return result, nil
		}),
	)

	// Demo the saga pattern
	fmt.Println("=== Saga Pattern Demo with Prep/Exec/Post ===")
	fmt.Println("Processing orders with distributed transactions...")
	fmt.Println()

	// Process multiple orders to show both success and failure scenarios
	orders := []*Order{
		{
			ID:         "ORD-001",
			CustomerID: "CUST-123",
			Amount:     99.99,
			Items:      []string{"item-1", "item-2"},
			Status:     "pending",
		},
		{
			ID:         "ORD-002",
			CustomerID: "CUST-456",
			Amount:     149.99,
			Items:      []string{"item-3"},
			Status:     "pending",
		},
		{
			ID:         "ORD-003",
			CustomerID: "CUST-789",
			Amount:     299.99,
			Items:      []string{"item-4", "item-5", "item-6"},
			Status:     "pending",
		},
	}

	for _, order := range orders {
		fmt.Printf("\n📦 Processing Order: %s\n", order.ID)
		fmt.Println("------------------------")

		// Clear previous transaction state
		_ = store.Delete(ctx, "transaction_state")
		if err := store.Set(ctx, "transaction_state", &TransactionState{}); err != nil {
			fmt.Printf("Failed to initialize transaction state: %v\n", err)
			continue
		}

		// Create saga graph
		graph := pocket.NewGraph(reserveInventory, store)
		result, err := graph.Run(ctx, order)

		if err != nil {
			// Handle error by triggering compensation
			errorData := map[string]interface{}{
				"error": err,
				"order": order,
			}

			errorGraph := pocket.NewGraph(errorHandler, store)
			compResult, _ := errorGraph.Run(ctx, errorData)

			fmt.Printf("\n❌ Order %s failed: %v\n", order.ID, err)
			if compResult != nil {
				fmt.Printf("💫 %s\n", compResult)
			}
		} else {
			fmt.Printf("\n✅ %s\n", result)
		}

		fmt.Println("\n" + strings.Repeat("-", 50))
	}

	// Demonstrate builder pattern for saga
	fmt.Println("\n=== Saga Builder Pattern ===")

	// Create a more complex saga with builder
	_, err := pocket.NewBuilder(store).
		Add(reserveInventory).
		Add(chargePayment).
		Add(createShipment).
		Add(sendConfirmation).
		Add(success).
		Add(compensate).
		Add(errorHandler).
		Connect("reserve_inventory", "charge_payment", "charge_payment").
		Connect("charge_payment", "create_shipment", "create_shipment").
		Connect("create_shipment", "send_confirmation", "send_confirmation").
		Connect("send_confirmation", "success", "success").
		Start("reserve_inventory").
		Build()

	if err != nil {
		fmt.Printf("Failed to build saga: %v\n", err)
	} else {
		fmt.Println("Successfully built saga graph with builder pattern")
	}

	fmt.Println("\n=== Saga Demo Complete ===")
}
