package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/yaml"
)

const (
	categoryData = "data"
	categoryIO   = "io"
	categoryCore = "core"
	categoryFlow = "flow"
)

func TestEchoNode(t *testing.T) {
	builder := &EchoNodeBuilder{}
	def := &yaml.NodeDefinition{
		Name: "test-echo",
		Config: map[string]interface{}{
			"message": "Test message",
		},
	}

	node, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Failed to build echo node: %v", err)
	}

	ctx := context.Background()
	store := pocket.NewStore()
	input := map[string]interface{}{"test": "data"}

	// Run lifecycle
	prepResult, err := node.Prep(ctx, store, input)
	if err != nil {
		t.Fatalf("Prep failed: %v", err)
	}

	execResult, err := node.Exec(ctx, prepResult)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	output, next, err := node.Post(ctx, store, input, prepResult, execResult)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	// Verify output
	result, ok := output.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map output, got %T", output)
	}

	if result["message"] != "Test message" {
		t.Errorf("Expected message 'Test message', got %v", result["message"])
	}

	if next != "default" {
		t.Errorf("Expected next 'default', got %v", next)
	}
}

func TestDelayNode(t *testing.T) {
	builder := &DelayNodeBuilder{}
	def := &yaml.NodeDefinition{
		Name: "test-delay",
		Config: map[string]interface{}{
			"duration": "100ms",
		},
	}

	node, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Failed to build delay node: %v", err)
	}

	ctx := context.Background()
	input := "test input"

	start := time.Now()
	result, err := node.Exec(ctx, input)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	if result != input {
		t.Errorf("Expected input to pass through, got %v", result)
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("Expected delay of at least 100ms, got %v", elapsed)
	}
}

func TestConditionalNode(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "high score",
			input:    map[string]interface{}{"score": 0.9},
			expected: "high",
		},
		{
			name:     "medium score",
			input:    map[string]interface{}{"score": 0.6},
			expected: "medium",
		},
		{
			name:     "low score",
			input:    map[string]interface{}{"score": 0.3},
			expected: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &ConditionalNodeBuilder{}
			def := &yaml.NodeDefinition{
				Name: "test-conditional",
				Config: map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{"if": "{{gt .score 0.8}}", "then": "high"},
						map[string]interface{}{"if": "{{gt .score 0.5}}", "then": "medium"},
					},
					"else": "low",
				},
			}

			node, err := builder.Build(def)
			if err != nil {
				t.Fatalf("Failed to build conditional node: %v", err)
			}

			ctx := context.Background()
			store := pocket.NewStore()

			_, next, err := node.Post(ctx, store, tt.input, tt.input, tt.input)
			if err != nil {
				t.Fatalf("Post failed: %v", err)
			}

			if next != tt.expected {
				t.Errorf("Expected route '%s', got '%s'", tt.expected, next)
			}
		})
	}
}

func TestRouterNode(t *testing.T) {
	builder := &RouterNodeBuilder{}
	def := &yaml.NodeDefinition{
		Name: "test-router",
		Config: map[string]interface{}{
			"route": "custom-route",
		},
	}

	node, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Failed to build router node: %v", err)
	}

	ctx := context.Background()
	store := pocket.NewStore()
	input := "test"

	_, next, err := node.Post(ctx, store, input, input, input)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if next != "custom-route" {
		t.Errorf("Expected route 'custom-route', got '%s'", next)
	}
}

func TestTransformNode(t *testing.T) {
	builder := &TransformNodeBuilder{}
	def := &yaml.NodeDefinition{
		Name: "test-transform",
	}

	node, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Failed to build transform node: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{"data": "test"}

	result, err := node.Exec(ctx, input)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	output, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map output, got %T", result)
	}

	if !output["transformed"].(bool) {
		t.Error("Expected transformed to be true")
	}

	if output["node"] != "test-transform" {
		t.Errorf("Expected node name 'test-transform', got %v", output["node"])
	}
}

func TestTransformNodeWithScore(t *testing.T) {
	builder := &TransformNodeBuilder{}
	def := &yaml.NodeDefinition{
		Name: "generate-score",
	}

	node, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Failed to build transform node: %v", err)
	}

	ctx := context.Background()
	result, err := node.Exec(ctx, nil)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	output, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map output, got %T", result)
	}

	score, ok := output["score"].(float64)
	if !ok {
		t.Fatal("Expected score in output")
	}

	if score < 0 || score > 1 {
		t.Errorf("Expected score between 0 and 1, got %f", score)
	}
}

func TestNodeMetadata(t *testing.T) {
	builders := []NodeBuilder{
		&EchoNodeBuilder{},
		&DelayNodeBuilder{},
		&RouterNodeBuilder{},
		&ConditionalNodeBuilder{},
		&TransformNodeBuilder{},
	}

	for _, builder := range builders {
		meta := builder.Metadata()

		// Check required fields
		if meta.Type == "" {
			t.Errorf("Node type is empty for %T", builder)
		}
		if meta.Category == "" {
			t.Errorf("Node category is empty for %T", builder)
		}
		if meta.Description == "" {
			t.Errorf("Node description is empty for %T", builder)
		}
		if meta.ConfigSchema == nil {
			t.Errorf("Node config schema is nil for %T", builder)
		}
	}
}

func TestInvalidConfigurations(t *testing.T) {
	t.Run("conditional without conditions", func(t *testing.T) {
		builder := &ConditionalNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name:   "test",
			Config: map[string]interface{}{},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for missing conditions")
		}
		if !strings.Contains(err.Error(), "conditions must be an array") {
			t.Errorf("Expected error about conditions, got: %v", err)
		}
	})

	t.Run("conditional with invalid condition", func(t *testing.T) {
		builder := &ConditionalNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test",
			Config: map[string]interface{}{
				"conditions": []interface{}{
					"invalid", // Not a map
				},
			},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for invalid condition")
		}
		if !strings.Contains(err.Error(), "must be an object") {
			t.Errorf("Expected error about object type, got: %v", err)
		}
	})
}

func TestTemplateNode(t *testing.T) {
	t.Run("simple string template", func(t *testing.T) {
		builder := &TemplateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-template",
			Config: map[string]interface{}{
				"template": "Hello, {{.name}}! You have {{.count}} messages.",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build template node: %v", err)
		}

		ctx := context.Background()
		input := map[string]interface{}{
			"name":  "Alice",
			"count": 5,
		}

		result, err := node.Exec(ctx, input)
		if err != nil {
			t.Fatalf("Exec failed: %v", err)
		}

		expected := "Hello, Alice! You have 5 messages."
		if result != expected {
			t.Errorf("Expected '%s', got '%v'", expected, result)
		}
	})

	t.Run("json output format", func(t *testing.T) {
		builder := &TemplateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-json-template",
			Config: map[string]interface{}{
				"template":      `{"user": "{{.name}}", "active": true}`,
				"output_format": "json",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build template node: %v", err)
		}

		ctx := context.Background()
		input := map[string]interface{}{"name": "Bob"}

		result, err := node.Exec(ctx, input)
		if err != nil {
			t.Fatalf("Exec failed: %v", err)
		}

		jsonResult, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result for JSON, got %T", result)
		}

		if jsonResult["user"] != "Bob" {
			t.Errorf("Expected user 'Bob', got '%v'", jsonResult["user"])
		}
		if jsonResult["active"] != true {
			t.Errorf("Expected active true, got '%v'", jsonResult["active"])
		}
	})

	t.Run("missing template", func(t *testing.T) {
		builder := &TemplateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name:   "test-missing",
			Config: map[string]interface{}{},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for missing template")
		}
		if !strings.Contains(err.Error(), "either 'template' or 'file' must be specified") {
			t.Errorf("Expected error about missing template, got: %v", err)
		}
	})
}

func TestHTTPNode(t *testing.T) {
	// Note: These are basic unit tests. For real HTTP testing,
	// we would use httptest to create a test server.

	t.Run("missing url", func(t *testing.T) {
		builder := &HTTPNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name:   "test-http",
			Config: map[string]interface{}{},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for missing URL")
		}
		if !strings.Contains(err.Error(), "url is required") {
			t.Errorf("Expected error about missing URL, got: %v", err)
		}
	})

	t.Run("config validation", func(t *testing.T) {
		builder := &HTTPNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-http",
			Config: map[string]interface{}{
				"url":     "https://example.com/api",
				"method":  "POST",
				"timeout": "10s",
				"headers": map[string]interface{}{
					"Authorization": "Bearer token",
				},
				"retry": map[string]interface{}{
					"max_attempts": 5,
					"delay":        "2s",
				},
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build HTTP node: %v", err)
		}

		// Just verify the node was created successfully
		if node == nil {
			t.Error("Expected non-nil node")
		}
	})

	t.Run("metadata", func(t *testing.T) {
		builder := &HTTPNodeBuilder{}
		meta := builder.Metadata()

		if meta.Type != "http" {
			t.Errorf("Expected type 'http', got '%s'", meta.Type)
		}
		if meta.Category != "io" {
			t.Errorf("Expected category 'io', got '%s'", meta.Category)
		}
		if len(meta.Examples) != 2 {
			t.Errorf("Expected 2 examples, got %d", len(meta.Examples))
		}
	})
}

func TestJSONPathNode(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	t.Run("extract simple field", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-jsonpath",
			Config: map[string]interface{}{
				"path": "$.user.name",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build JSONPath node: %v", err)
		}

		input := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "Alice",
				"age":  30,
			},
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if result != "Alice" {
			t.Errorf("Expected 'Alice', got %v", result)
		}
	})

	t.Run("extract from array", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-jsonpath",
			Config: map[string]interface{}{
				"path":     "$.items[*].price",
				"multiple": true,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build JSONPath node: %v", err)
		}

		input := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"name": "Book", "price": 10.99},
				map[string]interface{}{"name": "Pen", "price": 2.50},
				map[string]interface{}{"name": "Notebook", "price": 5.00},
			},
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		prices, ok := result.([]interface{})
		if !ok {
			t.Fatalf("Expected array result, got %T", result)
		}

		if len(prices) != 3 {
			t.Errorf("Expected 3 prices, got %d", len(prices))
		}

		expected := []float64{10.99, 2.50, 5.00}
		for i, price := range prices {
			if price != expected[i] {
				t.Errorf("Price[%d]: expected %.2f, got %v", i, expected[i], price)
			}
		}
	})

	t.Run("use default value", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-jsonpath",
			Config: map[string]interface{}{
				"path":    "$.missing.field",
				"default": "Not found",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build JSONPath node: %v", err)
		}

		input := map[string]interface{}{
			"other": "data",
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if result != "Not found" {
			t.Errorf("Expected 'Not found', got %v", result)
		}
	})

	t.Run("extract first element", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-jsonpath",
			Config: map[string]interface{}{
				"path":     "$.numbers[0]",
				"multiple": false,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build JSONPath node: %v", err)
		}

		input := map[string]interface{}{
			"numbers": []interface{}{42, 84, 126},
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		// When extracting a single element from array, it should return the value directly
		if result != 42 {
			t.Errorf("Expected 42, got %v", result)
		}
	})

	t.Run("unwrap single element array", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-jsonpath",
			Config: map[string]interface{}{
				"path":   "$.single",
				"unwrap": true,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build JSONPath node: %v", err)
		}

		input := map[string]interface{}{
			"single": []interface{}{"only-value"},
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		// With unwrap=true, single element arrays should be unwrapped
		if result != "only-value" {
			t.Errorf("Expected 'only-value', got %v", result)
		}
	})

	t.Run("missing path", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name:   "test-jsonpath",
			Config: map[string]interface{}{},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for missing path")
		}
		if !strings.Contains(err.Error(), "path is required") {
			t.Errorf("Expected error about missing path, got: %v", err)
		}
	})

	t.Run("invalid jsonpath expression", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-jsonpath",
			Config: map[string]interface{}{
				"path": "$[invalid syntax",
			},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for invalid JSONPath")
		}
		if !strings.Contains(err.Error(), "invalid JSONPath expression") {
			t.Errorf("Expected error about invalid JSONPath, got: %v", err)
		}
	})

	t.Run("metadata", func(t *testing.T) {
		builder := &JSONPathNodeBuilder{}
		meta := builder.Metadata()

		if meta.Type != "jsonpath" {
			t.Errorf("Expected type 'jsonpath', got %s", meta.Type)
		}
		if meta.Category != categoryData {
			t.Errorf("Expected category 'data', got %s", meta.Category)
		}
		if len(meta.Examples) == 0 {
			t.Error("Expected at least one example")
		}
	})
}

func TestValidateNode(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	t.Run("valid data passes schema", func(t *testing.T) {
		builder := &ValidateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-validate",
			Config: map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":  map[string]interface{}{"type": "string"},
						"email": map[string]interface{}{"type": "string", "format": "email"},
						"age":   map[string]interface{}{"type": "integer", "minimum": 0},
					},
					"required": []string{"name", "email"},
				},
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build validate node: %v", err)
		}

		input := map[string]interface{}{
			"name":  "Alice",
			"email": "alice@example.com",
			"age":   30,
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if !res["valid"].(bool) {
			t.Error("Expected validation to pass")
		}

		errors, ok := res["errors"].([]interface{})
		if !ok || len(errors) != 0 {
			t.Errorf("Expected empty errors array, got %v", res["errors"])
		}
	})

	t.Run("invalid data fails validation", func(t *testing.T) {
		builder := &ValidateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-validate",
			Config: map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"score": map[string]interface{}{
							"type":    "number",
							"minimum": 0,
							"maximum": 100,
						},
					},
					"required": []string{"score"},
				},
				"fail_on_error": false, // Don't fail, just return validation result
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build validate node: %v", err)
		}

		input := map[string]interface{}{
			"score": 150, // Exceeds maximum
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if res["valid"].(bool) {
			t.Error("Expected validation to fail")
		}

		errors, ok := res["errors"].([]interface{})
		if !ok || len(errors) == 0 {
			t.Error("Expected validation errors")
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		builder := &ValidateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-validate",
			Config: map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
					"required": []string{"name"},
				},
				"fail_on_error": false,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build validate node: %v", err)
		}

		input := map[string]interface{}{
			"other": "field",
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if res["valid"].(bool) {
			t.Error("Expected validation to fail for missing required field")
		}
	})

	t.Run("fail on error mode", func(t *testing.T) {
		builder := &ValidateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-validate",
			Config: map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"count": map[string]interface{}{"type": "integer"},
					},
				},
				"fail_on_error": true, // Default behavior
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build validate node: %v", err)
		}

		input := map[string]interface{}{
			"count": "not a number", // Invalid type
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, input)
		if err == nil {
			t.Error("Expected error when validation fails with fail_on_error=true")
		}
		if !strings.Contains(err.Error(), "validation failed") {
			t.Errorf("Expected validation failed error, got: %v", err)
		}
	})

	t.Run("missing schema config", func(t *testing.T) {
		builder := &ValidateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name:   "test-validate",
			Config: map[string]interface{}{},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for missing schema")
		}
		if !strings.Contains(err.Error(), "either 'schema' or 'schema_file' must be specified") {
			t.Errorf("Expected error about missing schema, got: %v", err)
		}
	})

	t.Run("metadata", func(t *testing.T) {
		builder := &ValidateNodeBuilder{}
		meta := builder.Metadata()

		if meta.Type != "validate" {
			t.Errorf("Expected type 'validate', got %s", meta.Type)
		}
		if meta.Category != categoryData {
			t.Errorf("Expected category 'data', got %s", meta.Category)
		}
		if len(meta.Examples) == 0 {
			t.Error("Expected at least one example")
		}
	})
}

func TestAggregateNode(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	t.Run("aggregate array mode", func(t *testing.T) {
		builder := &AggregateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-aggregate",
			Config: map[string]interface{}{
				"mode": "array",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build aggregate node: %v", err)
		}

		input := []interface{}{"item1", "item2", "item3"}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		data, ok := res["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected array data, got %T", res["data"])
		}

		if len(data) != 3 {
			t.Errorf("Expected 3 items, got %d", len(data))
		}

		if res["count"].(int) != 3 {
			t.Errorf("Expected count 3, got %v", res["count"])
		}
	})

	t.Run("aggregate object mode with template key", func(t *testing.T) {
		builder := &AggregateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-aggregate",
			Config: map[string]interface{}{
				"mode": "object",
				"key":  "{{.type}}",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build aggregate node: %v", err)
		}

		input := []interface{}{
			map[string]interface{}{"type": "user", "name": "Alice"},
			map[string]interface{}{"type": "product", "name": "Widget"},
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		data, ok := res["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected object data, got %T", res["data"])
		}

		if _, exists := data["user"]; !exists {
			t.Error("Expected 'user' key in result")
		}
		if _, exists := data["product"]; !exists {
			t.Error("Expected 'product' key in result")
		}
	})

	t.Run("aggregate merge mode", func(t *testing.T) {
		builder := &AggregateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-aggregate",
			Config: map[string]interface{}{
				"mode": "merge",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build aggregate node: %v", err)
		}

		input := []interface{}{
			map[string]interface{}{"user": "Alice", "role": "admin"},
			map[string]interface{}{"settings": map[string]interface{}{"theme": "dark"}},
			map[string]interface{}{"settings": map[string]interface{}{"lang": "en"}},
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		data, ok := res["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected object data, got %T", res["data"])
		}

		// Check merged result
		if data["user"] != "Alice" {
			t.Errorf("Expected user=Alice, got %v", data["user"])
		}
		if data["role"] != "admin" {
			t.Errorf("Expected role=admin, got %v", data["role"])
		}

		settings, ok := data["settings"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected settings to be a map")
		}
		if settings["theme"] != "dark" {
			t.Errorf("Expected theme=dark, got %v", settings["theme"])
		}
		if settings["lang"] != "en" {
			t.Errorf("Expected lang=en, got %v", settings["lang"])
		}
	})

	t.Run("aggregate concat mode", func(t *testing.T) {
		builder := &AggregateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-aggregate",
			Config: map[string]interface{}{
				"mode": "concat",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build aggregate node: %v", err)
		}

		input := []interface{}{
			[]interface{}{"a", "b"},
			[]interface{}{"c", "d"},
			"e", // Single item
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		data, ok := res["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected array data, got %T", res["data"])
		}

		expected := []interface{}{"a", "b", "c", "d", "e"}
		if len(data) != len(expected) {
			t.Errorf("Expected %d items, got %d", len(expected), len(data))
		}

		for i, v := range expected {
			if data[i] != v {
				t.Errorf("Item %d: expected %v, got %v", i, v, data[i])
			}
		}
	})

	t.Run("invalid mode", func(t *testing.T) {
		builder := &AggregateNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-aggregate",
			Config: map[string]interface{}{
				"mode": "invalid",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build aggregate node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, []interface{}{"test"})
		if err == nil {
			t.Error("Expected error for invalid mode")
		}
		if !strings.Contains(err.Error(), "unknown aggregation mode") {
			t.Errorf("Expected unknown mode error, got: %v", err)
		}
	})

	t.Run("metadata", func(t *testing.T) {
		builder := &AggregateNodeBuilder{}
		meta := builder.Metadata()

		if meta.Type != "aggregate" {
			t.Errorf("Expected type 'aggregate', got %s", meta.Type)
		}
		if meta.Category != categoryData {
			t.Errorf("Expected category 'data', got %s", meta.Category)
		}
		if len(meta.Examples) == 0 {
			t.Error("Expected at least one example")
		}
	})
}

func TestFileNode(t *testing.T) {
	store := pocket.NewStore()
	ctx := context.Background()

	// Create temp directory for tests
	tempDir := t.TempDir()

	t.Run("read existing file", func(t *testing.T) {
		// Create test file
		testContent := "Hello, World!"
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation": "read",
				"path":      "test.txt",
				"base_dir":  tempDir,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build file node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if !res["exists"].(bool) {
			t.Error("Expected file to exist")
		}

		if res["content"].(string) != testContent {
			t.Errorf("Expected content '%s', got '%s'", testContent, res["content"])
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation": "read",
				"path":      "missing.txt",
				"base_dir":  tempDir,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build file node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if res["exists"].(bool) {
			t.Error("Expected file to not exist")
		}
	})

	t.Run("write file with create dirs", func(t *testing.T) {
		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation":   "write",
				"path":        "output/result.txt",
				"content":     "Test output",
				"base_dir":    tempDir,
				"create_dirs": true,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build file node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if !res["exists"].(bool) {
			t.Error("Expected file to exist after write")
		}

		// Verify file was actually written
		writtenPath := filepath.Join(tempDir, "output", "result.txt")
		content, err := os.ReadFile(writtenPath)
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		}
		if string(content) != "Test output" {
			t.Errorf("Expected written content 'Test output', got '%s'", string(content))
		}
	})

	t.Run("append to file", func(t *testing.T) {
		// Create initial file
		testFile := filepath.Join(tempDir, "append.txt")
		if err := os.WriteFile(testFile, []byte("Line 1\n"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation": "append",
				"path":      "append.txt",
				"content":   "Line 2\n",
				"base_dir":  tempDir,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build file node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		// Verify append worked
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read appended file: %v", err)
		}
		if string(content) != "Line 1\nLine 2\n" {
			t.Errorf("Expected 'Line 1\\nLine 2\\n', got '%s'", string(content))
		}
	})

	t.Run("list directory", func(t *testing.T) {
		// Create test files
		listDir := filepath.Join(tempDir, "list")
		if err := os.MkdirAll(listDir, 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(listDir, "file1.txt"), []byte("content1"), 0o644); err != nil {
			t.Fatalf("Failed to write file1.txt: %v", err)
		}
		if err := os.WriteFile(filepath.Join(listDir, "file2.txt"), []byte("content2"), 0o644); err != nil {
			t.Fatalf("Failed to write file2.txt: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(listDir, "subdir"), 0o755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation": "list",
				"path":      "list",
				"base_dir":  tempDir,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build file node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		files, ok := res["files"].([]interface{})
		if !ok {
			t.Fatalf("Expected files array, got %T", res["files"])
		}

		if len(files) != 3 { // file1.txt, file2.txt, subdir
			t.Errorf("Expected 3 files, got %d", len(files))
		}
	})

	t.Run("sandbox path restriction", func(t *testing.T) {
		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation":      "read",
				"path":           "../../../etc/passwd",
				"base_dir":       tempDir,
				"allow_absolute": false,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build file node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, nil)
		if err == nil {
			t.Error("Expected error for path outside base directory")
		}
		if !strings.Contains(err.Error(), "outside base directory") {
			t.Errorf("Expected sandbox error, got: %v", err)
		}
	})

	t.Run("template content", func(t *testing.T) {
		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation": "write",
				"path":      "template.txt",
				"content":   "Hello, {{.name}}!",
				"base_dir":  tempDir,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build file node: %v", err)
		}

		input := map[string]interface{}{"name": "Alice"}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		// Verify templated content
		writtenPath := filepath.Join(tempDir, "template.txt")
		content, err := os.ReadFile(writtenPath)
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		}
		if string(content) != "Hello, Alice!" {
			t.Errorf("Expected 'Hello, Alice!', got '%s'", string(content))
		}
	})

	t.Run("missing path", func(t *testing.T) {
		builder := &FileNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-file",
			Config: map[string]interface{}{
				"operation": "read",
			},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected error for missing path")
		}
		if !strings.Contains(err.Error(), "path is required") {
			t.Errorf("Expected path required error, got: %v", err)
		}
	})

	t.Run("metadata", func(t *testing.T) {
		builder := &FileNodeBuilder{}
		meta := builder.Metadata()

		if meta.Type != "file" {
			t.Errorf("Expected type 'file', got %s", meta.Type)
		}
		if meta.Category != "io" {
			t.Errorf("Expected category 'io', got %s", meta.Category)
		}
		if len(meta.Examples) == 0 {
			t.Error("Expected at least one example")
		}
	})
}

func TestExecNode(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("simple command", func(t *testing.T) {
		builder := &ExecNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-exec",
			Config: map[string]interface{}{
				"command": "echo",
				"args":    []interface{}{"Hello, World!"},
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build exec node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if res["exit_code"].(int) != 0 {
			t.Errorf("Expected exit code 0, got %d", res["exit_code"])
		}

		stdout, ok := res["stdout"].(string)
		if !ok {
			t.Fatalf("Expected stdout to be string, got %T: %+v", res["stdout"], res)
		}
		if !strings.Contains(stdout, "Hello, World!") {
			t.Errorf("Expected stdout to contain 'Hello, World!', got: %q", stdout)
		}
	})

	t.Run("command with timeout", func(t *testing.T) {
		builder := &ExecNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-exec",
			Config: map[string]interface{}{
				"command": "sh",
				"args":    []interface{}{"-c", "sleep 10"},
				"timeout": "500ms",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build exec node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, nil)
		// Remove debug output
		if err == nil {
			t.Error("Expected timeout error")
		} else if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})

	t.Run("allowed commands whitelist", func(t *testing.T) {
		builder := &ExecNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-exec",
			Config: map[string]interface{}{
				"command":          "rm",
				"args":             []interface{}{"-rf", "/tmp/test"},
				"allowed_commands": []interface{}{"echo", "ls", "cat"},
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build exec node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, nil)
		if err == nil {
			t.Error("Expected error for disallowed command")
		} else if !strings.Contains(err.Error(), "not in allowed list") {
			t.Errorf("Expected whitelist error, got: %v", err)
		}
	})

	t.Run("allowed command executes", func(t *testing.T) {
		builder := &ExecNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-exec",
			Config: map[string]interface{}{
				"command":          "echo",
				"args":             []interface{}{"test"},
				"allowed_commands": []interface{}{"echo", "ls", "cat"},
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build exec node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res := result.(map[string]interface{})
		if res["exit_code"].(int) != 0 {
			t.Errorf("Expected exit code 0, got %d", res["exit_code"])
		}
	})

	t.Run("environment variables", func(t *testing.T) {
		builder := &ExecNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-exec",
			Config: map[string]interface{}{
				"command": "sh",
				"args":    []interface{}{"-c", "echo $TEST_VAR"},
				"env": map[string]interface{}{
					"TEST_VAR": "test_value",
				},
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build exec node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res := result.(map[string]interface{})
		stdout := res["stdout"].(string)
		if !strings.Contains(stdout, "test_value") {
			t.Errorf("Expected stdout to contain 'test_value', got: %s", stdout)
		}
	})

	t.Run("no output capture", func(t *testing.T) {
		builder := &ExecNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-exec",
			Config: map[string]interface{}{
				"command":        "echo",
				"args":           []interface{}{"test"},
				"capture_output": false,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build exec node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res := result.(map[string]interface{})
		if _, hasStdout := res["stdout"]; hasStdout {
			t.Error("Expected no stdout when capture_output is false")
		}
		if _, hasStderr := res["stderr"]; hasStderr {
			t.Error("Expected no stderr when capture_output is false")
		}
	})

	t.Run("metadata", func(t *testing.T) {
		builder := &ExecNodeBuilder{}
		meta := builder.Metadata()

		if meta.Type != "exec" {
			t.Errorf("Expected type 'exec', got %s", meta.Type)
		}
		if meta.Category != "io" {
			t.Errorf("Expected category 'io', got %s", meta.Category)
		}
		if len(meta.Examples) == 0 {
			t.Error("Expected at least one example")
		}
	})
}

func TestParallelNode(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("basic parallel execution", func(t *testing.T) {
		builder := &ParallelNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-parallel",
			Config: map[string]interface{}{
				"tasks": []interface{}{
					map[string]interface{}{
						"name":      "task1",
						"operation": "transform",
					},
					map[string]interface{}{
						"name":      "task2",
						"operation": "transform",
					},
					map[string]interface{}{
						"name":      "task3",
						"operation": "transform",
					},
				},
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build parallel node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, "test input")
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		res, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		summary := res["summary"].(map[string]interface{})
		if summary["total"].(int) != 3 {
			t.Errorf("Expected 3 total tasks, got %d", summary["total"])
		}
		if summary["successful"].(int) != 3 {
			t.Errorf("Expected 3 successful tasks, got %d", summary["successful"])
		}
	})

	t.Run("with concurrency limit", func(t *testing.T) {
		builder := &ParallelNodeBuilder{Verbose: true}
		def := &yaml.NodeDefinition{
			Name: "test-parallel",
			Config: map[string]interface{}{
				"tasks": []interface{}{
					map[string]interface{}{
						"name":      "task1",
						"operation": "http_get",
					},
					map[string]interface{}{
						"name":      "task2",
						"operation": "http_get",
					},
					map[string]interface{}{
						"name":      "task3",
						"operation": "http_get",
					},
					map[string]interface{}{
						"name":      "task4",
						"operation": "http_get",
					},
					map[string]interface{}{
						"name":      "task5",
						"operation": "http_get",
					},
				},
				"max_concurrency": 2,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build parallel node: %v", err)
		}

		start := time.Now()
		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}
		duration := time.Since(start)

		// With concurrency limit of 2, tasks should still run efficiently
		// The semaphore ensures no more than 2 run at once, but with
		// goroutines they can still complete quickly
		if duration > 500*time.Millisecond {
			t.Errorf("Expected execution to complete within 500ms, got %v", duration)
		}

		res := result.(map[string]interface{})
		summary := res["summary"].(map[string]interface{})
		if summary["successful"].(int) != 5 {
			t.Errorf("Expected 5 successful tasks, got %d", summary["successful"])
		}
	})

	t.Run("fail fast", func(t *testing.T) {
		builder := &ParallelNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-parallel",
			Config: map[string]interface{}{
				"tasks": []interface{}{
					map[string]interface{}{
						"name":      "task1",
						"operation": "transform",
					},
					map[string]interface{}{
						"name":      "task2",
						"operation": "error", // This will fail
					},
					map[string]interface{}{
						"name":      "task3",
						"operation": "transform",
					},
				},
				"fail_fast": true,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build parallel node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, nil)
		if err == nil {
			t.Error("Expected error with fail-fast")
		}
		if !strings.Contains(err.Error(), "fail-fast") {
			t.Errorf("Expected fail-fast error, got: %v", err)
		}
	})

	t.Run("continue on error", func(t *testing.T) {
		builder := &ParallelNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-parallel",
			Config: map[string]interface{}{
				"tasks": []interface{}{
					map[string]interface{}{
						"name":      "task1",
						"operation": "transform",
					},
					map[string]interface{}{
						"name":      "task2",
						"operation": "error", // This will fail
					},
					map[string]interface{}{
						"name":      "task3",
						"operation": "transform",
					},
				},
				"fail_fast": false,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build parallel node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Should not fail when fail_fast is false: %v", err)
		}

		res := result.(map[string]interface{})
		summary := res["summary"].(map[string]interface{})
		if summary["successful"].(int) != 2 {
			t.Errorf("Expected 2 successful tasks, got %d", summary["successful"])
		}
		if summary["failed"].(int) != 1 {
			t.Errorf("Expected 1 failed task, got %d", summary["failed"])
		}

		errors := res["errors"].([]interface{})
		if len(errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errors))
		}
	})

	t.Run("metadata", func(t *testing.T) {
		builder := &ParallelNodeBuilder{}
		meta := builder.Metadata()

		if meta.Type != "parallel" {
			t.Errorf("Expected type 'parallel', got %s", meta.Type)
		}
		if meta.Category != "flow" {
			t.Errorf("Expected category 'flow', got %s", meta.Category)
		}
		if len(meta.Examples) == 0 {
			t.Error("Expected at least one example")
		}
	})
}
