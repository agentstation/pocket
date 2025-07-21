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
	input := "test" //nolint:goconst // test string literal is fine

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

func TestLuaNode(t *testing.T) {
	ctx := context.Background()
	store := pocket.NewStore()

	t.Run("simple script", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua",
			Config: map[string]interface{}{
				"script": `return {result = "hello from lua"}`,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			if resultMap["result"] != "hello from lua" {
				t.Errorf("Expected result to be 'hello from lua', got %v", resultMap["result"])
			}
		} else {
			t.Errorf("Expected result to be a map, got %T", result)
		}
	})

	t.Run("transform input", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-transform",
			Config: map[string]interface{}{
				"script": `
					local result = {}
					for k, v in pairs(input) do
						if type(v) == "number" then
							result[k] = v * 2
						else
							result[k] = v
						end
					end
					return result
				`,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		input := map[string]interface{}{
			"value": 21,
			"name":  "test",
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			if resultMap["value"] != 42 {
				t.Errorf("Expected value to be 42, got %v", resultMap["value"])
			}
			if resultMap["name"] != "test" {
				t.Errorf("Expected name to be 'test', got %v", resultMap["name"])
			}
		} else {
			t.Errorf("Expected result to be a map, got %T", result)
		}
	})

	t.Run("json functions", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-json",
			Config: map[string]interface{}{
				"script": `
					-- Test JSON encoding
					local data = {name = "test", value = 42}
					local json_str = json_encode(data)
					
					-- Test JSON decoding
					local decoded = json_decode(json_str)
					
					return {
						encoded = json_str,
						decoded = decoded
					}
				`,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			// Check encoded JSON
			if encoded, ok := resultMap["encoded"].(string); ok {
				if !strings.Contains(encoded, "test") || !strings.Contains(encoded, "42") {
					t.Errorf("JSON encoding failed: %s", encoded)
				}
			}

			// Check decoded data
			if decoded, ok := resultMap["decoded"].(map[string]interface{}); ok {
				if decoded["name"] != "test" {
					t.Errorf("JSON decoding failed for name: %v", decoded["name"])
				}
				// JSON numbers could be int or float64 depending on how Lua returns them
				switch v := decoded["value"].(type) {
				case float64:
					if v != 42 {
						t.Errorf("JSON decoding failed for value: %v", v)
					}
				case int:
					if v != 42 {
						t.Errorf("JSON decoding failed for value: %v", v)
					}
				default:
					t.Errorf("JSON decoding failed for value: unexpected type %T", v)
				}
			}
		}
	})

	t.Run("string utilities", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-strings",
			Config: map[string]interface{}{
				"script": `
					local text = "  hello world  "
					return {
						trimmed = str_trim(text),
						split = str_split("one,two,three", ","),
						contains = str_contains("hello world", "world"),
						replaced = str_replace("hello world", "world", "lua")
					}
				`,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			if resultMap["trimmed"] != "hello world" {
				t.Errorf("str_trim failed: %v", resultMap["trimmed"])
			}
			if resultMap["contains"] != true {
				t.Errorf("str_contains failed: %v", resultMap["contains"])
			}
			if resultMap["replaced"] != "hello lua" {
				t.Errorf("str_replace failed: %v", resultMap["replaced"])
			}
			if split, ok := resultMap["split"].([]interface{}); ok {
				if len(split) != 3 || split[0] != "one" {
					t.Errorf("str_split failed: %v", split)
				}
			}
		}
	})

	t.Run("sandboxing", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-sandbox",
			Config: map[string]interface{}{
				"script": `
					-- These should all be nil in sandbox mode
					local dangerous = {
						io = io,
						os = os,
						debug = debug,
						package = package,
						require = require,
						dofile = dofile,
						loadfile = loadfile
					}
					
					local all_nil = true
					for name, func in pairs(dangerous) do
						if func ~= nil then
							all_nil = false
						end
					end
					
					return {sandboxed = all_nil}
				`,
				"sandbox": true,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			if resultMap["sandboxed"] != true {
				t.Error("Sandboxing failed - dangerous functions are available")
			}
		}
	})

	t.Run("no sandbox mode", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-no-sandbox",
			Config: map[string]interface{}{
				"script": `
					-- os should be available when sandbox is disabled
					return {has_os = os ~= nil}
				`,
				"sandbox": false,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			if resultMap["has_os"] != true {
				t.Error("No sandbox mode failed - os should be available")
			}
		}
	})

	t.Run("script timeout", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-timeout",
			Config: map[string]interface{}{
				"script": `
					-- Infinite loop
					while true do
					end
				`,
				"timeout": "100ms",
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		_, err = graph.Run(ctx, nil)
		if err == nil {
			t.Error("Expected timeout error")
		}
		if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})

	t.Run("script file", func(t *testing.T) {
		// Create a temporary script file
		scriptContent := `
			-- Test script from file
			return {
				message = "loaded from file",
				input_type = type_of(input)
			}
		`
		tmpFile, err := os.CreateTemp("", "test-lua-*.lua")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(scriptContent); err != nil {
			t.Fatalf("Failed to write script: %v", err)
		}
		_ = tmpFile.Close()

		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-file",
			Config: map[string]interface{}{
				"file": tmpFile.Name(),
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, "test input")
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			if resultMap["message"] != "loaded from file" {
				t.Errorf("Expected message 'loaded from file', got %v", resultMap["message"])
			}
			if resultMap["input_type"] != "string" {
				t.Errorf("Expected input_type 'string', got %v", resultMap["input_type"])
			}
		}
	})

	t.Run("validation", func(t *testing.T) {
		builder := &LuaNodeBuilder{}

		// No script or file
		def := &yaml.NodeDefinition{
			Name:   "test-lua-invalid",
			Config: map[string]interface{}{},
		}

		_, err := builder.Build(def)
		if err == nil {
			t.Error("Expected validation error for missing script/file")
		}

		// Both script and file
		def.Config = map[string]interface{}{
			"script": "return 1",
			"file":   "test.lua",
		}

		_, err = builder.Build(def)
		if err == nil {
			t.Error("Expected validation error for both script and file")
		}
	})

	t.Run("complex data structures", func(t *testing.T) {
		builder := &LuaNodeBuilder{}
		def := &yaml.NodeDefinition{
			Name: "test-lua-complex",
			Config: map[string]interface{}{
				"script": `
					-- Work with nested data
					local result = {
						users = {},
						total = 0
					}
					
					for i, user in ipairs(input.users) do
						table.insert(result.users, {
							id = user.id,
							name = string.upper(user.name),
							score = user.score * 2
						})
						result.total = result.total + user.score
					end
					
					return result
				`,
			},
		}

		node, err := builder.Build(def)
		if err != nil {
			t.Fatalf("Failed to build node: %v", err)
		}

		input := map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"id": 1, "name": "alice", "score": 10},
				map[string]interface{}{"id": 2, "name": "bob", "score": 20},
			},
		}

		graph := pocket.NewGraph(node, store)
		result, err := graph.Run(ctx, input)
		if err != nil {
			t.Fatalf("Failed to run graph: %v", err)
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			if resultMap["total"] != 30 {
				t.Errorf("Expected total 30, got %v", resultMap["total"])
			}

			if users, ok := resultMap["users"].([]interface{}); ok {
				if len(users) != 2 {
					t.Errorf("Expected 2 users, got %d", len(users))
				}

				if user0, ok := users[0].(map[string]interface{}); ok {
					if user0["name"] != "ALICE" {
						t.Errorf("Expected name ALICE, got %v", user0["name"])
					}
					if user0["score"] != 20 {
						t.Errorf("Expected score 20, got %v", user0["score"])
					}
				}
			}
		}
	})
}
