# CLI Configuration

Pocket CLI can be configured through configuration files, environment variables, and command-line flags. This guide covers all configuration options and their precedence.

## Configuration Precedence

Configuration values are resolved in the following order (highest to lowest precedence):

1. Command-line flags
2. Environment variables
3. Configuration file (`pocket.yaml` or `pocket.json`)
4. Default values

## Configuration File

### Location

Pocket looks for configuration files in the following locations:

1. `./pocket.yaml` or `./pocket.json` (current directory)
2. `~/.pocket/config.yaml` or `~/.pocket/config.json` (user home)
3. `/etc/pocket/config.yaml` or `/etc/pocket/config.json` (system-wide)

### Example Configuration

**YAML format (`pocket.yaml`):**
```yaml
# Default values for all workflows
defaults:
  timeout: "30s"
  retry:
    max_attempts: 3
    delay: "2s"
    multiplier: 2
  
# Plugin directories
plugins:
  directories:
    - "./plugins"
    - "~/.pocket/plugins"
  
# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "stderr"
  
# Store configuration
store:
  max_entries: 10000
  ttl: "30m"
  persist_path: "~/.pocket/store"
  
# HTTP client defaults
http:
  timeout: "10s"
  max_redirects: 5
  user_agent: "Pocket/1.0"
  
# Environment variables to load
environment:
  files:
    - ".env"
    - ".env.local"
```

**JSON format (`pocket.json`):**
```json
{
  "defaults": {
    "timeout": "30s",
    "retry": {
      "max_attempts": 3,
      "delay": "2s",
      "multiplier": 2
    }
  },
  "plugins": {
    "directories": [
      "./plugins",
      "~/.pocket/plugins"
    ]
  },
  "logging": {
    "level": "info",
    "format": "json",
    "output": "stderr"
  }
}
```

## Environment Variables

All configuration options can be set via environment variables using the `POCKET_` prefix:

| Environment Variable | Configuration Path | Description |
|---------------------|-------------------|-------------|
| `POCKET_LOG_LEVEL` | `logging.level` | Log level (debug, info, warn, error) |
| `POCKET_LOG_FORMAT` | `logging.format` | Log format (text, json) |
| `POCKET_PLUGINS_DIR` | `plugins.directories[0]` | Primary plugin directory |
| `POCKET_STORE_MAX_ENTRIES` | `store.max_entries` | Maximum store entries |
| `POCKET_STORE_TTL` | `store.ttl` | Store entry TTL |
| `POCKET_HTTP_TIMEOUT` | `http.timeout` | Default HTTP timeout |
| `POCKET_HTTP_USER_AGENT` | `http.user_agent` | HTTP User-Agent header |

### Examples

```bash
# Set log level
export POCKET_LOG_LEVEL=debug

# Configure store
export POCKET_STORE_MAX_ENTRIES=5000
export POCKET_STORE_TTL=1h

# Set HTTP timeout
export POCKET_HTTP_TIMEOUT=30s
```

## Command-Line Configuration

Most configuration options can be overridden via command-line flags:

```bash
# Override log level
pocket run workflow.yaml --log-level debug

# Set custom config file
pocket run workflow.yaml --config /path/to/config.yaml

# Disable config file loading
pocket run workflow.yaml --no-config

# Set plugin directory
pocket run workflow.yaml --plugins-dir ./custom-plugins
```

## Workflow-Specific Configuration

Configuration can also be embedded in workflow files:

```yaml
name: configured-workflow
description: Workflow with custom configuration

# Workflow-specific config
config:
  timeout: "60s"
  retry:
    max_attempts: 5
  store:
    persist: true
    path: "./workflow-state"

start: process

nodes:
  - name: process
    type: http
    config:
      url: "https://api.example.com"
      # Uses workflow timeout by default
```

## Plugin Configuration

### Plugin Directories

Specify multiple plugin directories:

```yaml
plugins:
  directories:
    - "./plugins"          # Project-specific
    - "~/.pocket/plugins"  # User plugins
    - "/opt/pocket/plugins" # System plugins
  
  # Auto-load plugins on startup
  autoload: true
  
  # Plugin-specific configuration
  config:
    my-plugin:
      api_key: "${MY_PLUGIN_API_KEY}"
      endpoint: "https://api.example.com"
```

### Plugin Loading

```bash
# Load specific plugin
pocket plugin load my-plugin

# Load from custom directory
pocket plugin load ./path/to/plugin.so

# List loaded plugins
pocket plugin list
```

## Security Configuration

### Secrets Management

```yaml
# Reference environment variables
secrets:
  # Load from environment
  from_env:
    - OPENAI_API_KEY
    - DATABASE_URL
  
  # Load from files
  from_files:
    - path: ~/.pocket/secrets.yaml
      format: yaml
    - path: /run/secrets/api-keys
      format: env
```

### Restricted Mode

Run Pocket in restricted mode for additional security:

```yaml
security:
  restricted_mode: true
  allowed_hosts:
    - "api.example.com"
    - "*.trusted-domain.com"
  forbidden_nodes:
    - "exec"
    - "file"
  max_workflow_duration: "5m"
```

## Performance Configuration

### Concurrency Limits

```yaml
performance:
  # Maximum concurrent nodes
  max_concurrent_nodes: 10
  
  # Maximum concurrent workflows
  max_concurrent_workflows: 5
  
  # Worker pool size
  worker_pool_size: 20
  
  # Queue sizes
  queue_size: 1000
```

### Resource Limits

```yaml
limits:
  # Maximum memory per workflow
  max_memory: "1GB"
  
  # Maximum execution time
  max_execution_time: "30m"
  
  # Maximum output size
  max_output_size: "10MB"
  
  # Rate limiting
  rate_limit:
    requests_per_second: 100
    burst: 200
```

## Development Configuration

### Debug Mode

```yaml
debug:
  enabled: true
  
  # Save execution traces
  trace_dir: "./traces"
  
  # Enable profiling
  profiling:
    cpu: true
    memory: true
    output: "./profiles"
  
  # Verbose node execution
  verbose_nodes: true
  
  # Save intermediate results
  save_intermediates: true
```

### Testing Configuration

```yaml
testing:
  # Mock external services
  mock_http: true
  mock_responses:
    "api.example.com":
      status: 200
      body: '{"mock": true}'
  
  # Deterministic execution
  seed: 12345
  
  # Disable timeouts in tests
  disable_timeouts: true
```

## Example: Production Configuration

Complete production configuration example:

```yaml
# pocket.yaml - Production configuration
defaults:
  timeout: "30s"
  retry:
    max_attempts: 3
    delay: "1s"
    multiplier: 2
    max_delay: "30s"

plugins:
  directories:
    - "/opt/pocket/plugins"
  autoload: true

logging:
  level: "info"
  format: "json"
  output: "stdout"
  # Send to centralized logging
  hooks:
    - type: "syslog"
      address: "logs.example.com:514"

store:
  max_entries: 50000
  ttl: "1h"
  persist_path: "/var/lib/pocket/store"
  cleanup_interval: "10m"

http:
  timeout: "10s"
  max_redirects: 5
  user_agent: "Pocket/1.0 (Production)"
  tls:
    insecure_skip_verify: false
    min_version: "1.2"

security:
  restricted_mode: true
  allowed_hosts:
    - "*.mycompany.com"
    - "api.trusted-vendor.com"

performance:
  max_concurrent_nodes: 50
  max_concurrent_workflows: 10
  worker_pool_size: 100

monitoring:
  metrics:
    enabled: true
    port: 9090
    path: "/metrics"
  
  health:
    enabled: true
    port: 8080
    path: "/health"
  
  tracing:
    enabled: true
    endpoint: "jaeger.example.com:6831"
    sample_rate: 0.1
```

## Configuration Validation

Validate your configuration:

```bash
# Validate config file
pocket config validate

# Show effective configuration
pocket config show

# Show configuration source
pocket config show --source
```

## Best Practices

1. **Use environment variables for secrets** - Never hardcode sensitive values
2. **Version control config files** - Track configuration changes
3. **Use different configs for environments** - Separate dev/staging/prod
4. **Set appropriate timeouts** - Prevent hanging workflows
5. **Configure logging appropriately** - Debug in dev, info in prod
6. **Monitor resource usage** - Set limits to prevent resource exhaustion
7. **Regular cleanup** - Configure store cleanup for long-running instances

## Troubleshooting

### Config not loading

```bash
# Check config search paths
pocket config paths

# Verify config syntax
pocket config validate --file pocket.yaml

# Debug config loading
POCKET_LOG_LEVEL=debug pocket config show
```

### Environment variables not working

```bash
# List all Pocket environment variables
env | grep POCKET_

# Test specific variable
POCKET_LOG_LEVEL=debug pocket run workflow.yaml
```

## Next Steps

- Learn about [Plugin Management](plugins.md)
- Explore [Troubleshooting Guide](troubleshooting.md)
- See [Command Reference](command-reference.md)