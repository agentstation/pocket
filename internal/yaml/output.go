package yaml

import (
	"context"
	"fmt"

	"github.com/agentstation/pocket"
	"gopkg.in/yaml.v3"
)

// YAMLNode creates a node that marshals its output to YAML format.
// If execFn is provided, it wraps the function to marshal its output as YAML.
// If execFn is nil, the node will marshal its input as YAML.
func YAMLNode(name string, execFn pocket.ExecFunc) *pocket.Node {
	// Create the YAML marshaling exec function
	yamlExec := func(ctx context.Context, input any) (any, error) {
		var dataToMarshal any
		
		if execFn != nil {
			// Execute the user's function first
			result, err := execFn(ctx, input)
			if err != nil {
				return nil, err
			}
			dataToMarshal = result
		} else {
			// No exec function provided, marshal the input
			dataToMarshal = input
		}
		
		// Marshal to YAML
		yamlBytes, err := yaml.Marshal(dataToMarshal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
		}
		
		// Return both the YAML string and the original data
		return YAMLOutput{
			YAML:   string(yamlBytes),
			Data:   dataToMarshal,
			Format: "yaml",
		}, nil
	}
	
	return pocket.NewNode[any, any](name,
		pocket.WithExec(yamlExec),
	)
}

// YAMLNodeWithLifecycle creates a YAML node with full lifecycle support.
// The exec function's output will be marshaled to YAML.
func YAMLNodeWithLifecycle(name string, prep pocket.PrepFunc, exec pocket.ExecFunc, post pocket.PostFunc) *pocket.Node {
	// Wrap the exec function to marshal output as YAML
	var yamlExec pocket.ExecFunc
	if exec != nil {
		yamlExec = func(ctx context.Context, input any) (any, error) {
			result, err := exec(ctx, input)
			if err != nil {
				return nil, err
			}
			
			yamlBytes, err := yaml.Marshal(result)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
			}
			
			return YAMLOutput{
				YAML:   string(yamlBytes),
				Data:   result,
				Format: "yaml",
			}, nil
		}
	} else {
		// Default: marshal input as YAML
		yamlExec = func(ctx context.Context, input any) (any, error) {
			yamlBytes, err := yaml.Marshal(input)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
			}
			
			return YAMLOutput{
				YAML:   string(yamlBytes),
				Data:   input,
				Format: "yaml",
			}, nil
		}
	}
	
	// Create node with all lifecycle functions
	opts := []pocket.Option{
		pocket.WithExec(yamlExec),
	}
	
	if prep != nil {
		opts = append(opts, pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			return prep(ctx, store, input)
		}))
	}
	
	if post != nil {
		opts = append(opts, pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			return post(ctx, store, input, prep, exec)
		}))
	}
	
	return pocket.NewNode[any, any](name, opts...)
}

// YAMLOutput represents structured output in YAML format.
type YAMLOutput struct {
	YAML   string
	Data   any
	Format string
}

// String returns the YAML representation.
func (y YAMLOutput) String() string {
	return y.YAML
}

// ToYAML converts any data to YAML string.
func ToYAML(data any) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	return string(yamlBytes), nil
}

// FromYAML parses YAML string into the target structure.
func FromYAML(yamlStr string, target any) error {
	return yaml.Unmarshal([]byte(yamlStr), target)
}