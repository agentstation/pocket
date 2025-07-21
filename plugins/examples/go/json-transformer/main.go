//go:build wasm

package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

// Plugin metadata types.
type Metadata struct {
	Name         string           `json:"name"`
	Version      string           `json:"version"`
	Description  string           `json:"description"`
	Author       string           `json:"author"`
	License      string           `json:"license"`
	Runtime      string           `json:"runtime"`
	Binary       string           `json:"binary"`
	Nodes        []NodeDefinition `json:"nodes"`
	Permissions  Permissions      `json:"permissions"`
	Requirements Requirements     `json:"requirements"`
}

type NodeDefinition struct {
	Type         string      `json:"type"`
	Category     string      `json:"category"`
	Description  string      `json:"description"`
	ConfigSchema interface{} `json:"configSchema,omitempty"`
	InputSchema  interface{} `json:"inputSchema,omitempty"`
	OutputSchema interface{} `json:"outputSchema,omitempty"`
}

type Permissions struct {
	Memory  string `json:"memory"`
	Timeout int    `json:"timeout"`
}

type Requirements struct {
	Pocket string `json:"pocket"`
}

// Request/Response types.
type Request struct {
	Node     string          `json:"node"`
	Function string          `json:"function"`
	Config   json.RawMessage `json:"config,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

type Response struct {
	Success bool            `json:"success"`
	Output  json.RawMessage `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
	Next    string          `json:"next,omitempty"`
}

// JSON transformer specific types.
type TransformerInput struct {
	Data      interface{} `json:"data"`
	Transform string      `json:"transform"`
}

type TransformerConfig struct {
	Transforms   map[string]TransformSpec `json:"transforms"`
	DefaultDepth int                      `json:"default_depth"`
}

type TransformSpec struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// Memory management exports
//
//export alloc
func alloc(size uint32) uint32 {
	buf := make([]byte, size)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

// Metadata export
//
//export metadata
func metadata(ptr uint32, size uint32) uint32 {
	meta := Metadata{
		Name:        "json-transformer",
		Version:     "1.0.0",
		Description: "JSON transformation plugin for Pocket",
		Author:      "Pocket Team",
		License:     "MIT",
		Runtime:     "wasm",
		Binary:      "plugin.wasm",
		Nodes: []NodeDefinition{
			{
				Type:        "json-transform",
				Category:    "data",
				Description: "Transform JSON data using various operations",
				ConfigSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"transforms": map[string]interface{}{
							"type":        "object",
							"description": "Named transform specifications",
							"additionalProperties": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"type": map[string]interface{}{
										"type": "string",
										"enum": []string{"flatten", "nest", "filter", "map", "reduce"},
									},
									"parameters": map[string]interface{}{
										"type": "object",
									},
								},
								"required": []string{"type"},
							},
						},
						"default_depth": map[string]interface{}{
							"type":        "integer",
							"default":     10,
							"description": "Default depth for nested operations",
						},
					},
				},
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"data": map[string]interface{}{
							"description": "JSON data to transform",
						},
						"transform": map[string]interface{}{
							"type":        "string",
							"description": "Name of transform to apply",
						},
					},
					"required": []string{"data", "transform"},
				},
				OutputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"result": map[string]interface{}{
							"description": "Transformed JSON data",
						},
						"metadata": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"operation": map[string]interface{}{
									"type": "string",
								},
								"changes": map[string]interface{}{
									"type": "integer",
								},
							},
						},
					},
					"required": []string{"result", "metadata"},
				},
			},
		},
		Permissions: Permissions{
			Memory:  "10MB",
			Timeout: 5000,
		},
		Requirements: Requirements{
			Pocket: ">=1.0.0",
		},
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return 0
	}

	copyToWasm(ptr, data)
	return uint32(len(data))
}

// Main call export
//
//export call
func call(ptr uint32, size uint32, outPtr uint32, outSize uint32) uint32 {
	input := readFromWasm(ptr, size)

	var req Request
	if err := json.Unmarshal(input, &req); err != nil {
		resp := Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse request: %v", err),
		}
		return writeResponse(resp, outPtr, outSize)
	}

	var resp Response
	switch req.Function {
	case "prep":
		resp = handlePrep(&req)
	case "exec":
		resp = handleExec(&req)
	case "post":
		resp = handlePost(&req)
	default:
		resp = Response{
			Success: false,
			Error:   fmt.Sprintf("Unknown function: %s", req.Function),
		}
	}

	return writeResponse(resp, outPtr, outSize)
}

func handlePrep(req *Request) Response {
	var input TransformerInput
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse input: %v", err),
		}
	}

	// Validate transform name
	var config TransformerConfig
	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &config); err != nil {
			return Response{
				Success: false,
				Error:   fmt.Sprintf("Failed to parse config: %v", err),
			}
		}
	}

	if config.Transforms == nil {
		config.Transforms = getDefaultTransforms()
	}

	if _, exists := config.Transforms[input.Transform]; !exists {
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Unknown transform: %s", input.Transform),
		}
	}

	prepData := map[string]interface{}{
		"data":      input.Data,
		"transform": input.Transform,
		"spec":      config.Transforms[input.Transform],
	}

	output, _ := json.Marshal(prepData)
	return Response{
		Success: true,
		Output:  output,
	}
}

func handleExec(req *Request) Response {
	var prepData map[string]interface{}
	if err := json.Unmarshal(req.Input, &prepData); err != nil {
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse prep data: %v", err),
		}
	}

	data := prepData["data"]
	transformName := prepData["transform"].(string)
	spec := prepData["spec"].(map[string]interface{})

	transformType := spec["type"].(string)
	params, _ := spec["parameters"].(map[string]interface{})

	var result interface{}
	var changes int

	switch transformType {
	case "flatten":
		result, changes = flattenJSON(data, params)
	case "nest":
		result, changes = nestJSON(data, params)
	case "filter":
		result, changes = filterJSON(data, params)
	case "map":
		result, changes = mapJSON(data, params)
	case "reduce":
		result, changes = reduceJSON(data, params)
	default:
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Unknown transform type: %s", transformType),
		}
	}

	output := map[string]interface{}{
		"result": result,
		"metadata": map[string]interface{}{
			"operation": transformType,
			"changes":   changes,
		},
	}

	outputData, _ := json.Marshal(output)
	return Response{
		Success: true,
		Output:  outputData,
	}
}

func handlePost(req *Request) Response {
	var execResult map[string]interface{}
	if err := json.Unmarshal(req.Input, &execResult); err != nil {
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse exec result: %v", err),
		}
	}

	metadata := execResult["metadata"].(map[string]interface{})
	changes := int(metadata["changes"].(float64))

	// Route based on changes
	next := "done"
	if changes == 0 {
		next = "no-changes"
	} else if changes > 100 {
		next = "many-changes"
	} else {
		next = "some-changes"
	}

	return Response{
		Success: true,
		Output:  req.Input,
		Next:    next,
	}
}

// Transform implementations.
func flattenJSON(data interface{}, params map[string]interface{}) (interface{}, int) {
	separator := "."
	if sep, ok := params["separator"].(string); ok {
		separator = sep
	}

	result := make(map[string]interface{})
	changes := 0
	flattenHelper(data, "", separator, result, &changes)
	return result, changes
}

func flattenHelper(data interface{}, prefix, separator string, result map[string]interface{}, changes *int) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + separator + key
			}
			flattenHelper(val, newKey, separator, result, changes)
			*changes++
		}
	case []interface{}:
		for i, val := range v {
			newKey := fmt.Sprintf("%s[%d]", prefix, i)
			flattenHelper(val, newKey, separator, result, changes)
			*changes++
		}
	default:
		result[prefix] = v
	}
}

func nestJSON(data interface{}, params map[string]interface{}) (interface{}, int) {
	separator := "."
	if sep, ok := params["separator"].(string); ok {
		separator = sep
	}

	if flat, ok := data.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		changes := 0

		for key, value := range flat {
			parts := strings.Split(key, separator)
			current := result

			for i, part := range parts[:len(parts)-1] {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
					changes++
				}
				current = current[part].(map[string]interface{})
			}

			current[parts[len(parts)-1]] = value
			changes++
		}

		return result, changes
	}

	return data, 0
}

func filterJSON(data interface{}, params map[string]interface{}) (interface{}, int) {
	fields, _ := params["fields"].([]interface{})
	exclude, _ := params["exclude"].(bool)

	if m, ok := data.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		changes := 0

		fieldSet := make(map[string]bool)
		for _, f := range fields {
			if field, ok := f.(string); ok {
				fieldSet[field] = true
			}
		}

		for key, value := range m {
			_, inSet := fieldSet[key]
			if (inSet && !exclude) || (!inSet && exclude) {
				result[key] = value
			} else {
				changes++
			}
		}

		return result, changes
	}

	return data, 0
}

func mapJSON(data interface{}, params map[string]interface{}) (interface{}, int) {
	// Simple mapping example - uppercase all string values
	changes := 0
	return mapHelper(data, &changes), changes
}

func mapHelper(data interface{}, changes *int) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = mapHelper(val, changes)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = mapHelper(val, changes)
		}
		return result
	case string:
		*changes++
		return strings.ToUpper(v)
	default:
		return v
	}
}

func reduceJSON(data interface{}, params map[string]interface{}) (interface{}, int) {
	// Simple reduction - count all elements
	count := 0
	countElements(data, &count)

	return map[string]interface{}{
		"total_elements": count,
		"type":           reflect.TypeOf(data).String(),
	}, count
}

func countElements(data interface{}, count *int) {
	switch v := data.(type) {
	case map[string]interface{}:
		*count += len(v)
		for _, val := range v {
			countElements(val, count)
		}
	case []interface{}:
		*count += len(v)
		for _, val := range v {
			countElements(val, count)
		}
	default:
		*count++
	}
}

// Helper functions.
func getDefaultTransforms() map[string]TransformSpec {
	return map[string]TransformSpec{
		"flatten": {
			Type: "flatten",
			Parameters: map[string]interface{}{
				"separator": ".",
			},
		},
		"nest": {
			Type: "nest",
			Parameters: map[string]interface{}{
				"separator": ".",
			},
		},
		"uppercase": {
			Type:       "map",
			Parameters: map[string]interface{}{},
		},
		"count": {
			Type:       "reduce",
			Parameters: map[string]interface{}{},
		},
	}
}

func readFromWasm(ptr uint32, size uint32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
}

func copyToWasm(ptr uint32, data []byte) {
	dst := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len(data))
	copy(dst, data)
}

func writeResponse(resp Response, ptr uint32, maxSize uint32) uint32 {
	data, err := json.Marshal(resp)
	if err != nil {
		// Write error response
		errResp := Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal response: %v", err),
		}
		data, _ = json.Marshal(errResp)
	}

	size := uint32(len(data))
	if size > maxSize {
		size = maxSize
	}

	copyToWasm(ptr, data[:size])
	return size
}

func main() {
	// Required for WASM
}
