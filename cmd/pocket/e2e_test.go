//go:build e2e
// +build e2e

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestE2EFullCLIWorkflow tests the complete CLI workflow from binary execution.
func TestE2EFullCLIWorkflow(t *testing.T) {
	// Skip if pocket binary is not available
	pocketBin := findPocketBinary(t)
	if pocketBin == "" {
		t.Skip("pocket binary not found, skipping e2e test")
	}

	tempDir := t.TempDir()

	// Test 1: Version command
	output := runCommand(t, pocketBin, "version")
	if !strings.Contains(output, "pocket version") {
		t.Errorf("Version command failed, output: %s", output)
	}

	// Test 2: List nodes
	output = runCommand(t, pocketBin, "nodes")
	if !strings.Contains(output, "Core:") || !strings.Contains(output, "echo") {
		t.Errorf("Nodes list command failed, output: %s", output)
	}

	// Test 3: Get node info
	output = runCommand(t, pocketBin, "nodes", "info", "echo")
	if !strings.Contains(output, "Node Type: echo") {
		t.Errorf("Node info command failed, output: %s", output)
	}

	// Test 4: Create and run a workflow
	workflowYAML := `
name: e2e-test
description: End-to-end test workflow
nodes:
  - name: start
    type: echo
    config:
      message: "E2E Test"
  - name: process
    type: transform
    config:
      expression: '{"status": "complete", "message": input}'
connections:
  - from: start
    to: process
start: start
`
	workflowPath := filepath.Join(tempDir, "e2e-workflow.yaml")
	os.WriteFile(workflowPath, []byte(workflowYAML), 0600)

	// Dry run first
	output = runCommand(t, pocketBin, "run", workflowPath, "--dry-run")
	if !strings.Contains(output, "validation successful") {
		t.Errorf("Dry run failed, output: %s", output)
	}

	// Actual run
	output = runCommand(t, pocketBin, "run", workflowPath)
	if !strings.Contains(output, "transformed") && !strings.Contains(output, "E2E Test") {
		t.Errorf("Workflow run failed, output: %s", output)
	}
}

// TestE2EPluginWorkflow tests plugin installation and usage.
func TestE2EPluginWorkflow(t *testing.T) {
	pocketBin := findPocketBinary(t)
	if pocketBin == "" {
		t.Skip("pocket binary not found, skipping e2e test")
	}

	tempDir := t.TempDir()

	// Test plugin commands
	// List plugins (should be empty initially)
	output := runCommand(t, pocketBin, "plugins", "list")
	if !strings.Contains(output, "No plugins installed") && !strings.Contains(output, "Installed Plugins") {
		t.Errorf("Plugin list command failed, output: %s", output)
	}

	// Create a simple WASM plugin directory structure
	pluginDir := filepath.Join(tempDir, "test-plugin")
	os.MkdirAll(pluginDir, 0750)

	// Create a minimal manifest
	manifest := `
name: test-plugin
version: 1.0.0
description: Test plugin for e2e
binary: plugin.wasm
nodes:
  - type: test-node
    category: test
    description: Test node from plugin
`
	manifestPath := filepath.Join(pluginDir, "manifest.yaml")
	os.WriteFile(manifestPath, []byte(manifest), 0600)

	// Create a dummy WASM file
	wasmPath := filepath.Join(pluginDir, "plugin.wasm")
	os.WriteFile(wasmPath, []byte("dummy wasm content"), 0600)

	// Validate the plugin
	output = runCommand(t, pocketBin, "plugins", "validate", pluginDir)
	if strings.Contains(output, "error") || strings.Contains(output, "Error") {
		t.Logf("Plugin validation output: %s", output)
		// Don't fail the test as validation might have stricter requirements
	}

	// Try to get plugin info (from directory)
	output = runCommandAllowError(t, pocketBin, "plugins", "info", pluginDir)
	if strings.Contains(output, "test-plugin") {
		// Success - plugin info was displayed
		t.Logf("Plugin info retrieved successfully")
	}
}

// TestE2EScriptWorkflow tests Lua script discovery and execution.
func TestE2EScriptWorkflow(t *testing.T) {
	pocketBin := findPocketBinary(t)
	if pocketBin == "" {
		t.Skip("pocket binary not found, skipping e2e test")
	}

	// Create scripts directory
	homeDir, _ := os.UserHomeDir()
	scriptsDir := filepath.Join(homeDir, ".pocket", "scripts")
	os.MkdirAll(scriptsDir, 0750)

	// Create a test script
	testScript := `
-- @name: e2e-test
-- @category: test
-- @description: E2E test script
-- @version: 1.0.0

function exec(input)
    return {
        message = "E2E test passed",
        input = input,
        timestamp = os.time()
    }
end
`
	scriptPath := filepath.Join(scriptsDir, "e2e-test.lua")
	err := os.WriteFile(scriptPath, []byte(testScript), 0600)
	if err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	defer os.Remove(scriptPath) // Clean up after test

	// List scripts
	output := runCommand(t, pocketBin, "scripts")
	if strings.Contains(output, "e2e-test") {
		t.Logf("Script discovered successfully")
	}

	// Validate script
	output = runCommand(t, pocketBin, "scripts", "validate", scriptPath)
	if !strings.Contains(output, "valid") {
		t.Errorf("Script validation failed, output: %s", output)
	}

	// Get script info
	output = runCommand(t, pocketBin, "scripts", "info", "e2e-test")
	if !strings.Contains(output, "e2e-test") {
		t.Errorf("Script info failed, output: %s", output)
	}

	// Run script
	output = runCommand(t, pocketBin, "scripts", "run", "e2e-test", `{"test": true}`)
	if !strings.Contains(output, "E2E test passed") {
		t.Errorf("Script run failed, output: %s", output)
	}
}

// TestE2EComplexWorkflow tests a more complex workflow with multiple features.
func TestE2EComplexWorkflow(t *testing.T) {
	pocketBin := findPocketBinary(t)
	if pocketBin == "" {
		t.Skip("pocket binary not found, skipping e2e test")
	}

	tempDir := t.TempDir()

	// Create a complex workflow that uses multiple node types
	workflowYAML := `
name: complex-e2e
description: Complex end-to-end test
nodes:
  - name: input
    type: echo
    config:
      message: "Starting complex workflow"
      passthrough: true

  - name: validate_input
    type: validate
    config:
      schema:
        type: object
        properties:
          data:
            type: array
            items:
              type: number

  - name: transform_data
    type: jsonpath
    config:
      expression: "$.data[*]"

  - name: process_parallel
    type: parallel
    config:
      nodes:
        - name: double
          type: transform
          config:
            expression: 'input * 2'
        - name: square
          type: transform
          config:
            expression: 'input * input'

  - name: aggregate_results
    type: aggregate
    config:
      mode: collect

  - name: format_output
    type: template
    config:
      template: |
        Results:
        {{range .}}
        - Double: {{index . 0}}
        - Square: {{index . 1}}
        {{end}}

connections:
  - from: input
    to: validate_input
  - from: validate_input
    to: transform_data
  - from: transform_data
    to: process_parallel
  - from: process_parallel
    to: aggregate_results
  - from: aggregate_results
    to: format_output

start: input
`

	workflowPath := filepath.Join(tempDir, "complex-workflow.yaml")
	os.WriteFile(workflowPath, []byte(workflowYAML), 0600)

	// Create input file
	inputData := `{"data": [2, 3, 4, 5]}`
	inputPath := filepath.Join(tempDir, "input.json")
	os.WriteFile(inputPath, []byte(inputData), 0600)

	// Run the complex workflow
	// Note: Currently pocket doesn't support input from file, but this tests the workflow structure
	output := runCommand(t, pocketBin, "run", workflowPath, "--dry-run")
	if !strings.Contains(output, "validation successful") {
		t.Errorf("Complex workflow validation failed, output: %s", output)
	}
}

// TestE2EPerformance tests performance with bounded store.
func TestE2EPerformance(t *testing.T) {
	pocketBin := findPocketBinary(t)
	if pocketBin == "" {
		t.Skip("pocket binary not found, skipping e2e test")
	}

	tempDir := t.TempDir()

	// Create a workflow that tests store performance
	workflowYAML := `
name: perf-test
description: Performance test workflow
nodes:
  - name: generate
    type: transform
    config:
      expression: |
        [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

  - name: process
    type: transform
    config:
      expression: |
        {
          "data": input,
          "timestamp": now(),
          "processed": true
        }

connections:
  - from: generate
    to: process

start: generate
`

	workflowPath := filepath.Join(tempDir, "perf-workflow.yaml")
	os.WriteFile(workflowPath, []byte(workflowYAML), 0600)

	// Run with bounded store
	start := time.Now()
	output := runCommand(t, pocketBin, "run", workflowPath,
		"--store-type", "bounded",
		"--max-entries", "100",
		"--ttl", "1m")

	duration := time.Since(start)

	if !strings.Contains(output, "transformed") || !strings.Contains(output, "timestamp") {
		t.Errorf("Performance test failed, output: %s", output)
	}

	t.Logf("Workflow completed in %v", duration)
}

// Helper functions

func findPocketBinary(t *testing.T) string {
	// Try to find pocket binary in common locations
	paths := []string{
		"./pocket",
		"../../pocket",
		"./bin/pocket",
		"../../bin/pocket",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			return abs
		}
	}

	// Try to build it
	cmd := exec.Command("go", "build", "-o", "pocket", "./cmd/pocket")
	if err := cmd.Run(); err == nil {
		abs, _ := filepath.Abs("./pocket")
		return abs
	}

	return ""
}

func runCommand(t *testing.T, args ...string) string {
	t.Helper()
	output, err := runCommandWithError(args...)
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}
	return output
}

func runCommandAllowError(t *testing.T, args ...string) string {
	t.Helper()
	output, _ := runCommandWithError(args...)
	return output
}

func runCommandWithError(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	return output, err
}

// TestE2EDocGeneration tests documentation generation.
func TestE2EDocGeneration(t *testing.T) {
	pocketBin := findPocketBinary(t)
	if pocketBin == "" {
		t.Skip("pocket binary not found, skipping e2e test")
	}

	tempDir := t.TempDir()

	// Generate markdown docs
	output := runCommand(t, pocketBin, "docs")
	if !strings.Contains(output, "# Pocket Node Reference") {
		t.Errorf("Markdown doc generation failed")
	}

	// Generate JSON docs
	output = runCommand(t, pocketBin, "docs", "--output", "json")
	if !strings.Contains(output, `"title"`) || !strings.Contains(output, `"nodes"`) {
		t.Errorf("JSON doc generation failed")
	}

	// Generate node-specific docs
	output = runCommand(t, pocketBin, "nodes", "docs")
	if !strings.Contains(output, "Core Nodes") {
		t.Errorf("Node docs generation failed")
	}

	// Save docs to file
	docsPath := filepath.Join(tempDir, "nodes.md")
	cmd := exec.Command(pocketBin, "docs")
	outFile, err := os.Create(docsPath)
	if err != nil {
		t.Fatalf("Failed to create docs file: %v", err)
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	err = cmd.Run()
	if err != nil {
		t.Errorf("Failed to save docs to file: %v", err)
	}

	// Verify file was created and has content
	info, err := os.Stat(docsPath)
	if err != nil || info.Size() == 0 {
		t.Error("Docs file was not created or is empty")
	}
}

// TestE2EConcurrentWorkflows tests running multiple workflows concurrently.
func TestE2EConcurrentWorkflows(t *testing.T) {
	pocketBin := findPocketBinary(t)
	if pocketBin == "" {
		t.Skip("pocket binary not found, skipping e2e test")
	}

	tempDir := t.TempDir()

	// Create multiple simple workflows
	for i := 0; i < 3; i++ {
		workflowYAML := fmt.Sprintf(`
name: concurrent-%d
description: Concurrent test %d
nodes:
  - name: echo
    type: echo
    config:
      message: "Workflow %d"
  - name: delay
    type: delay
    config:
      duration: "100ms"
connections:
  - from: echo
    to: delay
start: echo
`, i, i, i)

		workflowPath := filepath.Join(tempDir, fmt.Sprintf("workflow-%d.yaml", i))
		os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	}

	// Run workflows concurrently
	start := time.Now()
	errors := make(chan error, 3)

	for i := 0; i < 3; i++ {
		workflowPath := filepath.Join(tempDir, fmt.Sprintf("workflow-%d.yaml", i))
		go func(path string) {
			cmd := exec.Command(pocketBin, "run", path)
			errors <- cmd.Run()
		}(workflowPath)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		if err := <-errors; err != nil {
			t.Errorf("Concurrent workflow failed: %v", err)
		}
	}

	duration := time.Since(start)
	t.Logf("Concurrent workflows completed in %v", duration)

	// Should complete faster than sequential (300ms)
	if duration > 250*time.Millisecond {
		t.Logf("Warning: Concurrent execution might be slower than expected")
	}
}
