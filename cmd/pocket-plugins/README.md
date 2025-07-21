# Pocket Plugin CLI

Command-line tool for managing Pocket plugins.

## Installation

```bash
go install github.com/agentstation/pocket/cmd/pocket-plugins@latest
```

Or build from source:

```bash
go build -o pocket-plugins ./cmd/pocket-plugins
```

## Commands

### List Plugins

List all installed plugins:

```bash
pocket-plugins list
```

Output:
```
NAME                VERSION    RUNTIME    NODES    PATH
----                -------    -------    -----    ----
sentiment-analyzer  1.0.0      wasm       1        ~/.pocket/plugins/sentiment-analyzer
word-counter        1.0.0      wasm       1        ~/.pocket/plugins/word-counter
```

### Install Plugin

Install a plugin from a directory:

```bash
pocket-plugins install ./my-plugin

# Example output:
✓ Installed plugin 'my-plugin' version 1.0.0
  Location: ~/.pocket/plugins/my-plugin
  Nodes: 2
    - process (data): Process data with custom logic
    - validate (data): Validate input data
```

### Remove Plugin

Remove an installed plugin:

```bash
pocket-plugins remove my-plugin

# Prompts for confirmation:
Are you sure you want to remove plugin 'my-plugin'? [y/N]: y
✓ Removed plugin 'my-plugin'
```

### Plugin Info

Show detailed information about a plugin:

```bash
pocket-plugins info sentiment-analyzer

# Output:
Plugin: sentiment-analyzer
Version: 1.0.0
Description: Sentiment analysis plugin for Pocket
Author: Pocket Team
License: MIT
Runtime: wasm

Permissions:
  Memory: 10MB
  Timeout: 5s

Requirements:
  Pocket: >=1.0.0

Nodes (1):

  sentiment:
    Category: ai
    Description: Analyze text sentiment with ML
```

### Validate Plugin

Validate a plugin before installation:

```bash
pocket-plugins validate ./my-plugin

# Output:
Validating plugin at ./my-plugin...

✓ Manifest is valid
✓ Plugin name: my-plugin
✓ Version: 1.0.0
✓ Runtime: wasm
✓ Binary found: plugin.wasm
✓ Nodes defined: 2
✓ Plugin responds to calls (156 bytes)

Validation passed!
```

### Run Plugin (Testing)

Run a specific plugin function for testing:

```bash
pocket-plugins run sentiment-analyzer sentiment exec

# Note: This command is for testing only and requires additional implementation
```

## Plugin Directory Structure

Plugins are installed to standard locations:

1. **User plugins**: `~/.pocket/plugins/`
2. **System plugins**: `/usr/local/share/pocket/plugins/`
3. **Local plugins**: `./plugins/`

Each plugin directory should contain:
- `manifest.yaml` or `manifest.json` - Plugin metadata
- `plugin.wasm` - The compiled WebAssembly binary
- Additional assets (optional)

## Plugin Manifest

Example `manifest.yaml`:

```yaml
name: my-plugin
version: 1.0.0
description: My custom plugin
author: Your Name
license: MIT
runtime: wasm
binary: plugin.wasm

nodes:
  - type: process
    category: data
    description: Process data

permissions:
  memory: 10MB
  timeout: 5s

requirements:
  pocket: ">=1.0.0"
```

## Environment Variables

- `POCKET_PLUGIN_PATH`: Additional paths to search for plugins (colon-separated)

## Error Handling

The CLI provides clear error messages:

```bash
# Plugin not found
pocket-plugins info nonexistent
Error: plugin 'nonexistent' not found

# Invalid plugin
pocket-plugins validate ./invalid-plugin
Error: validation failed: plugin name is required

# Already installed
pocket-plugins install ./my-plugin
Error: plugin 'my-plugin' is already installed
```

## Contributing

To add new commands or features:

1. Edit `cmd/pocket-plugins/main.go`
2. Add command implementation
3. Update this README
4. Submit a pull request