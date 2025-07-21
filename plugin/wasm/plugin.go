// Package wasm implements WebAssembly plugin support using wazero.
package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/agentstation/pocket/plugin"
)

// wasmPlugin implements the Plugin interface for WebAssembly plugins.
type wasmPlugin struct {
	metadata plugin.Metadata
	runtime  wazero.Runtime
	module   api.Module

	// Exported functions
	callFunc api.Function

	// Mutex for thread safety
	mu sync.Mutex
}

// NewPlugin creates a new WebAssembly plugin from bytes.
func NewPlugin(ctx context.Context, wasmBytes []byte, metadata plugin.Metadata) (plugin.Plugin, error) {
	// Create runtime with configuration
	runtimeConfig := wazero.NewRuntimeConfig()

	// Set memory limit if specified
	if metadata.Permissions.Memory != "" {
		limit, err := parseMemoryLimit(metadata.Permissions.Memory)
		if err != nil {
			return nil, fmt.Errorf("invalid memory limit: %w", err)
		}
		runtimeConfig = runtimeConfig.WithMemoryLimitPages(uint32(limit / 65536)) // 64KB pages
	}

	// Create runtime
	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)

	// Initialize WASI if needed
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile the module
	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to compile WASM module: %w", err)
	}

	// Configure module with sandboxing
	moduleConfig := wazero.NewModuleConfig().
		WithName(metadata.Name).
		WithStartFunctions() // Don't auto-call _start

	// Add allowed environment variables
	for _, envVar := range metadata.Permissions.Env {
		if value := os.Getenv(envVar); value != "" {
			moduleConfig = moduleConfig.WithEnv(envVar, value)
		}
	}

	// Configure filesystem access if permitted
	if len(metadata.Permissions.Filesystem) > 0 {
		// For now, we'll mount allowed directories
		// In production, we'd want more fine-grained control
		for _, path := range metadata.Permissions.Filesystem {
			if stat, err := os.Stat(path); err == nil && stat.IsDir() {
				moduleConfig = moduleConfig.WithFS(os.DirFS(path))
				break // wazero only supports one FS mount currently
			}
		}
	}

	// Instantiate the module
	module, err := r.InstantiateModule(ctx, compiled, moduleConfig)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	// Get the main call function
	callFunc := module.ExportedFunction("__pocket_call")
	if callFunc == nil {
		module.Close(ctx)
		r.Close(ctx)
		return nil, fmt.Errorf("plugin does not export required function: __pocket_call")
	}

	// Get memory exports for passing data
	memory := module.ExportedMemory("memory")
	if memory == nil {
		module.Close(ctx)
		r.Close(ctx)
		return nil, fmt.Errorf("plugin does not export memory")
	}

	// Get allocation functions
	allocFunc := module.ExportedFunction("__pocket_alloc")
	if allocFunc == nil {
		module.Close(ctx)
		r.Close(ctx)
		return nil, fmt.Errorf("plugin does not export required function: __pocket_alloc")
	}

	return &wasmPlugin{
		metadata: metadata,
		runtime:  r,
		module:   module,
		callFunc: callFunc,
	}, nil
}

// Metadata returns the plugin's metadata.
func (p *wasmPlugin) Metadata() plugin.Metadata {
	return p.metadata
}

// Call invokes a function exported by the plugin.
func (p *wasmPlugin) Call(ctx context.Context, function string, input []byte) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Apply timeout if configured
	if p.metadata.Permissions.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.metadata.Permissions.Timeout)
		defer cancel()
	}

	// Get memory and allocation functions
	memory := p.module.ExportedMemory("memory")
	allocFunc := p.module.ExportedFunction("__pocket_alloc")
	freeFunc := p.module.ExportedFunction("__pocket_free")

	// Allocate memory for input
	inputLen := uint32(len(input))
	results, err := allocFunc.Call(ctx, uint64(inputLen))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory: %w", err)
	}

	inputPtr := uint32(results[0])

	// Write input to WASM memory
	if !memory.Write(inputPtr, input) {
		return nil, fmt.Errorf("failed to write input to memory")
	}

	// Call the function
	results, err = p.callFunc.Call(ctx, uint64(inputPtr), uint64(inputLen))
	if err != nil {
		return nil, fmt.Errorf("plugin call failed: %w", err)
	}

	// Free input memory
	if freeFunc != nil {
		freeFunc.Call(ctx, uint64(inputPtr), uint64(inputLen))
	}

	// Read the result
	resultPtr := uint32(results[0])
	resultLen := uint32(results[1])

	if resultLen == 0 {
		return nil, nil
	}

	output, ok := memory.Read(resultPtr, resultLen)
	if !ok {
		return nil, fmt.Errorf("failed to read output from memory")
	}

	// Free output memory
	if freeFunc != nil {
		freeFunc.Call(ctx, uint64(resultPtr), uint64(resultLen))
	}

	return output, nil
}

// Close releases plugin resources.
func (p *wasmPlugin) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.module != nil {
		p.module.Close(ctx)
	}
	if p.runtime != nil {
		return p.runtime.Close(ctx)
	}
	return nil
}

// LoadPlugin loads a WebAssembly plugin from a file.
func LoadPlugin(ctx context.Context, path string) (plugin.Plugin, error) {
	// Read manifest
	manifestPath := filepath.Join(filepath.Dir(path), "manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		// Try JSON format
		manifestPath = filepath.Join(filepath.Dir(path), "manifest.json")
		manifestData, err = os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read manifest: %w", err)
		}
	}

	// Parse manifest
	var metadata plugin.Metadata
	if filepath.Ext(manifestPath) == ".json" {
		err = json.Unmarshal(manifestData, &metadata)
	} else {
		// Use YAML unmarshaling
		// For now, we'll use JSON since we already have it
		// In production, we'd use the yaml package
		return nil, fmt.Errorf("YAML parsing not implemented yet")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Read WASM binary
	wasmPath := filepath.Join(filepath.Dir(path), metadata.Binary)
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM binary: %w", err)
	}

	return NewPlugin(ctx, wasmBytes, metadata)
}

// parseMemoryLimit parses a memory limit string (e.g., "100MB", "1GB").
func parseMemoryLimit(limit string) (uint64, error) {
	// Simple implementation for now
	var value uint64
	var unit string

	_, err := fmt.Sscanf(limit, "%d%s", &value, &unit)
	if err != nil {
		return 0, err
	}

	switch unit {
	case "KB":
		return value * 1024, nil
	case "MB":
		return value * 1024 * 1024, nil
	case "GB":
		return value * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}
}
