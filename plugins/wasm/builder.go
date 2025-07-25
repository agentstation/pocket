package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/nodes"
	"github.com/agentstation/pocket/plugins"
	"github.com/agentstation/pocket/yaml"
)

// PluginNodeBuilder creates Pocket nodes from WASM plugins.
type PluginNodeBuilder struct {
	plugin   plugins.Plugin
	nodeType plugins.NodeDefinition
}

// NewPluginNodeBuilder creates a new builder for a specific node type in a plugin.
func NewPluginNodeBuilder(p plugins.Plugin, nodeType *plugins.NodeDefinition) *PluginNodeBuilder {
	return &PluginNodeBuilder{
		plugin:   p,
		nodeType: *nodeType,
	}
}

// Metadata returns the node metadata.
func (b *PluginNodeBuilder) Metadata() nodes.Metadata {
	// Convert plugin node definition to nodes metadata
	return nodes.Metadata{
		Type:         b.nodeType.Type,
		Category:     b.nodeType.Category,
		Description:  b.nodeType.Description,
		ConfigSchema: b.nodeType.ConfigSchema,
		Examples:     convertExamples(b.nodeType.Examples),
		Since:        b.plugin.Metadata().Version,
	}
}

// Build creates a new node instance.
func (b *PluginNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	return pocket.NewNode[any, any](def.Name,
		pocket.Steps{
			Prep: b.prepFunc(def),
			Exec: b.execFunc(def),
			Post: b.postFunc(def),
		},
	), nil
}

// prepFunc creates the prep function for the node.
func (b *PluginNodeBuilder) prepFunc(def *yaml.NodeDefinition) pocket.PrepFunc {
	return func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
		// Create request
		inputJSON, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}

		req := plugins.Request{
			Node:     b.nodeType.Type,
			Function: "prep",
			Config:   def.Config,
			Input:    inputJSON,
		}

		// Marshal request
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		// Call plugin
		respJSON, err := b.plugin.Call(ctx, "prep", reqJSON)
		if err != nil {
			return nil, fmt.Errorf("plugin prep failed: %w", err)
		}

		// Unmarshal response
		var resp plugins.Response
		if err := json.Unmarshal(respJSON, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if !resp.Success {
			return nil, fmt.Errorf("plugin prep error: %s", resp.Error)
		}

		// Unmarshal output
		var output any
		if len(resp.Output) > 0 {
			if err := json.Unmarshal(resp.Output, &output); err != nil {
				return nil, fmt.Errorf("failed to unmarshal output: %w", err)
			}
		}

		return output, nil
	}
}

// execFunc creates the exec function for the node.
func (b *PluginNodeBuilder) execFunc(def *yaml.NodeDefinition) pocket.ExecFunc {
	return func(ctx context.Context, prepData any) (any, error) {
		// Create request
		prepJSON, err := json.Marshal(prepData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal prep data: %w", err)
		}

		req := plugins.Request{
			Node:       b.nodeType.Type,
			Function:   "exec",
			Config:     def.Config,
			PrepResult: prepJSON,
		}

		// Marshal request
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		// Call plugin
		respJSON, err := b.plugin.Call(ctx, "exec", reqJSON)
		if err != nil {
			return nil, fmt.Errorf("plugin exec failed: %w", err)
		}

		// Unmarshal response
		var resp plugins.Response
		if err := json.Unmarshal(respJSON, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if !resp.Success {
			return nil, fmt.Errorf("plugin exec error: %s", resp.Error)
		}

		// Unmarshal output
		var output any
		if len(resp.Output) > 0 {
			if err := json.Unmarshal(resp.Output, &output); err != nil {
				return nil, fmt.Errorf("failed to unmarshal output: %w", err)
			}
		}

		return output, nil
	}
}

// postFunc creates the post function for the node.
func (b *PluginNodeBuilder) postFunc(def *yaml.NodeDefinition) pocket.PostFunc {
	return func(ctx context.Context, store pocket.StoreWriter, input, prepData, execResult any) (any, string, error) {
		// Create request
		inputJSON, _ := json.Marshal(input)
		prepJSON, _ := json.Marshal(prepData)
		execJSON, _ := json.Marshal(execResult)

		req := plugins.Request{
			Node:       b.nodeType.Type,
			Function:   "post",
			Config:     def.Config,
			Input:      inputJSON,
			PrepResult: prepJSON,
			ExecResult: execJSON,
		}

		// Marshal request
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal request: %w", err)
		}

		// Call plugin
		respJSON, err := b.plugin.Call(ctx, "post", reqJSON)
		if err != nil {
			return nil, "", fmt.Errorf("plugin post failed: %w", err)
		}

		// Unmarshal response
		var resp plugins.Response
		if err := json.Unmarshal(respJSON, &resp); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if !resp.Success {
			return nil, "", fmt.Errorf("plugin post error: %s", resp.Error)
		}

		// Unmarshal output
		var output any
		if len(resp.Output) > 0 {
			if err := json.Unmarshal(resp.Output, &output); err != nil {
				return nil, "", fmt.Errorf("failed to unmarshal output: %w", err)
			}
		}

		// Default to "done" if no next specified
		next := resp.Next
		if next == "" {
			next = "done"
		}

		return output, next, nil
	}
}

// convertExamples converts plugin examples to nodes examples.
func convertExamples(examples []plugins.Example) []nodes.Example {
	result := make([]nodes.Example, len(examples))
	for i, ex := range examples {
		result[i] = nodes.Example{
			Name:        ex.Name,
			Description: ex.Description,
			Config:      ex.Config,
		}
	}
	return result
}
