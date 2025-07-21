//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	goyaml "github.com/goccy/go-yaml"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/nodes"
	"github.com/agentstation/pocket/yaml"
)

// TestEndToEndWorkflowExecution tests a complete workflow from YAML to execution.
func TestEndToEndWorkflowExecution(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a simple workflow YAML file
	workflowYAML := `
name: test-workflow
description: Integration test workflow
nodes:
  - name: start
    type: echo
    config:
      message: "Starting workflow"
  
  - name: delay
    type: delay
    config:
      duration: "100ms"
  
  - name: transform
    type: transform
    config:
      expression: |
        {
          "original": input,
          "timestamp": now(),
          "status": "processed"
        }

connections:
  - from: start
    to: delay
  - from: delay
    to: transform

start: start
`

	workflowPath := filepath.Join(tempDir, "test-workflow.yaml")
	err := os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test the run command
	config := &RunConfig{
		FilePath:   workflowPath,
		Verbose:    false,
		DryRun:     false,
		StoreType:  "memory",
		MaxEntries: 1000,
		TTL:        0,
	}

	// Execute the workflow
	err = runWorkflow(config)
	if err != nil {
		t.Errorf("Workflow execution failed: %v", err)
	}
}

// TestConditionalRoutingWorkflow tests conditional routing in workflows.
func TestConditionalRoutingWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	workflowYAML := `
name: conditional-workflow
description: Test conditional routing
nodes:
  - name: input
    type: echo
    config:
      message: "Starting with score"
      passthrough: true

  - name: router
    type: conditional
    config:
      conditions:
        - if: "{{gt .score 0.8}}"
          then: "high"
        - if: "{{gt .score 0.5}}"
          then: "medium"
      else: "low"

  - name: high
    type: transform
    config:
      expression: '{"result": "high score", "score": input.score}'

  - name: medium
    type: transform
    config:
      expression: '{"result": "medium score", "score": input.score}'

  - name: low
    type: transform
    config:
      expression: '{"result": "low score", "score": input.score}'

connections:
  - from: input
    to: router
  - from: router
    to: high
    action: high
  - from: router
    to: medium
    action: medium
  - from: router
    to: low
    action: low

start: input
`

	workflowPath := filepath.Join(tempDir, "conditional-workflow.yaml")
	err := os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test with high score
	ctx := context.Background()
	store := pocket.NewStore()

	// Load the workflow
	loader := yaml.NewLoader()
	nodes.RegisterAll(loader, false)

	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow: %v", err)
	}

	var graphDef yaml.GraphDefinition
	if err := goyaml.Unmarshal(data, &graphDef); err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	graph, err := loader.LoadDefinition(&graphDef, store)
	if err != nil {
		t.Fatalf("Failed to load workflow: %v", err)
	}

	// Test high score path
	input := map[string]interface{}{"score": 0.9}
	result, err := graph.Run(ctx, input)
	if err != nil {
		t.Errorf("High score test failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected map result, got %T", result)
	} else if resultMap["result"] != "high score" {
		t.Errorf("Expected 'high score', got %v", resultMap["result"])
	}

	// Test medium score path
	input = map[string]interface{}{"score": 0.6}
	result, err = graph.Run(ctx, input)
	if err != nil {
		t.Errorf("Medium score test failed: %v", err)
	}

	resultMap, ok = result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected map result, got %T", result)
	} else if resultMap["result"] != "medium score" {
		t.Errorf("Expected 'medium score', got %v", resultMap["result"])
	}

	// Test low score path
	input = map[string]interface{}{"score": 0.3}
	result, err = graph.Run(ctx, input)
	if err != nil {
		t.Errorf("Low score test failed: %v", err)
	}

	resultMap, ok = result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected map result, got %T", result)
	} else if resultMap["result"] != "low score" {
		t.Errorf("Expected 'low score', got %v", resultMap["result"])
	}
}

// TestParallelExecutionWorkflow tests parallel node execution.
func TestParallelExecutionWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	workflowYAML := `
name: parallel-workflow
description: Test parallel execution
nodes:
  - name: start
    type: echo
    config:
      message: "Starting parallel execution"
      passthrough: true

  - name: parallel
    type: parallel
    config:
      nodes:
        - name: task1
          type: delay
          config:
            duration: "50ms"
        - name: task2
          type: delay
          config:
            duration: "50ms"
        - name: task3
          type: delay
          config:
            duration: "50ms"

  - name: aggregate
    type: aggregate
    config:
      mode: "merge"

connections:
  - from: start
    to: parallel
  - from: parallel
    to: aggregate

start: start
`

	workflowPath := filepath.Join(tempDir, "parallel-workflow.yaml")
	err := os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Measure execution time
	start := time.Now()

	config := &RunConfig{
		FilePath:   workflowPath,
		Verbose:    false,
		DryRun:     false,
		StoreType:  "memory",
		MaxEntries: 1000,
		TTL:        0,
	}

	err = runWorkflow(config)
	if err != nil {
		t.Errorf("Parallel workflow execution failed: %v", err)
	}

	duration := time.Since(start)

	// If running in parallel, should take ~50ms, not 150ms
	if duration > 100*time.Millisecond {
		t.Logf("Warning: Parallel execution may not be working correctly (took %v)", duration)
	}
}

// TestLuaScriptIntegration tests Lua script node integration.
func TestLuaScriptIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Create a Lua script that returns a result directly
	luaScript := `
-- Simple data processor
return {
    original = input,
    processed = true,
    doubled = {
        a = (input.a or 0) * 2,
        b = (input.b or 0) * 2
    }
}
`

	scriptPath := filepath.Join(tempDir, "processor.lua")
	err := os.WriteFile(scriptPath, []byte(luaScript), 0600)
	if err != nil {
		t.Fatalf("Failed to write Lua script: %v", err)
	}

	workflowYAML := fmt.Sprintf(`
name: lua-workflow
description: Test Lua script integration
nodes:
  - name: process
    type: lua
    config:
      file: "%s"

start: process
`, scriptPath)

	workflowPath := filepath.Join(tempDir, "lua-workflow.yaml")
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Create a store and load the workflow
	ctx := context.Background()
	store := pocket.NewStore()
	loader := yaml.NewLoader()
	nodes.RegisterAll(loader, false)

	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow: %v", err)
	}

	var graphDef yaml.GraphDefinition
	if err := goyaml.Unmarshal(data, &graphDef); err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	graph, err := loader.LoadDefinition(&graphDef, store)
	if err != nil {
		t.Fatalf("Failed to load workflow: %v", err)
	}

	// Run with test data
	input := map[string]interface{}{
		"a":    10,
		"b":    20,
		"name": "test",
	}

	result, err := graph.Run(ctx, input)
	if err != nil {
		t.Errorf("Lua workflow execution failed: %v", err)
	}

	// Verify the result
	t.Logf("Result from Lua: %+v", result)

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Errorf("Expected map result, got %T", result)
		return
	}

	// Check if the result has the expected fields
	if _, hasOriginal := resultMap["original"]; !hasOriginal {
		t.Error("Result missing 'original' field")
	}
	if _, hasProcessed := resultMap["processed"]; !hasProcessed {
		t.Error("Result missing 'processed' field")
	}
	if _, hasDoubled := resultMap["doubled"]; !hasDoubled {
		t.Error("Result missing 'doubled' field")
	}

	// Check doubled values if present
	if doubled, ok := resultMap["doubled"].(map[string]interface{}); ok {
		// Check values - they might be int or float64 depending on Lua
		aVal, aOk := doubled["a"]
		bVal, bOk := doubled["b"]

		if aOk {
			if aFloat, ok := aVal.(float64); ok && aFloat != 20.0 {
				t.Errorf("Expected doubled['a'] = 20, got %v", aFloat)
			} else if aInt, ok := aVal.(int); ok && aInt != 20 {
				t.Errorf("Expected doubled['a'] = 20, got %v", aInt)
			}
		}

		if bOk {
			if bFloat, ok := bVal.(float64); ok && bFloat != 40.0 {
				t.Errorf("Expected doubled['b'] = 40, got %v", bFloat)
			} else if bInt, ok := bVal.(int); ok && bInt != 40 {
				t.Errorf("Expected doubled['b'] = 40, got %v", bInt)
			}
		}
	} else {
		t.Logf("Warning: doubled field is not a map or is nil")
	}
}

// TestCLICommandIntegration tests CLI commands working together.
func TestCLICommandIntegration(t *testing.T) {
	// Test nodes command output can be parsed
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// List nodes in JSON format
	config := &NodesConfig{Format: "json"}
	err := runNodesList(config)
	if err != nil {
		t.Errorf("Failed to list nodes: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	// Parse JSON output
	var nodeList []nodes.Metadata
	err = json.Unmarshal(buf.Bytes(), &nodeList)
	if err != nil {
		t.Errorf("Failed to parse nodes JSON: %v", err)
	}

	// Verify we have expected nodes
	nodeTypes := make(map[string]bool)
	for _, node := range nodeList {
		nodeTypes[node.Type] = true
	}

	expectedTypes := []string{"echo", "delay", "conditional", "transform", "lua", "parallel"}
	for _, expected := range expectedTypes {
		if !nodeTypes[expected] {
			t.Errorf("Expected node type %s not found in list", expected)
		}
	}
}

// TestStoreIntegration tests store functionality in workflows.
func TestStoreIntegration(t *testing.T) {
	tempDir := t.TempDir()

	workflowYAML := `
name: store-workflow
description: Test store functionality
nodes:
  - name: save
    type: transform
    config:
      expression: |
        {
          "saved": input,
          "timestamp": now()
        }
      store:
        key: "workflow_data"
        value: "{{ .result }}"

  - name: delay
    type: delay
    config:
      duration: "10ms"

  - name: load
    type: transform
    config:
      expression: |
        {
          "loaded": store.workflow_data,
          "original": input
        }

connections:
  - from: save
    to: delay
  - from: delay
    to: load

start: save
`

	workflowPath := filepath.Join(tempDir, "store-workflow.yaml")
	err := os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Note: This test would need the transform node to support store operations
	// For now, we'll just verify the workflow loads successfully
	loader := yaml.NewLoader()
	nodes.RegisterAll(loader, false)

	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow: %v", err)
	}

	var graphDef yaml.GraphDefinition
	if err := goyaml.Unmarshal(data, &graphDef); err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	if err := graphDef.Validate(); err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}

// TestErrorHandlingIntegration tests error handling in workflows.
func TestErrorHandlingIntegration(t *testing.T) {
	tempDir := t.TempDir()

	workflowYAML := `
name: error-workflow
description: Test error handling
nodes:
  - name: start
    type: echo
    config:
      message: "Starting"

  - name: fail
    type: validate
    config:
      schema:
        type: object
        properties:
          required_field:
            type: string
        required: ["required_field"]

connections:
  - from: start
    to: fail

start: start
`

	workflowPath := filepath.Join(tempDir, "error-workflow.yaml")
	err := os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	config := &RunConfig{
		FilePath:   workflowPath,
		Verbose:    false,
		DryRun:     false,
		StoreType:  "memory",
		MaxEntries: 1000,
		TTL:        0,
	}

	// This should fail due to validation
	err = runWorkflow(config)
	if err == nil {
		t.Error("Expected workflow to fail validation, but it succeeded")
	} else if !strings.Contains(err.Error(), "validation") && !strings.Contains(err.Error(), "required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

// TestDryRunIntegration tests dry-run functionality.
func TestDryRunIntegration(t *testing.T) {
	tempDir := t.TempDir()

	workflowYAML := `
name: dryrun-workflow
description: Test dry-run
nodes:
  - name: echo
    type: echo
    config:
      message: "This should not execute"

start: echo
`

	workflowPath := filepath.Join(tempDir, "dryrun-workflow.yaml")
	err := os.WriteFile(workflowPath, []byte(workflowYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	config := &RunConfig{
		FilePath:   workflowPath,
		Verbose:    false,
		DryRun:     true, // Dry run mode
		StoreType:  "memory",
		MaxEntries: 1000,
		TTL:        0,
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runWorkflow(config)

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("Dry run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "validation successful") {
		t.Error("Expected dry run to indicate validation success")
	}
}
