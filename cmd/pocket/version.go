package main

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information about the Pocket CLI.`,
	Example: `  # Show version
  pocket version

  # Show version in JSON format
  pocket version --output json

  # Show version in YAML format
  pocket version --output yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		versionInfo := map[string]string{
			"version":   version,
			"commit":    commit,
			"buildDate": buildDate,
			"goVersion": goVersion,
		}

		switch output {
		case jsonFormat:
			data, err := json.MarshalIndent(versionInfo, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal version info: %w", err)
			}
			fmt.Println(string(data))

		case yamlFormat:
			data, err := yaml.Marshal(versionInfo)
			if err != nil {
				return fmt.Errorf("failed to marshal version info: %w", err)
			}
			fmt.Print(string(data))

		default: // text
			fmt.Printf("pocket version %s\n", version)
			if version != "dev" {
				fmt.Printf("  commit:     %s\n", commit)
				fmt.Printf("  built:      %s\n", buildDate)
				fmt.Printf("  go version: %s\n", goVersion)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
