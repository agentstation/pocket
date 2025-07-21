package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/builtin/script"
	"github.com/spf13/cobra"
)

// scriptsCmd represents the scripts command.
var scriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "Manage Lua scripts",
	Long: `Discover, validate, and manage Lua scripts.

Scripts are discovered from ~/.pocket/scripts/ and can be used as nodes
in your workflows. Each script should have metadata comments describing
its purpose and configuration.`,
	Example: `  # List all discovered scripts
  pocket scripts

  # Validate a script
  pocket scripts validate my-script.lua

  # Get script information
  pocket scripts info data-processor

  # Run a script directly
  pocket scripts run data-processor '{"input": "data"}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default action is to list scripts
		return runScriptsList(verbose)
	},
}

// scriptsValidateCmd represents the scripts validate command.
var scriptsValidateCmd = &cobra.Command{
	Use:   "validate <script-path>",
	Short: "Validate a Lua script",
	Long: `Validate a Lua script's syntax and structure without executing it.

Checks for syntax errors and verifies that required functions are present.`,
	Example: `  # Validate a script
  pocket scripts validate my-script.lua

  # Validate with verbose output
  pocket scripts validate my-script.lua --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptPath := args[0]
		return runScriptsValidate(scriptPath, verbose)
	},
}

// scriptsInfoCmd represents the scripts info command.
var scriptsInfoCmd = &cobra.Command{
	Use:   "info <script-name>",
	Short: "Show script details",
	Long: `Display detailed information about a discovered script.

Shows the script's metadata, file path, size, and validation status.`,
	Example: `  # Get script information
  pocket scripts info data-processor

  # Get info with verbose output
  pocket scripts info data-processor --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptName := args[0]
		return runScriptsInfo(scriptName, verbose)
	},
}

// scriptsRunCmd represents the scripts run command.
var scriptsRunCmd = &cobra.Command{
	Use:   "run <script-name> [input-json]",
	Short: "Run a script directly",
	Long: `Execute a discovered script directly for testing purposes.

Optionally provide input data as a JSON string. Use --verbose to see
debug output from the script.`,
	Example: `  # Run a script without input
  pocket scripts run my-script

  # Run with JSON input
  pocket scripts run data-processor '{"text": "Hello", "value": 42}'

  # Run with verbose output for debugging
  pocket scripts run data-processor --verbose`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptName := args[0]
		inputJSON := ""
		if len(args) > 1 {
			inputJSON = args[1]
		}
		return runScriptsRun(scriptName, inputJSON, verbose)
	},
}

func init() {
	rootCmd.AddCommand(scriptsCmd)
	scriptsCmd.AddCommand(scriptsValidateCmd)
	scriptsCmd.AddCommand(scriptsInfoCmd)
	scriptsCmd.AddCommand(scriptsRunCmd)
}

// runScriptsList lists all discovered scripts.
func runScriptsList(verbose bool) error {
	manager := script.NewManager("", verbose)

	if err := manager.Discover(); err != nil {
		return fmt.Errorf("failed to discover scripts: %w", err)
	}

	scripts := manager.ListScripts()
	if len(scripts) == 0 {
		fmt.Println("No scripts found in ~/.pocket/scripts")
		fmt.Println("\nCreate a script with metadata like:")
		fmt.Println("-- @name: my-script")
		fmt.Println("-- @category: data")
		fmt.Println("-- @description: My custom script")
		fmt.Println("-- @version: 1.0.0")
		fmt.Println("")
		fmt.Println("function exec(input)")
		fmt.Println("    return input")
		fmt.Println("end")
		return nil
	}

	// Group by category
	byCategory := make(map[string][]*script.Script)
	for _, s := range scripts {
		byCategory[s.Category] = append(byCategory[s.Category], s)
	}

	// Sort categories
	categories := make([]string, 0, len(byCategory))
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	fmt.Printf("\nDiscovered %d scripts:\n\n", len(scripts))

	for _, cat := range categories {
		fmt.Printf("%s:\n", cat)
		fmt.Println(strings.Repeat("-", len(cat)+1))

		// Sort scripts in category
		catScripts := byCategory[cat]
		sort.Slice(catScripts, func(i, j int) bool {
			return catScripts[i].Name < catScripts[j].Name
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, s := range catScripts {
			desc := s.Description
			if desc == "" {
				desc = "(no description)"
			}
			if s.Version != "" {
				_, _ = fmt.Fprintf(w, "  %s\t%s\t(v%s)\n", s.Name, desc, s.Version)
			} else {
				_, _ = fmt.Fprintf(w, "  %s\t%s\n", s.Name, desc)
			}
		}
		_ = w.Flush()
		fmt.Println()
	}

	return nil
}

// runScriptsValidate validates a Lua script.
func runScriptsValidate(scriptPath string, verbose bool) error {
	// Resolve path
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("script not found: %w", err)
	}

	manager := script.NewManager("", verbose)

	fmt.Printf("Validating %s...\n", scriptPath)

	if err := manager.ValidateScript(absPath); err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
		return err
	}

	fmt.Println("✅ Script is valid!")

	// Also try to load metadata
	s, err := manager.LoadScript(absPath)
	if err == nil && s.Name != "" {
		fmt.Printf("\nMetadata:\n")
		fmt.Printf("  Name: %s\n", s.Name)
		if s.Category != "" {
			fmt.Printf("  Category: %s\n", s.Category)
		}
		if s.Description != "" {
			fmt.Printf("  Description: %s\n", s.Description)
		}
		if s.Version != "" {
			fmt.Printf("  Version: %s\n", s.Version)
		}
	}

	return nil
}

// runScriptsInfo shows information about a script.
func runScriptsInfo(scriptName string, verbose bool) error {
	manager := script.NewManager("", verbose)

	if err := manager.Discover(); err != nil {
		return fmt.Errorf("failed to discover scripts: %w", err)
	}

	s, found := manager.GetScript(scriptName)
	if !found {
		return fmt.Errorf("script '%s' not found", scriptName)
	}

	fmt.Printf("Script: %s\n", s.Name)
	fmt.Printf("Path: %s\n", s.Path)
	fmt.Printf("Category: %s\n", s.Category)
	if s.Description != "" {
		fmt.Printf("Description: %s\n", s.Description)
	}
	if s.Version != "" {
		fmt.Printf("Version: %s\n", s.Version)
	}

	// Show file info
	if info, err := os.Stat(s.Path); err == nil {
		fmt.Printf("Size: %d bytes\n", info.Size())
		fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	}

	// Validate the script
	fmt.Printf("\nValidation: ")
	if err := manager.ValidateScript(s.Path); err != nil {
		fmt.Printf("❌ %v\n", err)
	} else {
		fmt.Println("✅ Valid")
	}

	return nil
}

// runScriptsRun executes a script directly.
func runScriptsRun(scriptName, inputJSON string, verbose bool) error {
	var input interface{}
	if inputJSON != "" {
		// Parse input JSON
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return fmt.Errorf("invalid input JSON: %w", err)
		}
	}

	manager := script.NewManager("", verbose)

	if err := manager.Discover(); err != nil {
		return fmt.Errorf("failed to discover scripts: %w", err)
	}

	s, found := manager.GetScript(scriptName)
	if !found {
		return fmt.Errorf("script '%s' not found", scriptName)
	}

	fmt.Printf("Running script: %s\n", s.Name)
	if verbose {
		fmt.Println("Debug output:")
		fmt.Println(strings.Repeat("-", 40))
	}

	// Create and run node
	node, err := manager.CreateNode(s)
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	// Run the node using Pocket's graph execution
	ctx := context.Background()
	store := pocket.NewStore()
	graph := pocket.NewGraph(node, store)

	result, err := graph.Run(ctx, input)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	if verbose {
		fmt.Println(strings.Repeat("-", 40))
	}

	// Output result as JSON
	fmt.Println("\nResult:")
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	fmt.Println(string(output))

	return nil
}
