# Workflow Patterns

## Overview

Workflow patterns represent common solutions to recurring process automation challenges. This guide covers patterns like Saga, compensation, orchestration, and complex business process flows.

## Saga Pattern

The Saga pattern manages distributed transactions across multiple services, providing compensation for failures:

```go
type SagaStep struct {
    Name        string
    Execute     pocket.Node
    Compensate  pocket.Node
}

type Saga struct {
    Steps       []SagaStep
    CompletedSteps []string
}

// Order processing saga
func CreateOrderSaga() *pocket.Graph {
    // Step 1: Reserve inventory
    reserveInventory := pocket.NewNode[Order, InventoryReservation]("reserve-inventory",
        pocket.WithExec(func(ctx context.Context, order Order) (InventoryReservation, error) {
            reservation, err := inventoryService.Reserve(order.Items)
            if err != nil {
                return InventoryReservation{}, fmt.Errorf("inventory reservation failed: %w", err)
            }
            return reservation, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            order Order, prep, reservation any) (InventoryReservation, string, error) {
            
            // Track completed step
            steps, _ := store.Get(ctx, "saga:completed")
            completed := append(steps.([]string), "reserve-inventory")
            store.Set(ctx, "saga:completed", completed)
            
            // Store reservation for potential compensation
            store.Set(ctx, "reservation:inventory", reservation)
            
            return reservation.(InventoryReservation), "charge-payment", nil
        }),
    )
    
    // Compensation for inventory
    compensateInventory := pocket.NewNode[InventoryReservation, any]("compensate-inventory",
        pocket.WithExec(func(ctx context.Context, reservation InventoryReservation) (any, error) {
            err := inventoryService.Release(reservation.ID)
            if err != nil {
                log.Printf("Failed to compensate inventory: %v", err)
            }
            return nil, nil
        }),
    )
    
    // Step 2: Charge payment
    chargePayment := pocket.NewNode[Order, PaymentResult]("charge-payment",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, order Order) (any, error) {
            reservation, _ := store.Get(ctx, "reservation:inventory")
            return map[string]any{
                "order":       order,
                "reservation": reservation,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (PaymentResult, error) {
            data := prepData.(map[string]any)
            order := data["order"].(Order)
            
            result, err := paymentService.Charge(order.CustomerID, order.Total)
            if err != nil {
                return PaymentResult{}, fmt.Errorf("payment failed: %w", err)
            }
            
            return result, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            order Order, prep, payment any) (PaymentResult, string, error) {
            
            result := payment.(PaymentResult)
            
            if !result.Success {
                // Trigger compensation
                return result, "compensate", nil
            }
            
            // Track completed step
            steps, _ := store.Get(ctx, "saga:completed")
            completed := append(steps.([]string), "charge-payment")
            store.Set(ctx, "saga:completed", completed)
            
            store.Set(ctx, "payment:result", result)
            
            return result, "create-shipment", nil
        }),
    )
    
    // Compensation for payment
    compensatePayment := pocket.NewNode[PaymentResult, any]("compensate-payment",
        pocket.WithExec(func(ctx context.Context, payment PaymentResult) (any, error) {
            if payment.TransactionID != "" {
                err := paymentService.Refund(payment.TransactionID)
                if err != nil {
                    log.Printf("Failed to refund payment: %v", err)
                }
            }
            return nil, nil
        }),
    )
    
    // Step 3: Create shipment
    createShipment := pocket.NewNode[Order, ShipmentResult]("create-shipment",
        pocket.WithExec(func(ctx context.Context, order Order) (ShipmentResult, error) {
            shipment, err := shippingService.CreateShipment(order)
            if err != nil {
                return ShipmentResult{}, fmt.Errorf("shipment creation failed: %w", err)
            }
            return shipment, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            order Order, prep, shipment any) (ShipmentResult, string, error) {
            
            result := shipment.(ShipmentResult)
            
            if !result.Success {
                return result, "compensate", nil
            }
            
            // All steps completed successfully
            return result, "complete", nil
        }),
    )
    
    // Compensation orchestrator
    compensate := pocket.NewNode[any, any]("compensate",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
            completed, _ := store.Get(ctx, "saga:completed")
            return completed, nil
        }),
        pocket.WithExec(func(ctx context.Context, completed any) (any, error) {
            steps := completed.([]string)
            
            // Compensate in reverse order
            for i := len(steps) - 1; i >= 0; i-- {
                switch steps[i] {
                case "create-shipment":
                    // Cancel shipment
                    shippingService.CancelShipment()
                case "charge-payment":
                    // Refund payment
                    compensatePayment.Exec(ctx, nil)
                case "reserve-inventory":
                    // Release inventory
                    compensateInventory.Exec(ctx, nil)
                }
            }
            
            return "compensated", nil
        }),
    )
    
    // Connect saga flow
    reserveInventory.Connect("charge-payment", chargePayment)
    chargePayment.Connect("create-shipment", createShipment)
    chargePayment.Connect("compensate", compensate)
    createShipment.Connect("compensate", compensate)
    createShipment.Connect("complete", completeNode)
    
    return pocket.NewGraph(reserveInventory, pocket.NewStore())
}
```

## Orchestration Pattern

Central coordination of multiple services:

```go
type Orchestrator struct {
    Services map[string]ServiceNode
    Workflow WorkflowDefinition
}

// Loan approval orchestration
func CreateLoanOrchestrator() *pocket.Graph {
    // Orchestrator node
    orchestrate := pocket.NewNode[LoanApplication, LoanDecision]("orchestrate",
        pocket.WithExec(func(ctx context.Context, app LoanApplication) (LoanDecision, error) {
            // Parallel service calls
            var wg sync.WaitGroup
            results := make(map[string]any)
            errors := make(map[string]error)
            var mu sync.Mutex
            
            // Credit check
            wg.Add(1)
            go func() {
                defer wg.Done()
                result, err := creditService.CheckCredit(app.CustomerID)
                mu.Lock()
                results["credit"] = result
                errors["credit"] = err
                mu.Unlock()
            }()
            
            // Income verification
            wg.Add(1)
            go func() {
                defer wg.Done()
                result, err := incomeService.Verify(app.CustomerID, app.RequestedAmount)
                mu.Lock()
                results["income"] = result
                errors["income"] = err
                mu.Unlock()
            }()
            
            // Property valuation
            if app.PropertyID != "" {
                wg.Add(1)
                go func() {
                    defer wg.Done()
                    result, err := propertyService.Evaluate(app.PropertyID)
                    mu.Lock()
                    results["property"] = result
                    errors["property"] = err
                    mu.Unlock()
                }()
            }
            
            wg.Wait()
            
            // Check for errors
            for service, err := range errors {
                if err != nil {
                    return LoanDecision{
                        Approved: false,
                        Reason:   fmt.Sprintf("%s check failed: %v", service, err),
                    }, nil
                }
            }
            
            // Make decision based on all results
            decision := evaluateLoanApplication(app, results)
            return decision, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            app LoanApplication, prep, decision any) (LoanDecision, string, error) {
            
            dec := decision.(LoanDecision)
            
            // Store decision
            store.Set(ctx, "loan:decision:"+app.ID, dec)
            
            if dec.Approved {
                return dec, "prepare-documents", nil
            }
            return dec, "notify-rejection", nil
        }),
    )
    
    return pocket.NewGraph(orchestrate, pocket.NewStore())
}
```

## Choreography Pattern

Decentralized coordination through events:

```go
type Event struct {
    ID        string
    Type      string
    Source    string
    Timestamp time.Time
    Data      any
}

// Event-driven order fulfillment
func CreateEventDrivenWorkflow() *pocket.Graph {
    // Order service
    orderService := pocket.NewNode[Order, Event]("order-service",
        pocket.WithExec(func(ctx context.Context, order Order) (Event, error) {
            // Process order
            processedOrder := processOrder(order)
            
            // Emit event
            return Event{
                ID:        generateID(),
                Type:      "OrderCreated",
                Source:    "order-service",
                Timestamp: time.Now(),
                Data:      processedOrder,
            }, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            order Order, prep, event any) (Event, string, error) {
            
            evt := event.(Event)
            
            // Publish to event bus
            publishEvent(evt)
            
            return evt, "wait-for-events", nil
        }),
    )
    
    // Inventory service reacts to OrderCreated
    inventoryService := pocket.NewNode[Event, Event]("inventory-service",
        pocket.WithExec(func(ctx context.Context, event Event) (Event, error) {
            if event.Type != "OrderCreated" {
                return Event{}, nil
            }
            
            order := event.Data.(Order)
            
            // Check inventory
            available := checkInventory(order.Items)
            
            // Emit response event
            return Event{
                ID:        generateID(),
                Type:      "InventoryChecked",
                Source:    "inventory-service",
                Timestamp: time.Now(),
                Data: map[string]any{
                    "orderID":   order.ID,
                    "available": available,
                },
            }, nil
        }),
    )
    
    // Payment service reacts to InventoryChecked
    paymentService := pocket.NewNode[Event, Event]("payment-service",
        pocket.WithExec(func(ctx context.Context, event Event) (Event, error) {
            if event.Type != "InventoryChecked" {
                return Event{}, nil
            }
            
            data := event.Data.(map[string]any)
            if !data["available"].(bool) {
                return Event{}, nil
            }
            
            // Process payment
            paymentResult := processPayment(data["orderID"].(string))
            
            return Event{
                ID:        generateID(),
                Type:      "PaymentProcessed",
                Source:    "payment-service",
                Timestamp: time.Now(),
                Data:      paymentResult,
            }, nil
        }),
    )
    
    // Connect services through event routing
    eventRouter := pocket.NewNode[Event, Event]("event-router",
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            event Event, prep, result any) (Event, string, error) {
            
            evt := result.(Event)
            
            // Route to appropriate service
            switch evt.Type {
            case "OrderCreated":
                return evt, "inventory", nil
            case "InventoryChecked":
                return evt, "payment", nil
            case "PaymentProcessed":
                return evt, "shipping", nil
            default:
                return evt, "complete", nil
            }
        }),
    )
    
    // Connect event flow
    orderService.Connect("wait-for-events", eventRouter)
    eventRouter.Connect("inventory", inventoryService)
    eventRouter.Connect("payment", paymentService)
    
    return pocket.NewGraph(orderService, pocket.NewStore())
}
```

## Approval Workflow Pattern

Multi-level approval with escalation:

```go
type ApprovalRequest struct {
    ID          string
    Type        string
    Amount      float64
    Requester   string
    Details     map[string]any
    Approvals   []Approval
}

type Approval struct {
    ApproverID string
    Decision   string
    Comments   string
    Timestamp  time.Time
}

func CreateApprovalWorkflow() *pocket.Graph {
    // Determine required approvals
    routeApproval := pocket.NewNode[ApprovalRequest, ApprovalRequest]("route-approval",
        pocket.WithExec(func(ctx context.Context, req ApprovalRequest) (ApprovalRequest, error) {
            // Determine approval levels based on amount
            levels := determineApprovalLevels(req.Type, req.Amount)
            
            // Set first approver
            req.Details["approvalLevels"] = levels
            req.Details["currentLevel"] = 0
            
            return req, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            input, prep, result any) (ApprovalRequest, string, error) {
            
            req := result.(ApprovalRequest)
            levels := req.Details["approvalLevels"].([]string)
            
            if len(levels) == 0 {
                return req, "auto-approve", nil
            }
            
            return req, levels[0], nil
        }),
    )
    
    // Manager approval
    managerApproval := pocket.NewNode[ApprovalRequest, ApprovalRequest]("manager-approval",
        pocket.WithExec(func(ctx context.Context, req ApprovalRequest) (ApprovalRequest, error) {
            // Get manager decision
            decision := getManagerDecision(req)
            
            approval := Approval{
                ApproverID: "manager",
                Decision:   decision,
                Comments:   "Reviewed by direct manager",
                Timestamp:  time.Now(),
            }
            
            req.Approvals = append(req.Approvals, approval)
            
            return req, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            input, prep, result any) (ApprovalRequest, string, error) {
            
            req := result.(ApprovalRequest)
            lastApproval := req.Approvals[len(req.Approvals)-1]
            
            if lastApproval.Decision != "approved" {
                return req, "rejected", nil
            }
            
            // Check if more approvals needed
            currentLevel := req.Details["currentLevel"].(int)
            levels := req.Details["approvalLevels"].([]string)
            
            if currentLevel+1 < len(levels) {
                req.Details["currentLevel"] = currentLevel + 1
                return req, levels[currentLevel+1], nil
            }
            
            return req, "approved", nil
        }),
    )
    
    // Director approval (for higher amounts)
    directorApproval := pocket.NewNode[ApprovalRequest, ApprovalRequest]("director-approval",
        pocket.WithExec(func(ctx context.Context, req ApprovalRequest) (ApprovalRequest, error) {
            // Escalated approval
            decision := getDirectorDecision(req)
            
            approval := Approval{
                ApproverID: "director",
                Decision:   decision,
                Comments:   "Executive approval",
                Timestamp:  time.Now(),
            }
            
            req.Approvals = append(req.Approvals, approval)
            
            return req, nil
        }),
        // Similar post logic...
    )
    
    // Timeout handling
    timeoutHandler := pocket.NewNode[ApprovalRequest, ApprovalRequest]("timeout-handler",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, req ApprovalRequest) (any, error) {
            startTime, _ := store.Get(ctx, "approval:start:"+req.ID)
            return map[string]any{
                "request":   req,
                "startTime": startTime,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (ApprovalRequest, error) {
            data := prepData.(map[string]any)
            req := data["request"].(ApprovalRequest)
            startTime := data["startTime"].(time.Time)
            
            if time.Since(startTime) > 48*time.Hour {
                // Auto-escalate after 48 hours
                req.Details["escalated"] = true
                req.Details["escalationReason"] = "timeout"
            }
            
            return req, nil
        }),
    )
    
    // Connect approval flow
    routeApproval.Connect("manager", managerApproval)
    routeApproval.Connect("director", directorApproval)
    routeApproval.Connect("auto-approve", autoApprove)
    
    managerApproval.Connect("director", directorApproval)
    managerApproval.Connect("approved", finalizeApproval)
    managerApproval.Connect("rejected", notifyRejection)
    
    return pocket.NewGraph(routeApproval, pocket.NewStore())
}
```

## State Machine Pattern

Implement complex state transitions:

```go
type StateMachine struct {
    CurrentState string
    Transitions  map[string]map[string]string // state -> event -> nextState
    Actions      map[string]pocket.Node       // state -> action node
}

// Order state machine
func CreateOrderStateMachine() *pocket.Graph {
    // State transition node
    transition := pocket.NewNode[OrderEvent, OrderState]("state-transition",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, event OrderEvent) (any, error) {
            currentState, _ := store.Get(ctx, "order:"+event.OrderID+":state")
            if currentState == nil {
                currentState = "created"
            }
            
            return map[string]any{
                "event":        event,
                "currentState": currentState.(string),
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (OrderState, error) {
            data := prepData.(map[string]any)
            event := data["event"].(OrderEvent)
            currentState := data["currentState"].(string)
            
            // Define state transitions
            transitions := map[string]map[string]string{
                "created": {
                    "pay":    "paid",
                    "cancel": "cancelled",
                },
                "paid": {
                    "ship":   "shipped",
                    "refund": "refunded",
                },
                "shipped": {
                    "deliver": "delivered",
                    "return":  "returning",
                },
                "delivered": {
                    "return": "returning",
                },
                "returning": {
                    "receive": "returned",
                },
            }
            
            // Check if transition is valid
            stateTransitions, exists := transitions[currentState]
            if !exists {
                return OrderState{}, fmt.Errorf("unknown state: %s", currentState)
            }
            
            nextState, valid := stateTransitions[event.Type]
            if !valid {
                return OrderState{}, fmt.Errorf("invalid transition: %s -> %s", currentState, event.Type)
            }
            
            return OrderState{
                OrderID:      event.OrderID,
                State:        nextState,
                PreviousState: currentState,
                Timestamp:    time.Now(),
            }, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            event OrderEvent, prep, state any) (OrderState, string, error) {
            
            orderState := state.(OrderState)
            
            // Save new state
            store.Set(ctx, "order:"+orderState.OrderID+":state", orderState.State)
            
            // Route to state-specific action
            return orderState, orderState.State, nil
        }),
    )
    
    // State-specific actions
    paidAction := pocket.NewNode[OrderState, any]("paid-action",
        pocket.WithExec(func(ctx context.Context, state OrderState) (any, error) {
            // Notify warehouse
            notifyWarehouse(state.OrderID)
            
            // Update inventory
            updateInventory(state.OrderID)
            
            return nil, nil
        }),
    )
    
    shippedAction := pocket.NewNode[OrderState, any]("shipped-action",
        pocket.WithExec(func(ctx context.Context, state OrderState) (any, error) {
            // Send tracking info
            sendTrackingEmail(state.OrderID)
            
            // Update delivery estimates
            updateDeliveryEstimate(state.OrderID)
            
            return nil, nil
        }),
    )
    
    // Connect states to actions
    transition.Connect("paid", paidAction)
    transition.Connect("shipped", shippedAction)
    transition.Connect("delivered", deliveredAction)
    transition.Connect("cancelled", cancelledAction)
    
    return pocket.NewGraph(transition, pocket.NewStore())
}
```

## Batch Processing Pattern

Process items in configurable batches:

```go
type BatchProcessor struct {
    BatchSize     int
    Timeout       time.Duration
    MaxConcurrent int
}

func CreateBatchWorkflow(config BatchProcessor) *pocket.Graph {
    // Batch collector
    collectBatch := pocket.NewNode[Item, Batch]("collect-batch",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, item Item) (any, error) {
            batch, exists := store.Get(ctx, "current:batch")
            if !exists {
                batch = &Batch{
                    ID:        generateBatchID(),
                    Items:     []Item{},
                    StartTime: time.Now(),
                }
            }
            
            return map[string]any{
                "item":  item,
                "batch": batch,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (Batch, error) {
            data := prepData.(map[string]any)
            item := data["item"].(Item)
            batch := data["batch"].(*Batch)
            
            batch.Items = append(batch.Items, item)
            
            return *batch, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            item Item, prep, batch any) (Batch, string, error) {
            
            b := batch.(Batch)
            
            // Check if batch is ready
            ready := len(b.Items) >= config.BatchSize ||
                     time.Since(b.StartTime) > config.Timeout
            
            if ready {
                // Process batch
                store.Delete(ctx, "current:batch")
                return b, "process", nil
            }
            
            // Continue collecting
            store.Set(ctx, "current:batch", &b)
            return b, "collect", nil
        }),
    )
    
    // Batch processor
    processBatch := pocket.NewNode[Batch, BatchResult]("process-batch",
        pocket.WithExec(func(ctx context.Context, batch Batch) (BatchResult, error) {
            results := make([]ItemResult, len(batch.Items))
            errors := make([]error, len(batch.Items))
            
            // Process items concurrently with limit
            sem := make(chan struct{}, config.MaxConcurrent)
            var wg sync.WaitGroup
            
            for i, item := range batch.Items {
                wg.Add(1)
                sem <- struct{}{}
                
                go func(idx int, itm Item) {
                    defer wg.Done()
                    defer func() { <-sem }()
                    
                    result, err := processItem(itm)
                    results[idx] = result
                    errors[idx] = err
                }(i, item)
            }
            
            wg.Wait()
            
            // Aggregate results
            return BatchResult{
                BatchID:    batch.ID,
                Results:    results,
                Errors:     errors,
                ProcessedAt: time.Now(),
            }, nil
        }),
    )
    
    // Connect flow
    collectBatch.Connect("process", processBatch)
    collectBatch.Connect("collect", collectBatch) // Self-loop for collection
    
    return pocket.NewGraph(collectBatch, pocket.NewStore())
}
```

## Conditional Workflow Pattern

Dynamic branching based on conditions:

```go
type ConditionalFlow struct {
    Conditions []Condition
    Branches   map[string]pocket.Node
}

func CreateConditionalWorkflow() *pocket.Graph {
    // Decision node
    decide := pocket.NewNode[Request, Request]("decide",
        pocket.WithExec(func(ctx context.Context, req Request) (Request, error) {
            // Evaluate conditions
            req.Metadata = evaluateConditions(req)
            return req, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            input, prep, result any) (Request, string, error) {
            
            req := result.(Request)
            
            // Complex routing logic
            if req.Priority == "urgent" && req.Value > 10000 {
                return req, "executive-review", nil
            } else if req.Type == "refund" {
                if req.Value < 100 {
                    return req, "auto-approve", nil
                }
                return req, "refund-review", nil
            } else if req.RiskScore > 0.8 {
                return req, "risk-review", nil
            }
            
            return req, "standard-process", nil
        }),
    )
    
    // Create branches
    executiveReview := createExecutiveReviewBranch()
    refundReview := createRefundReviewBranch()
    riskReview := createRiskReviewBranch()
    standardProcess := createStandardProcessBranch()
    
    // Connect branches
    decide.Connect("executive-review", executiveReview)
    decide.Connect("refund-review", refundReview)
    decide.Connect("risk-review", riskReview)
    decide.Connect("standard-process", standardProcess)
    
    return pocket.NewGraph(decide, pocket.NewStore())
}
```

## Best Practices

### 1. Idempotency

Ensure operations can be safely retried:

```go
pocket.WithExec(func(ctx context.Context, req Request) (Response, error) {
    // Check if already processed
    if result := checkIdempotencyKey(req.ID); result != nil {
        return *result, nil
    }
    
    // Process request
    response := processRequest(req)
    
    // Store result with idempotency key
    storeIdempotencyResult(req.ID, response)
    
    return response, nil
})
```

### 2. Timeout Management

```go
pocket.WithExec(func(ctx context.Context, input Input) (Output, error) {
    // Create timeout context
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Execute with timeout
    result := make(chan Output)
    errCh := make(chan error)
    
    go func() {
        output, err := longRunningOperation(ctx, input)
        if err != nil {
            errCh <- err
            return
        }
        result <- output
    }()
    
    select {
    case output := <-result:
        return output, nil
    case err := <-errCh:
        return Output{}, err
    case <-ctx.Done():
        return Output{}, ctx.Err()
    }
})
```

### 3. Audit Trail

```go
pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
    input, prep, result any) (any, string, error) {
    
    // Create audit entry
    audit := AuditEntry{
        Timestamp: time.Now(),
        Node:      "process-order",
        Input:     input,
        Output:    result,
        User:      getUserFromContext(ctx),
    }
    
    // Store audit trail
    store.Set(ctx, fmt.Sprintf("audit:%s:%d", audit.Node, time.Now().Unix()), audit)
    
    return result, "next", nil
})
```

### 4. Error Aggregation

```go
type WorkflowErrors struct {
    Errors []WorkflowError
    mu     sync.Mutex
}

func (w *WorkflowErrors) Add(node string, err error) {
    w.mu.Lock()
    defer w.mu.Unlock()
    
    w.Errors = append(w.Errors, WorkflowError{
        Node:      node,
        Error:     err,
        Timestamp: time.Now(),
    })
}

func (w *WorkflowErrors) HasCritical() bool {
    for _, err := range w.Errors {
        if err.IsCritical() {
            return true
        }
    }
    return false
}
```

## Summary

Workflow patterns in Pocket enable:

1. **Distributed transactions** with Saga pattern and compensation
2. **Service coordination** through orchestration and choreography
3. **Complex approvals** with multi-level routing and escalation
4. **State management** with explicit state machines
5. **Efficient processing** with batching and conditional flows

These patterns provide blueprints for building robust, scalable business process automation while maintaining clarity and maintainability.