# Plugin Management

Pocket's plugin system allows you to extend the graph execution engine with custom nodes. Plugins can be written in Go (native), Lua (scripts), or WebAssembly (any language that compiles to WASM).

## Overview

Plugins extend Pocket by:
- Adding new node types
- Integrating with external services
- Implementing custom business logic
- Providing language-specific functionality

## Plugin Types

### 1. Native Go Plugins

High-performance plugins written in Go:

```bash
# Install a Go plugin
pocket plugin install github.com/example/pocket-plugin-redis

# Load a local plugin
pocket plugin load ./plugins/my-plugin.so
```

### 2. Lua Scripts

Lightweight scripting for simple logic:

```bash
# Load a Lua script as a plugin
pocket plugin load ./scripts/processor.lua

# Use in workflow
pocket run workflow.yaml
```

### 3. WebAssembly Modules

Language-agnostic plugins compiled to WASM:

```bash
# Load a WASM plugin
pocket plugin load ./plugins/analyzer.wasm

# Supports plugins written in:
# - Rust, C/C++, Go, AssemblyScript, etc.
```

## Plugin Commands

### List Plugins

```bash
# List all loaded plugins
pocket plugin list

# Output:
# NAME                TYPE      VERSION   STATUS    NODES
# redis-cache         native    1.2.0     loaded    redis-get, redis-set
# data-processor      lua       1.0.0     loaded    process-data
# ml-analyzer         wasm      0.5.0     loaded    analyze-sentiment
```

### Install Plugins

```bash
# Install from registry (coming soon)
pocket plugin install redis-cache

# Install from URL
pocket plugin install https://github.com/example/pocket-plugin/releases/latest

# Install specific version
pocket plugin install redis-cache@1.2.0

# Install to custom directory
pocket plugin install redis-cache --dir ~/.pocket/custom-plugins
```

### Load Plugins

```bash
# Load a plugin file
pocket plugin load ./my-plugin.so

# Load with alias
pocket plugin load ./my-plugin.so --alias custom-processor

# Load all plugins from directory
pocket plugin load ./plugins/

# Load with configuration
pocket plugin load ./my-plugin.so --config plugin-config.yaml
```

### Unload Plugins

```bash
# Unload a specific plugin
pocket plugin unload redis-cache

# Unload all plugins
pocket plugin unload --all
```

### Plugin Information

```bash
# Show plugin details
pocket plugin info redis-cache

# Output:
# Plugin: redis-cache
# Type: native
# Version: 1.2.0
# Author: Example Corp
# Description: Redis integration for Pocket workflows
# 
# Provided Nodes:
#   - redis-get: Retrieve value from Redis
#   - redis-set: Store value in Redis
#   - redis-del: Delete key from Redis
# 
# Configuration:
#   host: Redis server host (default: localhost)
#   port: Redis server port (default: 6379)
#   password: Redis password (optional)
```

## Using Plugins in Workflows

### Native Plugin Nodes

```yaml
name: cache-workflow
start: get-data

nodes:
  - name: get-data
    type: redis-get  # Node from redis-cache plugin
    config:
      key: "user:{{.user_id}}"
      
  - name: process
    type: conditional
    config:
      conditions:
        - if: "{{.found}}"
          then: use-cache
        - else: fetch-fresh
```

### Lua Plugin Nodes

```yaml
name: lua-processing
start: transform

nodes:
  - name: transform
    type: lua  # Built-in lua node
    config:
      script: "./scripts/transform.lua"
      # Or inline:
      # source: |
      #   local data = input.data
      #   return {transformed = data * 2}
```

### WASM Plugin Nodes

```yaml
name: wasm-analysis
start: analyze

nodes:
  - name: analyze
    type: sentiment-analyzer  # Node from WASM plugin
    config:
      model: "en-US"
      threshold: 0.7
```

## Plugin Configuration

### Global Plugin Configuration

In `pocket.yaml`:

```yaml
plugins:
  # Plugin directories
  directories:
    - "./plugins"
    - "~/.pocket/plugins"
    - "/opt/pocket/plugins"
  
  # Auto-load plugins on startup
  autoload: true
  
  # Plugin-specific configuration
  config:
    redis-cache:
      host: "redis.example.com"
      port: 6379
      password: "${REDIS_PASSWORD}"
    
    ml-analyzer:
      api_endpoint: "https://ml.example.com"
      api_key: "${ML_API_KEY}"
```

### Per-Workflow Configuration

Override plugin config in workflows:

```yaml
name: configured-workflow
start: cache-get

# Workflow-specific plugin config
plugins:
  redis-cache:
    host: "redis-prod.example.com"
    database: 1

nodes:
  - name: cache-get
    type: redis-get
    config:
      key: "data:{{.id}}"
```

### Environment Variables

Plugins can access environment variables:

```bash
# Set plugin configuration via environment
export POCKET_PLUGIN_REDIS_HOST=redis.example.com
export POCKET_PLUGIN_REDIS_PORT=6380

# Run workflow
pocket run workflow.yaml
```

## Developing Plugins

### Quick Start

1. **Go Native Plugin**:
```bash
# Create plugin project
pocket plugin create my-plugin --type native

# Generates:
# my-plugin/
#   ├── go.mod
#   ├── plugin.go
#   ├── nodes.go
#   └── README.md
```

2. **Lua Script**:
```lua
-- my-processor.lua
function process(input)
    -- Your logic here
    return {
        result = input.value * 2,
        processed = true
    }
end
```

3. **WebAssembly**:
```bash
# Create WASM plugin (Rust example)
pocket plugin create my-wasm-plugin --type wasm --lang rust
```

### Plugin Interface

Native Go plugins must implement:

```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Nodes() []Node
}
```

### Testing Plugins

```bash
# Test plugin functionality
pocket plugin test ./my-plugin.so

# Test with specific input
pocket plugin test ./my-plugin.so --input data.json

# Run plugin benchmarks
pocket plugin bench ./my-plugin.so
```

## Plugin Security

### Sandboxing

Plugins run in sandboxed environments:

- **Lua**: Restricted environment, no file/network access by default
- **WASM**: Full sandboxing with capability-based security
- **Native**: OS-level protections, review code carefully

### Permissions

Configure plugin permissions:

```yaml
plugins:
  security:
    # Default permissions for all plugins
    default_permissions:
      - read_env
      - http_client
    
    # Plugin-specific permissions
    permissions:
      untrusted-plugin:
        - none  # No permissions
      
      trusted-plugin:
        - read_env
        - write_files
        - network_access
```

### Verification

```bash
# Verify plugin signature (coming soon)
pocket plugin verify ./my-plugin.so

# Show plugin permissions
pocket plugin permissions redis-cache
```

## Plugin Registry (Coming Soon)

### Browse Plugins

```bash
# Search registry
pocket plugin search redis

# Show categories
pocket plugin categories

# List popular plugins
pocket plugin popular
```

### Publish Plugins

```bash
# Login to registry
pocket plugin login

# Publish plugin
pocket plugin publish ./my-plugin.so

# Update plugin
pocket plugin publish ./my-plugin.so --version 1.2.0
```

## Best Practices

1. **Version plugins** - Use semantic versioning
2. **Document node types** - Clear descriptions and examples
3. **Handle errors gracefully** - Don't crash the engine
4. **Respect timeouts** - Honor context cancellation
5. **Minimize dependencies** - Keep plugins lightweight
6. **Test thoroughly** - Unit and integration tests
7. **Provide examples** - Show how to use your plugin

## Troubleshooting

### Plugin won't load

```bash
# Check plugin compatibility
pocket plugin check ./my-plugin.so

# View detailed error
POCKET_LOG_LEVEL=debug pocket plugin load ./my-plugin.so

# Verify plugin file
file ./my-plugin.so
```

### Plugin crashes

```bash
# Run with debugging
pocket run workflow.yaml --debug-plugins

# Disable problematic plugin
pocket run workflow.yaml --disable-plugin problematic-plugin
```

### Performance issues

```bash
# Profile plugin execution
pocket plugin profile redis-cache --workflow cache-heavy.yaml

# Show plugin metrics
pocket plugin metrics
```

## Examples

### Redis Cache Plugin

```yaml
name: user-cache
start: check-cache

nodes:
  - name: check-cache
    type: redis-get
    config:
      key: "user:{{.user_id}}"
      
  - name: fetch-from-db
    type: http
    config:
      url: "{{.db_api}}/users/{{.user_id}}"
      
  - name: update-cache
    type: redis-set
    config:
      key: "user:{{.user_id}}"
      value: "{{.}}"
      ttl: "1h"

connections:
  - from: check-cache
    to: return-cached
    when: found
  - from: check-cache
    to: fetch-from-db
    when: not_found
  - from: fetch-from-db
    to: update-cache
```

### Data Processing with Lua

```yaml
name: data-pipeline
start: process

nodes:
  - name: process
    type: lua
    config:
      script: |
        -- Complex data transformation
        local result = {}
        for i, item in ipairs(input.items) do
          if item.active then
            table.insert(result, {
              id = item.id,
              value = item.value * 1.1,
              processed_at = os.time()
            })
          end
        end
        return {items = result}
```

## Next Steps

- Read [Plugin Development Guide](../development/plugin-development.md)
- Explore [Lua Scripting](../nodes/lua-scripts.md)
- Learn about [WebAssembly Plugins](../nodes/wasm-plugins.md)
- See [Plugin Examples](../../plugins/examples/)