# YAML Integration

## Overview

Pocket provides YAML support for defining workflows declaratively, formatting structured output for LLMs, and loading configurations. This enables non-programmers to create workflows and makes it easier to generate workflows programmatically.

## YAML Workflow Definition

### Basic Workflow Structure

Define workflows in YAML:

```yaml
# workflow.yaml
name: order-processing
version: 1.0
description: Process customer orders

nodes:
  - id: validate
    type: processor
    config:
      handler: validateOrder
    connections:
      success: enrich
      failure: reject

  - id: enrich
    type: processor
    config:
      handler: enrichOrder
    connections:
      default: process-payment

  - id: process-payment
    type: processor
    config:
      handler: processPayment
      timeout: 30s
      retries: 3
    connections:
      success: fulfill
      failure: compensate

  - id: fulfill
    type: processor
    config:
      handler: fulfillOrder
    connections:
      default: notify

  - id: notify
    type: processor
    config:
      handler: sendNotification

start: validate
```

### Loading YAML Workflows

```go
import "github.com/agentstation/pocket/yaml"

// Load workflow from file
loader := yaml.NewLoader()
loader.RegisterHandler("validateOrder", validateOrderFunc)
loader.RegisterHandler("enrichOrder", enrichOrderFunc)
loader.RegisterHandler("processPayment", processPaymentFunc)
loader.RegisterHandler("fulfillOrder", fulfillOrderFunc)
loader.RegisterHandler("sendNotification", sendNotificationFunc)

graph, err := loader.LoadFile("workflow.yaml", store)
if err != nil {
    log.Fatal(err)
}

// Run the workflow
result, err := graph.Run(ctx, order)
```

### Advanced YAML Features

#### Node Types and Configurations

```yaml
nodes:
  # Simple processor
  - id: simple
    type: processor
    config:
      handler: simpleHandler

  # With retry and timeout
  - id: resilient
    type: processor
    config:
      handler: apiCall
      timeout: 10s
      retries: 3
      retry_delay: 1s

  # With fallback
  - id: with-fallback
    type: processor
    config:
      handler: primaryHandler
      fallback: backupHandler

  # Parallel execution
  - id: parallel
    type: parallel
    nodes:
      - fetch-user
      - fetch-inventory
      - check-payment

  # Conditional routing
  - id: router
    type: router
    config:
      handler: routeDecision
    routes:
      premium: premium-flow
      standard: standard-flow
      default: basic-flow

  # Sub-workflow
  - id: sub-workflow
    type: graph
    graph: authentication-workflow.yaml
```

#### Environment Variables and Templating

```yaml
# Use environment variables
nodes:
  - id: api-call
    type: processor
    config:
      handler: callAPI
      endpoint: ${API_ENDPOINT}
      timeout: ${TIMEOUT:30s}  # Default value

# Template support
  - id: dynamic
    type: processor
    config:
      handler: process
      params:
        region: "{{ .Region }}"
        tier: "{{ .UserTier }}"
```

## YAML Output Formatting

### Structured Output for LLMs

Create YAML-formatted output nodes:

```go
// Create a YAML output node
yamlFormatter := pocket.NewNode[ProcessResult, string]("yaml-formatter",
    pocket.WithExec(func(ctx context.Context, result ProcessResult) (string, error) {
        // Structure for YAML output
        output := map[string]any{
            "summary": map[string]any{
                "status":    result.Status,
                "timestamp": result.Timestamp.Format(time.RFC3339),
                "duration":  result.Duration.String(),
            },
            "data": map[string]any{
                "processed_items": result.ProcessedCount,
                "failed_items":    result.FailedCount,
                "success_rate":    result.SuccessRate(),
            },
            "next_steps": result.NextSteps,
            "metadata": result.Metadata,
        }
        
        // Convert to YAML
        yamlBytes, err := yaml.Marshal(output)
        if err != nil {
            return "", fmt.Errorf("yaml formatting failed: %w", err)
        }
        
        return string(yamlBytes), nil
    }),
)
```

### YAML Helper Functions

```go
import "github.com/agentstation/pocket/yaml"

// Create YAML node with schema validation
yamlNode := yaml.YAMLNode("formatter",
    pocket.WithExec(func(ctx context.Context, input any) (any, error) {
        // Ensure output conforms to expected schema
        return yaml.Structure{
            "analysis": yaml.Object{
                "sentiment": "positive",
                "confidence": 0.95,
                "keywords": []string{"excellent", "recommended"},
            },
            "recommendations": yaml.Array{
                yaml.Object{
                    "action": "follow-up",
                    "priority": "high",
                    "reason": "high satisfaction score",
                },
            },
        }, nil
    }),
    yaml.WithSchema(analysisSchema), // Optional schema validation
)
```

## Dynamic Workflow Generation

### Template-Based Workflows

Generate workflows from templates:

```go
type WorkflowTemplate struct {
    BaseYAML string
    Params   map[string]any
}

func GenerateWorkflow(template WorkflowTemplate) (*pocket.Graph, error) {
    // Parse template
    tmpl, err := template.New("workflow").Parse(template.BaseYAML)
    if err != nil {
        return nil, err
    }
    
    // Execute template with parameters
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, template.Params); err != nil {
        return nil, err
    }
    
    // Load generated YAML
    loader := yaml.NewLoader()
    return loader.Load(buf.Bytes(), store)
}

// Usage
template := WorkflowTemplate{
    BaseYAML: `
name: {{ .Name }}-workflow
nodes:
  {{- range .Steps }}
  - id: {{ .ID }}
    type: processor
    config:
      handler: {{ .Handler }}
    {{- if .Connections }}
    connections:
      {{- range $key, $value := .Connections }}
      {{ $key }}: {{ $value }}
      {{- end }}
    {{- end }}
  {{- end }}
start: {{ .Start }}
`,
    Params: map[string]any{
        "Name":  "dynamic",
        "Start": "step1",
        "Steps": []map[string]any{
            {
                "ID":      "step1",
                "Handler": "process",
                "Connections": map[string]string{
                    "default": "step2",
                },
            },
            {
                "ID":      "step2",
                "Handler": "complete",
            },
        },
    },
}

graph, err := GenerateWorkflow(template)
```

### Programmatic YAML Building

```go
// Build YAML workflow programmatically
builder := yaml.NewWorkflowBuilder("data-pipeline")

builder.
    AddNode("extract", yaml.NodeConfig{
        Type:    "processor",
        Handler: "extractData",
        Config: map[string]any{
            "source": "database",
            "query":  "SELECT * FROM orders",
        },
    }).
    AddNode("transform", yaml.NodeConfig{
        Type:    "processor",
        Handler: "transformData",
    }).
    AddNode("load", yaml.NodeConfig{
        Type:    "processor",
        Handler: "loadData",
        Config: map[string]any{
            "destination": "warehouse",
        },
    }).
    Connect("extract", "success", "transform").
    Connect("transform", "success", "load").
    SetStart("extract")

// Generate YAML
yamlBytes, err := builder.ToYAML()

// Or create graph directly
graph, err := builder.Build(store)
```

## Schema Validation

### Define YAML Schemas

Ensure YAML configurations are valid:

```go
// Define schema for workflow validation
workflowSchema := yaml.Schema{
    Type: "object",
    Properties: map[string]yaml.Schema{
        "name": {Type: "string", Required: true},
        "version": {Type: "string", Pattern: `^\d+\.\d+$`},
        "nodes": {
            Type: "array",
            Items: yaml.Schema{
                Type: "object",
                Properties: map[string]yaml.Schema{
                    "id":   {Type: "string", Required: true},
                    "type": {Type: "string", Enum: []string{"processor", "router", "parallel"}},
                    "config": {Type: "object"},
                    "connections": {
                        Type: "object",
                        AdditionalProperties: yaml.Schema{Type: "string"},
                    },
                },
            },
        },
        "start": {Type: "string", Required: true},
    },
}

// Validate YAML
validator := yaml.NewValidator(workflowSchema)
if err := validator.ValidateFile("workflow.yaml"); err != nil {
    log.Fatalf("Invalid workflow: %v", err)
}
```

### Runtime Schema Validation

```go
// Validate data at runtime
dataSchema := yaml.Schema{
    Type: "object",
    Properties: map[string]yaml.Schema{
        "user_id":   {Type: "string", Format: "uuid"},
        "amount":    {Type: "number", Minimum: 0},
        "currency":  {Type: "string", Enum: []string{"USD", "EUR", "GBP"}},
        "items": {
            Type: "array",
            MinItems: 1,
            Items: yaml.Schema{
                Type: "object",
                Properties: map[string]yaml.Schema{
                    "sku":      {Type: "string"},
                    "quantity": {Type: "integer", Minimum: 1},
                },
            },
        },
    },
}

validateNode := pocket.NewNode[map[string]any, map[string]any]("validate",
    pocket.WithExec(func(ctx context.Context, data map[string]any) (map[string]any, error) {
        validator := yaml.NewValidator(dataSchema)
        if err := validator.Validate(data); err != nil {
            return nil, fmt.Errorf("validation failed: %w", err)
        }
        return data, nil
    }),
)
```

## Configuration Management

### Multi-Environment Configs

```yaml
# base.yaml
defaults:
  timeout: 30s
  retries: 3

# development.yaml
extends: base.yaml
api:
  endpoint: http://localhost:8080
  debug: true

# production.yaml
extends: base.yaml
api:
  endpoint: https://api.example.com
  debug: false
  rate_limit: 1000
```

Load environment-specific configuration:

```go
// Configuration loader
type ConfigLoader struct {
    environment string
}

func (c *ConfigLoader) Load() (*Config, error) {
    // Load base configuration
    base, err := c.loadYAML("config/base.yaml")
    if err != nil {
        return nil, err
    }
    
    // Load environment-specific
    envFile := fmt.Sprintf("config/%s.yaml", c.environment)
    envConfig, err := c.loadYAML(envFile)
    if err != nil {
        return nil, err
    }
    
    // Merge configurations
    merged := c.merge(base, envConfig)
    
    // Parse into struct
    var config Config
    if err := yaml.Unmarshal(merged, &config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## Advanced Patterns

### Conditional YAML Loading

```go
// Load different workflows based on conditions
func LoadConditionalWorkflow(condition string) (*pocket.Graph, error) {
    loader := yaml.NewLoader()
    
    var workflowFile string
    switch condition {
    case "premium":
        workflowFile = "workflows/premium-flow.yaml"
    case "batch":
        workflowFile = "workflows/batch-flow.yaml"
    default:
        workflowFile = "workflows/standard-flow.yaml"
    }
    
    // Load with includes
    loader.SetIncludePath("workflows/includes")
    return loader.LoadFile(workflowFile, store)
}
```

### YAML Preprocessing

```go
// Preprocess YAML before loading
func PreprocessYAML(input []byte) ([]byte, error) {
    // Replace variables
    processed := os.Expand(string(input), func(key string) string {
        if value := os.Getenv(key); value != "" {
            return value
        }
        // Check custom sources
        if value := getFromVault(key); value != "" {
            return value
        }
        return "${" + key + "}" // Keep original if not found
    })
    
    // Process includes
    includeRegex := regexp.MustCompile(`!include\s+(.+)`)
    processed = includeRegex.ReplaceAllStringFunc(processed, func(match string) string {
        parts := includeRegex.FindStringSubmatch(match)
        if len(parts) > 1 {
            included, _ := os.ReadFile(parts[1])
            return string(included)
        }
        return match
    })
    
    return []byte(processed), nil
}
```

### YAML Workflow Versioning

```go
// Version-aware workflow loading
type VersionedLoader struct {
    versions map[string]WorkflowVersion
}

type WorkflowVersion struct {
    Version    string
    Deprecated bool
    Migrator   func(old map[string]any) map[string]any
}

func (v *VersionedLoader) Load(yamlData []byte) (*pocket.Graph, error) {
    // Parse to get version
    var meta struct {
        Version string `yaml:"version"`
    }
    yaml.Unmarshal(yamlData, &meta)
    
    // Check version
    version, exists := v.versions[meta.Version]
    if !exists {
        return nil, fmt.Errorf("unsupported version: %s", meta.Version)
    }
    
    if version.Deprecated {
        log.Printf("Warning: version %s is deprecated", meta.Version)
    }
    
    // Migrate if needed
    var data map[string]any
    yaml.Unmarshal(yamlData, &data)
    
    if version.Migrator != nil {
        data = version.Migrator(data)
    }
    
    // Load workflow
    return loadFromMap(data)
}
```

## Best Practices

### 1. Validate Early

```go
// Validate on startup
func init() {
    if err := validateAllWorkflows("workflows/"); err != nil {
        log.Fatal("Workflow validation failed:", err)
    }
}
```

### 2. Use Meaningful IDs

```yaml
# Good: descriptive IDs
nodes:
  - id: validate-user-input
  - id: check-inventory-availability
  - id: process-payment-transaction

# Avoid: generic IDs
nodes:
  - id: step1
  - id: step2
  - id: step3
```

### 3. Document in YAML

```yaml
# Order processing workflow
# This workflow handles customer orders from validation to fulfillment
name: order-processing
version: 1.2
author: team@example.com

nodes:
  # Validates order data and customer information
  - id: validate
    type: processor
    description: |
      Checks that the order contains valid:
      - Customer information
      - Product SKUs
      - Quantities
      - Pricing
```

### 4. Modular Workflows

```yaml
# auth-check.yaml - Reusable authentication
name: auth-check
nodes:
  - id: verify-token
    type: processor
    config:
      handler: verifyAuthToken

# main-workflow.yaml
name: main
nodes:
  - id: auth
    type: include
    workflow: auth-check.yaml
    
  - id: process
    type: processor
    config:
      handler: mainProcess
```

### 5. Environment Separation

```go
// Separate configs by environment
type Environment string

const (
    Development Environment = "development"
    Staging     Environment = "staging"
    Production  Environment = "production"
)

func LoadWorkflowForEnv(name string, env Environment) (*pocket.Graph, error) {
    configDir := fmt.Sprintf("config/%s", env)
    workflowFile := filepath.Join(configDir, name+".yaml")
    
    return loader.LoadFile(workflowFile, store)
}
```

## Summary

YAML integration in Pocket enables:

1. **Declarative workflows** that non-programmers can understand
2. **Structured output** formatting for LLM interactions
3. **Configuration management** with environment support
4. **Dynamic generation** of workflows from templates
5. **Schema validation** for correctness and safety

This makes Pocket workflows more accessible, maintainable, and suitable for various deployment scenarios.