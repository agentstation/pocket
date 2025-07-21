# Lua Script Examples

This directory contains example Lua scripts and workflows demonstrating the Pocket Lua scripting capabilities.

## Workflow Examples

### lua-scripting.yaml
Basic Lua scripting demonstration showing:
- Data transformation
- JSON encoding/decoding
- String manipulation
- Validation logic

### lua-file-script.yaml
Example of using external Lua script files:
- Loading scripts from files
- Complex data processing
- Template integration

## Script Examples

### scripts/data_processor.lua
Comprehensive data processing script showing:
- Number statistics calculation
- Text analysis and transformation
- JSON data handling
- Error handling patterns

## Running the Examples

```bash
# Run a workflow with inline Lua script
pocket run docs/plugins/examples/lua/lua-scripting.yaml

# Run a workflow with external script
pocket run docs/plugins/examples/lua/lua-file-script.yaml

# Run a script directly (coming soon)
pocket scripts run docs/plugins/examples/lua/scripts/data_processor.lua
```

## Creating Your Own Scripts

1. Start with a simple inline script in your workflow
2. Move complex logic to external `.lua` files
3. Use the provided utility functions (json_encode, str_trim, etc.)
4. Test thoroughly with different inputs
5. Add error handling for production use

See the [Lua Scripting Guide](../../LUA.md) for detailed documentation.