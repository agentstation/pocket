# Phase 2: Script Management Features

## Overview

This PR implements the missing script management features from Phase 2 of the Pocket plugin system. While the core Lua scripting functionality was completed, the convenience features for managing scripts were not implemented.

## Features to Implement

### 1. Script Discovery from ~/.pocket/scripts
- Automatic discovery of Lua scripts from the user's script directory
- Scripts will be available as node types without explicit registration
- Support for organized subdirectories (e.g., ~/.pocket/scripts/data/, ~/.pocket/scripts/ai/)

### 2. Script Validation Command
- New CLI command: `pocket scripts validate <script-path>`
- Validates Lua syntax without executing
- Checks for required functions/structure
- Reports errors with line numbers

### 3. Script Debugging Support
- Add debug mode for Lua execution
- Support for print/log statements during development
- Better error messages with stack traces
- Optional verbose output showing script execution flow

## Implementation Plan

### Script Discovery

1. Add script discovery to the registry initialization
2. Scan ~/.pocket/scripts directory recursively
3. Load script metadata from comments or companion files
4. Register discovered scripts as node types

### CLI Commands

Add new script subcommand to the CLI:
```
pocket scripts list              # List discovered scripts
pocket scripts validate <path>   # Validate a script
pocket scripts info <name>       # Show script details
pocket scripts run <name>        # Run a script directly (for testing)
```

### Script Structure

Standardize script structure for discovery:
```lua
-- @name: my-processor
-- @category: data
-- @description: Process data with custom logic
-- @version: 1.0.0

function prep(input, store)
    -- Preparation logic
    return prepared_data
end

function exec(prep_data)
    -- Execution logic
    return result
end

function post(input, prep_data, result, store)
    -- Post-processing logic
    return result, "next"
end
```

### Debugging Features

1. Add `--debug` flag to workflow execution
2. Enable Lua debug library in debug mode
3. Capture and display print statements
4. Show execution timing for each phase
5. Add stack traces for errors

## Benefits

1. **Developer Experience**: Easier to manage and organize scripts
2. **Validation**: Catch errors before runtime
3. **Debugging**: Faster development cycle with better error information
4. **Discovery**: Scripts automatically available without manual registration

## Testing

- Unit tests for script discovery
- Integration tests for CLI commands
- Test scripts with various error conditions
- Performance tests for script loading

## Documentation

- Update Lua scripting guide with new features
- Add examples of discoverable scripts
- Document debugging workflow
- Create script development best practices

## Migration

No breaking changes - existing inline and file-based scripts continue to work. The new features are additive.

## Future Enhancements

- Script hot-reloading during development
- Script package manager for sharing
- Visual debugging interface
- Performance profiling for scripts