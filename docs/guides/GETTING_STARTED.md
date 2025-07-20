# Getting Started with Pocket

## Introduction

This guide will walk you through building your first Pocket workflow, from simple transformations to complex decision graphs. By the end, you'll understand how to create, connect, and compose nodes into powerful workflows.

## Installation

```bash
go get github.com/agentstation/pocket
```

## Your First Node

Let's start with the simplest possible node - one that transforms text to uppercase:

```go
package main

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/agentstation/pocket"
)

func main() {
    // Create a simple node
    uppercase := pocket.NewNode[string, string]("uppercase",
        pocket.WithExec(func(ctx context.Context, input string) (string, error) {
            return strings.ToUpper(input), nil
        }),
    )
    
    // Create a graph with a store
    store := pocket.NewStore()
    graph := pocket.NewGraph(uppercase, store)
    
    // Run it
    result, err := graph.Run(context.Background(), "hello world")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(result) // Output: HELLO WORLD
}
```

## Understanding the Prep/Exec/Post Pattern

Most real nodes use all three phases. Let's build a more realistic example:

```go
// A node that processes user registration
type User struct {
    Email    string
    Name     string
    Password string
}

type RegistrationResult struct {
    UserID  string
    Success bool
    Message string
}

func main() {
    registerUser := pocket.NewNode[User, RegistrationResult]("register",
        // Prep: Validate and prepare data
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, user User) (any, error) {
            // Validate email
            if !strings.Contains(user.Email, "@") {
                return nil, fmt.Errorf("invalid email format")
            }
            
            // Check if user exists
            if existing, exists := store.Get(ctx, "user:"+user.Email); exists {
                return nil, fmt.Errorf("user already exists")
            }
            
            // Prepare data for execution
            return map[string]any{
                "user":       user,
                "timestamp": time.Now(),
            }, nil
        }),
        
        // Exec: Pure business logic
        pocket.WithExec(func(ctx context.Context, prepData any) (RegistrationResult, error) {
            data := prepData.(map[string]any)
            user := data["user"].(User)
            
            // Generate user ID (in practice, use UUID)
            userID := fmt.Sprintf("user_%d", data["timestamp"].(time.Time).Unix())
            
            // Hash password (simplified for example)
            hashedPassword := fmt.Sprintf("hashed_%s", user.Password)
            
            return RegistrationResult{
                UserID:  userID,
                Success: true,
                Message: "Registration successful",
            }, nil
        }),
        
        // Post: Save state and decide routing
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, 
            user User, prep any, result RegistrationResult) (RegistrationResult, string, error) {
            
            if result.Success {
                // Save user data
                store.Set(ctx, "user:"+user.Email, map[string]any{
                    "id":    result.UserID,
                    "email": user.Email,
                    "name":  user.Name,
                })
                
                return result, "send-welcome", nil
            }
            
            return result, "handle-error", nil
        }),
    )
    
    // Run the registration
    store := pocket.NewStore()
    graph := pocket.NewGraph(registerUser, store)
    
    newUser := User{
        Email:    "alice@example.com",
        Name:     "Alice",
        Password: "secret123",
    }
    
    result, err := graph.Run(context.Background(), newUser)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Registration result: %+v\n", result)
}
```

## Building a Multi-Node Workflow

Now let's connect multiple nodes to create a workflow:

```go
func main() {
    // Node 1: Validate user input
    validate := pocket.NewNode[User, User]("validate",
        pocket.WithExec(func(ctx context.Context, user User) (User, error) {
            if user.Email == "" || user.Name == "" {
                return User{}, fmt.Errorf("missing required fields")
            }
            return user, nil
        }),
    )
    
    // Node 2: Create user account
    createAccount := pocket.NewNode[User, Account]("create-account",
        pocket.WithExec(func(ctx context.Context, user User) (Account, error) {
            return Account{
                ID:     generateID(),
                Email:  user.Email,
                Name:   user.Name,
                Status: "active",
            }, nil
        }),
    )
    
    // Node 3: Send welcome email
    sendWelcome := pocket.NewNode[Account, EmailResult]("send-welcome",
        pocket.WithExec(func(ctx context.Context, account Account) (EmailResult, error) {
            // Simulate sending email
            fmt.Printf("Sending welcome email to %s\n", account.Email)
            return EmailResult{
                Sent:      true,
                MessageID: "msg_" + generateID(),
            }, nil
        }),
    )
    
    // Connect the nodes
    validate.Connect("default", createAccount)
    createAccount.Connect("default", sendWelcome)
    
    // Create and run the workflow
    store := pocket.NewStore()
    graph := pocket.NewGraph(validate, store)
    
    user := User{
        Email: "bob@example.com",
        Name:  "Bob",
    }
    
    result, err := graph.Run(context.Background(), user)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Workflow completed: %+v\n", result)
}
```

## Decision-Based Routing

Nodes can make decisions and route to different paths:

```go
func main() {
    // Risk assessment node
    assessRisk := pocket.NewNode[Transaction, RiskScore]("assess-risk",
        pocket.WithExec(func(ctx context.Context, tx Transaction) (RiskScore, error) {
            score := calculateRiskScore(tx)
            return RiskScore{
                Value:       score,
                Transaction: tx,
            }, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            tx Transaction, prep any, score RiskScore) (RiskScore, string, error) {
            
            // Route based on risk level
            switch {
            case score.Value > 80:
                return score, "high-risk", nil
            case score.Value > 50:
                return score, "medium-risk", nil
            default:
                return score, "low-risk", nil
            }
        }),
    )
    
    // Different handlers for each risk level
    highRiskHandler := pocket.NewNode[RiskScore, Result]("high-risk-handler",
        pocket.WithExec(func(ctx context.Context, score RiskScore) (Result, error) {
            return Result{
                Action:  "block",
                Message: "Transaction blocked due to high risk",
            }, nil
        }),
    )
    
    mediumRiskHandler := pocket.NewNode[RiskScore, Result]("medium-risk-handler",
        pocket.WithExec(func(ctx context.Context, score RiskScore) (Result, error) {
            return Result{
                Action:  "review",
                Message: "Transaction requires manual review",
            }, nil
        }),
    )
    
    lowRiskHandler := pocket.NewNode[RiskScore, Result]("low-risk-handler",
        pocket.WithExec(func(ctx context.Context, score RiskScore) (Result, error) {
            return Result{
                Action:  "approve",
                Message: "Transaction approved",
            }, nil
        }),
    )
    
    // Connect based on risk levels
    assessRisk.Connect("high-risk", highRiskHandler)
    assessRisk.Connect("medium-risk", mediumRiskHandler)
    assessRisk.Connect("low-risk", lowRiskHandler)
    
    // Run with different transactions
    store := pocket.NewStore()
    graph := pocket.NewGraph(assessRisk, store)
    
    transactions := []Transaction{
        {Amount: 10000, Country: "high-risk-country"},
        {Amount: 500, Country: "trusted-country"},
        {Amount: 2000, Country: "medium-risk-country"},
    }
    
    for _, tx := range transactions {
        result, err := graph.Run(context.Background(), tx)
        if err != nil {
            fmt.Printf("Error processing transaction: %v\n", err)
            continue
        }
        fmt.Printf("Transaction %+v -> %+v\n", tx, result)
    }
}
```

## Using State Management

The store allows nodes to share state within a workflow:

```go
func main() {
    // Node 1: Fetch user data
    fetchUser := pocket.NewNode[string, User]("fetch-user",
        pocket.WithExec(func(ctx context.Context, userID string) (User, error) {
            // Simulate fetching from database
            return User{
                ID:    userID,
                Name:  "Alice",
                Email: "alice@example.com",
            }, nil
        }),
        pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
            userID string, prep any, user User) (User, string, error) {
            
            // Store user data for other nodes
            store.Set(ctx, "current-user", user)
            return user, "enrich", nil
        }),
    )
    
    // Node 2: Enrich with additional data
    enrichUser := pocket.NewNode[User, EnrichedUser]("enrich-user",
        pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, user User) (any, error) {
            // Read shared state
            currentUser, _ := store.Get(ctx, "current-user")
            
            return map[string]any{
                "user":          user,
                "originalUser": currentUser,
            }, nil
        }),
        pocket.WithExec(func(ctx context.Context, prepData any) (EnrichedUser, error) {
            data := prepData.(map[string]any)
            user := data["user"].(User)
            
            return EnrichedUser{
                User:         user,
                Preferences:  fetchPreferences(user.ID),
                LastActivity: fetchLastActivity(user.ID),
            }, nil
        }),
    )
    
    // Connect nodes
    fetchUser.Connect("enrich", enrichUser)
    
    // Create store with configuration
    store := pocket.NewStore(
        pocket.WithMaxEntries(1000),
        pocket.WithTTL(5 * time.Minute),
    )
    
    graph := pocket.NewGraph(fetchUser, store)
    result, err := graph.Run(context.Background(), "user123")
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Enriched user: %+v\n", result)
}
```

## Error Handling and Fallbacks

Build resilient workflows with error handling:

```go
func main() {
    // Main processing node with potential failures
    processPayment := pocket.NewNode[Payment, PaymentResult]("process-payment",
        pocket.WithExec(func(ctx context.Context, payment Payment) (PaymentResult, error) {
            // Simulate occasional failures
            if payment.Amount > 1000 {
                return PaymentResult{}, fmt.Errorf("payment processor unavailable")
            }
            
            return PaymentResult{
                TransactionID: generateID(),
                Status:       "completed",
            }, nil
        }),
        pocket.WithFallback(func(ctx context.Context, payment Payment, err error) (PaymentResult, error) {
            // Fallback to alternative processor
            fmt.Printf("Primary processor failed: %v, using fallback\n", err)
            
            return PaymentResult{
                TransactionID: "fallback_" + generateID(),
                Status:       "completed_via_fallback",
            }, nil
        }),
    )
    
    // Add retry capability
    processWithRetry := pocket.NewNode[Payment, PaymentResult]("process-with-retry",
        pocket.WithExec(func(ctx context.Context, payment Payment) (PaymentResult, error) {
            return processPayment.Exec(ctx, payment)
        }),
        pocket.WithRetry(3, time.Second),
    )
    
    // Test with various payments
    payments := []Payment{
        {Amount: 100, Currency: "USD"},
        {Amount: 2000, Currency: "USD"}, // Will trigger fallback
    }
    
    store := pocket.NewStore()
    graph := pocket.NewGraph(processWithRetry, store)
    
    for _, payment := range payments {
        result, err := graph.Run(context.Background(), payment)
        if err != nil {
            fmt.Printf("Payment failed: %v\n", err)
            continue
        }
        fmt.Printf("Payment processed: %+v\n", result)
    }
}
```

## Using the Builder API

For complex workflows, use the Builder API:

```go
func main() {
    store := pocket.NewStore()
    
    // Create nodes
    input := pocket.NewNode[Request, ValidatedRequest]("input", ...)
    process := pocket.NewNode[ValidatedRequest, ProcessedData]("process", ...)
    transform := pocket.NewNode[ProcessedData, TransformedData]("transform", ...)
    output := pocket.NewNode[TransformedData, Response]("output", ...)
    errorHandler := pocket.NewNode[any, ErrorResponse]("error", ...)
    
    // Build the graph
    graph, err := pocket.NewBuilder(store).
        Add(input).
        Add(process).
        Add(transform).
        Add(output).
        Add(errorHandler).
        Connect("input", "valid", "process").
        Connect("input", "invalid", "error").
        Connect("process", "success", "transform").
        Connect("process", "failure", "error").
        Connect("transform", "default", "output").
        Start("input").
        Build()
    
    if err != nil {
        panic(err)
    }
    
    // Run the workflow
    result, err := graph.Run(context.Background(), Request{Data: "test"})
    fmt.Printf("Result: %+v, Error: %v\n", result, err)
}
```

## Next Steps

Now that you understand the basics:

1. **Explore Type Safety**: Learn about Pocket's [type system](TYPE_SAFETY.md)
2. **Master State Management**: Understand [stores and scoping](STATE_MANAGEMENT.md)
3. **Build Resilient Workflows**: Study [error handling patterns](ERROR_HANDLING.md)
4. **Learn Patterns**: Check out [concurrency patterns](../patterns/CONCURRENCY.md)
5. **See Examples**: Browse the [example projects](../examples/README.md)

## Quick Reference

### Creating Nodes

```go
// Simple exec-only node
node := pocket.NewNode[In, Out]("name",
    pocket.WithExec(execFunc),
)

// Full lifecycle node
node := pocket.NewNode[In, Out]("name",
    pocket.WithPrep(prepFunc),
    pocket.WithExec(execFunc),
    pocket.WithPost(postFunc),
)

// With error handling
node := pocket.NewNode[In, Out]("name",
    pocket.WithExec(execFunc),
    pocket.WithFallback(fallbackFunc),
    pocket.WithRetry(3, time.Second),
)
```

### Connecting Nodes

```go
// Simple connection
nodeA.Connect("default", nodeB)

// Multiple routes
router.Connect("success", successHandler)
router.Connect("failure", failureHandler)
router.Connect("retry", retryHandler)
```

### Running Workflows

```go
// Create and run
store := pocket.NewStore()
graph := pocket.NewGraph(startNode, store)
result, err := graph.Run(ctx, input)

// With options
store := pocket.NewStore(
    pocket.WithMaxEntries(1000),
    pocket.WithTTL(5 * time.Minute),
)
```

Happy workflow building!