package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// pluginsCmd represents the plugins command.
var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage WebAssembly plugins",
	Long: `Manage WebAssembly plugins for Pocket.

Plugins extend Pocket's functionality by providing custom nodes that can be
used in workflows. They are WebAssembly modules that implement the Pocket
plugin interface.`,
	Example: `  # List all installed plugins
  pocket plugins list

  # Install a plugin
  pocket plugins install my-plugin.wasm

  # Get plugin information
  pocket plugins info my-plugin

  # Remove a plugin
  pocket plugins remove my-plugin`,
}

// pluginsListCmd represents the plugins list command.
var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long:  `List all installed WebAssembly plugins.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listPlugins(verbose)
	},
}

// pluginsInstallCmd represents the plugins install command.
var pluginsInstallCmd = &cobra.Command{
	Use:   "install <plugin.wasm>",
	Short: "Install a plugin",
	Long: `Install a WebAssembly plugin.

The plugin file will be copied to the plugins directory and made available
for use in workflows.`,
	Example: `  # Install a local plugin
  pocket plugins install ./my-plugin.wasm

  # Install with a custom name
  pocket plugins install ./plugin.wasm --name custom-name`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginPath := args[0]
		name, _ := cmd.Flags().GetString("name")
		return installPlugin(pluginPath, name, verbose)
	},
}

// pluginsInfoCmd represents the plugins info command.
var pluginsInfoCmd = &cobra.Command{
	Use:   "info <plugin-name>",
	Short: "Show plugin information",
	Long:  `Display detailed information about an installed plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		return showPluginInfo(pluginName, verbose)
	},
}

// pluginsRemoveCmd represents the plugins remove command.
var pluginsRemoveCmd = &cobra.Command{
	Use:     "remove <plugin-name>",
	Aliases: []string{"rm", "uninstall"},
	Short:   "Remove a plugin",
	Long:    `Remove an installed WebAssembly plugin.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		force, _ := cmd.Flags().GetBool("force")
		return removePlugin(pluginName, force, verbose)
	},
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsInstallCmd)
	pluginsCmd.AddCommand(pluginsInfoCmd)
	pluginsCmd.AddCommand(pluginsRemoveCmd)

	// Install command flags
	pluginsInstallCmd.Flags().String("name", "", "Custom name for the plugin")

	// Remove command flags
	pluginsRemoveCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
}

// getPluginsDir returns the plugins directory path.
func getPluginsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pocket", "plugins"), nil
}

// listPlugins lists all installed plugins.
func listPlugins(verbose bool) error {
	pluginsDir, err := getPluginsDir()
	if err != nil {
		return fmt.Errorf("failed to get plugins directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Read directory
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	// Filter .wasm files
	var plugins []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".wasm") {
			plugins = append(plugins, entry)
		}
	}

	if len(plugins) == 0 {
		fmt.Println("No plugins installed.")
		fmt.Println("\nInstall plugins with: pocket plugins install <plugin.wasm>")
		return nil
	}

	fmt.Printf("Installed plugins (%d):\n\n", len(plugins))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tSIZE\tMODIFIED\n")
	fmt.Fprintf(w, "----\t----\t--------\n")

	for _, plugin := range plugins {
		info, err := plugin.Info()
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(plugin.Name(), ".wasm")
		size := formatSize(info.Size())
		modified := info.ModTime().Format("2006-01-02 15:04")

		fmt.Fprintf(w, "%s\t%s\t%s\n", name, size, modified)
	}

	w.Flush()
	return nil
}

// installPlugin installs a WebAssembly plugin.
func installPlugin(pluginPath, customName string, verbose bool) error {
	// Expand path
	expandedPath, err := expandPath(pluginPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(expandedPath); err != nil {
		return fmt.Errorf("plugin file not found: %w", err)
	}

	// Get plugins directory
	pluginsDir, err := getPluginsDir()
	if err != nil {
		return fmt.Errorf("failed to get plugins directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Determine target name
	targetName := customName
	if targetName == "" {
		targetName = filepath.Base(expandedPath)
	}
	if !strings.HasSuffix(targetName, ".wasm") {
		targetName += ".wasm"
	}

	targetPath := filepath.Join(pluginsDir, targetName)

	// Check if already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("plugin already exists: %s", targetName)
	}

	// Read source file
	data, err := os.ReadFile(expandedPath) //nolint:gosec // User-provided plugin file
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %w", err)
	}

	// Write to target
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	fmt.Printf("✅ Installed plugin: %s\n", strings.TrimSuffix(targetName, ".wasm"))
	return nil
}

// showPluginInfo displays information about a plugin.
func showPluginInfo(pluginName string, verbose bool) error {
	pluginsDir, err := getPluginsDir()
	if err != nil {
		return fmt.Errorf("failed to get plugins directory: %w", err)
	}

	// Add .wasm extension if not present
	if !strings.HasSuffix(pluginName, ".wasm") {
		pluginName += ".wasm"
	}

	pluginPath := filepath.Join(pluginsDir, pluginName)

	// Check if exists
	info, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin not found: %s", strings.TrimSuffix(pluginName, ".wasm"))
	}

	// Display info
	fmt.Printf("Plugin: %s\n", strings.TrimSuffix(pluginName, ".wasm"))
	fmt.Printf("Path: %s\n", pluginPath)
	fmt.Printf("Size: %s\n", formatSize(info.Size()))
	fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	// TODO: In the future, we could:
	// - Load the WASM module and extract metadata
	// - Show exported functions
	// - Validate the plugin interface

	return nil
}

// removePlugin removes an installed plugin.
func removePlugin(pluginName string, force bool, verbose bool) error {
	pluginsDir, err := getPluginsDir()
	if err != nil {
		return fmt.Errorf("failed to get plugins directory: %w", err)
	}

	// Add .wasm extension if not present
	if !strings.HasSuffix(pluginName, ".wasm") {
		pluginName += ".wasm"
	}

	pluginPath := filepath.Join(pluginsDir, pluginName)

	// Check if exists
	if _, err := os.Stat(pluginPath); err != nil {
		return fmt.Errorf("plugin not found: %s", strings.TrimSuffix(pluginName, ".wasm"))
	}

	// Confirm removal if not forced
	if !force {
		fmt.Printf("Remove plugin '%s'? [y/N]: ", strings.TrimSuffix(pluginName, ".wasm"))
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove the file
	if err := os.Remove(pluginPath); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	fmt.Printf("✅ Removed plugin: %s\n", strings.TrimSuffix(pluginName, ".wasm"))
	return nil
}

// formatSize formats a file size in human-readable format.
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
