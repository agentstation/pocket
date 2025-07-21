package script

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shopify/go-lua"

	"github.com/agentstation/pocket"
)

// Manager handles script discovery and management.
type Manager struct {
	scriptsDir string
	scripts    map[string]*Script
	verbose    bool
}

// Script represents a discovered Lua script.
type Script struct {
	Name        string
	Path        string
	Category    string
	Description string
	Version     string
	Content     string
}

// NewManager creates a new script manager.
func NewManager(scriptsDir string, verbose bool) *Manager {
	if scriptsDir == "" {
		home, _ := os.UserHomeDir()
		scriptsDir = filepath.Join(home, ".pocket", "scripts")
	}
	return &Manager{
		scriptsDir: scriptsDir,
		scripts:    make(map[string]*Script),
		verbose:    verbose,
	}
}

// Discover finds all Lua scripts in the scripts directory.
func (m *Manager) Discover() error {
	// Create scripts directory if it doesn't exist
	if err := os.MkdirAll(m.scriptsDir, 0o750); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Walk the directory tree
	err := filepath.WalkDir(m.scriptsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Lua files
		if d.IsDir() || !strings.HasSuffix(path, ".lua") {
			return nil
		}

		// Load and parse the script
		script, err := m.LoadScript(path)
		if err != nil {
			if m.verbose {
				fmt.Printf("Warning: failed to load script %s: %v\n", path, err)
			}
			return nil // Continue discovering other scripts
		}

		m.scripts[script.Name] = script
		if m.verbose {
			fmt.Printf("Discovered script: %s (%s)\n", script.Name, script.Path)
		}

		return nil
	})

	return err
}

// LoadScript loads and parses a Lua script.
func (m *Manager) LoadScript(path string) (*Script, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is user-provided and validated
	if err != nil {
		return nil, err
	}

	// Extract metadata from comments
	script := &Script{
		Path:    path,
		Content: string(content),
	}

	// Parse metadata comments at the top of the file
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Stop at first non-comment line
		if !strings.HasPrefix(line, "--") {
			break
		}

		// Parse metadata comments
		switch {
		case strings.HasPrefix(line, "-- @name:"):
			script.Name = strings.TrimSpace(strings.TrimPrefix(line, "-- @name:"))
		case strings.HasPrefix(line, "-- @category:"):
			script.Category = strings.TrimSpace(strings.TrimPrefix(line, "-- @category:"))
		case strings.HasPrefix(line, "-- @description:"):
			script.Description = strings.TrimSpace(strings.TrimPrefix(line, "-- @description:"))
		case strings.HasPrefix(line, "-- @version:"):
			script.Version = strings.TrimSpace(strings.TrimPrefix(line, "-- @version:"))
		}
	}

	// Use filename as name if not specified
	if script.Name == "" {
		base := filepath.Base(path)
		script.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// Default category
	if script.Category == "" {
		script.Category = "script"
	}

	return script, nil
}

// GetScript returns a discovered script by name.
func (m *Manager) GetScript(name string) (*Script, bool) {
	script, ok := m.scripts[name]
	return script, ok
}

// ListScripts returns all discovered scripts.
func (m *Manager) ListScripts() []*Script {
	scripts := make([]*Script, 0, len(m.scripts))
	for _, script := range m.scripts {
		scripts = append(scripts, script)
	}
	return scripts
}

// ValidateScript validates a Lua script without executing it.
func (m *Manager) ValidateScript(path string) error {
	content, err := os.ReadFile(path) //nolint:gosec // Path is user-provided and validated
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	l := lua.NewState()
	// Note: go-lua doesn't have a Close method

	// Just try to load the script without running it
	if err := lua.LoadString(l, string(content)); err != nil {
		return fmt.Errorf("script validation failed: %w", err)
	}

	// Check for required functions
	l.Pop(1) // Pop the loaded chunk

	// Load again and execute to define functions
	if err := lua.DoString(l, string(content)); err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	// Check for required functions
	requiredFuncs := []string{"exec"} // At minimum, exec is required
	for _, fn := range requiredFuncs {
		l.Global(fn)
		if l.TypeOf(-1) != lua.TypeFunction {
			l.Pop(1)
			return fmt.Errorf("required function '%s' not found", fn)
		}
		l.Pop(1)
	}

	return nil
}

// CreateNode creates a Pocket node from a script.
func (m *Manager) CreateNode(script *Script) (pocket.Node, error) {
	return pocket.NewNode[any, any](script.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			return ExecuteLuaScript(ctx, script.Content, input, m.verbose)
		}),
	), nil
}

// ExecuteLuaScript executes a Lua script with the given input.
func ExecuteLuaScript(ctx context.Context, scriptContent string, input any, verbose bool) (any, error) {
	l := lua.NewState()
	// Note: go-lua doesn't have a Close method

	// Set up sandboxed environment
	setupSandbox(l)

	// Add debug capabilities if verbose
	if verbose {
		// Enable print function for debugging
		l.Register("print", func(l *lua.State) int {
			n := l.Top()
			fmt.Print("[DEBUG] ")
			for i := 1; i <= n; i++ {
				if i > 1 {
					fmt.Print("\t")
				}
				fmt.Print(lua.CheckString(l, i))
			}
			fmt.Println()
			return 0
		})

		// Add debug info function
		l.Register("debug_info", func(l *lua.State) int {
			info := fmt.Sprintf("Script debug info: Stack size=%d", l.Top())
			l.PushString(info)
			return 1
		})
	}

	// Convert input to Lua
	pushValue(l, input)
	l.SetGlobal("input")

	// Execute the script
	if err := lua.DoString(l, scriptContent); err != nil {
		return nil, fmt.Errorf("script error: %w", err)
	}

	// Call exec function if it exists
	l.Global("exec")
	if l.TypeOf(-1) == lua.TypeFunction {
		pushValue(l, input)
		if err := l.ProtectedCall(1, 1, 0); err != nil {
			return nil, fmt.Errorf("exec error: %w", err)
		}
		result := pullValue(l, -1)
		l.Pop(1)
		return result, nil
	}
	l.Pop(1)

	// No exec function, check if script returned a value
	if l.Top() > 0 {
		result := pullValue(l, -1)
		l.Pop(1)
		return result, nil
	}

	return input, nil
}
