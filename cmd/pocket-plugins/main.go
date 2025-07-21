package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/agentstation/pocket/plugin"
	"github.com/agentstation/pocket/plugin/loader"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "pocket-plugins",
	Short: "Manage Pocket plugins",
	Long: `Pocket plugin management tool

Discover, install, and manage WebAssembly plugins for the Pocket workflow engine.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(runCmd)
}

// List command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long:  `List all installed plugins in the default plugin directories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		l := loader.New()

		// Get default plugin paths
		paths := loader.DefaultPluginPaths()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tVERSION\tRUNTIME\tNODES\tPATH")
		_, _ = fmt.Fprintln(w, "----\t-------\t-------\t-----\t----")

		found := false
		for _, path := range paths {
			plugins, err := l.Discover(path)
			if err != nil {
				continue
			}

			for _, p := range plugins {
				found = true
				nodeCount := len(p.Nodes)
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					p.Name,
					p.Version,
					p.Runtime,
					nodeCount,
					filepath.Dir(p.Binary),
				)
			}
		}

		_ = w.Flush()

		if !found {
			fmt.Println("\nNo plugins found. Install plugins with: pocket-plugins install <path>")
		}

		return nil
	},
}

// Install command.
var installCmd = &cobra.Command{
	Use:   "install <path>",
	Short: "Install a plugin",
	Long: `Install a plugin from a directory or archive.

The plugin directory should contain:
- manifest.yaml or manifest.json
- plugin.wasm (or the binary specified in manifest)

Examples:
  pocket-plugins install ./my-plugin
  pocket-plugins install ./my-plugin.tar.gz`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourcePath := args[0]

		// Check if path exists
		info, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("cannot access '%s': %w", sourcePath, err)
		}

		// Determine installation directory
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		pluginsDir := filepath.Join(home, ".pocket", "plugins")
		if err := os.MkdirAll(pluginsDir, 0o750); err != nil {
			return fmt.Errorf("cannot create plugins directory: %w", err)
		}

		// If it's a directory, validate and copy
		if info.IsDir() {
			// Load and validate the plugin
			ctx := context.Background()
			l := loader.New()

			plugin, err := l.Load(ctx, sourcePath)
			if err != nil {
				return fmt.Errorf("failed to load plugin: %w", err)
			}

			metadata := plugin.Metadata()
			targetDir := filepath.Join(pluginsDir, metadata.Name)

			// Check if already installed
			if _, err := os.Stat(targetDir); err == nil {
				return fmt.Errorf("plugin '%s' is already installed", metadata.Name)
			}

			// Copy plugin directory
			if err := copyDir(sourcePath, targetDir); err != nil {
				return fmt.Errorf("failed to install plugin: %w", err)
			}

			fmt.Printf("✓ Installed plugin '%s' version %s\n", metadata.Name, metadata.Version)
			fmt.Printf("  Location: %s\n", targetDir)
			fmt.Printf("  Nodes: %d\n", len(metadata.Nodes))

			for _, node := range metadata.Nodes {
				fmt.Printf("    - %s (%s): %s\n", node.Type, node.Category, node.Description)
			}
		} else {
			return fmt.Errorf("archive installation not yet implemented")
		}

		return nil
	},
}

// Remove command.
var removeCmd = &cobra.Command{
	Use:   "remove <plugin-name>",
	Short: "Remove an installed plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		pluginPath := filepath.Join(home, ".pocket", "plugins", pluginName)

		// Check if plugin exists
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			return fmt.Errorf("plugin '%s' is not installed", pluginName)
		}

		// Confirm removal
		fmt.Printf("Are you sure you want to remove plugin '%s'? [y/N]: ", pluginName)
		var response string
		_, _ = fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			fmt.Println("Removal cancelled.")
			return nil
		}

		// Remove plugin directory
		if err := os.RemoveAll(pluginPath); err != nil {
			return fmt.Errorf("failed to remove plugin: %w", err)
		}

		fmt.Printf("✓ Removed plugin '%s'\n", pluginName)
		return nil
	},
}

// Info command.
var infoCmd = &cobra.Command{
	Use:   "info <plugin-name>",
	Short: "Show detailed information about a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		ctx := context.Background()
		l := loader.New()

		// Search for plugin
		paths := loader.DefaultPluginPaths()

		var metadata plugin.Metadata
		found := false

		for _, path := range paths {
			pluginPath := filepath.Join(path, pluginName)
			if p, err := l.Load(ctx, pluginPath); err == nil {
				metadata = p.Metadata()
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("plugin '%s' not found", pluginName)
		}

		fmt.Printf("Plugin: %s\n", metadata.Name)
		fmt.Printf("Version: %s\n", metadata.Version)
		fmt.Printf("Description: %s\n", metadata.Description)
		fmt.Printf("Author: %s\n", metadata.Author)
		fmt.Printf("License: %s\n", metadata.License)
		fmt.Printf("Runtime: %s\n", metadata.Runtime)
		fmt.Printf("\nPermissions:\n")
		fmt.Printf("  Memory: %s\n", metadata.Permissions.Memory)
		fmt.Printf("  Timeout: %v\n", metadata.Permissions.Timeout)

		if len(metadata.Permissions.Env) > 0 {
			fmt.Printf("  Environment: %v\n", metadata.Permissions.Env)
		}

		if len(metadata.Permissions.Filesystem) > 0 {
			fmt.Printf("  Filesystem: %v\n", metadata.Permissions.Filesystem)
		}

		fmt.Printf("\nRequirements:\n")
		fmt.Printf("  Pocket: %s\n", metadata.Requirements.Pocket)

		if metadata.Requirements.Memory != "" {
			fmt.Printf("  Memory: %s\n", metadata.Requirements.Memory)
		}

		fmt.Printf("\nNodes (%d):\n", len(metadata.Nodes))
		for _, node := range metadata.Nodes {
			fmt.Printf("\n  %s:\n", node.Type)
			fmt.Printf("    Category: %s\n", node.Category)
			fmt.Printf("    Description: %s\n", node.Description)
		}

		return nil
	},
}

// Validate command.
var validateCmd = &cobra.Command{
	Use:   "validate <path>",
	Short: "Validate a plugin without installing",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginPath := args[0]
		ctx := context.Background()
		l := loader.New()

		fmt.Printf("Validating plugin at %s...\n\n", pluginPath)

		// Try to load the plugin
		plugin, err := l.Load(ctx, pluginPath)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		metadata := plugin.Metadata()

		// Basic validation checks
		fmt.Println("✓ Manifest is valid")
		fmt.Printf("✓ Plugin name: %s\n", metadata.Name)
		fmt.Printf("✓ Version: %s\n", metadata.Version)
		fmt.Printf("✓ Runtime: %s\n", metadata.Runtime)
		fmt.Printf("✓ Binary found: %s\n", metadata.Binary)
		fmt.Printf("✓ Nodes defined: %d\n", len(metadata.Nodes))

		// Test plugin call (metadata)
		metadataJSON, err := plugin.Call(ctx, "metadata", nil)
		if err != nil {
			fmt.Printf("⚠ Warning: Plugin metadata call failed: %v\n", err)
		} else {
			fmt.Printf("✓ Plugin responds to calls (%d bytes)\n", len(metadataJSON))
		}

		fmt.Println("\nValidation passed!")
		return nil
	},
}

// Run command (for testing).
var runCmd = &cobra.Command{
	Use:   "run <plugin-name> <node-type> <function>",
	Short: "Run a plugin node function (for testing)",
	Long: `Run a specific function of a plugin node for testing.

Functions:
  prep - Run the preparation phase
  exec - Run the execution phase
  post - Run the post-processing phase

Example:
  pocket-plugins run sentiment-analyzer sentiment exec`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		nodeType := args[1]
		function := args[2]

		fmt.Printf("Running %s.%s.%s()...\n", pluginName, nodeType, function)
		fmt.Println("(This command is for testing only)")

		// This is a placeholder for actual implementation
		return fmt.Errorf("not yet implemented")
	},
}

// Helper function to copy directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path) //nolint:gosec // Path is validated through filepath.Walk
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
