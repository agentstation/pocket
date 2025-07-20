package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Version information set by ldflags.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
	goVersion = "unknown"
)

//nolint:gocyclo // Main function handles multiple commands and their arguments
func main() {
	// Define flags
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		verbose     = flag.Bool("verbose", false, "Enable verbose logging")
		vShort      = flag.Bool("v", false, "Enable verbose logging (short)")
		dryRun      = flag.Bool("dry-run", false, "Validate workflow without executing")
		storeType   = flag.String("store-type", "memory", "Store type: memory or bounded")
		maxEntries  = flag.Int("max-entries", 10000, "Max entries for bounded store")
		ttl         = flag.Duration("ttl", 0, "TTL for store entries (0 = no expiration)")
	)

	// Custom usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pocket - A minimalist LLM decision graph framework\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  pocket [flags] <command> [arguments]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  run <file.yaml>    Execute a workflow from a YAML file\n")
		fmt.Fprintf(os.Stderr, "  nodes              List available node types\n")
		fmt.Fprintf(os.Stderr, "  nodes info <type>  Show detailed info about a node type\n")
		fmt.Fprintf(os.Stderr, "  nodes docs         Generate node documentation\n")
		fmt.Fprintf(os.Stderr, "  version            Show version information\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  pocket run workflow.yaml\n")
		fmt.Fprintf(os.Stderr, "  pocket run workflow.yaml --verbose\n")
		fmt.Fprintf(os.Stderr, "  pocket run workflow.yaml --dry-run\n")
		fmt.Fprintf(os.Stderr, "  pocket run workflow.yaml --store-type bounded --max-entries 1000\n")
		fmt.Fprintf(os.Stderr, "  pocket nodes\n")
		fmt.Fprintf(os.Stderr, "  pocket nodes info echo\n")
		fmt.Fprintf(os.Stderr, "  pocket nodes docs\n")
		fmt.Fprintf(os.Stderr, "  pocket version\n")
	}

	flag.Parse()

	// Handle verbose flag
	isVerbose := *verbose || *vShort

	// Handle version flag
	if *showVersion {
		printVersion()
		return
	}

	// Get command and args
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "version":
		printVersion()
	case "run":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: run command requires a YAML file path\n")
			fmt.Fprintf(os.Stderr, "Usage: pocket run <file.yaml>\n")
			os.Exit(1)
		}

		config := &RunConfig{
			FilePath:   args[1],
			Verbose:    isVerbose,
			DryRun:     *dryRun,
			StoreType:  *storeType,
			MaxEntries: *maxEntries,
			TTL:        *ttl,
		}

		if err := runWorkflow(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "nodes":
		// Handle nodes subcommands
		var subCommand string
		if len(args) > 1 {
			subCommand = args[1]
		}

		switch subCommand {
		case "info":
			if len(args) < 3 {
				fmt.Fprintf(os.Stderr, "Error: nodes info requires a node type\n")
				fmt.Fprintf(os.Stderr, "Usage: pocket nodes info <type>\n")
				os.Exit(1)
			}
			if err := runNodesInfo(args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		case "docs":
			// Generate documentation
			config := &DocsConfig{
				Format: "markdown",
			}
			if err := runGenerateDocs(config); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		default:
			// List all nodes
			config := &NodesConfig{
				Format: "table",
			}
			if err := runNodesList(config); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("pocket version %s\n", version)
	if version != "dev" {
		fmt.Printf("  commit:     %s\n", commit)
		fmt.Printf("  built:      %s\n", buildDate)
		fmt.Printf("  go version: %s\n", goVersion)
	}
}

// expandPath expands ~ to home directory.
func expandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
