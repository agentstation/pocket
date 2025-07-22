# Pocket CLI Command Reference

Complete reference for all Pocket CLI commands and options.

## Global Options

These options are available for all commands:

```bash
pocket [command] [flags]

Global Flags:
  -h, --help           Help for any command
  -v, --verbose        Enable verbose output
      --output string  Output format: text, json, yaml (default "text")
      --no-color       Disable colored output
```

## Commands

### pocket run

Execute a workflow from a YAML or JSON file.

```bash
pocket run <workflow-file> [flags]
```

**Flags:**
- `--dry-run` - Validate workflow without executing
- `--store-type string` - Store type: memory, bounded (default "memory")
- `--max-entries int` - Max entries for bounded store (default 10000)
- `--ttl duration` - TTL for store entries
- `--input string` - Input data as JSON string
- `--input-file string` - Input data from file

**Examples:**
```bash
# Run a workflow
pocket run workflow.yaml

# Dry run to validate
pocket run workflow.yaml --dry-run

# With verbose output
pocket run workflow.yaml --verbose

# With custom store settings
pocket run workflow.yaml --store-type bounded --max-entries 5000 --ttl 30m

# With input data
pocket run workflow.yaml --input '{"name": "test"}'
pocket run workflow.yaml --input-file data.json
```

### pocket nodes

Manage and inspect available nodes.

#### pocket nodes list

List all available node types.

```bash
pocket nodes list [flags]
```

**Flags:**
- `--category string` - Filter by category (core, data, io, flow, script)
- `--format string` - Output format: table, json, yaml (default "table")

**Examples:**
```bash
# List all nodes
pocket nodes list

# List only data nodes
pocket nodes list --category data

# Output as JSON
pocket nodes list --format json
```

#### pocket nodes info

Show detailed information about a node type.

```bash
pocket nodes info <node-type> [flags]
```

**Examples:**
```bash
# Get info about http node
pocket nodes info http

# Get info as JSON
pocket nodes info transform --output json
```

#### pocket nodes docs

Generate node documentation.

```bash
pocket nodes docs [flags]
```

**Flags:**
- `--output string` - Output file (default stdout)
- `--format string` - Format: markdown, json (default "markdown")

### pocket scripts

Manage Lua scripts.

#### pocket scripts list

List discovered Lua scripts.

```bash
pocket scripts list [flags]
```

**Flags:**
- `--path string` - Script directory (default "~/.pocket/scripts")

**Examples:**
```bash
# List all scripts
pocket scripts list

# List from custom directory
pocket scripts list --path ./my-scripts
```

#### pocket scripts validate

Validate a Lua script.

```bash
pocket scripts validate <script-path> [flags]
```

**Examples:**
```bash
# Validate a script
pocket scripts validate ~/.pocket/scripts/processor.lua

# Validate with verbose output
pocket scripts validate script.lua --verbose
```

#### pocket scripts info

Show script metadata and information.

```bash
pocket scripts info <script-name> [flags]
```

**Examples:**
```bash
# Get script info
pocket scripts info data-processor

# As JSON
pocket scripts info sentiment-analyzer --output json
```

#### pocket scripts run

Run a script directly (for testing).

```bash
pocket scripts run <script-name> [input] [flags]
```

**Examples:**
```bash
# Run with test input
pocket scripts run processor '{"test": "data"}'

# Run with input from file
pocket scripts run analyzer --input-file test.json
```

### pocket plugins

Manage WebAssembly plugins.

#### pocket plugins list

List installed plugins.

```bash
pocket plugins list [flags]
```

**Flags:**
- `--path string` - Plugin directory (default "~/.pocket/plugins")

**Examples:**
```bash
# List all plugins
pocket plugins list

# List with details
pocket plugins list --verbose
```

#### pocket plugins install

Install a plugin from a directory or archive.

```bash
pocket plugins install <path> [flags]
```

**Flags:**
- `--name string` - Override plugin name
- `--force` - Overwrite existing plugin

**Examples:**
```bash
# Install from directory
pocket plugins install ./my-plugin

# Install with custom name
pocket plugins install ./plugin --name custom-processor

# Force overwrite
pocket plugins install ./updated-plugin --force
```

#### pocket plugins remove

Remove an installed plugin.

```bash
pocket plugins remove <plugin-name> [flags]
```

**Examples:**
```bash
# Remove a plugin
pocket plugins remove sentiment-analyzer

# Remove with confirmation skip
pocket plugins remove old-plugin --yes
```

#### pocket plugins info

Show detailed plugin information.

```bash
pocket plugins info <plugin-name> [flags]
```

**Examples:**
```bash
# Get plugin details
pocket plugins info word-counter

# Show as JSON
pocket plugins info transformer --output json
```

#### pocket plugins validate

Validate a plugin without installing.

```bash
pocket plugins validate <path> [flags]
```

**Examples:**
```bash
# Validate plugin structure
pocket plugins validate ./new-plugin

# Validate with verbose output
pocket plugins validate ./plugin --verbose
```

#### pocket plugins run

Test a plugin function directly.

```bash
pocket plugins run <plugin> <node> <function> [input] [flags]
```

**Examples:**
```bash
# Test plugin function
pocket plugins run analyzer sentiment exec '{"text": "Great product!"}'

# Test with prep/post
pocket plugins run processor transform prep '{"data": "test"}'
```

### pocket version

Display version information.

```bash
pocket version [flags]
```

**Flags:**
- `--json` - Output as JSON
- `--check-update` - Check for updates (coming soon)

**Examples:**
```bash
# Show version
pocket version

# As JSON
pocket version --json
```

## Configuration

### Configuration File

Pocket looks for configuration in these locations (in order):
1. `./pocket.yaml`
2. `~/.pocket/config.yaml`
3. `/etc/pocket/config.yaml`

Example configuration:
```yaml
# pocket.yaml
log_level: info
store:
  type: bounded
  max_entries: 10000
  ttl: 30m
  
plugins:
  path: ~/.pocket/plugins
  
scripts:
  path: ~/.pocket/scripts
  
defaults:
  timeout: 30s
  retry:
    max_attempts: 3
    delay: 1s
```

### Environment Variables

```bash
# Set Pocket home directory
export POCKET_HOME="$HOME/.pocket"

# Set log level
export POCKET_LOG_LEVEL="debug"

# Set default store type
export POCKET_STORE_TYPE="bounded"

# Disable color output
export POCKET_NO_COLOR="true"
```

## Exit Codes

Pocket uses standard exit codes:

- `0` - Success
- `1` - General error
- `2` - Usage error (invalid arguments)
- `3` - Workflow validation error
- `4` - Workflow execution error
- `5` - Plugin error

## Examples

### Complete Workflow Example

```bash
# Create a workflow
cat > pipeline.yaml << EOF
name: data-pipeline
start: fetch

nodes:
  - name: fetch
    type: http
    config:
      url: "https://api.example.com/data"
      
  - name: validate
    type: validate
    config:
      schema:
        type: object
        required: [id, value]
        
  - name: transform
    type: transform
    config:
      jq: ".value * 1.1"

connections:
  - from: fetch
    to: validate
  - from: validate
    to: transform
EOF

# Validate it
pocket run pipeline.yaml --dry-run

# Run it
pocket run pipeline.yaml --verbose

# Run with custom settings
pocket run pipeline.yaml \
  --store-type bounded \
  --max-entries 1000 \
  --ttl 10m
```

### Plugin Management Example

```bash
# Install a plugin
pocket plugins install ./sentiment-analyzer

# Verify installation
pocket plugins list
pocket plugins info sentiment-analyzer

# Use in workflow
cat > analyze.yaml << EOF
name: sentiment-analysis
start: analyze

nodes:
  - name: analyze
    type: sentiment
    config:
      threshold: 0.7
EOF

pocket run analyze.yaml --input '{"text": "This is amazing!"}'

# Remove when done
pocket plugins remove sentiment-analyzer
```

## Troubleshooting

### Common Issues

**Workflow not found:**
```bash
pocket run workflow.yaml
# Error: workflow file not found

# Fix: Check file path
ls -la workflow.yaml
```

**Invalid YAML:**
```bash
pocket run workflow.yaml --dry-run
# Error: yaml: line 5: found unexpected ':'

# Fix: Validate YAML syntax
```

**Node type not found:**
```bash
# Check available nodes
pocket nodes list

# Check if plugin is installed
pocket plugins list
```

**Permission denied:**
```bash
# Fix: Check file permissions
chmod +r workflow.yaml

# For plugins
chmod +x ~/.pocket/plugins/*/plugin.wasm
```

### Debug Mode

For detailed debugging:
```bash
# Maximum verbosity
POCKET_LOG_LEVEL=debug pocket run workflow.yaml --verbose

# With execution trace
POCKET_TRACE=true pocket run workflow.yaml
```