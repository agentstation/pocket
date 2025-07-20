# Type Safety Guide

## Overview

Pocket leverages Go's type system to provide safety at multiple levels:

1. **Compile-time**: Generic type parameters catch errors during compilation
2. **Initialization-time**: ValidateGraph ensures type compatibility across nodes
3. **Runtime**: Type assertions validate data during execution

This guide explores how to use Pocket's type system effectively.

## Compile-Time Type Safety

### Basic Typed Nodes

Use generics to specify input and output types:

```go
// Strongly typed node
userProcessor := pocket.NewNode[User, ProcessedUser]("process-user",
    pocket.WithExec(func(ctx context.Context, user User) (ProcessedUser, error) {
        // 'user' is typed as User - no casting needed
        return ProcessedUser{
            ID:        user.ID,
            Name:      strings.ToUpper(user.Name),
            Processed: time.Now(),
        }, nil
    }),
)

// This won't compile - type mismatch
// userProcessor := pocket.NewNode[User, ProcessedUser]("process-user",
//     pocket.WithExec(func(ctx context.Context, wrong WrongType) (ProcessedUser, error) {
//         // Compiler error: cannot use func literal...
//     }),
// )
```

### Type-Safe Options

All lifecycle options support generics:

```go
validator := pocket.NewNode[User, ValidationResult]("validate",
    // Prep with typed input
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, user User) (UserData, error) {
        // All parameters are typed
        rules, _ := store.Get(ctx, "validation:rules")
        return UserData{
            User:  user,
            Rules: rules.(ValidationRules),
        }, nil
    }),
    
    // Exec with typed prep result
    pocket.WithExec(func(ctx context.Context, data UserData) (ValidationResult, error) {
        // No type assertions needed
        return validateWithRules(data.User, data.Rules), nil
    }),
    
    // Post with all typed parameters
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        user User, prep UserData, result ValidationResult) (ValidationResult, string, error) {
        
        store.Set(ctx, "validation:"+user.ID, result)
        
        if result.Valid {
            return result, "success", nil
        }
        return result, "failure", nil
    }),
)
```

### Type-Safe Store Operations

Use TypedStore for compile-time safe storage:

```go
// Create a typed store
userStore := pocket.NewTypedStore[User](store)

// Compile-time type checking
user := User{ID: "123", Name: "Alice"}
err := userStore.Set(ctx, "user:123", user)              // ✅ Correct type

// This won't compile
// err = userStore.Set(ctx, "user:456", "not a user")    // ❌ Compile error

// Retrieved value is typed
retrieved, exists, err := userStore.Get(ctx, "user:123") // retrieved is User
if exists {
    fmt.Printf("User name: %s\n", retrieved.Name) // Direct field access
}
```

## Initialization-Time Validation

### ValidateGraph Function

Before running a workflow, validate type compatibility:

```go
// Build your workflow
fetchUser := pocket.NewNode[UserID, User]("fetch", ...)
enrichUser := pocket.NewNode[User, EnrichedUser]("enrich", ...)
notifyUser := pocket.NewNode[EnrichedUser, NotificationResult]("notify", ...)

// Connect nodes
fetchUser.Connect("default", enrichUser)    // ✅ User → User
enrichUser.Connect("default", notifyUser)   // ✅ EnrichedUser → EnrichedUser

// Validate before execution
if err := pocket.ValidateGraph(fetchUser); err != nil {
    // Error: type mismatch at connection 'fetch'->'wrong':
    //        node 'fetch' outputs User but node 'wrong' expects Product
    log.Fatal(err)
}

// Safe to run
graph := pocket.NewGraph(fetchUser, store)
```

### How ValidateGraph Works

The validation performs a depth-first traversal:

```go
// Simplified validation logic
func validateConnection(from, to Node) error {
    outputType := from.OutputType()
    inputType := to.InputType()
    
    // Check compatibility
    if !isTypeCompatible(outputType, inputType) {
        return fmt.Errorf("type mismatch: %s outputs %v but %s expects %v",
            from.Name(), outputType, to.Name(), inputType)
    }
    
    return nil
}

// Type compatibility checks:
// 1. Exact match
// 2. Interface satisfaction
// 3. Assignability
// 4. Special handling for 'any'
```

### Mixed Type Scenarios

ValidateGraph handles mixed typed/untyped nodes:

```go
// Typed node
validator := pocket.NewNode[User, ValidationResult]("validator",
    pocket.WithExec(validateUser),
)

// Untyped node (accepts any input)
logger := pocket.NewNode[any, any]("logger",
    pocket.WithExec(func(ctx context.Context, input any) (any, error) {
        log.Printf("Processing: %+v", input)
        return input, nil // Pass through
    }),
)

// Connect them - validation passes
validator.Connect("default", logger) // ✅ any accepts ValidationResult

// ValidateGraph understands this is valid
err := pocket.ValidateGraph(validator) // nil - no error
```

## Runtime Type Safety

### Automatic Type Checking

Even with dynamic types, Pocket validates at runtime:

```go
// Node declared with specific types
processor := pocket.NewNode[User, Result]("processor",
    pocket.WithExec(func(ctx context.Context, user User) (Result, error) {
        return processUser(user), nil
    }),
)

// Runtime validation
graph := pocket.NewGraph(processor, store)

// Correct type - works
result, err := graph.Run(ctx, User{ID: "123"}) // ✅ Success

// Wrong type - runtime error
result, err = graph.Run(ctx, "not a user")     // ❌ Error: invalid input type
```

### Dynamic Type Handling

For truly dynamic scenarios:

```go
// Dynamic processor
dynamic := pocket.NewNode[any, any]("dynamic",
    pocket.WithExec(func(ctx context.Context, input any) (any, error) {
        // Handle multiple types
        switch v := input.(type) {
        case User:
            return processUser(v), nil
        case Product:
            return processProduct(v), nil
        case string:
            return processString(v), nil
        default:
            return nil, fmt.Errorf("unsupported type: %T", input)
        }
    }),
)
```

## Type Safety Patterns

### 1. Pipeline with Type Flow

Build pipelines where types flow naturally:

```go
// Each step has clear input/output types
type (
    RawData      struct { Content []byte }
    ParsedData   struct { Fields map[string]any }
    ValidatedData struct { ParsedData; Valid bool }
    ProcessedData struct { Result any }
)

// Type-safe pipeline
parse := pocket.NewNode[RawData, ParsedData]("parse",
    pocket.WithExec(func(ctx context.Context, raw RawData) (ParsedData, error) {
        return parseContent(raw.Content)
    }),
)

validate := pocket.NewNode[ParsedData, ValidatedData]("validate",
    pocket.WithExec(func(ctx context.Context, parsed ParsedData) (ValidatedData, error) {
        return ValidatedData{
            ParsedData: parsed,
            Valid:      validateFields(parsed.Fields),
        }, nil
    }),
)

process := pocket.NewNode[ValidatedData, ProcessedData]("process",
    pocket.WithExec(func(ctx context.Context, validated ValidatedData) (ProcessedData, error) {
        if !validated.Valid {
            return ProcessedData{}, errors.New("invalid data")
        }
        return ProcessedData{Result: transform(validated)}, nil
    }),
)

// Connect with type safety
parse.Connect("default", validate)     // ParsedData matches
validate.Connect("default", process)   // ValidatedData matches
```

### 2. Generic Node Builders

Create reusable, type-safe node builders:

```go
// Generic transformer builder
func TransformNode[In any, Out any](
    name string,
    transform func(In) (Out, error),
) pocket.Node {
    return pocket.NewNode[In, Out](name,
        pocket.WithExec(func(ctx context.Context, input In) (Out, error) {
            return transform(input)
        }),
    )
}

// Type-safe usage
upperCase := TransformNode("uppercase", strings.ToUpper)
doubleInt := TransformNode("double", func(n int) (int, error) {
    return n * 2, nil
})

// Generic validator builder
func ValidatorNode[T any](
    name string,
    validate func(T) error,
) pocket.Node {
    return pocket.NewNode[T, T](name,
        pocket.WithExec(func(ctx context.Context, input T) (T, error) {
            if err := validate(input); err != nil {
                return input, fmt.Errorf("validation failed: %w", err)
            }
            return input, nil
        }),
    )
}
```

### 3. Type-Safe Error Handling

Maintain type safety in error scenarios:

```go
type Result[T any] struct {
    Value T
    Error error
}

processor := pocket.NewNode[Request, Result[Response]]("safe-processor",
    pocket.WithExec(func(ctx context.Context, req Request) (Result[Response], error) {
        resp, err := processRequest(req)
        return Result[Response]{
            Value: resp,
            Error: err,
        }, nil // Never returns error, encapsulates it
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        req Request, prep any, result Result[Response]) (Result[Response], string, error) {
        
        if result.Error != nil {
            return result, "error", nil
        }
        return result, "success", nil
    }),
)
```

### 4. Composed Type Transformations

Build complex transformations with type safety:

```go
// Type definitions
type (
    UserInput    struct { Name, Email string }
    UserRecord   struct { ID string; UserInput }
    UserProfile  struct { UserRecord; CreatedAt time.Time }
    UserResponse struct { Profile UserProfile; Token string }
)

// Composed transformation
createUser := pocket.NewNode[UserInput, UserRecord]("create",
    pocket.WithExec(func(ctx context.Context, input UserInput) (UserRecord, error) {
        return UserRecord{
            ID:        generateID(),
            UserInput: input,
        }, nil
    }),
)

enrichProfile := pocket.NewNode[UserRecord, UserProfile]("enrich",
    pocket.WithExec(func(ctx context.Context, record UserRecord) (UserProfile, error) {
        return UserProfile{
            UserRecord: record,
            CreatedAt:  time.Now(),
        }, nil
    }),
)

generateResponse := pocket.NewNode[UserProfile, UserResponse]("response",
    pocket.WithExec(func(ctx context.Context, profile UserProfile) (UserResponse, error) {
        return UserResponse{
            Profile: profile,
            Token:   generateToken(profile.ID),
        }, nil
    }),
)

// Type-safe composition
createUser.Connect("default", enrichProfile)
enrichProfile.Connect("default", generateResponse)
```

## Best Practices

### 1. Prefer Specific Types

```go
// ✅ Good - specific types
node := pocket.NewNode[User, ProcessedUser]("processor", ...)

// ❌ Avoid - unless truly dynamic
node := pocket.NewNode[any, any]("processor", ...)
```

### 2. Validate Early

```go
// Always validate after building your graph
if err := pocket.ValidateGraph(startNode); err != nil {
    return fmt.Errorf("invalid workflow: %w", err)
}
```

### 3. Use Type Aliases for Clarity

```go
type (
    UserID   string
    Email    string
    Password string
)

// Clear intent in node signatures
fetchUser := pocket.NewNode[UserID, User]("fetch", ...)
validateEmail := pocket.NewNode[Email, bool]("validate-email", ...)
```

### 4. Document Type Transformations

```go
// enrichUser transforms a basic User into an EnrichedUser by adding:
// - Profile information from the profile service
// - Preferences from the preference store  
// - Recent activity from the activity log
enrichUser := pocket.NewNode[User, EnrichedUser]("enrich", ...)
```

### 5. Handle nil Appropriately

```go
processor := pocket.NewNode[*User, *ProcessedUser]("processor",
    pocket.WithExec(func(ctx context.Context, user *User) (*ProcessedUser, error) {
        if user == nil {
            return nil, errors.New("user cannot be nil")
        }
        return &ProcessedUser{
            ID:   user.ID,
            Name: user.Name,
        }, nil
    }),
)
```

## Common Patterns

### Type Converters

```go
// Generic type converter
func ConverterNode[In, Out any](name string, convert func(In) Out) pocket.Node {
    return pocket.NewNode[In, Out](name,
        pocket.WithExec(func(ctx context.Context, input In) (Out, error) {
            return convert(input), nil
        }),
    )
}

// Usage
jsonToUser := ConverterNode("json-to-user", func(data []byte) User {
    var user User
    json.Unmarshal(data, &user)
    return user
})
```

### Type Guards

```go
// Ensure type at runtime
func TypeGuard[T any](name string) pocket.Node {
    return pocket.NewNode[any, T](name,
        pocket.WithExec(func(ctx context.Context, input any) (T, error) {
            typed, ok := input.(T)
            if !ok {
                var zero T
                return zero, fmt.Errorf("expected %T, got %T", zero, input)
            }
            return typed, nil
        }),
    )
}

// Usage
ensureUser := TypeGuard[User]("ensure-user")
```

### Type-Safe Branching

```go
// Route based on type
router := pocket.NewNode[any, any]("type-router",
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input any, prep any, exec any) (any, string, error) {
        
        switch input.(type) {
        case User:
            return input, "user-handler", nil
        case Product:
            return input, "product-handler", nil
        default:
            return input, "unknown-handler", nil
        }
    }),
)
```

## Summary

Pocket's type system provides:

1. **Compile-time safety** through generics
2. **Initialization validation** with ValidateGraph
3. **Runtime checking** for dynamic scenarios
4. **Flexible patterns** for real-world use cases

By leveraging these features, you can build workflows that are both type-safe and maintainable, catching errors early in development rather than in production.