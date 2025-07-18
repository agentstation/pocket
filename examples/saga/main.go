// Package main demonstrates the Saga pattern with Pocket.
// This example shows how to implement distributed transactions with
// compensating actions for rollback.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/agentstation/pocket"
)

// Transaction represents a step in the saga.
type Transaction struct {
	Name       string
	Execute    func(ctx context.Context, data any) error
	Compensate func(ctx context.Context, data any) error
}

// SagaProcessor handles transaction execution with compensation.
type SagaProcessor struct {
	transaction Transaction
	store       pocket.Store
}

func (s *SagaProcessor) Process(ctx context.Context, input any) (any, error) {
	// Execute the transaction
	err := s.transaction.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w", s.transaction.Name, err)
	}

	// Record successful transaction for potential rollback
	completedTxns, _ := s.store.Get("completed_transactions")
	completed := append(getTransactionList(completedTxns), s.transaction.Name)
	s.store.Set("completed_transactions", completed)

	return input, nil
}

// CompensatingProcessor handles rollback.
type CompensatingProcessor struct {
	transactions map[string]Transaction
	store        pocket.Store
}

func (c *CompensatingProcessor) Process(ctx context.Context, input any) (any, error) {
	// Get list of completed transactions
	completedTxns, _ := c.store.Get("completed_transactions")
	completed := getTransactionList(completedTxns)

	// Compensate in reverse order
	for i := len(completed) - 1; i >= 0; i-- {
		txnName := completed[i]
		if txn, exists := c.transactions[txnName]; exists {
			fmt.Printf("Compensating: %s\n", txnName)
			if err := txn.Compensate(ctx, input); err != nil {
				return nil, fmt.Errorf("compensation failed for %s: %w", txnName, err)
			}
		}
	}

	// Clear completed transactions
	c.store.Delete("completed_transactions")
	return "Saga rolled back successfully", nil
}

func getTransactionList(data any) []string {
	if list, ok := data.([]string); ok {
		return list
	}
	return []string{}
}

// Order represents our domain object.
type Order struct {
	ID         string
	CustomerID string
	Amount     float64
	Items      []string
	Status     string
}

func main() {
	// Initialize random number generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create store
	store := pocket.NewStore()

	// Define saga transactions
	transactions := map[string]Transaction{
		"reserve_inventory": {
			Name: "reserve_inventory",
			Execute: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Reserving inventory for order %s...\n", order.ID)
				store.Set("inventory_reserved", true)

				// Simulate potential failure
				if rng.Float32() > 0.7 {
					return fmt.Errorf("insufficient inventory")
				}
				return nil
			},
			Compensate: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Releasing inventory reservation for order %s\n", order.ID)
				store.Delete("inventory_reserved")
				return nil
			},
		},
		"charge_payment": {
			Name: "charge_payment",
			Execute: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Charging payment of $%.2f for order %s...\n", order.Amount, order.ID)
				store.Set("payment_charged", order.Amount)

				// Simulate potential failure
				if rng.Float32() > 0.8 {
					return fmt.Errorf("payment declined")
				}
				return nil
			},
			Compensate: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Refunding payment of $%.2f for order %s\n", order.Amount, order.ID)
				store.Delete("payment_charged")
				return nil
			},
		},
		"create_shipment": {
			Name: "create_shipment",
			Execute: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Creating shipment for order %s...\n", order.ID)
				store.Set("shipment_created", order.ID)

				// Simulate potential failure
				if rng.Float32() > 0.9 {
					return fmt.Errorf("shipping service unavailable")
				}
				return nil
			},
			Compensate: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Cancelling shipment for order %s\n", order.ID)
				store.Delete("shipment_created")
				return nil
			},
		},
		"send_confirmation": {
			Name: "send_confirmation",
			Execute: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Sending confirmation email for order %s...\n", order.ID)
				store.Set("confirmation_sent", true)
				return nil
			},
			Compensate: func(ctx context.Context, data any) error {
				order := data.(*Order)
				fmt.Printf("Sending cancellation email for order %s\n", order.ID)
				store.Delete("confirmation_sent")
				return nil
			},
		},
	}

	// Create saga nodes
	reserveInventory := pocket.NewNode("reserve_inventory",
		&SagaProcessor{transaction: transactions["reserve_inventory"], store: store})

	chargePayment := pocket.NewNode("charge_payment",
		&SagaProcessor{transaction: transactions["charge_payment"], store: store})

	createShipment := pocket.NewNode("create_shipment",
		&SagaProcessor{transaction: transactions["create_shipment"], store: store})

	sendConfirmation := pocket.NewNode("send_confirmation",
		&SagaProcessor{transaction: transactions["send_confirmation"], store: store})

	// Create compensation node
	compensate := pocket.NewNode("compensate",
		&CompensatingProcessor{transactions: transactions, store: store})

	// Create success node
	success := pocket.NewNode("success", pocket.ProcessorFunc(func(ctx context.Context, input any) (any, error) {
		order := input.(*Order)
		order.Status = "completed"
		return fmt.Sprintf("Order %s completed successfully!", order.ID), nil
	}))

	// Build saga flow with error handling
	// Each node connects to the next on success, or to compensate on error
	reserveInventory.Default(chargePayment)
	chargePayment.Default(createShipment)
	createShipment.Default(sendConfirmation)
	sendConfirmation.Default(success)

	// Create a wrapper that handles errors and triggers compensation
	sagaFlow := func(ctx context.Context, order *Order) (string, error) {
		// Try each step in sequence
		nodes := []*pocket.Node{reserveInventory, chargePayment, createShipment, sendConfirmation, success}

		for i, node := range nodes {
			flow := pocket.NewFlow(node, store)
			result, err := flow.Run(ctx, order)

			if err != nil {
				fmt.Printf("\n❌ Error at step %d: %v\n", i+1, err)
				fmt.Println("\n🔄 Starting compensation...")

				// Run compensation
				compensateFlow := pocket.NewFlow(compensate, store)
				compResult, compErr := compensateFlow.Run(ctx, order)

				if compErr != nil {
					return "", fmt.Errorf("saga failed and compensation failed: %w", compErr)
				}

				return compResult.(string), err
			}

			// For the last node, return the result
			if i == len(nodes)-1 {
				return result.(string), nil
			}
		}

		return "", fmt.Errorf("unexpected saga state")
	}

	// Demo the saga pattern
	fmt.Println("=== Saga Pattern Demo ===")
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

	ctx := context.Background()

	for _, order := range orders {
		fmt.Printf("\n📦 Processing Order: %s\n", order.ID)
		fmt.Println("------------------------")

		// Clear previous transaction state
		store.Delete("completed_transactions")

		result, err := sagaFlow(ctx, order)

		if err != nil {
			fmt.Printf("\n❌ Order %s failed: %v\n", order.ID, err)
			fmt.Printf("💫 %s\n", result)
		} else {
			fmt.Printf("\n✅ %s\n", result)
		}

		fmt.Println("\n" + strings.Repeat("-", 50))
	}

	fmt.Println("\n=== Saga Demo Complete ===")
}
