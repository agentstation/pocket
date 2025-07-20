package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/agentstation/pocket"
	"github.com/agentstation/pocket/yaml"
	"github.com/ohler55/ojg/jp"
	"github.com/xeipuuv/gojsonschema"
)

// EchoNodeBuilder builds echo nodes.
type EchoNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *EchoNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "echo",
		Category:    "core",
		Description: "Outputs a message and passes through input",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Message to output",
					"default":     "Hello from echo node",
				},
			},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{"type": "string"},
				"input":   map[string]interface{}{"type": []string{"null", "object", "string", "number", "boolean", "array"}},
				"node":    map[string]interface{}{"type": "string"},
			},
		},
		Examples: []Example{
			{
				Name:        "Simple echo",
				Description: "Output a message",
				Config: map[string]interface{}{
					"message": "Hello, World!",
				},
				Output: map[string]interface{}{
					"message": "Hello, World!",
					"input":   nil,
					"node":    "echo1",
				},
			},
			{
				Name:        "Echo with input",
				Description: "Echo message with input passthrough",
				Config: map[string]interface{}{
					"message": "Processing complete",
				},
				Input: map[string]interface{}{"data": "test"},
				Output: map[string]interface{}{
					"message": "Processing complete",
					"input":   map[string]interface{}{"data": "test"},
					"node":    "echo2",
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates an echo node from a definition.
func (b *EchoNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	message := "Hello from echo node"
	if msgInterface, ok := def.Config["message"]; ok {
		if msg, ok := msgInterface.(string); ok {
			message = msg
		}
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			if b.Verbose {
				log.Printf("[%s] Echo: %s", def.Name, message)
			}
			return map[string]interface{}{
				"message": message,
				"input":   input,
				"node":    def.Name,
			}, nil
		}),
	), nil
}

// DelayNodeBuilder builds delay nodes.
type DelayNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *DelayNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "delay",
		Category:    "core",
		Description: "Delays execution for a specified duration",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"duration": map[string]interface{}{
					"type":        "string",
					"description": "Duration to delay (e.g., '1s', '500ms')",
					"default":     "1s",
					"pattern":     "^[0-9]+[a-z]+$",
				},
			},
		},
		Examples: []Example{
			{
				Name:        "Simple delay",
				Description: "Delay for 1 second",
				Config: map[string]interface{}{
					"duration": "1s",
				},
			},
			{
				Name:        "Short delay",
				Description: "Delay for 500 milliseconds",
				Config: map[string]interface{}{
					"duration": "500ms",
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a delay node from a definition.
func (b *DelayNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	duration := 1 * time.Second
	if durInterface, ok := def.Config["duration"]; ok {
		if durStr, ok := durInterface.(string); ok {
			if d, err := time.ParseDuration(durStr); err == nil {
				duration = d
			}
		}
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			if b.Verbose {
				log.Printf("[%s] Delaying for %v", def.Name, duration)
			}
			select {
			case <-time.After(duration):
				return input, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}),
	), nil
}

// RouterNodeBuilder builds router nodes.
type RouterNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *RouterNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "router",
		Category:    "core",
		Description: "Routes to a specific node based on configuration",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"route": map[string]interface{}{
					"type":        "string",
					"description": "The route/action to take",
					"default":     "default",
				},
			},
		},
		Examples: []Example{
			{
				Name:        "Simple routing",
				Description: "Route to a specific action",
				Config: map[string]interface{}{
					"route": "success",
				},
			},
			{
				Name:        "Default routing",
				Description: "Use default route",
				Config:      map[string]interface{}{},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a router node from a definition.
func (b *RouterNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	route := "default"
	if routeInterface, ok := def.Config["route"]; ok {
		if r, ok := routeInterface.(string); ok {
			route = r
		}
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			if b.Verbose {
				log.Printf("[%s] Routing to: %s", def.Name, route)
			}
			return exec, route, nil
		}),
	), nil
}

// TransformNodeBuilder builds transform nodes.
type TransformNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *TransformNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "transform",
		Category:    "data",
		Description: "Transforms input data",
		ConfigSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{
				// TODO: Add expression support
			},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"transformed": map[string]interface{}{"type": "boolean"},
				"original":    map[string]interface{}{"type": []string{"null", "object", "string", "number", "boolean", "array"}},
				"timestamp":   map[string]interface{}{"type": "string", "format": "date-time"},
				"node":        map[string]interface{}{"type": "string"},
			},
		},
		Examples: []Example{
			{
				Name:        "Simple transform",
				Description: "Wrap input with metadata",
				Config:      map[string]interface{}{},
				Input:       map[string]interface{}{"value": 42},
				Output: map[string]interface{}{
					"transformed": true,
					"original":    map[string]interface{}{"value": 42},
					"timestamp":   "2024-01-01T00:00:00Z",
					"node":        "transform1",
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a transform node from a definition.
func (b *TransformNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			if b.Verbose {
				log.Printf("[%s] Transforming input", def.Name)
			}

			// Simple transformation: wrap input in a result
			result := map[string]interface{}{
				"transformed": true,
				"original":    input,
				"timestamp":   time.Now().Format(time.RFC3339),
				"node":        def.Name,
			}

			// For testing conditional, add a score if the node name suggests it
			if strings.Contains(def.Name, "score") {
				// Generate a random score
				score := rand.Float64() // #nosec G404 - This is for example data generation, not security
				result["score"] = score
				if b.Verbose {
					log.Printf("[%s] Generated score: %.2f", def.Name, score)
				}
			}

			return result, nil
		}),
	), nil
}

// ConditionalNodeBuilder builds conditional routing nodes.
type ConditionalNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *ConditionalNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "conditional",
		Category:    "core",
		Description: "Routes to different nodes based on conditions",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"conditions": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"if":   map[string]interface{}{"type": "string"},
							"then": map[string]interface{}{"type": "string"},
						},
						"required": []string{"if", "then"},
					},
				},
				"else": map[string]interface{}{
					"type":        "string",
					"description": "Default route if no conditions match",
				},
			},
			"required": []string{"conditions"},
		},
		Examples: []Example{
			{
				Name: "Route by score",
				Config: map[string]interface{}{
					"conditions": []map[string]interface{}{
						{"if": "{{gt .score 0.8}}", "then": "high"},
						{"if": "{{gt .score 0.5}}", "then": "medium"},
					},
					"else": "low",
				},
			},
			{
				Name: "Route by type",
				Config: map[string]interface{}{
					"conditions": []map[string]interface{}{
						{"if": "{{eq .type \"error\"}}", "then": "error-handler"},
						{"if": "{{eq .type \"warning\"}}", "then": "warning-handler"},
					},
					"else": "success",
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a conditional node from a definition.
func (b *ConditionalNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	// Extract conditions
	conditionsRaw, ok := def.Config["conditions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("conditions must be an array")
	}

	type condition struct {
		expr  *template.Template
		route string
	}

	conditions := make([]condition, 0, len(conditionsRaw))
	for i, c := range conditionsRaw {
		cond, ok := c.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("condition %d must be an object", i)
		}

		ifExpr, ok := cond["if"].(string)
		if !ok {
			return nil, fmt.Errorf("condition %d missing 'if'", i)
		}

		thenRoute, ok := cond["then"].(string)
		if !ok {
			return nil, fmt.Errorf("condition %d missing 'then'", i)
		}

		tmpl, err := template.New(fmt.Sprintf("cond_%d", i)).Parse(ifExpr)
		if err != nil {
			return nil, fmt.Errorf("condition %d invalid template: %w", i, err)
		}

		conditions = append(conditions, condition{
			expr:  tmpl,
			route: thenRoute,
		})
	}

	defaultRoute, _ := def.Config["else"].(string)

	return pocket.NewNode[any, any](def.Name,
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, exec any) (any, string, error) {
			// Evaluate conditions in order
			for _, cond := range conditions {
				var buf bytes.Buffer
				if err := cond.expr.Execute(&buf, exec); err != nil {
					if b.Verbose {
						log.Printf("[%s] Condition evaluation error: %v", def.Name, err)
					}
					continue // Skip failed conditions
				}

				// Check if result is truthy
				result := strings.TrimSpace(buf.String())
				if result == "true" || result == "1" {
					if b.Verbose {
						log.Printf("[%s] Condition matched, routing to: %s", def.Name, cond.route)
					}
					return exec, cond.route, nil
				}
			}

			// No conditions matched, use default
			if b.Verbose {
				log.Printf("[%s] No conditions matched, routing to: %s", def.Name, defaultRoute)
			}
			return exec, defaultRoute, nil
		}),
	), nil
}

// TemplateNodeBuilder builds template rendering nodes.
type TemplateNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *TemplateNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "template",
		Category:    "data",
		Description: "Renders Go templates with input data",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"template": map[string]interface{}{
					"type":        "string",
					"description": "Go template string to render",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Path to template file (alternative to inline template)",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"string", "json", "yaml"},
					"default":     "string",
					"description": "Output format for the rendered template",
				},
			},
			"oneOf": []map[string]interface{}{
				{"required": []string{"template"}},
				{"required": []string{"file"}},
			},
		},
		Examples: []Example{
			{
				Name:        "Simple greeting",
				Description: "Render a greeting message",
				Config: map[string]interface{}{
					"template": "Hello, {{.name}}! Your score is {{.score}}.",
				},
				Input: map[string]interface{}{
					"name":  "Alice",
					"score": 95,
				},
				Output: "Hello, Alice! Your score is 95.",
			},
			{
				Name:        "JSON output",
				Description: "Render template and output as JSON",
				Config: map[string]interface{}{
					"template":      `{"message": "Welcome {{.user}}", "timestamp": "{{.time}}"}`,
					"output_format": "json",
				},
				Input: map[string]interface{}{
					"user": "Bob",
					"time": "2024-01-01T00:00:00Z",
				},
				Output: map[string]interface{}{
					"message":   "Welcome Bob",
					"timestamp": "2024-01-01T00:00:00Z",
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a template node from a definition.
func (b *TemplateNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	// Get template configuration
	templateStr, hasTemplate := def.Config["template"].(string)
	templateFile, hasFile := def.Config["file"].(string)

	if !hasTemplate && !hasFile {
		return nil, fmt.Errorf("either 'template' or 'file' must be specified")
	}

	outputFormat, _ := def.Config["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "string"
	}

	// Parse template at build time for validation
	var tmpl *template.Template
	var err error

	if hasTemplate {
		tmpl, err = template.New(def.Name).Parse(templateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid template: %w", err)
		}
	} else {
		// For file templates, we'll parse at execution time
		// to allow for dynamic template updates
		tmpl = nil
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			var execTemplate *template.Template

			// Use pre-parsed template or load from file
			if tmpl != nil {
				execTemplate = tmpl
			} else {
				// Read and parse template file
				content, err := os.ReadFile(templateFile) // #nosec G304 - Template files are user-configured
				if err != nil {
					return nil, fmt.Errorf("failed to read template file: %w", err)
				}

				execTemplate, err = template.New(def.Name).Parse(string(content))
				if err != nil {
					return nil, fmt.Errorf("failed to parse template file: %w", err)
				}
			}

			// Execute template
			var buf bytes.Buffer
			if err := execTemplate.Execute(&buf, input); err != nil {
				return nil, fmt.Errorf("template execution failed: %w", err)
			}

			result := buf.String()

			if b.Verbose {
				log.Printf("[%s] Rendered template: %s", def.Name, result)
			}

			// Format output based on output_format
			switch outputFormat {
			case "json":
				var jsonData interface{}
				if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
					return nil, fmt.Errorf("failed to parse JSON output: %w", err)
				}
				return jsonData, nil

			case "yaml":
				// For YAML, we'll keep it as string since we don't have a YAML parser imported
				// in the builtin package. The downstream can parse it if needed.
				return map[string]interface{}{
					"yaml":   result,
					"format": "yaml",
				}, nil

			default: // "string"
				return result, nil
			}
		}),
	), nil
}

// HTTPNodeBuilder builds HTTP client nodes.
type HTTPNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *HTTPNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "http",
		Category:    "io",
		Description: "Makes HTTP requests with retry and timeout support",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to request (supports templating)",
				},
				"method": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
					"default": "GET",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "HTTP headers",
				},
				"body": map[string]interface{}{
					"type":        []string{"string", "object"},
					"description": "Request body (for POST/PUT/PATCH)",
				},
				"timeout": map[string]interface{}{
					"type":        "string",
					"default":     "30s",
					"description": "Request timeout",
				},
				"retry": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"max_attempts": map[string]interface{}{"type": "integer", "default": 3},
						"delay":        map[string]interface{}{"type": "string", "default": "1s"},
					},
				},
			},
			"required": []string{"url"},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status":  map[string]interface{}{"type": "integer"},
				"headers": map[string]interface{}{"type": "object"},
				"body":    map[string]interface{}{"type": []string{"object", "string"}},
			},
		},
		Examples: []Example{
			{
				Name: "GET request",
				Config: map[string]interface{}{
					"url":    "https://api.example.com/data",
					"method": "GET",
				},
			},
			{
				Name: "POST with retry",
				Config: map[string]interface{}{
					"url":    "https://api.example.com/submit",
					"method": "POST",
					"headers": map[string]interface{}{
						"Content-Type": "application/json",
					},
					"body": map[string]interface{}{
						"key": "value",
					},
					"retry": map[string]interface{}{
						"max_attempts": 5,
						"delay":        "2s",
					},
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates an HTTP node from a definition.
//
//nolint:gocyclo // Configuration parsing requires handling many options
func (b *HTTPNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	url, _ := def.Config["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	method, _ := def.Config["method"].(string)
	if method == "" {
		method = "GET"
	}

	headers := make(map[string]string)
	if h, ok := def.Config["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			headers[k] = fmt.Sprint(v)
		}
	}

	body := def.Config["body"]

	timeoutStr, _ := def.Config["timeout"].(string)
	timeout, _ := time.ParseDuration(timeoutStr)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Parse retry config
	maxAttempts := 3
	retryDelay := time.Second
	if retry, ok := def.Config["retry"].(map[string]interface{}); ok {
		if ma, ok := retry["max_attempts"].(int); ok {
			maxAttempts = ma
		}
		if d, ok := retry["delay"].(string); ok {
			if pd, err := time.ParseDuration(d); err == nil {
				retryDelay = pd
			}
		}
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Support URL templating with input data
			finalURL := url
			if strings.Contains(url, "{{") {
				tmpl, err := template.New("url").Parse(url)
				if err != nil {
					return nil, fmt.Errorf("invalid URL template: %w", err)
				}
				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, input); err != nil {
					return nil, fmt.Errorf("URL template execution failed: %w", err)
				}
				finalURL = buf.String()
			}

			client := &http.Client{
				Timeout: timeout,
			}

			var lastErr error
			for attempt := 0; attempt < maxAttempts; attempt++ {
				if attempt > 0 {
					if b.Verbose {
						log.Printf("[%s] Retry attempt %d/%d", def.Name, attempt+1, maxAttempts)
					}
					time.Sleep(retryDelay)
				}

				// Prepare request body
				var bodyReader io.Reader
				if body != nil && method != "GET" && method != "DELETE" {
					switch v := body.(type) {
					case string:
						bodyReader = strings.NewReader(v)
					default:
						jsonBody, err := json.Marshal(v)
						if err != nil {
							return nil, fmt.Errorf("failed to marshal body: %w", err)
						}
						bodyReader = bytes.NewReader(jsonBody)
						if headers["Content-Type"] == "" {
							headers["Content-Type"] = "application/json"
						}
					}
				}

				req, err := http.NewRequestWithContext(ctx, method, finalURL, bodyReader)
				if err != nil {
					return nil, err
				}

				// Add headers
				for k, v := range headers {
					req.Header.Set(k, v)
				}

				resp, err := client.Do(req)
				if err != nil {
					lastErr = err
					continue
				}

				// Read and close body immediately to avoid defer in loop
				respBody, err := io.ReadAll(resp.Body)
				closeErr := resp.Body.Close()
				if closeErr != nil && b.Verbose {
					log.Printf("[%s] Failed to close response body: %v", def.Name, closeErr)
				}
				if err != nil {
					lastErr = err
					continue
				}

				// Parse JSON if content type is JSON
				var bodyData interface{} = string(respBody)
				contentType := resp.Header.Get("Content-Type")
				if strings.Contains(contentType, "application/json") {
					var jsonData interface{}
					if err := json.Unmarshal(respBody, &jsonData); err == nil {
						bodyData = jsonData
					}
				}

				result := map[string]interface{}{
					"status":  resp.StatusCode,
					"headers": resp.Header,
					"body":    bodyData,
				}

				if b.Verbose {
					log.Printf("[%s] HTTP %s %s - Status: %d", def.Name, method, finalURL, resp.StatusCode)
				}

				// Retry on 5xx errors
				if resp.StatusCode >= 500 && attempt < maxAttempts-1 {
					lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
					continue
				}

				return result, nil
			}

			return nil, fmt.Errorf("all attempts failed: %w", lastErr)
		}),
	), nil
}

// JSONPathNodeBuilder builds JSONPath extraction nodes.
type JSONPathNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *JSONPathNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "jsonpath",
		Category:    "data",
		Description: "Extracts data from JSON using JSONPath expressions",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "JSONPath expression to extract data",
				},
				"multiple": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Return all matches as array (true) or first match only (false)",
				},
				"default": map[string]interface{}{
					"description": "Default value if path not found",
				},
				"unwrap": map[string]interface{}{
					"type":        "boolean",
					"default":     true,
					"description": "Unwrap single-element arrays",
				},
			},
			"required": []string{"path"},
		},
		OutputSchema: map[string]interface{}{
			"description": "Extracted value(s) from the JSONPath query",
		},
		Examples: []Example{
			{
				Name:        "Extract user name",
				Description: "Get user name from nested object",
				Config: map[string]interface{}{
					"path": "$.user.name",
				},
				Input: map[string]interface{}{
					"user": map[string]interface{}{
						"name": "Alice",
						"age":  30,
					},
				},
				Output: "Alice",
			},
			{
				Name:        "Extract all prices",
				Description: "Get all prices from array of items",
				Config: map[string]interface{}{
					"path":     "$.items[*].price",
					"multiple": true,
				},
				Input: map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{"name": "Book", "price": 10.99},
						map[string]interface{}{"name": "Pen", "price": 2.50},
					},
				},
				Output: []interface{}{10.99, 2.50},
			},
			{
				Name:        "Extract with default",
				Description: "Use default value when path not found",
				Config: map[string]interface{}{
					"path":    "$.missing.field",
					"default": "Not found",
				},
				Input:  map[string]interface{}{"other": "data"},
				Output: "Not found",
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a JSONPath node from a definition.
func (b *JSONPathNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	pathStr, ok := def.Config["path"].(string)
	if !ok || pathStr == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Parse JSONPath expression at build time for validation
	expr, err := jp.ParseString(pathStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JSONPath expression: %w", err)
	}

	multiple, _ := def.Config["multiple"].(bool)
	defaultValue := def.Config["default"]
	unwrap := true
	if u, ok := def.Config["unwrap"].(bool); ok {
		unwrap = u
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Find matches using JSONPath
			results := expr.Get(input)

			if b.Verbose {
				log.Printf("[%s] JSONPath '%s' found %d matches", def.Name, pathStr, len(results))
			}

			// Handle no matches
			if len(results) == 0 {
				if defaultValue != nil {
					if b.Verbose {
						log.Printf("[%s] No matches, returning default value", def.Name)
					}
					return defaultValue, nil
				}
				if multiple {
					return []interface{}{}, nil
				}
				return nil, nil
			}

			// Return results based on configuration
			if multiple {
				// Return all matches as array
				return results, nil
			}

			// Return first match only
			result := results[0]

			// Unwrap single-element arrays if configured
			if unwrap {
				if arr, ok := result.([]interface{}); ok && len(arr) == 1 {
					result = arr[0]
				}
			}

			return result, nil
		}),
	), nil
}

// ValidateNodeBuilder builds JSON Schema validation nodes.
type ValidateNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *ValidateNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "validate",
		Category:    "data",
		Description: "Validates data against JSON Schema",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"schema": map[string]interface{}{
					"type":        "object",
					"description": "JSON Schema to validate against",
				},
				"schema_file": map[string]interface{}{
					"type":        "string",
					"description": "Path to JSON Schema file (alternative to inline schema)",
				},
				"fail_on_error": map[string]interface{}{
					"type":        "boolean",
					"default":     true,
					"description": "Return error on validation failure (true) or continue with validation result (false)",
				},
			},
			"oneOf": []map[string]interface{}{
				{"required": []string{"schema"}},
				{"required": []string{"schema_file"}},
			},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"valid": map[string]interface{}{"type": "boolean"},
				"errors": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"field":       map[string]interface{}{"type": "string"},
							"type":        map[string]interface{}{"type": "string"},
							"description": map[string]interface{}{"type": "string"},
						},
					},
				},
				"data": map[string]interface{}{
					"description": "The original input data",
				},
			},
		},
		Examples: []Example{
			{
				Name:        "Validate user object",
				Description: "Ensure user data matches expected schema",
				Config: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":  map[string]interface{}{"type": "string"},
							"email": map[string]interface{}{"type": "string", "format": "email"},
							"age":   map[string]interface{}{"type": "integer", "minimum": 0},
						},
						"required": []string{"name", "email"},
					},
				},
				Input: map[string]interface{}{
					"name":  "Alice",
					"email": "alice@example.com",
					"age":   30,
				},
				Output: map[string]interface{}{
					"valid":  true,
					"errors": []interface{}{},
					"data": map[string]interface{}{
						"name":  "Alice",
						"email": "alice@example.com",
						"age":   30,
					},
				},
			},
			{
				Name:        "Validation failure",
				Description: "Handle invalid data gracefully",
				Config: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"score": map[string]interface{}{
								"type":    "number",
								"minimum": 0,
								"maximum": 100,
							},
						},
						"required": []string{"score"},
					},
					"fail_on_error": false,
				},
				Input: map[string]interface{}{
					"score": 150,
				},
				Output: map[string]interface{}{
					"valid": false,
					"errors": []interface{}{
						map[string]interface{}{
							"field":       "score",
							"type":        "number_gte",
							"description": "Must be less than or equal to 100",
						},
					},
					"data": map[string]interface{}{
						"score": 150,
					},
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a validate node from a definition.
func (b *ValidateNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	// Get schema configuration
	schema, hasSchema := def.Config["schema"]
	schemaFile, hasFile := def.Config["schema_file"].(string)

	if !hasSchema && !hasFile {
		return nil, fmt.Errorf("either 'schema' or 'schema_file' must be specified")
	}

	failOnError := true
	if f, ok := def.Config["fail_on_error"].(bool); ok {
		failOnError = f
	}

	// Pre-compile schema if provided inline
	var schemaLoader gojsonschema.JSONLoader
	if hasSchema {
		schemaLoader = gojsonschema.NewGoLoader(schema)
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Load schema from file if needed
			var loader gojsonschema.JSONLoader
			if schemaLoader != nil {
				loader = schemaLoader
			} else {
				// Read schema file
				schemaContent, err := os.ReadFile(schemaFile) // #nosec G304 - Schema files are user-configured
				if err != nil {
					return nil, fmt.Errorf("failed to read schema file: %w", err)
				}
				loader = gojsonschema.NewBytesLoader(schemaContent)
			}

			// Create document loader from input
			documentLoader := gojsonschema.NewGoLoader(input)

			// Validate
			result, err := gojsonschema.Validate(loader, documentLoader)
			if err != nil {
				return nil, fmt.Errorf("validation error: %w", err)
			}

			// Build response
			response := map[string]interface{}{
				"valid":  result.Valid(),
				"errors": []interface{}{},
				"data":   input,
			}

			// Add validation errors if any
			if !result.Valid() {
				errors := []interface{}{}
				for _, err := range result.Errors() {
					errors = append(errors, map[string]interface{}{
						"field":       err.Field(),
						"type":        err.Type(),
						"description": err.Description(),
					})
				}
				response["errors"] = errors

				if b.Verbose {
					log.Printf("[%s] Validation failed with %d errors", def.Name, len(errors))
				}
			} else if b.Verbose {
				log.Printf("[%s] Validation passed", def.Name)
			}

			// Return error if configured to fail on validation error
			if !result.Valid() && failOnError {
				return response, fmt.Errorf("validation failed: %d errors", len(result.Errors()))
			}

			return response, nil
		}),
	), nil
}

// AggregateNodeBuilder builds data aggregation nodes.
type AggregateNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *AggregateNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "aggregate",
		Category:    "data",
		Description: "Collects and combines data from multiple inputs",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"array", "object", "merge", "concat"},
					"default":     "array",
					"description": "How to aggregate inputs: array (collect all), object (key-value pairs), merge (deep merge objects), concat (concatenate arrays)",
				},
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Key to use for object mode (supports templates)",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"minimum":     1,
					"description": "Number of inputs to collect before continuing",
				},
				"timeout": map[string]interface{}{
					"type":        "string",
					"default":     "30s",
					"description": "Maximum time to wait for all inputs",
				},
				"partial": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Allow partial results if timeout occurs",
				},
			},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"description": "Aggregated data (array, object, or merged result)",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of items collected",
				},
				"complete": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether all expected inputs were received",
				},
			},
		},
		Examples: []Example{
			{
				Name:        "Collect array of results",
				Description: "Aggregate multiple inputs into an array",
				Config: map[string]interface{}{
					"mode":  "array",
					"count": 3,
				},
				Output: map[string]interface{}{
					"data":     []interface{}{"input1", "input2", "input3"},
					"count":    3,
					"complete": true,
				},
			},
			{
				Name:        "Build object from inputs",
				Description: "Create object with dynamic keys",
				Config: map[string]interface{}{
					"mode": "object",
					"key":  "{{.type}}",
				},
				Output: map[string]interface{}{
					"data": map[string]interface{}{
						"user":    map[string]interface{}{"type": "user", "name": "Alice"},
						"product": map[string]interface{}{"type": "product", "name": "Widget"},
					},
					"count":    2,
					"complete": true,
				},
			},
			{
				Name:        "Merge objects deeply",
				Description: "Deep merge multiple objects",
				Config: map[string]interface{}{
					"mode": "merge",
				},
				Output: map[string]interface{}{
					"data": map[string]interface{}{
						"user":     "Alice",
						"role":     "admin",
						"settings": map[string]interface{}{"theme": "dark", "lang": "en"},
					},
					"count":    3,
					"complete": true,
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates an aggregate node from a definition.
//
//nolint:gocyclo // Configuration parsing requires handling multiple aggregation strategies
func (b *AggregateNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	mode, _ := def.Config["mode"].(string)
	if mode == "" {
		mode = "array"
	}

	keyTemplate, _ := def.Config["key"].(string)

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// For single-pass aggregation, the input should already be
			// an array or a structured object we can aggregate

			var items []interface{}

			// Handle different input types
			switch v := input.(type) {
			case []interface{}:
				items = v
			case map[string]interface{}:
				// If input is a map with a "data" field containing array
				if data, ok := v["data"].([]interface{}); ok {
					items = data
				} else {
					// Treat single map as one item
					items = []interface{}{v}
				}
			default:
				// Single item
				items = []interface{}{input}
			}

			if b.Verbose {
				log.Printf("[%s] Aggregating %d items in %s mode", def.Name, len(items), mode)
			}

			// Aggregate based on mode
			var result interface{}
			switch mode {
			case "array":
				result = items

			case "object":
				obj := make(map[string]interface{})
				for i, item := range items {
					key := fmt.Sprintf("item_%d", i)
					if keyTemplate != "" {
						// Execute key template
						tmpl, err := template.New("key").Parse(keyTemplate)
						if err == nil {
							var buf bytes.Buffer
							if err := tmpl.Execute(&buf, item); err == nil {
								key = buf.String()
							}
						}
					}
					obj[key] = item
				}
				result = obj

			case "merge":
				result = make(map[string]interface{})
				for _, item := range items {
					if m, ok := item.(map[string]interface{}); ok {
						result = deepMerge(result.(map[string]interface{}), m)
					}
				}

			case "concat":
				var concatenated []interface{}
				for _, item := range items {
					if arr, ok := item.([]interface{}); ok {
						concatenated = append(concatenated, arr...)
					} else {
						concatenated = append(concatenated, item)
					}
				}
				result = concatenated

			default:
				return nil, fmt.Errorf("unknown aggregation mode: %s", mode)
			}

			response := map[string]interface{}{
				"data":     result,
				"count":    len(items),
				"complete": true,
			}

			return response, nil
		}),
	), nil
}

// deepMerge recursively merges two maps.
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for key, srcVal := range src {
		if dstVal, exists := dst[key]; exists {
			// If both are maps, merge recursively
			if srcMap, srcOk := srcVal.(map[string]interface{}); srcOk {
				if dstMap, dstOk := dstVal.(map[string]interface{}); dstOk {
					dst[key] = deepMerge(dstMap, srcMap)
					continue
				}
			}
		}
		// Otherwise, overwrite with source value
		dst[key] = srcVal
	}
	return dst
}

// FileNodeBuilder builds file I/O nodes with sandboxing.
type FileNodeBuilder struct {
	Verbose bool
}

// Metadata returns the node metadata.
func (b *FileNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "file",
		Category:    "io",
		Description: "Reads or writes files with path restrictions",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"read", "write", "append", "exists", "list"},
					"default":     "read",
					"description": "File operation to perform",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path (relative to working directory or absolute if allowed)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write (for write/append operations)",
				},
				"encoding": map[string]interface{}{
					"type":        "string",
					"default":     "utf-8",
					"description": "File encoding",
				},
				"base_dir": map[string]interface{}{
					"type":        "string",
					"description": "Base directory for sandboxing (defaults to current working directory)",
				},
				"allow_absolute": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Allow absolute paths outside base directory",
				},
				"create_dirs": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "Create parent directories if they don't exist",
				},
			},
			"required": []string{"operation", "path"},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Resolved file path",
				},
				"exists": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether the file exists",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "File content (for read operations)",
				},
				"size": map[string]interface{}{
					"type":        "integer",
					"description": "File size in bytes",
				},
				"modified": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "Last modification time",
				},
				"files": map[string]interface{}{
					"type":        "array",
					"description": "List of files (for list operation)",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":     map[string]interface{}{"type": "string"},
							"path":     map[string]interface{}{"type": "string"},
							"size":     map[string]interface{}{"type": "integer"},
							"modified": map[string]interface{}{"type": "string", "format": "date-time"},
							"isDir":    map[string]interface{}{"type": "boolean"},
						},
					},
				},
			},
		},
		Examples: []Example{
			{
				Name:        "Read file",
				Description: "Read contents of a text file",
				Config: map[string]interface{}{
					"operation": "read",
					"path":      "config.json",
				},
				Output: map[string]interface{}{
					"path":     "/app/config.json",
					"exists":   true,
					"content":  "{\"version\": \"1.0.0\"}",
					"size":     20,
					"modified": "2024-01-15T10:30:00Z",
				},
			},
			{
				Name:        "Write file",
				Description: "Write content to a file",
				Config: map[string]interface{}{
					"operation":   "write",
					"path":        "output/result.txt",
					"content":     "Processing complete",
					"create_dirs": true,
				},
				Output: map[string]interface{}{
					"path":     "/app/output/result.txt",
					"exists":   true,
					"size":     19,
					"modified": "2024-01-15T10:35:00Z",
				},
			},
			{
				Name:        "List directory",
				Description: "List files in a directory",
				Config: map[string]interface{}{
					"operation": "list",
					"path":      "data/",
				},
				Output: map[string]interface{}{
					"path":   "/app/data",
					"exists": true,
					"files": []interface{}{
						map[string]interface{}{
							"name":     "file1.txt",
							"path":     "/app/data/file1.txt",
							"size":     1024,
							"modified": "2024-01-14T09:00:00Z",
							"isDir":    false,
						},
					},
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a file node from a definition.
//
//nolint:gocyclo // Complex due to multiple operations (read/write/append/list) and security validations
func (b *FileNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	operation, _ := def.Config["operation"].(string)
	if operation == "" {
		operation = "read"
	}

	pathStr, ok := def.Config["path"].(string)
	if !ok || pathStr == "" {
		return nil, fmt.Errorf("path is required")
	}

	content, _ := def.Config["content"].(string)
	// Note: encoding config is reserved for future UTF-16/32 support

	baseDir, _ := def.Config["base_dir"].(string)
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	allowAbsolute, _ := def.Config["allow_absolute"].(bool)
	createDirs, _ := def.Config["create_dirs"].(bool)

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Resolve path with sandboxing
			resolvedPath, err := resolvePath(pathStr, baseDir, allowAbsolute)
			if err != nil {
				return nil, fmt.Errorf("path resolution failed: %w", err)
			}

			if b.Verbose {
				log.Printf("[%s] File operation '%s' on path: %s", def.Name, operation, resolvedPath)
			}

			switch operation {
			case "read":
				data, err := os.ReadFile(resolvedPath) // #nosec G304 - Path is validated and sandboxed
				if err != nil {
					if os.IsNotExist(err) {
						return map[string]interface{}{
							"path":   resolvedPath,
							"exists": false,
						}, nil
					}
					return nil, fmt.Errorf("read failed: %w", err)
				}

				info, _ := os.Stat(resolvedPath)
				return map[string]interface{}{
					"path":     resolvedPath,
					"exists":   true,
					"content":  string(data),
					"size":     info.Size(),
					"modified": info.ModTime().Format(time.RFC3339),
				}, nil

			case "write":
				// Create directories if needed
				if createDirs {
					dir := filepath.Dir(resolvedPath)
					if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // Standard directory permissions
						return nil, fmt.Errorf("failed to create directories: %w", err)
					}
				}

				// Support template content with input data
				finalContent := content
				if strings.Contains(content, "{{") {
					tmpl, err := template.New("content").Parse(content)
					if err == nil {
						var buf bytes.Buffer
						if err := tmpl.Execute(&buf, input); err == nil {
							finalContent = buf.String()
						}
					}
				}

				if err := os.WriteFile(resolvedPath, []byte(finalContent), 0o644); err != nil { //nolint:gosec // Standard file permissions
					return nil, fmt.Errorf("write failed: %w", err)
				}

				info, _ := os.Stat(resolvedPath)
				return map[string]interface{}{
					"path":     resolvedPath,
					"exists":   true,
					"size":     info.Size(),
					"modified": info.ModTime().Format(time.RFC3339),
				}, nil

			case "append":
				// Create directories if needed
				if createDirs {
					dir := filepath.Dir(resolvedPath)
					if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // Standard directory permissions
						return nil, fmt.Errorf("failed to create directories: %w", err)
					}
				}

				// Support template content
				finalContent := content
				if strings.Contains(content, "{{") {
					tmpl, err := template.New("content").Parse(content)
					if err == nil {
						var buf bytes.Buffer
						if err := tmpl.Execute(&buf, input); err == nil {
							finalContent = buf.String()
						}
					}
				}

				file, err := os.OpenFile(resolvedPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // #nosec G302,G304 - Path is validated and sandboxed
				if err != nil {
					return nil, fmt.Errorf("append failed: %w", err)
				}
				defer func() {
					// Check for errors during file close
					if cerr := file.Close(); err == nil && cerr != nil {
						err = fmt.Errorf("failed to close file: %w", cerr)
					}
				}()

				if _, err := file.WriteString(finalContent); err != nil {
					return nil, fmt.Errorf("append write failed: %w", err)
				}

				info, _ := os.Stat(resolvedPath)
				return map[string]interface{}{
					"path":     resolvedPath,
					"exists":   true,
					"size":     info.Size(),
					"modified": info.ModTime().Format(time.RFC3339),
				}, nil

			case "exists":
				info, err := os.Stat(resolvedPath)
				if err != nil {
					if os.IsNotExist(err) {
						return map[string]interface{}{
							"path":   resolvedPath,
							"exists": false,
						}, nil
					}
					return nil, fmt.Errorf("stat failed: %w", err)
				}

				return map[string]interface{}{
					"path":     resolvedPath,
					"exists":   true,
					"size":     info.Size(),
					"modified": info.ModTime().Format(time.RFC3339),
					"isDir":    info.IsDir(),
				}, nil

			case "list":
				entries, err := os.ReadDir(resolvedPath)
				if err != nil {
					if os.IsNotExist(err) {
						return map[string]interface{}{
							"path":   resolvedPath,
							"exists": false,
						}, nil
					}
					return nil, fmt.Errorf("list failed: %w", err)
				}

				files := []interface{}{}
				for _, entry := range entries {
					info, err := entry.Info()
					if err != nil {
						continue
					}

					files = append(files, map[string]interface{}{
						"name":     entry.Name(),
						"path":     filepath.Join(resolvedPath, entry.Name()),
						"size":     info.Size(),
						"modified": info.ModTime().Format(time.RFC3339),
						"isDir":    entry.IsDir(),
					})
				}

				return map[string]interface{}{
					"path":   resolvedPath,
					"exists": true,
					"files":  files,
				}, nil

			default:
				return nil, fmt.Errorf("unknown operation: %s", operation)
			}
		}),
	), nil
}

// resolvePath resolves a file path with sandboxing.
func resolvePath(path, baseDir string, allowAbsolute bool) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// If absolute paths are allowed and path is absolute, return it
	if allowAbsolute && filepath.IsAbs(cleanPath) {
		return cleanPath, nil
	}

	// Otherwise, resolve relative to base directory
	resolvedPath := filepath.Join(baseDir, cleanPath)
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", err
	}

	// Ensure the resolved path is within the base directory
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	// Check if path is within base directory
	relPath, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", err
	}

	// Reject paths that go outside base directory
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path '%s' is outside base directory", path)
	}

	return absPath, nil
}

// ExecNodeBuilder builds command execution nodes.
type ExecNodeBuilder struct {
	Verbose bool
}

func (b *ExecNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "exec",
		Category:    "io",
		Description: "Executes shell commands with restrictions",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Command to execute",
				},
				"args": map[string]interface{}{
					"type":        "array",
					"description": "Command arguments",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"timeout": map[string]interface{}{
					"type":        "string",
					"description": "Execution timeout",
					"default":     "30s",
				},
				"allowed_commands": map[string]interface{}{
					"type":        "array",
					"description": "List of allowed commands (whitelist)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"env": map[string]interface{}{
					"type":        "object",
					"description": "Environment variables to set",
					"additionalProperties": map[string]interface{}{
						"type": "string",
					},
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for command",
				},
				"capture_output": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to capture command output",
					"default":     true,
				},
			},
			"required": []string{"command"},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"stdout": map[string]interface{}{
					"type": "string",
				},
				"stderr": map[string]interface{}{
					"type": "string",
				},
				"exit_code": map[string]interface{}{
					"type": "integer",
				},
				"duration": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Examples: []Example{
			{
				Name:        "List files",
				Description: "List files in current directory",
				Config: map[string]interface{}{
					"command": "ls",
					"args":    []interface{}{"-la"},
				},
			},
			{
				Name:        "Run with timeout",
				Description: "Execute command with timeout",
				Config: map[string]interface{}{
					"command": "sleep",
					"args":    []interface{}{"5"},
					"timeout": "2s",
				},
			},
			{
				Name:        "Restricted commands",
				Description: "Only allow specific commands",
				Config: map[string]interface{}{
					"command":          "echo",
					"args":             []interface{}{"Hello, World!"},
					"allowed_commands": []string{"echo", "ls", "cat"},
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates an exec node from a definition.
//
//nolint:gocyclo // Complex due to security validations, restrictions, and error handling
func (b *ExecNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	// Get command configuration
	command, ok := def.Config["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Get arguments
	var args []string
	if argsRaw, ok := def.Config["args"].([]interface{}); ok {
		for i, arg := range argsRaw {
			argStr, ok := arg.(string)
			if !ok {
				return nil, fmt.Errorf("argument %d must be a string", i)
			}
			args = append(args, argStr)
		}
	}

	// Get timeout
	timeoutStr, _ := def.Config["timeout"].(string)
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil || timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Get allowed commands whitelist
	var allowedCommands []string
	if allowed, ok := def.Config["allowed_commands"].([]interface{}); ok {
		for _, cmd := range allowed {
			if cmdStr, ok := cmd.(string); ok {
				allowedCommands = append(allowedCommands, cmdStr)
			}
		}
	}

	// Get environment variables
	env := make(map[string]string)
	if envMap, ok := def.Config["env"].(map[string]interface{}); ok {
		for k, v := range envMap {
			if vStr, ok := v.(string); ok {
				env[k] = vStr
			}
		}
	}

	// Get working directory
	workingDir, _ := def.Config["working_dir"].(string)

	// Get capture output flag
	captureOutput := true
	if capture, ok := def.Config["capture_output"].(bool); ok {
		captureOutput = capture
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {

			// Check if command is in whitelist
			if len(allowedCommands) > 0 {
				allowed := false
				for _, allowedCmd := range allowedCommands {
					if command == allowedCmd {
						allowed = true
						break
					}
				}
				if !allowed {
					return nil, fmt.Errorf("command '%s' is not in allowed list", command)
				}
			}

			// Create command with timeout context
			execCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Create command
			cmd := exec.CommandContext(execCtx, command, args...) // #nosec G204 - Command is user-configured with restrictions

			// Set working directory if specified
			if workingDir != "" {
				cmd.Dir = workingDir
			}

			// Set environment variables
			if len(env) > 0 {
				cmd.Env = os.Environ() // Start with current environment
				for k, v := range env {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
				}
			}

			// Setup output capture
			var stdout, stderr bytes.Buffer
			if captureOutput {
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
			}

			// Record start time
			startTime := time.Now()

			// Run command
			err := cmd.Run()
			duration := time.Since(startTime)

			// Get exit code
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
					// Check if it was killed due to timeout
					if execCtx.Err() == context.DeadlineExceeded {
						return nil, fmt.Errorf("command timed out after %v", timeout)
					}
				} else if execCtx.Err() == context.DeadlineExceeded {
					return nil, fmt.Errorf("command timed out after %v", timeout)
				} else {
					return nil, fmt.Errorf("command failed: %w", err)
				}
			}

			if b.Verbose {
				log.Printf("[%s] Command executed: %s %v (exit: %d, duration: %v)",
					def.Name, command, args, exitCode, duration)
			}

			result := map[string]interface{}{
				"command":   command,
				"args":      args,
				"exit_code": exitCode,
				"duration":  duration.String(),
			}

			if captureOutput {
				result["stdout"] = stdout.String()
				result["stderr"] = stderr.String()
			}

			return result, nil
		}),
	), nil
}

// ParallelNodeBuilder builds parallel execution nodes.
type ParallelNodeBuilder struct {
	Verbose bool
}

func (b *ParallelNodeBuilder) Metadata() NodeMetadata {
	return NodeMetadata{
		Type:        "parallel",
		Category:    "flow",
		Description: "Executes multiple operations in parallel",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tasks": map[string]interface{}{
					"type":        "array",
					"description": "List of tasks to execute in parallel",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Task name",
							},
							"operation": map[string]interface{}{
								"type":        "string",
								"description": "Operation to perform",
							},
							"config": map[string]interface{}{
								"type":        "object",
								"description": "Task-specific configuration",
							},
						},
						"required": []string{"name", "operation"},
					},
				},
				"max_concurrency": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of concurrent tasks",
					"minimum":     1,
					"default":     10,
				},
				"fail_fast": map[string]interface{}{
					"type":        "boolean",
					"description": "Stop all tasks if one fails",
					"default":     false,
				},
				"timeout": map[string]interface{}{
					"type":        "string",
					"description": "Overall timeout for all tasks",
					"default":     "5m",
				},
			},
			"required": []string{"tasks"},
		},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"results": map[string]interface{}{
					"type":        "array",
					"description": "Results from all tasks",
				},
				"errors": map[string]interface{}{
					"type":        "array",
					"description": "Errors from failed tasks",
				},
				"duration": map[string]interface{}{
					"type":        "string",
					"description": "Total execution time",
				},
			},
		},
		Examples: []Example{
			{
				Name:        "Parallel API calls",
				Description: "Fetch data from multiple APIs concurrently",
				Config: map[string]interface{}{
					"tasks": []interface{}{
						map[string]interface{}{
							"name":      "fetch_users",
							"operation": "http_get",
							"config": map[string]interface{}{
								"url": "https://api.example.com/users",
							},
						},
						map[string]interface{}{
							"name":      "fetch_posts",
							"operation": "http_get",
							"config": map[string]interface{}{
								"url": "https://api.example.com/posts",
							},
						},
					},
					"max_concurrency": 5,
				},
			},
			{
				Name:        "Parallel processing with fail-fast",
				Description: "Process multiple items, stop on first failure",
				Config: map[string]interface{}{
					"tasks": []interface{}{
						map[string]interface{}{
							"name":      "process_1",
							"operation": "transform",
						},
						map[string]interface{}{
							"name":      "process_2",
							"operation": "transform",
						},
					},
					"fail_fast": true,
					"timeout":   "30s",
				},
			},
		},
		Since: "1.0.0",
	}
}

// Build creates a parallel node from a definition.
//
//nolint:gocyclo // Complex due to concurrent execution handling and error aggregation
func (b *ParallelNodeBuilder) Build(def *yaml.NodeDefinition) (pocket.Node, error) {
	// Get tasks configuration
	tasksRaw, ok := def.Config["tasks"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("tasks must be an array")
	}

	// Parse tasks
	type Task struct {
		Name      string
		Operation string
		Config    map[string]interface{}
	}

	tasks := make([]Task, 0, len(tasksRaw))
	for i, taskRaw := range tasksRaw {
		taskMap, ok := taskRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("task %d must be an object", i)
		}

		name, ok := taskMap["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("task %d missing name", i)
		}

		operation, ok := taskMap["operation"].(string)
		if !ok || operation == "" {
			return nil, fmt.Errorf("task %d missing operation", i)
		}

		config, _ := taskMap["config"].(map[string]interface{})

		tasks = append(tasks, Task{
			Name:      name,
			Operation: operation,
			Config:    config,
		})
	}

	// Get other configuration
	maxConcurrency := 10
	if mc, ok := def.Config["max_concurrency"].(float64); ok {
		maxConcurrency = int(mc)
	}

	failFast := false
	if ff, ok := def.Config["fail_fast"].(bool); ok {
		failFast = ff
	}

	timeoutStr, _ := def.Config["timeout"].(string)
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil || timeout <= 0 {
		timeout = 5 * time.Minute
	}

	return pocket.NewNode[any, any](def.Name,
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			// Create timeout context
			execCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Start time tracking
			startTime := time.Now()

			// Create semaphore for concurrency control
			sem := make(chan struct{}, maxConcurrency)

			// Results collection
			type taskResult struct {
				Name   string
				Result interface{}
				Error  error
			}

			resultsChan := make(chan taskResult, len(tasks))

			// Error channel for fail-fast
			var errChan chan error
			if failFast {
				errChan = make(chan error, 1)
			}

			// Launch tasks
			var wg sync.WaitGroup
			for _, task := range tasks {
				wg.Add(1)

				task := task // Capture loop variable
				go func() {
					defer wg.Done()

					// Acquire semaphore
					select {
					case sem <- struct{}{}:
						defer func() { <-sem }()
					case <-execCtx.Done():
						resultsChan <- taskResult{
							Name:  task.Name,
							Error: fmt.Errorf("timeout waiting for concurrency slot"),
						}
						return
					}

					// Check for early termination
					if failFast {
						select {
						case <-execCtx.Done():
							return
						default:
						}
					}

					// Execute task
					result, err := executeTask(execCtx, task, input)

					// Send result
					resultsChan <- taskResult{
						Name:   task.Name,
						Result: result,
						Error:  err,
					}

					// Handle fail-fast
					if failFast && err != nil {
						select {
						case errChan <- err:
							cancel() // Cancel all other tasks
						default:
						}
					}
				}()
			}

			// Wait for all tasks to complete
			go func() {
				wg.Wait()
				close(resultsChan)
			}()

			// Collect results
			var results []interface{}
			var errors []interface{}
			resultMap := make(map[string]interface{})

			for res := range resultsChan {
				if res.Error != nil {
					errors = append(errors, map[string]interface{}{
						"task":  res.Name,
						"error": res.Error.Error(),
					})
					if b.Verbose {
						log.Printf("[%s] Task %s failed: %v", def.Name, res.Name, res.Error)
					}
				} else {
					results = append(results, map[string]interface{}{
						"task":   res.Name,
						"result": res.Result,
					})
					resultMap[res.Name] = res.Result
					if b.Verbose {
						log.Printf("[%s] Task %s completed successfully", def.Name, res.Name)
					}
				}
			}

			duration := time.Since(startTime)

			// Check if we should fail
			if failFast && len(errors) > 0 {
				return nil, fmt.Errorf("parallel execution failed (fail-fast): %d errors", len(errors))
			}

			if b.Verbose {
				log.Printf("[%s] Parallel execution completed: %d successful, %d failed, duration: %v",
					def.Name, len(results), len(errors), duration)
			}

			return map[string]interface{}{
				"results":  results,
				"errors":   errors,
				"duration": duration.String(),
				"summary": map[string]interface{}{
					"total":      len(tasks),
					"successful": len(results),
					"failed":     len(errors),
				},
			}, nil
		}),
	), nil
}

// executeTask simulates task execution - in a real implementation,
// this would delegate to actual node implementations.
func executeTask(_ context.Context, task struct {
	Name      string
	Operation string
	Config    map[string]interface{}
}, input interface{}) (interface{}, error) {
	// Simulate different operations
	switch task.Operation {
	case "http_get":
		// Simulate HTTP GET
		time.Sleep(100 * time.Millisecond)
		return map[string]interface{}{
			"status": 200,
			"data":   fmt.Sprintf("Data from %s", task.Name),
		}, nil

	case "transform":
		// Simulate data transformation
		time.Sleep(50 * time.Millisecond)
		return map[string]interface{}{
			"transformed": true,
			"task":        task.Name,
			"input":       input,
		}, nil

	case "error":
		// Simulate error for testing
		return nil, fmt.Errorf("simulated error for task %s", task.Name)

	default:
		return map[string]interface{}{
			"task":      task.Name,
			"operation": task.Operation,
			"input":     input,
		}, nil
	}
}
