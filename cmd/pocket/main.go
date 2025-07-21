package main

import (
	"fmt"
	"os"
)

// Version information set by ldflags.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
	goVersion = "unknown"
)

func main() {
	Execute()
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}