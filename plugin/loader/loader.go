// Package loader provides plugin discovery and loading functionality.
package loader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/agentstation/pocket/plugin"
	"github.com/agentstation/pocket/plugin/wasm"
)

// DefaultPluginPaths returns the default paths to search for plugins.
func DefaultPluginPaths() []string {
	paths := []string{}

	// User home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".pocket", "plugins"))
	}

	// System-wide location and current directory
	paths = append(paths, "/usr/local/share/pocket/plugins", "./plugins")

	return paths
}

// loader implements the plugin.Loader interface.
type loader struct {
	// Cache of discovered plugins
	discovered map[string]plugin.Metadata
}

// New creates a new plugin loader.
func New() plugin.Loader {
	return &loader{
		discovered: make(map[string]plugin.Metadata),
	}
}

// Discover finds all plugins in the given paths.
func (l *loader) Discover(paths ...string) ([]plugin.Metadata, error) {
	if len(paths) == 0 {
		paths = DefaultPluginPaths()
	}

	var plugins []plugin.Metadata

	for _, path := range paths {
		// Skip if path doesn't exist
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// Walk the directory
		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil //nolint:nilerr // Skip errors intentionally
			}

			// Look for manifest files
			if info.IsDir() {
				return nil
			}

			if info.Name() == "manifest.yaml" || info.Name() == "manifest.json" {
				metadata, err := l.loadManifest(p)
				if err != nil {
					// Log error but continue
					fmt.Fprintf(os.Stderr, "Warning: failed to load manifest %s: %v\n", p, err)
					return nil
				}

				// Validate metadata
				if err := validateMetadata(metadata); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: invalid plugin metadata in %s: %v\n", p, err)
					return nil
				}

				// Store in cache
				l.discovered[metadata.Name] = metadata
				plugins = append(plugins, metadata)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
		}
	}

	return plugins, nil
}

// Load loads a plugin from the given path.
func (l *loader) Load(ctx context.Context, path string) (plugin.Plugin, error) {
	// Check if it's a manifest file
	if strings.HasSuffix(path, "manifest.yaml") || strings.HasSuffix(path, "manifest.json") {
		metadata, err := l.loadManifest(path)
		if err != nil {
			return nil, err
		}
		return l.LoadFromMetadata(ctx, metadata)
	}

	// Check if it's a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// Look for manifest in directory
		manifestPath := filepath.Join(path, "manifest.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			manifestPath = filepath.Join(path, "manifest.json")
		}

		metadata, err := l.loadManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		return l.LoadFromMetadata(ctx, metadata)
	}

	// If it's a .wasm file, look for manifest in same directory
	if strings.HasSuffix(path, ".wasm") {
		dir := filepath.Dir(path)
		manifestPath := filepath.Join(dir, "manifest.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			manifestPath = filepath.Join(dir, "manifest.json")
		}

		metadata, err := l.loadManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		return l.LoadFromMetadata(ctx, metadata)
	}

	return nil, fmt.Errorf("unable to load plugin from path: %s", path)
}

// LoadFromMetadata loads a plugin using its metadata.
//
//nolint:gocritic // hugeParam: metadata is copied intentionally for safety
func (l *loader) LoadFromMetadata(ctx context.Context, metadata plugin.Metadata) (plugin.Plugin, error) {
	// Validate metadata
	if err := validateMetadata(metadata); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	// Only support WASM for now
	if metadata.Runtime != "wasm" {
		return nil, fmt.Errorf("unsupported runtime: %s", metadata.Runtime)
	}

	// Load WASM plugin
	// The binary path is relative to the manifest location
	// We need to resolve it properly
	var wasmPath string
	if filepath.IsAbs(metadata.Binary) {
		wasmPath = metadata.Binary
	} else {
		// If we have a cached location from discovery, use that
		if cached, ok := l.discovered[metadata.Name]; ok && cached.Binary != metadata.Binary {
			// The cached version has the resolved path
			wasmPath = cached.Binary
		} else {
			// Assume it's relative to current directory
			wasmPath = metadata.Binary
		}
	}

	// Read WASM bytes
	wasmBytes, err := os.ReadFile(wasmPath) // nolint:gosec // Path is validated
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM binary: %w", err)
	}

	// Create WASM plugin
	return wasm.NewPlugin(ctx, wasmBytes, &metadata)
}

// loadManifest loads a plugin manifest from a file.
func (l *loader) loadManifest(path string) (plugin.Metadata, error) {
	data, err := os.ReadFile(path) // nolint:gosec // Path is from manifest
	if err != nil {
		return plugin.Metadata{}, fmt.Errorf("failed to read manifest: %w", err)
	}

	var metadata plugin.Metadata

	// YAML parser can handle both YAML and JSON
	err = yaml.Unmarshal(data, &metadata)

	if err != nil {
		return plugin.Metadata{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Resolve binary path relative to manifest
	if !filepath.IsAbs(metadata.Binary) {
		metadata.Binary = filepath.Join(filepath.Dir(path), metadata.Binary)
	}

	return metadata, nil
}

// validateMetadata validates plugin metadata.
//
//nolint:gocritic // hugeParam: metadata is copied intentionally for validation
func validateMetadata(metadata plugin.Metadata) error {
	if metadata.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if metadata.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	if metadata.Runtime == "" {
		return fmt.Errorf("plugin runtime is required")
	}

	if metadata.Binary == "" {
		return fmt.Errorf("plugin binary is required")
	}

	if len(metadata.Nodes) == 0 {
		return fmt.Errorf("plugin must export at least one node")
	}

	// Validate each node
	for _, node := range metadata.Nodes {
		if node.Type == "" {
			return fmt.Errorf("node type is required")
		}
		if node.Category == "" {
			return fmt.Errorf("node category is required for type %s", node.Type)
		}
		if node.Description == "" {
			return fmt.Errorf("node description is required for type %s", node.Type)
		}
	}

	return nil
}
