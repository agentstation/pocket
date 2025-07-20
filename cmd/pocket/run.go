package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/builtin"
	"github.com/agentstation/pocket/yaml"
	goyaml "github.com/goccy/go-yaml"
)

// RunConfig holds configuration for the run command.
type RunConfig struct {
	FilePath   string
	Verbose    bool
	DryRun     bool
	StoreType  string
	MaxEntries int
	TTL        time.Duration
}

// runWorkflow executes a workflow from a YAML file.
//
//nolint:gocyclo // Complex due to workflow parsing, validation, and execution handling
func runWorkflow(config *RunConfig) error {
	// Expand path (handle ~)
	filePath, err := expandPath(config.FilePath)
	if err != nil {
		return fmt.Errorf("expand path: %w", err)
	}

	// Make path absolute
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	if config.Verbose {
		log.Printf("Loading workflow from: %s", absPath)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", config.FilePath)
		}
		return fmt.Errorf("access file: %w", err)
	}

	// Read the YAML file
	data, err := os.ReadFile(absPath) // #nosec G304 - User-provided workflow file
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Parse YAML into GraphDefinition
	var graphDef yaml.GraphDefinition
	if err := goyaml.Unmarshal(data, &graphDef); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	// Validate the graph definition
	if err := graphDef.Validate(); err != nil {
		return fmt.Errorf("invalid workflow: %w", err)
	}

	if config.Verbose {
		log.Printf("Loaded workflow: %s", graphDef.Name)
		if graphDef.Description != "" {
			log.Printf("Description: %s", graphDef.Description)
		}
		log.Printf("Nodes: %d", len(graphDef.Nodes))
		log.Printf("Connections: %d", len(graphDef.Connections))
	}

	// If dry run, stop here
	if config.DryRun {
		fmt.Println("Workflow validation successful (dry run)")
		return nil
	}

	// Create store based on configuration
	var store pocket.Store
	switch config.StoreType {
	case "memory":
		store = pocket.NewStore()
		if config.Verbose {
			log.Println("Using in-memory store")
		}
	case "bounded":
		opts := []pocket.StoreOption{
			pocket.WithMaxEntries(config.MaxEntries),
		}
		if config.TTL > 0 {
			opts = append(opts, pocket.WithTTL(config.TTL))
		}
		if config.Verbose {
			opts = append(opts, pocket.WithEvictionCallback(func(key string, value any) {
				log.Printf("Evicted: %s", key)
			}))
		}
		store = pocket.NewStore(opts...)
		if config.Verbose {
			log.Printf("Using bounded store (max entries: %d, TTL: %v)", config.MaxEntries, config.TTL)
		}
	default:
		return fmt.Errorf("unknown store type: %s", config.StoreType)
	}

	// Create a loader and register built-in nodes
	loader := yaml.NewLoader()
	builtin.RegisterAll(loader, config.Verbose)

	// Load the graph
	graph, err := loader.LoadDefinition(&graphDef, store)
	if err != nil {
		return fmt.Errorf("load workflow: %w", err)
	}

	if config.Verbose {
		log.Println("Starting workflow execution...")
	}

	// Create context
	ctx := context.Background()

	// TODO: In the future, we could accept input from:
	// - Command line args
	// - JSON file
	// - stdin
	// For now, we'll use nil input
	var input interface{}

	// Run the workflow
	start := time.Now()
	result, err := graph.Run(ctx, input)
	duration := time.Since(start)

	if err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	if config.Verbose {
		log.Printf("Workflow completed in %v", duration)
	}

	// Output the result
	if result != nil {
		// Try to format as YAML for nice output
		output, err := goyaml.Marshal(result)
		if err != nil {
			// Fallback to simple print
			fmt.Println(result)
		} else {
			fmt.Println(string(output))
		}
	}

	return nil
}
