package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Global flags.
	verbose bool
	output  string
	noColor bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "pocket",
	Short: "A minimalist LLM decision graph framework",
	Long: `Pocket is a minimalist LLM decision graph framework for Go.

Build composable workflows with powerful decision graphs, type safety,
and clean separation of concerns.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate),
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&output, "output", "text", "Output format (text, json, yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
