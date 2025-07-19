# Type Safety Guide for Pocket

This guide explains the three levels of type safety in the Pocket workflow framework and how to use the new unified API.

## Overview

Pocket provides type safety at three distinct levels:

1. **Compile-time**: Generic type parameters ensure function signatures match
2. **Initialization-time**: ValidateFlow checks type compatibility across connected nodes
3. **Runtime**: Type assertions ensure data integrity during execution

## The New Unified API

As of the latest version, Pocket uses a single generic `NewNode` function:

```go
// Typed node - provides full type safety
node := pocket.NewNode[User, Response]("processor", opts...)

// Untyped node - explicitly shows dynamic typing
node := pocket.NewNode[any, any]("processor", opts...)
```

The explicit `[any, any]` for untyped nodes encourages developers to add types for better safety.

## Type Safety Levels Explained

### 1. Compile-Time Type Safety

When you declare a node with specific types, the Go compiler enforces type correctness:

```go
// Declare a typed node
validator := pocket.NewNode[User, ValidationResult]("validator",
    pocket.WithExec(func(ctx context.Context, store pocket.Store, user User) (ValidationResult, error) {
        // Compile-time: 'user' is guaranteed to be User type
        // No casting needed!
        return ValidationResult{Valid: user.Email != ""}, nil
    }),
)

// This would NOT compile (type mismatch):
/*
badNode := pocket.NewNode[User, ValidationResult]("bad",
    pocket.WithExec(func(ctx context.Context, store pocket.Store, wrong WrongType) (ValidationResult, error) {
        // COMPILE ERROR: Cannot use WrongType where User is expected
        return ValidationResult{}, nil
    }),
)
*/
```

### 2. Initialization-Time Type Safety (ValidateFlow)

Before running your workflow, use `ValidateFlow` to check type compatibility between connected nodes:

```go
// Build your workflow
userProcessor := pocket.NewNode[User, ProcessedUser]("process", ...)
emailSender := pocket.NewNode[ProcessedUser, EmailResult]("email", ...)
userProcessor.Connect("success", emailSender)

// Validate the entire graph before execution
if err := pocket.ValidateFlow(userProcessor); err != nil {
    // Error: "type mismatch: node 'process' outputs ProcessedUser 
    //         but node 'wrongNode' expects DifferentType"
    log.Fatal(err)
}

// Now safe to execute
flow := pocket.NewFlow(userProcessor, store)
result, err := flow.Run(ctx, User{ID: "123"})
```

### 3. Runtime Type Safety

When using WithExec with 'any' type parameters on a typed node, the framework provides runtime safety:

```go
// Using WithExec[any, any] with a typed node
node := pocket.NewNode[User, Response]("processor",
    pocket.WithExec[any, any](func(ctx context.Context, store pocket.Store, input any) (any, error) {
        // The framework ensures 'input' is User type at runtime
        user := input.(User)  // This cast is safe due to node's type declaration
        return Response{Message: "Processed " + user.Name}, nil
    }),
)

// Runtime check prevents type errors
flow := pocket.NewFlow(node, store)
_, err := flow.Run(ctx, WrongType{}) // Error: invalid input type
```

## Migration Guide

### From TypedNode to NewNode

```go
// Old API (deprecated)
node := pocket.TypedNode[In, Out]("name", opts...)

// New API (recommended)
node := pocket.NewNode[In, Out]("name", opts...)
```

### Simplified Generic Options

```go
// The new unified API - all options are generic by default
node := pocket.NewNode[User, Response]("processor",
    pocket.WithExec(func(ctx, store, user User) (Response, error) {
        // Go infers the types from the function signature
        return Response{}, nil
    }),
    pocket.WithPrep(func(ctx, store, user User) (any, error) {
        // Prep can modify the input before execution
        return user, nil
    }),
    pocket.WithPost(func(ctx, store, user User, prep any, resp Response) (Response, string, error) {
        // Post can route based on the response
        return resp, "default", nil
    }),
)
```

## Best Practices

### 1. Always Use Types When Possible

```go
// ✅ Good - provides type safety
userNode := pocket.NewNode[User, ProcessedUser]("process", ...)

// ❌ Avoid - loses compile-time safety
userNode := pocket.NewNode[any, any]("process", ...)
```

### 2. Validate Flows Before Execution

```go
// Always validate your workflow graph
if err := pocket.ValidateFlow(startNode); err != nil {
    return fmt.Errorf("workflow validation failed: %w", err)
}
```

### 3. Use Generic Options for Type Safety

```go
node := pocket.NewNode[Input, Output]("processor",
    // Generic options eliminate casting when types match
    pocket.WithPrep(func(ctx context.Context, store pocket.Store, in Input) (any, error) {
        // Direct access to typed input
        return preprocessData(in), nil
    }),
    pocket.WithExec(func(ctx context.Context, store pocket.Store, in Input) (Output, error) {
        // Type-safe processing
        return processData(in), nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.Store, in Input, prep any, out Output) (Output, string, error) {
        // Routing based on typed output
        if out.Success {
            return out, "success", nil
        }
        return out, "retry", nil
    }),
)
```

### 4. Combine Lifecycle and Configuration Options

```go
node := pocket.NewNode[Request, Response]("api-call",
    // Generic option for exec
    pocket.WithExec(func(ctx context.Context, store pocket.Store, req Request) (Response, error) {
        return callAPI(req)
    }),
    // Configuration options work seamlessly
    pocket.WithRetry(3, time.Second),
    pocket.WithTimeout(30 * time.Second),
    pocket.WithFallback(func(ctx context.Context, store pocket.Store, req Request, err error) (Response, error) {
        // Generic fallback on error
        return getCachedResponse(req), nil
    }),
)
```

## Common Patterns

### Pattern 1: Type-Safe Pipeline

```go
// Each step has clear input/output types
fetchUser := pocket.NewNode[UserID, User]("fetch", ...)
validateUser := pocket.NewNode[User, ValidatedUser]("validate", ...)  
enrichUser := pocket.NewNode[ValidatedUser, EnrichedUser]("enrich", ...)
notifyUser := pocket.NewNode[EnrichedUser, NotificationResult]("notify", ...)

// Connect the pipeline
fetchUser.Connect("default", validateUser)
validateUser.Connect("valid", enrichUser)
enrichUser.Connect("default", notifyUser)

// Validate ensures type compatibility
if err := pocket.ValidateFlow(fetchUser); err != nil {
    log.Fatal("Pipeline type mismatch:", err)
}
```

### Pattern 2: Mixed Typed and Untyped Nodes

```go
// Typed nodes for critical paths
validator := pocket.NewNode[User, ValidationResult]("validate", ...)

// Untyped for flexible handling - note explicit [any, any]
logger := pocket.NewNode[any, any]("logger",
    pocket.WithExec[any, any](func(ctx context.Context, store pocket.Store, input any) (any, error) {
        log.Printf("Processing: %+v", input)
        return input, nil  // Pass through
    }),
)

// Connect them - ValidateFlow will skip type checking for untyped nodes
validator.Connect("default", logger)
```

### Pattern 3: Error Handling with Types

```go
processor := pocket.NewNode[Request, Response]("processor",
    pocket.WithExec(func(ctx context.Context, store pocket.Store, req Request) (Response, error) {
        if !req.Valid {
            return Response{}, errors.New("invalid request")
        }
        return process(req), nil
    }),
    pocket.WithFallback(func(ctx context.Context, store pocket.Store, req Request, err error) (Response, error) {
        // Type-safe error recovery
        return Response{
            Status: "fallback",
            Error:  err.Error(),
        }, nil
    }),
)
```

## Running the Examples

The `examples/typed` directory contains several demonstrations:

1. **main.go** - Complete workflow with validation, enrichment, and notification
2. **type_safety_demo.go** - Demonstrates all three levels of type safety

To run the examples:

```bash
# Run the main typed example
go run examples/typed/main.go

# Run the type safety demonstration
go run examples/typed/type_safety_demo.go
```

## Summary

The Pocket type system provides:
- **Safety**: Catch type errors early, not in production
- **Clarity**: Types document expected data flow
- **Flexibility**: Mix typed and untyped nodes as needed
- **Performance**: Type checks happen once, not on every execution

By leveraging Go's generics with the unified `NewNode[In, Out]` API and generic option functions, Pocket ensures your workflows are both type-safe and maintainable. The explicit `[any, any]` requirement for untyped nodes encourages developers to add types for better compile-time safety.