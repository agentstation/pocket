# CLI Migration Guide

This guide helps users migrate from the separate `pocket-plugins` binary to the unified `pocket` CLI.

## Overview

Starting from version X.X.X, the `pocket-plugins` binary has been integrated into the main `pocket` CLI. All plugin functionality is now available through the `pocket plugins` command.

## Command Mapping

### Old Commands (pocket-plugins)

```bash
# List plugins
pocket-plugins list

# Install a plugin
pocket-plugins install plugin.wasm

# Remove a plugin
pocket-plugins remove plugin-name

# Get plugin info
pocket-plugins info plugin-name
```

### New Commands (pocket)

```bash
# List plugins
pocket plugins list

# Install a plugin
pocket plugins install plugin.wasm

# Remove a plugin
pocket plugins remove plugin-name

# Get plugin info
pocket plugins info plugin-name
```

## Migration Steps

1. **Update to the latest version**
   ```bash
   go install github.com/agentstation/pocket/cmd/pocket@latest
   ```

2. **Remove the old binary**
   ```bash
   rm $(which pocket-plugins)
   ```

3. **Verify installation**
   ```bash
   pocket --version
   pocket plugins --help
   ```

## Feature Parity

All features from `pocket-plugins` are available in the unified CLI:

- ✅ List installed plugins
- ✅ Install new plugins
- ✅ Remove plugins
- ✅ View plugin information
- ✅ Custom plugin names during installation

## Additional Features

The unified CLI provides additional benefits:

1. **Single binary** - Simpler installation and distribution
2. **Consistent interface** - Same flags and options across all commands
3. **Better help** - Improved command documentation and examples
4. **Global flags** - Verbose output and format options work everywhere

## Examples

### Installing a plugin with custom name
```bash
# Old way
pocket-plugins install ./my-plugin.wasm --name custom-name

# New way
pocket plugins install ./my-plugin.wasm --name custom-name
```

### Listing plugins with verbose output
```bash
# Old way
pocket-plugins list -v

# New way
pocket plugins list --verbose
```

### Removing a plugin with force
```bash
# Old way
pocket-plugins remove my-plugin -f

# New way
pocket plugins remove my-plugin --force
```

## Plugin Location

Plugins continue to be stored in the same location:
- `~/.pocket/plugins/`

No migration of existing plugins is needed.

## Troubleshooting

### Command not found
If you get "command not found" errors:
1. Ensure you've installed the latest version
2. Check that `$GOPATH/bin` is in your PATH
3. Try using the full path: `$(go env GOPATH)/bin/pocket`

### Permissions issues
If you encounter permission errors:
```bash
chmod +x $(which pocket)
```

### Plugin compatibility
All existing WebAssembly plugins remain compatible. No changes to plugins are required.

## Getting Help

```bash
# General help
pocket --help

# Plugin command help
pocket plugins --help

# Specific command help
pocket plugins install --help
```

## Reporting Issues

If you encounter any issues during migration, please report them at:
https://github.com/agentstation/pocket/issues