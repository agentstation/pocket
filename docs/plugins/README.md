# Pocket Plugin System

The Pocket plugin system enables extending the framework with custom nodes through a progressive enhancement strategy. From built-in nodes to embedded scripting and future WebAssembly support, Pocket provides multiple ways to add custom functionality while maintaining security and performance.

## Overview

Pocket's plugin architecture consists of three levels:

1. **Built-in Nodes** - Production-ready nodes included with Pocket
2. **Lua Scripts** - Embedded scripting for custom logic (Phase 2)
3. **External Plugins** - WebAssembly and RPC plugins (Phase 3 - Future)

## Quick Start

### Using Built-in Nodes

```yaml
# workflow.yaml
nodes:
  - name: fetch-data
    type: http
    config:
      url: https://api.example.com/data
      method: GET
    
  - name: process
    type: transform
    config:
      jq: '.items | map({id, name: .title | ascii_upcase})'
      
  - name: validate
    type: validate
    config:
      schema:
        type: array
        items:
          type: object
          required: [id, name]
```

### Writing Lua Scripts

```yaml
nodes:
  - name: custom-logic
    type: lua
    config:
      script: |
        -- Access input data
        local items = input.items or {}
        
        -- Process with Lua
        local results = {}
        for i, item in ipairs(items) do
          if item.score > 0.8 then
            table.insert(results, {
              id = item.id,
              status = "high_priority",
              processed_at = os.time()
            })
          end
        end
        
        -- Return processed data
        return {
          results = results,
          count = #results
        }
```

## Available Node Types

### Built-in Nodes (14 types)

| Category | Node Type | Description |
|----------|-----------|-------------|
| **Core** | echo | Output messages and pass through data |
| | delay | Add delays to workflow execution |
| | router | Static routing based on configuration |
| | conditional | Dynamic routing with template expressions |
| **Data** | transform | Transform data using JQ expressions |
| | template | Render Go templates with data |
| | jsonpath | Extract data using JSONPath |
| | validate | Validate against JSON Schema |
| | aggregate | Collect and combine multiple inputs |
| **I/O** | http | Make HTTP requests with retry support |
| | file | Read/write files with sandboxing |
| | exec | Execute shell commands (restricted) |
| **Flow** | parallel | Execute multiple tasks concurrently |
| **Script** | lua | Execute Lua scripts |

[Full built-in node reference →](BUILT_IN.md)

## Lua Scripting

Lua scripts provide a safe, sandboxed environment for custom logic:

### Features
- Sandboxed execution (no file/network access)
- JSON encode/decode functions
- String manipulation utilities
- Timeout protection
- Access to input data and workflow state

### Example: Data Processing

```lua
-- Filter and transform data
local filtered = {}
for _, item in ipairs(input.items) do
  if item.active and item.score > threshold then
    table.insert(filtered, {
      id = item.id,
      name = string.upper(item.name),
      category = item.score > 0.9 and "premium" or "standard"
    })
  end
end

-- Return with metadata
return {
  items = filtered,
  metadata = {
    total = #input.items,
    filtered = #filtered,
    timestamp = os.time()
  }
}
```

[Lua scripting guide →](LUA.md)

## Plugin Discovery

Pocket discovers plugins from multiple locations:

1. **Built-in nodes** - Always available
2. **Script files** - From `~/.pocket/scripts/` (Phase 2)
3. **Plugin packages** - From `~/.pocket/plugins/` (Phase 3)

## CLI Commands

### Node Management
```bash
# List all available nodes
pocket nodes list

# Get detailed info about a node
pocket nodes info lua

# Generate node documentation
pocket nodes docs
```

### Script Management (Coming Soon)
```bash
# List discovered scripts
pocket scripts list

# Validate a script
pocket scripts validate my-script.lua

# Run a script directly
pocket scripts run process-data.lua < input.json
```

## Security Model

### Built-in Nodes
- Full trust and system access
- Validated configurations
- Production-ready error handling

### Lua Scripts
- Sandboxed environment
- No file system access
- No network access
- No dangerous functions (io, os.execute, require)
- CPU time limits

### Future: WebAssembly Plugins
- Capability-based permissions
- Memory isolation
- Explicit resource grants

## Best Practices

### 1. Choose the Right Level

- Use **built-in nodes** when they meet your needs
- Use **Lua scripts** for custom business logic
- Reserve **external plugins** for complex integrations

### 2. Keep Scripts Simple

Lua scripts should focus on data transformation and business logic:

```lua
-- Good: Simple, focused logic
if input.type == "order" and input.total > 100 then
  return {status = "priority", discount = 0.1}
else
  return {status = "standard", discount = 0}
end
```

### 3. Validate Inputs

Always validate data in your scripts:

```lua
-- Check for required fields
if not input.id or not input.data then
  error("Missing required fields: id and data")
end

-- Validate types
if type(input.data) ~= "table" then
  error("Data must be a table")
end
```

### 4. Handle Errors Gracefully

```lua
-- Use pcall for safe operations
local success, result = pcall(json_decode, input.raw_json)
if not success then
  return {
    error = "Invalid JSON",
    input = input.raw_json
  }
end
```

## Examples

### Data Enrichment Pipeline

```yaml
name: data-enrichment
nodes:
  - name: fetch
    type: http
    config:
      url: "https://api.source.com/data"
      
  - name: enrich
    type: lua
    config:
      file: "scripts/enrich-data.lua"
      
  - name: validate
    type: validate
    config:
      schema:
        $ref: "schemas/enriched-data.json"
        
  - name: store
    type: file
    config:
      path: "output/enriched-{{.timestamp}}.json"
      mode: write
```

### Conditional Processing

```yaml
nodes:
  - name: classify
    type: lua
    config:
      script: |
        -- Classify based on multiple criteria
        local score = 0
        
        if input.value > 1000 then score = score + 1 end
        if input.priority == "high" then score = score + 2 end
        if input.customer_tier == "premium" then score = score + 3 end
        
        local classification = "normal"
        if score >= 5 then
          classification = "urgent"
        elseif score >= 3 then
          classification = "high"
        end
        
        return {
          classification = classification,
          score = score,
          original = input
        }
```

## Roadmap

### Phase 2 (Current)
- ✅ Lua script execution
- ✅ Built-in utility functions
- ⏳ Script discovery from filesystem
- ⏳ CLI script management

### Phase 3 (Future)
- WebAssembly plugin support
- TypeScript/JavaScript SDK
- Plugin marketplace
- Hot reloading

## Next Steps

- [Built-in Node Reference](BUILT_IN.md) - Detailed documentation for all built-in nodes
- [Lua Scripting Guide](LUA.md) - Complete guide to Lua scripting
- [Go Node Development](GO.md) - Creating custom nodes in Go
- [WebAssembly Plugins](WEBASSEMBLY.md) - Future plugin development (coming soon)