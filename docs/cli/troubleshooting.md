# Troubleshooting Guide

This guide helps you diagnose and resolve common issues when using Pocket CLI.

## Common Issues

### Installation Problems

#### Go install fails

**Problem**: `go install` command fails with errors

**Solutions**:

1. Check Go version:
```bash
go version
# Should be 1.21 or higher
```

2. Update Go modules:
```bash
go clean -modcache
go install github.com/agentstation/pocket/cmd/pocket@latest
```

3. Use explicit version:
```bash
go install github.com/agentstation/pocket/cmd/pocket@v1.0.0
```

#### Binary not found

**Problem**: `pocket: command not found` after installation

**Solutions**:

1. Check if Go bin is in PATH:
```bash
echo $PATH | grep -q "$(go env GOPATH)/bin" && echo "Go bin is in PATH" || echo "Go bin is NOT in PATH"
```

2. Add Go bin to PATH:
```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$PATH:$(go env GOPATH)/bin"
```

3. Use full path:
```bash
$(go env GOPATH)/bin/pocket run workflow.yaml
```

### Workflow Execution Issues

#### Workflow file not found

**Problem**: `Error: workflow file not found`

**Solutions**:

1. Check file path:
```bash
ls -la workflow.yaml
pwd  # Verify current directory
```

2. Use absolute path:
```bash
pocket run /full/path/to/workflow.yaml
```

3. Check file extension:
```bash
# Pocket supports .yaml, .yml, and .json
pocket run workflow.yml  # Try alternative extension
```

#### YAML parsing errors

**Problem**: `Error: yaml: line X: found character that cannot start any token`

**Solutions**:

1. Validate YAML syntax:
```bash
# Use yamllint if available
yamllint workflow.yaml

# Or use Pocket's validate command
pocket validate workflow.yaml
```

2. Common YAML issues:
```yaml
# Wrong - tabs not allowed
nodes:
	- name: test  # Tab used here

# Correct - use spaces
nodes:
  - name: test  # 2 spaces

# Wrong - missing space after colon
key:value

# Correct
key: value
```

3. Check for special characters:
```yaml
# Wrong - unquoted special characters
config:
  message: Hello: World  # Colon needs quotes

# Correct
config:
  message: "Hello: World"
```

#### Node type not found

**Problem**: `Error: unknown node type: custom-node`

**Solutions**:

1. List available nodes:
```bash
pocket nodes list
```

2. Check if plugin is loaded:
```bash
pocket plugin list
```

3. Load required plugin:
```bash
pocket plugin load ./plugins/custom-plugin.so
```

4. Use built-in node:
```yaml
# Check spelling of built-in nodes
nodes:
  - name: echo-message
    type: echo  # Not "print" or "log"
```

### Runtime Errors

#### Context deadline exceeded

**Problem**: `Error: context deadline exceeded`

**Solutions**:

1. Increase timeout:
```bash
# Global timeout
pocket run workflow.yaml --timeout 5m

# Node-specific timeout
```
```yaml
nodes:
  - name: slow-operation
    type: http
    config:
      url: "https://slow-api.example.com"
      timeout: "30s"  # Increase node timeout
```

2. Check network connectivity:
```bash
# Test endpoint manually
curl -v https://api.example.com/endpoint
```

3. Enable debug logging:
```bash
POCKET_LOG_LEVEL=debug pocket run workflow.yaml
```

#### Memory issues

**Problem**: `runtime: out of memory`

**Solutions**:

1. Limit store size:
```yaml
# In pocket.yaml
store:
  max_entries: 1000  # Reduce from default
  ttl: "5m"          # Add TTL for cleanup
```

2. Process data in batches:
```yaml
nodes:
  - name: batch-processor
    type: batch
    config:
      size: 100  # Process 100 items at a time
```

3. Monitor memory usage:
```bash
# Run with memory profiling
pocket run workflow.yaml --profile-mem
```

#### Permission denied

**Problem**: `Error: permission denied`

**Solutions**:

1. Check file permissions:
```bash
ls -la workflow.yaml
chmod 644 workflow.yaml  # Make readable
```

2. For exec nodes:
```bash
# Make script executable
chmod +x ./scripts/process.sh
```

3. For plugin loading:
```bash
# Check plugin file permissions
ls -la ./plugins/
chmod 755 ./plugins/my-plugin.so
```

### Plugin Issues

#### Plugin fails to load

**Problem**: `Error: failed to load plugin: symbol not found`

**Solutions**:

1. Check plugin compatibility:
```bash
# Verify plugin was built with same Go version
go version
pocket plugin check ./plugin.so
```

2. Rebuild plugin:
```bash
cd plugin-source/
go build -buildmode=plugin -o plugin.so
```

3. Check dependencies:
```bash
# List plugin dependencies
ldd ./plugin.so  # Linux
otool -L ./plugin.so  # macOS
```

#### Lua script errors

**Problem**: `Error in Lua script: attempt to index nil value`

**Solutions**:

1. Check input data:
```lua
-- Add defensive checks
function process(input)
    if not input then
        error("Input is nil")
    end
    
    if not input.data then
        return {error = "Missing data field"}
    end
    
    -- Process safely
    return {result = input.data * 2}
end
```

2. Debug Lua script:
```yaml
nodes:
  - name: debug-lua
    type: lua
    config:
      debug: true  # Enable debug output
      script: |
        print("Input:", json.encode(input))
        -- Your logic here
```

### Configuration Issues

#### Config file not loading

**Problem**: Configuration in `pocket.yaml` not being applied

**Solutions**:

1. Check config file location:
```bash
# Show where Pocket looks for config
pocket config paths

# Show loaded configuration
pocket config show
```

2. Validate config syntax:
```bash
pocket config validate
```

3. Force config file:
```bash
pocket run workflow.yaml --config ./my-config.yaml
```

#### Environment variables not working

**Problem**: `${VARIABLE}` not being replaced

**Solutions**:

1. Check variable is set:
```bash
echo $MY_VARIABLE
env | grep MY_VARIABLE
```

2. Export variable:
```bash
export MY_VARIABLE="value"
pocket run workflow.yaml
```

3. Use env file:
```bash
# .env file
MY_VARIABLE=value

# Run with env file
pocket run workflow.yaml --env-file .env
```

## Debugging Techniques

### Enable Debug Logging

```bash
# Maximum verbosity
POCKET_LOG_LEVEL=debug pocket run workflow.yaml

# Or via flag
pocket run workflow.yaml --log-level debug

# Log to file
pocket run workflow.yaml --log-file debug.log
```

### Dry Run Mode

Test without execution:

```bash
# Validate and show execution plan
pocket run workflow.yaml --dry-run

# Shows:
# - Node execution order
# - Configuration validation
# - Type checking results
```

### Step-by-Step Execution

```bash
# Pause after each node
pocket run workflow.yaml --step

# Interactive mode
pocket run workflow.yaml --interactive
```

### Export Execution Trace

```bash
# Save execution details
pocket run workflow.yaml --trace trace.json

# Analyze trace
pocket trace analyze trace.json
```

### Use Verbose Output

```bash
# Show all node inputs/outputs
pocket run workflow.yaml --verbose

# Show specific node details
pocket run workflow.yaml --verbose-node process-data
```

## Performance Troubleshooting

### Slow Execution

1. **Profile execution**:
```bash
pocket run workflow.yaml --profile
pocket profile view profile.out
```

2. **Check bottlenecks**:
```bash
# Time each node
pocket run workflow.yaml --timing
```

3. **Optimize parallel execution**:
```yaml
nodes:
  - name: parallel-process
    type: parallel
    config:
      max_concurrency: 10  # Increase concurrency
```

### High Memory Usage

1. **Monitor memory**:
```bash
pocket run workflow.yaml --metrics
```

2. **Limit store size**:
```yaml
config:
  store:
    max_entries: 1000
    cleanup_interval: "1m"
```

3. **Stream large data**:
```yaml
nodes:
  - name: stream-process
    type: stream
    config:
      chunk_size: 1024
```

## Getting Help

### Built-in Help

```bash
# General help
pocket help

# Command-specific help
pocket run --help
pocket plugin --help

# Show version info
pocket version --verbose
```

### Diagnostic Information

Collect for bug reports:

```bash
# Generate diagnostic bundle
pocket diagnose --output diagnose.tar.gz

# Includes:
# - Version info
# - Configuration
# - System details
# - Recent logs
```

### Community Support

1. **GitHub Issues**: [Report bugs](https://github.com/agentstation/pocket/issues)
2. **Discussions**: [Ask questions](https://github.com/agentstation/pocket/discussions)
3. **Examples**: Check `/examples` directory

### Useful Commands for Debugging

```bash
# Validate workflow syntax
pocket validate workflow.yaml

# Check node connections
pocket graph workflow.yaml

# List all available nodes
pocket nodes list --verbose

# Test specific node
pocket test node echo --input '{"message": "test"}'

# Check system compatibility
pocket doctor
```

## Quick Fixes Checklist

When something goes wrong, try these in order:

1. ✓ Check syntax: `pocket validate workflow.yaml`
2. ✓ Enable debug logs: `POCKET_LOG_LEVEL=debug pocket run workflow.yaml`
3. ✓ Verify file paths are correct
4. ✓ Ensure plugins are loaded: `pocket plugin list`
5. ✓ Check environment variables are set
6. ✓ Try with increased timeout: `--timeout 5m`
7. ✓ Run with `--dry-run` to check execution plan
8. ✓ Simplify workflow to isolate issue
9. ✓ Check [examples](../../examples/cli/) for working patterns
10. ✓ Search [GitHub issues](https://github.com/agentstation/pocket/issues) for similar problems

## Next Steps

- Review [Configuration Guide](configuration.md)
- Learn about [Plugin Management](plugins.md)
- See [Command Reference](command-reference.md)