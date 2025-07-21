# Pocket Unified CLI Specification

## Overview

This document specifies the design and implementation plan for unifying the `pocket` and `pocket-plugins` binaries into a single, cohesive command-line interface. The unified CLI will provide a better user experience while maintaining backward compatibility where possible.

## Goals

1. **Single Binary**: Combine `pocket` and `pocket-plugins` into one executable
2. **Consistent Interface**: Use Cobra framework throughout for uniform command structure
3. **Backward Compatibility**: Maintain existing `pocket` command behavior
4. **Modular Design**: Keep plugin functionality separate and optional
5. **Improved UX**: Natural command hierarchy and better help/documentation

## Command Structure

### Top-Level Commands

```
pocket
├── run          # Execute workflows (existing)
├── nodes        # Node management (existing)
├── scripts      # Script management (existing)
├── plugins      # Plugin management (new, from pocket-plugins)
├── version      # Version information (existing)
└── help         # Help about any command
```

### Detailed Command Hierarchy

#### 1. Run Command (Existing)
```bash
pocket run <workflow.yaml> [flags]
  --verbose, -v       Enable verbose output
  --dry-run          Validate without executing
  --store-type       Store type: memory or bounded
  --max-entries      Max entries for bounded store
  --ttl              TTL for store entries
```

#### 2. Nodes Command (Existing)
```bash
pocket nodes                    # List all available nodes
pocket nodes info <type>        # Show detailed info about a node type
pocket nodes docs               # Generate node documentation
```

#### 3. Scripts Command (Existing)
```bash
pocket scripts                       # List discovered scripts
pocket scripts validate <path>       # Validate a Lua script
pocket scripts info <name>          # Show script details
pocket scripts run <name> [input]   # Run a script directly
```

#### 4. Plugins Command (New - from pocket-plugins)
```bash
pocket plugins                      # Show plugin help
pocket plugins list                 # List installed plugins
pocket plugins install <path>       # Install a plugin
pocket plugins remove <name>        # Remove a plugin
pocket plugins info <name>          # Show plugin details
pocket plugins validate <path>      # Validate plugin without installing
pocket plugins run <name> <node> <function>  # Test plugin function
```

#### 5. Version Command (Enhanced)
```bash
pocket version                      # Show version info
pocket version --json              # JSON output
pocket version --check-update      # Check for updates (future)
```

## Implementation Plan

### Phase 1: Cobra Migration for Existing Commands

1. **Create Cobra command structure**
   - Root command with global flags
   - Subcommands for run, nodes, scripts, version
   - Maintain exact same functionality

2. **File Structure**
   ```
   cmd/pocket/
   ├── main.go              # Entry point
   ├── root.go              # Root command setup
   ├── run.go               # Run command
   ├── nodes.go             # Nodes commands
   ├── scripts.go           # Scripts commands
   ├── plugins.go           # Plugins commands (new)
   ├── version.go           # Version command
   └── helpers.go           # Shared utilities
   ```

3. **Global Flags**
   ```go
   // Global flags available to all commands
   var (
       verbose bool
       output  string  // "text", "json", "yaml"
       noColor bool
   )
   ```

### Phase 2: Integrate Plugin Commands

1. **Move plugin functionality**
   - Copy plugin commands from pocket-plugins
   - Integrate under `pocket plugins` subcommand
   - Reuse existing plugin/loader packages

2. **Conditional compilation (optional)**
   ```go
   // +build !minimal
   
   // Plugin commands only included in full build
   ```

### Phase 3: Testing Strategy

1. **Unit Tests**
   - Test each command handler function
   - Mock file system and plugin operations
   - Verify command parsing and flag handling

2. **Integration Tests**
   ```go
   // Test actual command execution
   func TestCLICommands(t *testing.T) {
       tests := []struct {
           name     string
           args     []string
           wantErr  bool
           contains string
       }{
           {
               name:     "run workflow",
               args:     []string{"run", "testdata/workflow.yaml"},
               wantErr:  false,
               contains: "completed",
           },
           {
               name:     "list nodes",
               args:     []string{"nodes"},
               wantErr:  false,
               contains: "echo",
           },
           // ... more tests
       }
   }
   ```

3. **Backward Compatibility Tests**
   - Ensure old command formats still work
   - Test flag compatibility
   - Verify output format consistency

### Phase 4: Documentation Updates

1. **Update README.md**
   - Single installation instruction
   - New command structure
   - Migration guide

2. **Generate command documentation**
   - Use Cobra's built-in doc generation
   - Create man pages
   - Update online docs

## Migration Strategy

### For Existing Users

1. **Deprecation Notice**
   - Add deprecation warning to pocket-plugins
   - Point users to unified `pocket plugins` command
   - Provide migration timeline

2. **Transition Period**
   - Keep pocket-plugins binary for 2-3 releases
   - Show deprecation warnings
   - Eventually remove in major version

3. **Migration Guide**
   ```
   Old Command                    → New Command
   pocket-plugins list            → pocket plugins list
   pocket-plugins install <path>  → pocket plugins install <path>
   pocket-plugins remove <name>   → pocket plugins remove <name>
   ```

## Technical Considerations

### 1. Binary Size

- Current pocket: ~15MB
- Current pocket-plugins: ~11MB
- Expected unified: ~20-25MB
- Acceptable increase for improved functionality

### 2. Dependencies

```go
// go.mod additions/changes
require (
    github.com/spf13/cobra v1.9.1      // CLI framework
    github.com/spf13/pflag v1.0.5      // Enhanced flags
    github.com/tetratelabs/wazero v1.9.0  // WASM runtime (existing)
)
```

### 3. Build Options

```makefile
# Makefile targets
build:          ## Build full CLI with all features
build-minimal:  ## Build without plugin support (smaller binary)
```

### 4. Error Handling

- Consistent error messages across all commands
- Proper exit codes (0 = success, 1 = error, 2 = usage error)
- Structured errors for JSON output mode

## Success Criteria

1. **Functionality**: All existing commands work identically
2. **Performance**: No regression in command execution time
3. **Size**: Binary size increase < 50%
4. **Testing**: >80% code coverage for CLI commands
5. **Documentation**: Complete command reference
6. **Compatibility**: Existing scripts/workflows continue to work

## Implementation Timeline

1. **Week 1**: Cobra migration for existing commands
2. **Week 2**: Plugin command integration
3. **Week 3**: Testing and documentation
4. **Week 4**: Release preparation and migration guide

## Future Enhancements

1. **Interactive Mode**: `pocket repl` for interactive workflow development
2. **Auto-completion**: Shell completion for bash/zsh/fish
3. **Update Command**: `pocket update` to self-update
4. **Config Management**: `pocket config` for managing settings
5. **Plugin Registry**: `pocket plugins search` for discovering plugins

## Example Usage

### Before (Two Binaries)
```bash
# Install both tools
go install github.com/agentstation/pocket/cmd/pocket@latest
go install github.com/agentstation/pocket/cmd/pocket-plugins@latest

# Use separately
pocket run workflow.yaml
pocket-plugins list
```

### After (Single Binary)
```bash
# Install one tool
go install github.com/agentstation/pocket/cmd/pocket@latest

# Unified commands
pocket run workflow.yaml
pocket plugins list
```

## Testing Plan

### Unit Test Coverage

1. **Command Tests**
   - Each command has dedicated test file
   - Mock external dependencies
   - Test flag parsing and validation

2. **Integration Tests**
   - Full command execution tests
   - Test with real files and plugins
   - Verify end-to-end workflows

3. **Regression Tests**
   - Ensure backward compatibility
   - Test deprecated command formats
   - Verify output consistency

### Test Structure
```
cmd/pocket/
├── run_test.go         # Run command tests
├── nodes_test.go       # Nodes command tests
├── scripts_test.go     # Scripts command tests
├── plugins_test.go     # Plugins command tests
├── integration_test.go # End-to-end tests
└── testdata/          # Test fixtures
    ├── workflows/
    ├── scripts/
    └── plugins/
```

## Risks and Mitigation

1. **Risk**: Breaking existing user workflows
   - **Mitigation**: Extensive backward compatibility testing
   
2. **Risk**: Increased binary size affects deployment
   - **Mitigation**: Provide minimal build option
   
3. **Risk**: Command naming conflicts
   - **Mitigation**: Careful command hierarchy design
   
4. **Risk**: Plugin functionality adds complexity
   - **Mitigation**: Keep plugin code isolated and optional

## Conclusion

This unified CLI design will provide a better user experience while maintaining the simplicity and power of Pocket. The migration to Cobra enables more sophisticated command structures and better help documentation, while the integration of plugin commands creates a single, cohesive tool for all Pocket functionality.