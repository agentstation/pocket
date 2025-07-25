package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	"github.com/spf13/cobra"

	"github.com/agentstation/pocket/nodes"
	"github.com/agentstation/pocket/yaml"
)

// nodesCmd represents the nodes command.
var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Manage and explore available node types",
	Long: `Explore and manage Pocket node types.

List all available nodes, get detailed information about specific node types,
or generate documentation for all nodes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default action is to list nodes
		config := &NodesConfig{
			Format: output,
		}
		return runNodesList(config)
	},
}

// nodesInfoCmd represents the nodes info command.
var nodesInfoCmd = &cobra.Command{
	Use:   "info <node-type>",
	Short: "Show detailed information about a node type",
	Long: `Display detailed information about a specific node type.

Shows the node's description, configuration schema, input/output schemas,
and usage examples.`,
	Example: `  # Get info about the echo node
  pocket nodes info echo

  # Get info in JSON format
  pocket nodes info transform --output json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeType := args[0]
		return runNodesInfo(nodeType)
	},
}

// nodesDocsCmd represents the nodes docs command.
var nodesDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate node documentation",
	Long: `Generate comprehensive documentation for all available node types.

The documentation includes descriptions, schemas, and examples for each node.`,
	Example: `  # Generate markdown documentation
  pocket nodes docs

  # Generate JSON documentation
  pocket nodes docs --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := &DocsConfig{
			Format: output,
		}
		return runGenerateDocs(config)
	},
}

func init() {
	rootCmd.AddCommand(nodesCmd)
	nodesCmd.AddCommand(nodesInfoCmd)
	nodesCmd.AddCommand(nodesDocsCmd)
}

// NodesConfig holds configuration for the nodes command.
type NodesConfig struct {
	Format string // "table", "json", "yaml"
	Type   string // Filter by specific node type
}

// runNodesList lists all available node types.
func runNodesList(config *NodesConfig) error {
	// Create a loader and registry to get all nodes
	loader := yaml.NewLoader()
	nodes.RegisterAll(loader, false)

	// Get all registered nodes
	allNodes := getBuiltinNodes()

	// Filter by type if specified
	if config.Type != "" {
		filtered := []nodes.Metadata{}
		for _, node := range allNodes {
			if node.Type == config.Type {
				filtered = append(filtered, node)
			}
		}
		allNodes = filtered
	}

	// Sort by category then type
	sort.Slice(allNodes, func(i, j int) bool {
		if allNodes[i].Category != allNodes[j].Category {
			return allNodes[i].Category < allNodes[j].Category
		}
		return allNodes[i].Type < allNodes[j].Type
	})

	switch config.Format {
	case jsonFormat:
		return outputJSON(allNodes)
	case yamlFormat:
		return outputYAML(allNodes)
	default:
		return outputTable(allNodes)
	}
}

// runNodesInfo shows detailed information about a specific node type.
func runNodesInfo(nodeType string) error {
	allNodes := getBuiltinNodes()

	for _, node := range allNodes {
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

// getBuiltinNodes returns metadata for all built-in nodes.
func getBuiltinNodes() []nodes.Metadata {
	// Create instances to get metadata
	return []nodes.Metadata{
		(&nodes.EchoNodeBuilder{}).Metadata(),
		(&nodes.DelayNodeBuilder{}).Metadata(),
		(&nodes.RouterNodeBuilder{}).Metadata(),
		(&nodes.ConditionalNodeBuilder{}).Metadata(),
		(&nodes.TransformNodeBuilder{}).Metadata(),
		(&nodes.TemplateNodeBuilder{}).Metadata(),
		(&nodes.JSONPathNodeBuilder{}).Metadata(),
		(&nodes.ValidateNodeBuilder{}).Metadata(),
		(&nodes.AggregateNodeBuilder{}).Metadata(),
		(&nodes.HTTPNodeBuilder{}).Metadata(),
		(&nodes.FileNodeBuilder{}).Metadata(),
		(&nodes.ExecNodeBuilder{}).Metadata(),
		(&nodes.ParallelNodeBuilder{}).Metadata(),
		(&nodes.LuaNodeBuilder{}).Metadata(),
	}
}

// outputTable outputs nodes in table format.
func outputTable(nodeList []nodes.Metadata) error {
	// Group by category
	categories := make(map[string][]nodes.Metadata)
	for _, node := range nodeList {
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

	fmt.Printf("\nTotal: %d node types\n", len(nodeList))
	fmt.Println("\nUse 'pocket nodes info <type>' for detailed information about a specific node.")

	return nil
}

// outputJSON outputs nodes in JSON format.
func outputJSON(nodeList []nodes.Metadata) error {
	data, err := json.MarshalIndent(nodeList, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// outputYAML outputs nodes in YAML format.
func outputYAML(nodeList []nodes.Metadata) error {
	// Convert to YAML-friendly format
	output := make([]map[string]interface{}, len(nodeList))
	for i, node := range nodeList {
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
