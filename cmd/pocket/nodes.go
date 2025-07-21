package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	goyaml "github.com/goccy/go-yaml"

	"github.com/agentstation/pocket/builtin"
	"github.com/agentstation/pocket/yaml"
)

// NodesConfig holds configuration for the nodes command.
type NodesConfig struct {
	Format string // "table", "json", "yaml"
	Type   string // Filter by specific node type
}

// runNodesList lists all available node types.
func runNodesList(config *NodesConfig) error {
	// Create a loader and registry to get all nodes
	loader := yaml.NewLoader()
	builtin.RegisterAll(loader, false)

	// Get all registered nodes from builtin registry
	// Note: Since the registry doesn't expose its contents, we use a hardcoded list

	// We need to expose the registered nodes. For now, let's use the metadata
	// from a hardcoded list since the registry doesn't expose its contents
	nodes := getBuiltinNodes()

	// Filter by type if specified
	if config.Type != "" {
		filtered := []builtin.NodeMetadata{}
		for _, node := range nodes {
			if node.Type == config.Type {
				filtered = append(filtered, node)
			}
		}
		nodes = filtered
	}

	// Sort by category then type
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Category != nodes[j].Category {
			return nodes[i].Category < nodes[j].Category
		}
		return nodes[i].Type < nodes[j].Type
	})

	switch config.Format {
	case "json":
		return outputJSON(nodes)
	case "yaml":
		return outputYAML(nodes)
	default:
		return outputTable(nodes)
	}
}

// runNodesInfo shows detailed information about a specific node type.
func runNodesInfo(nodeType string) error {
	nodes := getBuiltinNodes()

	for _, node := range nodes {
		if node.Type != nodeType {
			continue
		}

		// Output detailed node information
		fmt.Printf("Node Type: %s\n", node.Type)
		fmt.Printf("Category: %s\n", node.Category)
		fmt.Printf("Description: %s\n", node.Description)
		if node.Since != "" {
			fmt.Printf("Since: %s\n", node.Since)
		}
		fmt.Println()

		// Config schema
		if len(node.ConfigSchema) > 0 {
			fmt.Println("Configuration:")
			schemaJSON, _ := json.MarshalIndent(node.ConfigSchema, "  ", "  ")
			fmt.Printf("  %s\n", schemaJSON)
			fmt.Println()
		}

		// Examples
		if len(node.Examples) > 0 {
			fmt.Println("Examples:")
			for i, example := range node.Examples {
				fmt.Printf("  %d. %s\n", i+1, example.Name)
				if example.Description != "" {
					fmt.Printf("     %s\n", example.Description)
				}
				if len(example.Config) > 0 {
					configYAML, _ := goyaml.Marshal(example.Config)
					fmt.Printf("     Config:\n")
					for _, line := range strings.Split(string(configYAML), "\n") {
						if line != "" {
							fmt.Printf("       %s\n", line)
						}
					}
				}
			}
		}

		return nil
	}

	return fmt.Errorf("node type '%s' not found", nodeType)
}

// getBuiltinNodes returns metadata for all builtin nodes.
func getBuiltinNodes() []builtin.NodeMetadata {
	// Create instances to get metadata
	return []builtin.NodeMetadata{
		(&builtin.EchoNodeBuilder{}).Metadata(),
		(&builtin.DelayNodeBuilder{}).Metadata(),
		(&builtin.RouterNodeBuilder{}).Metadata(),
		(&builtin.ConditionalNodeBuilder{}).Metadata(),
		(&builtin.TransformNodeBuilder{}).Metadata(),
		(&builtin.TemplateNodeBuilder{}).Metadata(),
		(&builtin.JSONPathNodeBuilder{}).Metadata(),
		(&builtin.ValidateNodeBuilder{}).Metadata(),
		(&builtin.AggregateNodeBuilder{}).Metadata(),
		(&builtin.HTTPNodeBuilder{}).Metadata(),
		(&builtin.FileNodeBuilder{}).Metadata(),
		(&builtin.ExecNodeBuilder{}).Metadata(),
		(&builtin.ParallelNodeBuilder{}).Metadata(),
		(&builtin.LuaNodeBuilder{}).Metadata(),
	}
}

// outputTable outputs nodes in table format.
func outputTable(nodes []builtin.NodeMetadata) error {
	// Group by category
	categories := make(map[string][]builtin.NodeMetadata)
	for _, node := range nodes {
		categories[node.Category] = append(categories[node.Category], node)
	}

	// Get sorted category names
	categoryNames := make([]string, 0, len(categories))
	for cat := range categories {
		categoryNames = append(categoryNames, cat)
	}
	sort.Strings(categoryNames)

	// Output each category
	for _, cat := range categoryNames {
		fmt.Printf("\n%s:\n", strings.ToUpper(cat[:1])+cat[1:])
		fmt.Println(strings.Repeat("-", len(cat)+1))

		for _, node := range categories[cat] {
			fmt.Printf("  %-20s %s\n", node.Type, node.Description)
		}
	}

	fmt.Printf("\nTotal: %d node types\n", len(nodes))
	fmt.Println("\nUse 'pocket nodes info <type>' for detailed information about a specific node.")

	return nil
}

// outputJSON outputs nodes in JSON format.
func outputJSON(nodes []builtin.NodeMetadata) error {
	data, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// outputYAML outputs nodes in YAML format.
func outputYAML(nodes []builtin.NodeMetadata) error {
	// Convert to YAML-friendly format
	output := make([]map[string]interface{}, len(nodes))
	for i, node := range nodes {
		output[i] = map[string]interface{}{
			"type":        node.Type,
			"category":    node.Category,
			"description": node.Description,
		}
		if node.Since != "" {
			output[i]["since"] = node.Since
		}
		if len(node.Examples) > 0 {
			output[i]["examples"] = len(node.Examples)
		}
	}

	yamlData, err := goyaml.Marshal(output)
	if err != nil {
		return err
	}

	fmt.Print(string(yamlData))
	return nil
}
