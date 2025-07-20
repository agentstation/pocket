package builtin

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateNodeConfig validates a node configuration against its schema.
func ValidateNodeConfig(meta *NodeMetadata, config map[string]interface{}) error {
	if len(meta.ConfigSchema) == 0 {
		// No schema defined, skip validation
		return nil
	}

	// Convert schema and config to JSON for validation
	schemaJSON, err := json.Marshal(meta.ConfigSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create JSON Schema loader
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)
	documentLoader := gojsonschema.NewBytesLoader(configJSON)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		// Collect all validation errors
		var errMsg string
		for i, err := range result.Errors() {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.String()
		}
		return fmt.Errorf("config validation failed: %s", errMsg)
	}

	return nil
}

// ValidateAllNodeConfigs validates all node configurations in a registry.
func ValidateAllNodeConfigs(registry *Registry, configs map[string]map[string]interface{}) error {
	for nodeType, config := range configs {
		builder, exists := registry.Get(nodeType)
		if !exists {
			return fmt.Errorf("unknown node type: %s", nodeType)
		}

		meta := builder.Metadata()
		if err := ValidateNodeConfig(&meta, config); err != nil {
			return fmt.Errorf("node '%s' config validation failed: %w", nodeType, err)
		}
	}

	return nil
}
