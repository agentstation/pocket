package nodes

import (
	"strings"
	"testing"
)

func TestValidateNodeConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		meta := Metadata{
			Type: "test",
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type": "string",
					},
					"count": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
				},
				"required": []string{"message"},
			},
		}

		config := map[string]interface{}{
			"message": "hello",
			"count":   5,
		}

		err := ValidateNodeConfig(&meta, config)
		if err != nil {
			t.Errorf("Expected valid config, got error: %v", err)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		meta := Metadata{
			Type: "test",
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"message"},
			},
		}

		config := map[string]interface{}{
			"other": "value",
		}

		err := ValidateNodeConfig(&meta, config)
		if err == nil {
			t.Error("Expected validation error for missing required field")
		}
		if !strings.Contains(err.Error(), "message") {
			t.Errorf("Expected error to mention 'message', got: %v", err)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		meta := Metadata{
			Type: "test",
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{
						"type": "integer",
					},
				},
			},
		}

		config := map[string]interface{}{
			"count": "not a number",
		}

		err := ValidateNodeConfig(&meta, config)
		if err == nil {
			t.Error("Expected validation error for invalid type")
		}
		if !strings.Contains(err.Error(), "count") {
			t.Errorf("Expected error to mention 'count', got: %v", err)
		}
	})

	t.Run("enum validation", func(t *testing.T) {
		meta := Metadata{
			Type: "test",
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"method": map[string]interface{}{
						"type": "string",
						"enum": []interface{}{"GET", "POST", "PUT"},
					},
				},
			},
		}

		// Valid enum value
		config := map[string]interface{}{
			"method": "POST",
		}
		err := ValidateNodeConfig(&meta, config)
		if err != nil {
			t.Errorf("Expected valid enum value, got error: %v", err)
		}

		// Invalid enum value
		config = map[string]interface{}{
			"method": "DELETE",
		}
		err = ValidateNodeConfig(&meta, config)
		if err == nil {
			t.Error("Expected validation error for invalid enum value")
		}
	})

	t.Run("no schema", func(t *testing.T) {
		meta := Metadata{
			Type: "test",
			// No ConfigSchema
		}

		config := map[string]interface{}{
			"any": "value",
		}

		err := ValidateNodeConfig(&meta, config)
		if err != nil {
			t.Errorf("Expected no error when schema is not defined, got: %v", err)
		}
	})

	t.Run("additional properties", func(t *testing.T) {
		meta := Metadata{
			Type: "test",
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"additionalProperties": false,
			},
		}

		config := map[string]interface{}{
			"name":  "test",
			"extra": "not allowed",
		}

		err := ValidateNodeConfig(&meta, config)
		if err == nil {
			t.Error("Expected validation error for additional properties")
		}
	})
}

func TestEchoNodeValidation(t *testing.T) {
	// Test real node schema validation
	builder := &EchoNodeBuilder{}
	meta := builder.Metadata()

	t.Run("valid echo config", func(t *testing.T) {
		config := map[string]interface{}{
			"message": "Hello, World!",
		}

		err := ValidateNodeConfig(&meta, config)
		if err != nil {
			t.Errorf("Expected valid config, got error: %v", err)
		}
	})

	t.Run("empty config with default", func(t *testing.T) {
		config := map[string]interface{}{}

		// Echo node has default message, so empty config is valid
		err := ValidateNodeConfig(&meta, config)
		if err != nil {
			t.Errorf("Expected valid config with default, got error: %v", err)
		}
	})
}

func TestHTTPNodeValidation(t *testing.T) {
	builder := &HTTPNodeBuilder{}
	meta := builder.Metadata()

	t.Run("valid http config", func(t *testing.T) {
		config := map[string]interface{}{
			"url":    "https://api.example.com",
			"method": "POST",
		}

		err := ValidateNodeConfig(&meta, config)
		if err != nil {
			t.Errorf("Expected valid config, got error: %v", err)
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		config := map[string]interface{}{
			"url":    "https://api.example.com",
			"method": "INVALID",
		}

		err := ValidateNodeConfig(&meta, config)
		if err == nil {
			t.Error("Expected validation error for invalid method")
		}
	})
}
