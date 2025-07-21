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
)

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
