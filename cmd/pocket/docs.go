package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/agentstation/pocket/builtin"
	"github.com/agentstation/pocket/yaml"
	"github.com/spf13/cobra"
)

// docsCmd represents the docs command.
var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation",
	Long: `Generate comprehensive documentation for Pocket nodes.

The documentation includes descriptions, schemas, and examples for each node.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := &DocsConfig{
			Format: output,
		}
		return runGenerateDocs(config)
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}

// DocsConfig holds configuration for the docs command.
type DocsConfig struct {
	Format   string // "markdown", "json"
	Output   string // Output file path (empty for stdout)
	Category string // Filter by category
}

// runGenerateDocs generates documentation from node metadata.
func runGenerateDocs(config *DocsConfig) error {
	// Create a loader and registry to get all nodes
	loader := yaml.NewLoader()
	builtin.RegisterAll(loader, false)

	// Get all registered nodes
	nodes := getBuiltinNodes()

	// Filter by category if specified
	if config.Category != "" {
		filtered := []builtin.NodeMetadata{}
		for _, node := range nodes {
			if node.Category == config.Category {
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
		return generateJSONDocs(nodes, config.Output)
	default:
		return generateMarkdownDocs(nodes, config.Output)
	}
}

// generateMarkdownDocs generates Markdown documentation.
//
//nolint:gocyclo // Complex due to comprehensive documentation generation with multiple sections
func generateMarkdownDocs(nodes []builtin.NodeMetadata, output string) error {
	var sb strings.Builder

	sb.WriteString("# Pocket Node Reference\n\n")
	sb.WriteString("This document provides a comprehensive reference for all built-in nodes in the Pocket framework.\n\n")
	sb.WriteString("## Table of Contents\n\n")

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

	// Generate TOC
	for _, cat := range categoryNames {
		sb.WriteString(fmt.Sprintf("- [%s Nodes](#%s-nodes)\n", strings.ToUpper(cat[:1])+cat[1:], cat))
		for _, node := range categories[cat] {
			sb.WriteString(fmt.Sprintf("  - [%s](#%s)\n", node.Type, node.Type))
		}
	}

	sb.WriteString("\n---\n\n")

	// Generate detailed documentation for each category
	for _, cat := range categoryNames {
		sb.WriteString(fmt.Sprintf("## %s Nodes\n\n", strings.ToUpper(cat[:1])+cat[1:]))

		for _, node := range categories[cat] {
			sb.WriteString(fmt.Sprintf("### %s\n\n", node.Type))
			sb.WriteString(fmt.Sprintf("%s\n\n", node.Description))

			if node.Since != "" {
				sb.WriteString(fmt.Sprintf("**Since:** %s\n\n", node.Since))
			}

			// Configuration
			if len(node.ConfigSchema) > 0 {
				sb.WriteString("#### Configuration\n\n")
				sb.WriteString("```json\n")
				schemaJSON, _ := json.MarshalIndent(node.ConfigSchema, "", "  ")
				sb.WriteString(string(schemaJSON))
				sb.WriteString("\n```\n\n")

				// Extract properties
				if props, ok := node.ConfigSchema["properties"].(map[string]interface{}); ok {
					sb.WriteString("**Properties:**\n\n")

					// Get sorted property names
					var propNames []string
					for name := range props {
						propNames = append(propNames, name)
					}
					sort.Strings(propNames)

					for _, name := range propNames {
						prop := props[name].(map[string]interface{})
						desc := ""
						if d, ok := prop["description"].(string); ok {
							desc = d
						}

						required := false
						if reqList, ok := node.ConfigSchema["required"].([]string); ok {
							for _, req := range reqList {
								if req == name {
									required = true
									break
								}
							}
						}

						sb.WriteString(fmt.Sprintf("- **%s**", name))
						if required {
							sb.WriteString(" *(required)*")
						}
						sb.WriteString(fmt.Sprintf(": %s\n", desc))

						if t, ok := prop["type"].(string); ok {
							sb.WriteString(fmt.Sprintf("  - Type: `%s`\n", t))
						}
						if def, ok := prop["default"]; ok {
							sb.WriteString(fmt.Sprintf("  - Default: `%v`\n", def))
						}
						if enum, ok := prop["enum"].([]interface{}); ok {
							values := []string{}
							for _, v := range enum {
								values = append(values, fmt.Sprintf("`%v`", v))
							}
							sb.WriteString(fmt.Sprintf("  - Allowed values: %s\n", strings.Join(values, ", ")))
						}
					}
					sb.WriteString("\n")
				}
			}

			// Input Schema
			if len(node.InputSchema) > 0 {
				sb.WriteString("#### Input Schema\n\n")
				sb.WriteString("```json\n")
				schemaJSON, _ := json.MarshalIndent(node.InputSchema, "", "  ")
				sb.WriteString(string(schemaJSON))
				sb.WriteString("\n```\n\n")
			}

			// Output Schema
			if len(node.OutputSchema) > 0 {
				sb.WriteString("#### Output Schema\n\n")
				sb.WriteString("```json\n")
				schemaJSON, _ := json.MarshalIndent(node.OutputSchema, "", "  ")
				sb.WriteString(string(schemaJSON))
				sb.WriteString("\n```\n\n")
			}

			// Examples
			if len(node.Examples) > 0 {
				sb.WriteString("#### Examples\n\n")
				for i, example := range node.Examples {
					sb.WriteString(fmt.Sprintf("**Example %d: %s**\n\n", i+1, example.Name))
					if example.Description != "" {
						sb.WriteString(fmt.Sprintf("%s\n\n", example.Description))
					}

					sb.WriteString("```yaml\n")
					sb.WriteString(fmt.Sprintf("type: %s\n", node.Type))
					sb.WriteString("config:\n")

					// Convert config to YAML-like format
					for k, v := range example.Config {
						writeYAMLValue(&sb, k, v, "  ")
					}
					sb.WriteString("```\n\n")

					if example.Input != nil {
						sb.WriteString("Input:\n```json\n")
						inputJSON, _ := json.MarshalIndent(example.Input, "", "  ")
						sb.WriteString(string(inputJSON))
						sb.WriteString("\n```\n\n")
					}

					if example.Output != nil {
						sb.WriteString("Output:\n```json\n")
						outputJSON, _ := json.MarshalIndent(example.Output, "", "  ")
						sb.WriteString(string(outputJSON))
						sb.WriteString("\n```\n\n")
					}
				}
			}

			sb.WriteString("---\n\n")
		}
	}

	// Write output
	if output == "" {
		fmt.Print(sb.String())
	} else {
		// TODO: Write to file
		return fmt.Errorf("file output not implemented yet")
	}

	return nil
}

// writeYAMLValue writes a value in YAML format.
func writeYAMLValue(sb *strings.Builder, key string, value interface{}, indent string) {
	switch v := value.(type) {
	case string:
		fmt.Fprintf(sb, "%s%s: %s\n", indent, key, v)
	case []interface{}:
		fmt.Fprintf(sb, "%s%s:\n", indent, key)
		for _, item := range v {
			fmt.Fprintf(sb, "%s  - %v\n", indent, item)
		}
	case map[string]interface{}:
		fmt.Fprintf(sb, "%s%s:\n", indent, key)
		for k, val := range v {
			writeYAMLValue(sb, k, val, indent+"  ")
		}
	default:
		fmt.Fprintf(sb, "%s%s: %v\n", indent, key, value)
	}
}

// generateJSONDocs generates JSON documentation.
func generateJSONDocs(nodes []builtin.NodeMetadata, output string) error {
	// Create documentation structure
	doc := map[string]interface{}{
		"title":       "Pocket Node Reference",
		"description": "Comprehensive reference for all built-in nodes in the Pocket framework",
		"version":     "1.0.0",
		"nodes":       nodes,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	// Write output
	if output == "" {
		fmt.Println(string(data))
	} else {
		// TODO: Write to file
		return fmt.Errorf("file output not implemented yet")
	}

	return nil
}