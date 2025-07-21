package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/agentstation/pocket/nodes"
)

func TestGetBuiltinNodes(t *testing.T) {
	allNodes := getBuiltinNodes()

	// Check that we have nodes
	if len(allNodes) == 0 {
		t.Error("Expected at least one built-in node")
	}

	// Check that all nodes have required fields
	for _, node := range allNodes {
		if node.Type == "" {
			t.Error("Node missing type")
		}
		if node.Category == "" {
			t.Error("Node missing category")
		}
		if node.Description == "" {
			t.Errorf("Node %s missing description", node.Type)
		}
	}

	// Check for expected node types
	expectedTypes := []string{"echo", "delay", "router", "conditional", "transform"}
	typeMap := make(map[string]bool)
	for _, node := range allNodes {
		typeMap[node.Type] = true
	}

	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("Expected node type %s not found", expected)
		}
	}
}

func TestOutputTable(t *testing.T) {
	// Create test nodes
	testNodes := []nodes.Metadata{
		{
			Type:        "test1",
			Category:    "core",
			Description: "Test node 1",
		},
		{
			Type:        "test2",
			Category:    "data",
			Description: "Test node 2",
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputTable(testNodes)
	if err != nil {
		t.Errorf("outputTable() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check output contains expected content
	if !strings.Contains(output, "Core:") {
		t.Error("Output missing Core category")
	}
	if !strings.Contains(output, "Data:") {
		t.Error("Output missing Data category")
	}
	if !strings.Contains(output, "test1") {
		t.Error("Output missing test1 node")
	}
	if !strings.Contains(output, "test2") {
		t.Error("Output missing test2 node")
	}
	if !strings.Contains(output, "Total: 2 node types") {
		t.Error("Output missing total count")
	}
}

func TestOutputJSON(t *testing.T) {
	// Create test nodes
	testNodes := []nodes.Metadata{
		{
			Type:        "test1",
			Category:    "core",
			Description: "Test node 1",
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(testNodes)
	if err != nil {
		t.Errorf("outputJSON() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON to verify it's valid
	var result []nodes.Metadata
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}

	// Check content
	if len(result) != 1 {
		t.Errorf("Expected 1 node, got %d", len(result))
	}
	if result[0].Type != "test1" {
		t.Errorf("Expected type test1, got %s", result[0].Type)
	}
}

func TestOutputYAML(t *testing.T) {
	// Create test nodes
	testNodes := []nodes.Metadata{
		{
			Type:        "test1",
			Category:    "core",
			Description: "Test node 1",
			Since:       "v1.0.0",
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputYAML(testNodes)
	if err != nil {
		t.Errorf("outputYAML() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check YAML content
	if !strings.Contains(output, "type: test1") {
		t.Error("Output missing type field")
	}
	if !strings.Contains(output, "category: core") {
		t.Error("Output missing category field")
	}
	if !strings.Contains(output, "description: Test node 1") {
		t.Error("Output missing description field")
	}
	if !strings.Contains(output, "since: v1.0.0") {
		t.Error("Output missing since field")
	}
}
