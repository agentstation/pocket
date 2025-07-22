# Advanced Workflow Examples

Complex workflows demonstrating advanced Pocket features like parallel processing, data transformation, and external integrations.

## Available Examples

### 1. Parallel Task Execution (`parallel-tasks.yaml`)
Execute multiple operations concurrently for better performance.

```yaml
name: parallel-data-fetch
start: fetch-all

nodes:
  - name: fetch-all
    type: parallel
    config:
      max_concurrency: 3
      fail_fast: false
      timeout: "30s"
      tasks:
        - name: fetch-users
          node: http
          config:
            url: "https://api.example.com/users"
            method: GET
            
        - name: fetch-orders
          node: http
          config:
            url: "https://api.example.com/orders"
            method: GET
            
        - name: fetch-inventory
          node: http
          config:
            url: "https://api.example.com/inventory"
            method: GET
```

### 2. Data Aggregation (`aggregate-data.yaml`)
Collect and combine data from multiple sources.

```yaml
name: data-aggregation
start: start

nodes:
  - name: start
    type: echo
    config:
      message: "Starting aggregation..."
      
  - name: aggregate
    type: aggregate
    config:
      mode: object
      key: "{{.source}}"
      count: 3
      timeout: "10s"
      partial: true
      
connections:
  - from: start
    to: source1
  - from: start
    to: source2
  - from: start
    to: source3
  - from: source1
    to: aggregate
  - from: source2
    to: aggregate
  - from: source3
    to: aggregate
```

### 3. HTTP API Integration (`http-api.yaml`)
Make HTTP requests with retries and error handling.

```yaml
name: api-integration
start: fetch-data

nodes:
  - name: fetch-data
    type: http
    config:
      url: "https://api.github.com/users/{{.username}}"
      method: GET
      headers:
        Accept: "application/vnd.github.v3+json"
        User-Agent: "Pocket-Workflow"
      timeout: "10s"
      retry:
        max_attempts: 3
        delay: "2s"
        
  - name: extract-info
    type: jsonpath
    config:
      path: "$.{login: login, name: name, public_repos: public_repos}"
      
  - name: format-output
    type: template
    config:
      template: |
        GitHub User Information:
        - Username: {{.login}}
        - Name: {{.name}}
        - Public Repos: {{.public_repos}}

connections:
  - from: fetch-data
    to: extract-info
  - from: extract-info
    to: format-output
```

### 4. Data Validation Pipeline (`validate-api-response.yaml`)
Validate data against JSON Schema before processing.

```yaml
name: validation-pipeline
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
        properties:
          id:
            type: string
            pattern: "^[A-Z0-9]+$"
          status:
            type: string
            enum: ["active", "pending", "inactive"]
          data:
            type: object
            required: ["value", "timestamp"]
        required: ["id", "status", "data"]
      fail_on_error: false
      
  - name: process-valid
    type: transform
    config:
      jq: ".data | {id, processed: true, value: .data.value * 1.1}"
      
  - name: handle-invalid
    type: template
    config:
      template: "Validation failed: {{.errors | join \", \"}}"

connections:
  - from: fetch
    to: validate
  - from: validate
    to: process-valid
    action: valid
  - from: validate
    to: handle-invalid
    action: invalid
```

### 5. JSONPath Data Extraction (`jsonpath-extract.yaml`)
Extract specific data from complex JSON structures.

```yaml
name: data-extraction
start: extract

nodes:
  - name: extract
    type: jsonpath
    config:
      path: "$.users[?(@.active==true)].{id: id, email: email, lastLogin: last_login}"
      multiple: true
      default: []
      
  - name: process-users
    type: template
    config:
      template: |
        Active Users ({{len .}}):
        {{range .}}
        - {{.email}} (ID: {{.id}}, Last login: {{.lastLogin}})
        {{end}}
```

### 6. File Operations (`file-operations.yaml`)
Read, transform, and write files with proper error handling.

```yaml
name: file-processing
start: read

nodes:
  - name: read
    type: file
    config:
      path: "./input.json"
      operation: read
      
  - name: transform
    type: transform
    config:
      jq: ". | map({id, value: .value * 2, processed: true})"
      
  - name: write
    type: file
    config:
      path: "./output.json"
      operation: write
      create_dirs: true
      
connections:
  - from: read
    to: transform
  - from: transform
    to: write
```

### 7. Command Execution (`exec-commands.yaml`)
Execute shell commands safely with timeout and output capture.

```yaml
name: command-execution
start: check-files

nodes:
  - name: check-files
    type: exec
    config:
      command: ls
      args: ["-la", "./data"]
      timeout: "5s"
      capture_output: true
      
  - name: process-list
    type: template
    config:
      template: |
        Files found:
        {{.stdout}}
```

## Key Concepts Demonstrated

### Parallel Processing
- Concurrent task execution
- Concurrency limits
- Fail-fast vs. continue-on-error
- Timeout management

### Data Processing
- JSONPath extraction
- JQ transformations
- Template rendering
- Data validation

### External Integrations
- HTTP requests with retry
- File I/O operations
- Command execution
- API authentication

### Error Handling
- Validation with schema
- Conditional routing on errors
- Fallback paths
- Partial results

## Best Practices

1. **Use Timeouts**: Always set appropriate timeouts for I/O operations
2. **Handle Errors**: Define error paths for robust workflows
3. **Validate Early**: Validate data before processing
4. **Limit Concurrency**: Control parallel execution to avoid overwhelming resources
5. **Secure Commands**: Be careful with exec nodes - validate inputs

## Running the Examples

```bash
# Run with sample data
pocket run examples/cli/http-api.yaml --input '{"username": "octocat"}'

# Run with verbose output
pocket run examples/cli/parallel-tasks.yaml --verbose

# Test with dry run
pocket run examples/cli/validate-api-response.yaml --dry-run
```

## Next Steps

- Explore [Real-World Examples](../real-world/)
- Learn about [Plugin Development](../../development/plugin-development.md)
- Study [Performance Optimization](../../advanced/PERFORMANCE.md)