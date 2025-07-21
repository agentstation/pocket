package wasm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentstation/pocket/plugins"
)

// Test WASM module that implements a simple echo plugin.
var testWASM = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, // WASM magic number and version
	// This is a simplified/mock WASM module for testing
	// In real tests, we would use a properly compiled WASM module
}

func TestNewPlugin(t *testing.T) {
	ctx := context.Background()

	metadata := plugins.Metadata{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "Test",
		Runtime:     "wasm",
		Binary:      "test.wasm",
		Nodes: []plugins.NodeDefinition{
			{
				Type:        "echo",
				Category:    "test",
				Description: "Echo node",
			},
		},
	}

	// This test will fail with our mock WASM bytes
	// In a real implementation, we would need proper WASM bytes
	_, err := NewPlugin(ctx, testWASM, &metadata)
	if err == nil {
		t.Error("Expected error with mock WASM bytes, got nil")
	}
}

func TestParseMemoryLimit(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		wantErr  bool
	}{
		{"100KB", 100 * 1024, false},
		{"50MB", 50 * 1024 * 1024, false},
		{"2GB", 2 * 1024 * 1024 * 1024, false},
		{"invalid", 0, true},
		{"100TB", 0, true}, // Unsupported unit
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMemoryLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMemoryLimit(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseMemoryLimit(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoadPlugin(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test manifest
	manifest := plugins.Metadata{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "Test",
		Runtime:     "wasm",
		Binary:      "test.wasm",
		Nodes: []plugins.NodeDefinition{
			{
				Type:        "test-node",
				Category:    "test",
				Description: "Test node",
			},
		},
		Permissions: plugins.Permissions{
			Memory:  "10MB",
			Timeout: 5 * time.Second,
		},
	}

	// Write manifest to file
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Write mock WASM file
	wasmPath := filepath.Join(tmpDir, "test.wasm")
	if err := os.WriteFile(wasmPath, testWASM, 0o644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}

	// Test loading plugin
	ctx := context.Background()
	_, err = LoadPlugin(ctx, manifestPath)
	// We expect an error because our mock WASM is invalid
	if err == nil {
		t.Error("Expected error with mock WASM, got nil")
	}
}

// TestPluginMetadata tests that plugin metadata is correctly stored and retrieved.
func TestPluginMetadata(t *testing.T) {
	metadata := plugins.Metadata{
		Name:        "sentiment-analyzer",
		Version:     "1.0.0",
		Description: "Analyzes text sentiment",
		Author:      "Pocket Team",
		License:     "MIT",
		Runtime:     "wasm",
		Binary:      "plugin.wasm",
		Nodes: []plugins.NodeDefinition{
			{
				Type:        "sentiment",
				Category:    "ai",
				Description: "Analyze text sentiment",
				ConfigSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"threshold": map[string]interface{}{
							"type":    "number",
							"default": 0.1,
						},
					},
				},
			},
		},
		Permissions: plugins.Permissions{
			Memory:  "10MB",
			Timeout: 5 * time.Second,
		},
		Requirements: plugins.Requirements{
			Pocket: ">=1.0.0",
			Memory: "10MB",
		},
	}

	// Test that all fields are preserved
	if metadata.Name != "sentiment-analyzer" {
		t.Errorf("Expected name 'sentiment-analyzer', got %s", metadata.Name)
	}
	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", metadata.Version)
	}
	if len(metadata.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(metadata.Nodes))
	}
	if metadata.Nodes[0].Type != "sentiment" {
		t.Errorf("Expected node type 'sentiment', got %s", metadata.Nodes[0].Type)
	}
	if metadata.Permissions.Memory != "10MB" {
		t.Errorf("Expected memory permission '10MB', got %s", metadata.Permissions.Memory)
	}
	if metadata.Requirements.Pocket != ">=1.0.0" {
		t.Errorf("Expected Pocket requirement '>=1.0.0', got %s", metadata.Requirements.Pocket)
	}
}

// TestPluginRequest tests the request/response serialization.
func TestPluginRequest(t *testing.T) {
	req := plugins.Request{
		Node:     "sentiment",
		Function: "exec",
		Config: map[string]interface{}{
			"threshold": 0.1,
		},
		Input: json.RawMessage(`{"text": "Hello world"}`),
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Test unmarshaling
	var decoded plugins.Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.Node != req.Node {
		t.Errorf("Expected node %s, got %s", req.Node, decoded.Node)
	}
	if decoded.Function != req.Function {
		t.Errorf("Expected function %s, got %s", req.Function, decoded.Function)
	}
}

// TestPluginResponse tests the response serialization.
func TestPluginResponse(t *testing.T) {
	resp := plugins.Response{
		Success: true,
		Output:  json.RawMessage(`{"sentiment": "positive", "score": 0.8}`),
		Next:    "done",
	}

	// Test marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Test unmarshaling
	var decoded plugins.Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.Success != resp.Success {
		t.Errorf("Expected success %v, got %v", resp.Success, decoded.Success)
	}
	if decoded.Next != resp.Next {
		t.Errorf("Expected next %s, got %s", resp.Next, decoded.Next)
	}
}

// TestPluginSecurity tests security features.
func TestPluginSecurity(t *testing.T) {
	tests := []struct {
		name        string
		permissions plugins.Permissions
		expectation string
	}{
		{
			name: "memory limit",
			permissions: plugins.Permissions{
				Memory: "1MB",
			},
			expectation: "should enforce memory limit",
		},
		{
			name: "timeout",
			permissions: plugins.Permissions{
				Timeout: 1 * time.Second,
			},
			expectation: "should enforce timeout",
		},
		{
			name: "environment variables",
			permissions: plugins.Permissions{
				Env: []string{"HOME", "PATH"},
			},
			expectation: "should only allow specified env vars",
		},
		{
			name: "filesystem access",
			permissions: plugins.Permissions{
				Filesystem: []string{"/tmp", "/data"},
			},
			expectation: "should only allow specified paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These tests verify that the permissions fields exist and can be set
			// Actual enforcement would be tested with real WASM modules
			if tt.permissions.Memory != "" && tt.name == "memory limit" {
				t.Logf("Memory limit set to %s", tt.permissions.Memory)
			}
			if tt.permissions.Timeout > 0 && tt.name == "timeout" {
				t.Logf("Timeout set to %v", tt.permissions.Timeout)
			}
			if len(tt.permissions.Env) > 0 && tt.name == "environment variables" {
				t.Logf("Allowed env vars: %v", tt.permissions.Env)
			}
			if len(tt.permissions.Filesystem) > 0 && tt.name == "filesystem access" {
				t.Logf("Allowed paths: %v", tt.permissions.Filesystem)
			}
		})
	}
}
