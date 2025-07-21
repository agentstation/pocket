package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"

	"github.com/agentstation/pocket/plugins"
)

func TestDefaultPluginPaths(t *testing.T) {
	paths := DefaultPluginPaths()

	// Should have at least the system and current directory paths
	if len(paths) < 2 {
		t.Errorf("Expected at least 2 default paths, got %d", len(paths))
	}

	// Should include current directory
	found := false
	for _, p := range paths {
		if p == "./plugins" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected default paths to include ./plugins")
	}
}

func TestDiscover(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "plugins")

	// Create plugin directories
	plugin1Dir := filepath.Join(pluginDir, "plugin1")
	plugin2Dir := filepath.Join(pluginDir, "plugin2")

	if err := os.MkdirAll(plugin1Dir, 0o755); err != nil {
		t.Fatalf("Failed to create plugin1 dir: %v", err)
	}
	if err := os.MkdirAll(plugin2Dir, 0o755); err != nil {
		t.Fatalf("Failed to create plugin2 dir: %v", err)
	}

	// Create valid manifest for plugin1
	manifest1 := plugins.Metadata{
		Name:        "test-plugin-1",
		Version:     "1.0.0",
		Description: "Test plugin 1",
		Author:      "Test",
		Runtime:     "wasm",
		Binary:      "plugin.wasm",
		Nodes: []plugins.NodeDefinition{
			{
				Type:        "test-node-1",
				Category:    "test",
				Description: "Test node 1",
			},
		},
	}

	manifest1Data, err := yaml.Marshal(manifest1)
	if err != nil {
		t.Fatalf("Failed to marshal manifest1: %v", err)
	}

	if err := os.WriteFile(filepath.Join(plugin1Dir, "manifest.yaml"), manifest1Data, 0o644); err != nil {
		t.Fatalf("Failed to write manifest1: %v", err)
	}

	// Create invalid manifest for plugin2 (missing required fields)
	invalidManifest := map[string]interface{}{
		"name": "invalid-plugin",
		// Missing version, runtime, binary, nodes
	}

	invalidData, err := yaml.Marshal(invalidManifest)
	if err != nil {
		t.Fatalf("Failed to marshal invalid manifest: %v", err)
	}

	if err := os.WriteFile(filepath.Join(plugin2Dir, "manifest.yaml"), invalidData, 0o644); err != nil {
		t.Fatalf("Failed to write invalid manifest: %v", err)
	}

	// Test discovery
	l := New()
	discovered, err := l.Discover(pluginDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should find only the valid plugin
	if len(discovered) != 1 {
		t.Errorf("Expected 1 valid plugin, got %d", len(discovered))
	}

	if len(discovered) > 0 && discovered[0].Name != "test-plugin-1" {
		t.Errorf("Expected plugin name 'test-plugin-1', got %s", discovered[0].Name)
	}
}

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test manifest
	manifest := plugins.Metadata{
		Name:        "load-test-plugin",
		Version:     "1.0.0",
		Description: "Plugin for load testing",
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
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	// Test 1: Load from manifest file
	manifestPath := filepath.Join(tmpDir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Create dummy WASM file
	wasmPath := filepath.Join(tmpDir, "test.wasm")
	if err := os.WriteFile(wasmPath, []byte{0x00, 0x61, 0x73, 0x6d}, 0o644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}

	l := New()
	ctx := context.Background()

	// This will fail because the WASM is invalid, but it tests the loading logic
	_, err = l.Load(ctx, manifestPath)
	if err == nil {
		t.Error("Expected error with invalid WASM, got nil")
	}

	// Test 2: Load from directory
	pluginDir := filepath.Join(tmpDir, "plugin-dir")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}

	manifestPath2 := filepath.Join(pluginDir, "manifest.yaml")
	if err := os.WriteFile(manifestPath2, manifestData, 0o644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	wasmPath2 := filepath.Join(pluginDir, "test.wasm")
	if err := os.WriteFile(wasmPath2, []byte{0x00, 0x61, 0x73, 0x6d}, 0o644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}

	_, err = l.Load(ctx, pluginDir)
	if err == nil {
		t.Error("Expected error with invalid WASM, got nil")
	}

	// Test 3: Load from WASM file
	_, err = l.Load(ctx, wasmPath2)
	if err == nil {
		t.Error("Expected error with invalid WASM, got nil")
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata plugins.Metadata
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid metadata",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Version: "1.0.0",
				Runtime: "wasm",
				Binary:  "plugin.wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Type:        "test-node",
						Category:    "test",
						Description: "Test node",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			metadata: plugins.Metadata{
				Version: "1.0.0",
				Runtime: "wasm",
				Binary:  "plugin.wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Type:        "test-node",
						Category:    "test",
						Description: "Test node",
					},
				},
			},
			wantErr: true,
			errMsg:  "plugin name is required",
		},
		{
			name: "missing version",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Runtime: "wasm",
				Binary:  "plugin.wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Type:        "test-node",
						Category:    "test",
						Description: "Test node",
					},
				},
			},
			wantErr: true,
			errMsg:  "plugin version is required",
		},
		{
			name: "missing runtime",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Version: "1.0.0",
				Binary:  "plugin.wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Type:        "test-node",
						Category:    "test",
						Description: "Test node",
					},
				},
			},
			wantErr: true,
			errMsg:  "plugin runtime is required",
		},
		{
			name: "missing binary",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Version: "1.0.0",
				Runtime: "wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Type:        "test-node",
						Category:    "test",
						Description: "Test node",
					},
				},
			},
			wantErr: true,
			errMsg:  "plugin binary is required",
		},
		{
			name: "no nodes",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Version: "1.0.0",
				Runtime: "wasm",
				Binary:  "plugin.wasm",
				Nodes:   []plugins.NodeDefinition{},
			},
			wantErr: true,
			errMsg:  "plugin must export at least one node",
		},
		{
			name: "node missing type",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Version: "1.0.0",
				Runtime: "wasm",
				Binary:  "plugin.wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Category:    "test",
						Description: "Test node",
					},
				},
			},
			wantErr: true,
			errMsg:  "node type is required",
		},
		{
			name: "node missing category",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Version: "1.0.0",
				Runtime: "wasm",
				Binary:  "plugin.wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Type:        "test-node",
						Description: "Test node",
					},
				},
			},
			wantErr: true,
			errMsg:  "node category is required for type test-node",
		},
		{
			name: "node missing description",
			metadata: plugins.Metadata{
				Name:    "test-plugin",
				Version: "1.0.0",
				Runtime: "wasm",
				Binary:  "plugin.wasm",
				Nodes: []plugins.NodeDefinition{
					{
						Type:     "test-node",
						Category: "test",
					},
				},
			},
			wantErr: true,
			errMsg:  "node description is required for type test-node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetadata(tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("validateMetadata() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLoadManifest(t *testing.T) {
	tmpDir := t.TempDir()
	l := &loader{
		discovered: make(map[string]plugins.Metadata),
	}

	// Test YAML manifest
	yamlManifest := plugins.Metadata{
		Name:        "yaml-plugin",
		Version:     "1.0.0",
		Description: "YAML test plugin",
		Runtime:     "wasm",
		Binary:      "plugin.wasm",
		Nodes: []plugins.NodeDefinition{
			{
				Type:        "test",
				Category:    "test",
				Description: "Test node",
			},
		},
	}

	yamlData, err := yaml.Marshal(yamlManifest)
	if err != nil {
		t.Fatalf("Failed to marshal YAML: %v", err)
	}

	yamlPath := filepath.Join(tmpDir, "manifest.yaml")
	if err := os.WriteFile(yamlPath, yamlData, 0o644); err != nil {
		t.Fatalf("Failed to write YAML manifest: %v", err)
	}

	loaded, err := l.loadManifest(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load YAML manifest: %v", err)
	}

	if loaded.Name != yamlManifest.Name {
		t.Errorf("Expected name %s, got %s", yamlManifest.Name, loaded.Name)
	}

	// Check that binary path was resolved to absolute
	expectedBinary := filepath.Join(tmpDir, "plugin.wasm")
	if loaded.Binary != expectedBinary {
		t.Errorf("Expected binary path %s, got %s", expectedBinary, loaded.Binary)
	}
}

func TestUnsupportedRuntime(t *testing.T) {
	l := New()
	ctx := context.Background()

	metadata := plugins.Metadata{
		Name:    "unsupported-plugin",
		Version: "1.0.0",
		Runtime: "python", // Unsupported runtime
		Binary:  "plugin.py",
		Nodes: []plugins.NodeDefinition{
			{
				Type:        "test",
				Category:    "test",
				Description: "Test",
			},
		},
	}

	_, err := l.LoadFromMetadata(ctx, metadata)
	if err == nil {
		t.Error("Expected error for unsupported runtime, got nil")
	}

	if err != nil && err.Error() != "unsupported runtime: python" {
		t.Errorf("Expected 'unsupported runtime' error, got: %v", err)
	}
}
