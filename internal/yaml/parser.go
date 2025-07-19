package yaml

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// Parser handles parsing YAML flow definitions.
type Parser struct {
	// Future: Add schema validation, custom marshalers, etc.
}

// NewParser creates a new YAML parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads and parses a YAML flow definition from a reader.
func (p *Parser) Parse(r io.Reader) (*FlowDefinition, error) {
	// Note: We're not importing yaml.v3 to keep the package dependency-free.
	// This is a placeholder that would need a YAML library in practice.
	// For now, we'll return an error indicating YAML support needs to be enabled.
	return nil, fmt.Errorf("YAML parsing requires importing a YAML library (e.g., gopkg.in/yaml.v3)")
}

// ParseFile reads and parses a YAML flow definition from a file.
func (p *Parser) ParseFile(filename string) (*FlowDefinition, error) {
	// #nosec G304 - This is a parser that needs to accept arbitrary file paths
	// In production, callers should validate the path based on their security requirements
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return p.Parse(file)
}

// ParseString parses a YAML flow definition from a string.
func (p *Parser) ParseString(s string) (*FlowDefinition, error) {
	return p.Parse(bytes.NewReader([]byte(s)))
}

// Marshal converts a flow definition to YAML format.
func (p *Parser) Marshal(fd *FlowDefinition) ([]byte, error) {
	// Placeholder - would use YAML library
	return nil, fmt.Errorf("YAML marshaling requires importing a YAML library")
}

// MarshalToFile writes a flow definition to a YAML file.
func (p *Parser) MarshalToFile(fd *FlowDefinition, filename string) error {
	data, err := p.Marshal(fd)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0o600)
}

// Example shows what a YAML flow definition would look like.
func Example() string {
	return `name: chat_workflow
description: Multi-agent chat workflow with routing
version: "1.0.0"
start: input_validator

nodes:
  - name: input_validator
    type: validator
    config:
      required_fields: ["message", "user_id"]
    timeout: 5s
    
  - name: intent_classifier
    type: llm
    config:
      model: "gpt-4"
      prompt: "Classify the intent of this message: {{.message}}"
    retry:
      max_attempts: 3
      delay: 1s
      multiplier: 2
      
  - name: knowledge_search
    type: rag
    config:
      index: "knowledge_base"
      top_k: 5
      
  - name: response_generator
    type: llm
    config:
      model: "gpt-4"
      prompt: "Generate a response based on: {{.context}}"
      
  - name: output_formatter
    type: formatter
    config:
      format: "markdown"

connections:
  - from: input_validator
    to: intent_classifier
    action: valid
    
  - from: intent_classifier
    to: knowledge_search
    action: needs_info
    
  - from: intent_classifier
    to: response_generator
    action: direct_response
    
  - from: knowledge_search
    to: response_generator
    action: default
    
  - from: response_generator
    to: output_formatter
    action: default
`
}
