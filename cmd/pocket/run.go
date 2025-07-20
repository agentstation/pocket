package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/yaml"
	goyaml "github.com/goccy/go-yaml"
)

// RunConfig holds configuration for the run command
type RunConfig struct {
	FilePath   string
	Verbose    bool
	DryRun     bool
	StoreType  string
	MaxEntries int
	TTL        time.Duration
}

// runWorkflow executes a workflow from a YAML file
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
	data, err := os.ReadFile(absPath)
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

	// Create a loader and register default node builders
	loader := yaml.NewLoader()
	registerDefaultBuilders(loader, config.Verbose)

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

// registerDefaultBuilders registers built-in node types
func registerDefaultBuilders(loader *yaml.Loader, verbose bool) {
	// Register a simple echo node type
	loader.RegisterNodeType("echo", func(def *yaml.NodeDefinition) (pocket.Node, error) {
		message := "Hello from echo node"
		if msgInterface, ok := def.Config["message"]; ok {
			if msg, ok := msgInterface.(string); ok {
				message = msg
			}
		}

		return pocket.NewNode[any, any](def.Name,
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				if verbose {
					log.Printf("[%s] Echo: %s", def.Name, message)
				}
				return map[string]interface{}{
					"message": message,
					"input":   input,
					"node":    def.Name,
				}, nil
			}),
		), nil
	})

	// Register a delay node type
	loader.RegisterNodeType("delay", func(def *yaml.NodeDefinition) (pocket.Node, error) {
		duration := 1 * time.Second
		if durInterface, ok := def.Config["duration"]; ok {
			if durStr, ok := durInterface.(string); ok {
				if d, err := time.ParseDuration(durStr); err == nil {
					duration = d
				}
			}
		}

		return pocket.NewNode[any, any](def.Name,
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				if verbose {
					log.Printf("[%s] Delaying for %v", def.Name, duration)
				}
				select {
				case <-time.After(duration):
					return input, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}),
		), nil
	})

	// Register a transform node type
	loader.RegisterNodeType("transform", func(def *yaml.NodeDefinition) (pocket.Node, error) {
		return pocket.NewNode[any, any](def.Name,
			pocket.WithExec(func(ctx context.Context, input any) (any, error) {
				if verbose {
					log.Printf("[%s] Transforming input", def.Name)
				}
				
				// Simple transformation: wrap input in a result
				return map[string]interface{}{
					"transformed": true,
					"original":    input,
					"timestamp":   time.Now().Format(time.RFC3339),
					"node":        def.Name,
				}, nil
			}),
		), nil
	})

	// Register a router node type
	loader.RegisterNodeType("router", func(def *yaml.NodeDefinition) (pocket.Node, error) {
		defaultRoute := "default"
		if routeInterface, ok := def.Config["default_route"]; ok {
			if route, ok := routeInterface.(string); ok {
				defaultRoute = route
			}
		}

		return pocket.NewNode[any, any](def.Name,
			pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
				if verbose {
					log.Printf("[%s] Routing to: %s", def.Name, defaultRoute)
				}
				return exec, defaultRoute, nil
			}),
		), nil
	})

	// TODO: Add more built-in node types:
	// - http: Make HTTP requests
	// - conditional: Route based on conditions
	// - aggregate: Collect results
	// - parallel: Run nodes in parallel
}