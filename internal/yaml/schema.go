// Package yaml provides YAML-based flow definition support for pocket.
package yaml

import (
	"fmt"
	"time"
)

// FlowDefinition represents a complete flow defined in YAML.
type FlowDefinition struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description,omitempty"`
	Version     string                 `yaml:"version,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
	Nodes       []NodeDefinition       `yaml:"nodes"`
	Connections []Connection           `yaml:"connections,omitempty"`
	Start       string                 `yaml:"start"`
}

// NodeDefinition represents a node in YAML format.
type NodeDefinition struct {
	Name        string                 `yaml:"name"`
	Type        string                 `yaml:"type"`
	Description string                 `yaml:"description,omitempty"`
	Config      map[string]interface{} `yaml:"config,omitempty"`
	Retry       *RetryConfig           `yaml:"retry,omitempty"`
	Timeout     string                 `yaml:"timeout,omitempty"`
	InputType   string                 `yaml:"input_type,omitempty"`
	OutputType  string                 `yaml:"output_type,omitempty"`
}

// Connection represents a connection between nodes.
type Connection struct {
	From   string `yaml:"from"`
	To     string `yaml:"to"`
	Action string `yaml:"action,omitempty"`
}

// RetryConfig represents retry configuration in YAML.
type RetryConfig struct {
	MaxAttempts int    `yaml:"max_attempts"`
	Delay       string `yaml:"delay"`
	Multiplier  float64 `yaml:"multiplier,omitempty"`
	MaxDelay    string `yaml:"max_delay,omitempty"`
}

// Validate checks if the flow definition is valid.
func (fd *FlowDefinition) Validate() error {
	if fd.Name == "" {
		return fmt.Errorf("flow name is required")
	}
	if fd.Start == "" {
		return fmt.Errorf("start node is required")
	}
	if len(fd.Nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}

	// Check if start node exists
	nodeMap := make(map[string]bool)
	for _, node := range fd.Nodes {
		if node.Name == "" {
			return fmt.Errorf("node name is required")
		}
		if node.Type == "" {
			return fmt.Errorf("node type is required for node %s", node.Name)
		}
		nodeMap[node.Name] = true
	}

	if !nodeMap[fd.Start] {
		return fmt.Errorf("start node %s not found", fd.Start)
	}

	// Validate connections
	for _, conn := range fd.Connections {
		if !nodeMap[conn.From] {
			return fmt.Errorf("connection from node %s not found", conn.From)
		}
		if !nodeMap[conn.To] {
			return fmt.Errorf("connection to node %s not found", conn.To)
		}
	}

	// Validate individual nodes
	for _, node := range fd.Nodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("node %s: %w", node.Name, err)
		}
	}

	return nil
}

// Validate checks if the node definition is valid.
func (nd *NodeDefinition) Validate() error {
	// Validate timeout if specified
	if nd.Timeout != "" {
		if _, err := time.ParseDuration(nd.Timeout); err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
	}

	// Validate retry config if specified
	if nd.Retry != nil {
		if err := nd.Retry.Validate(); err != nil {
			return fmt.Errorf("invalid retry config: %w", err)
		}
	}

	return nil
}

// Validate checks if the retry config is valid.
func (rc *RetryConfig) Validate() error {
	if rc.MaxAttempts <= 0 {
		return fmt.Errorf("max_attempts must be positive")
	}

	if rc.Delay == "" {
		return fmt.Errorf("delay is required")
	}
	if _, err := time.ParseDuration(rc.Delay); err != nil {
		return fmt.Errorf("invalid delay: %w", err)
	}

	if rc.MaxDelay != "" {
		if _, err := time.ParseDuration(rc.MaxDelay); err != nil {
			return fmt.Errorf("invalid max_delay: %w", err)
		}
	}

	if rc.Multiplier < 0 {
		return fmt.Errorf("multiplier cannot be negative")
	}

	return nil
}

// GetTimeout returns the parsed timeout duration.
func (nd *NodeDefinition) GetTimeout() (time.Duration, error) {
	if nd.Timeout == "" {
		return 0, nil
	}
	return time.ParseDuration(nd.Timeout)
}

// GetRetryDelay returns the parsed retry delay duration.
func (rc *RetryConfig) GetRetryDelay() (time.Duration, error) {
	return time.ParseDuration(rc.Delay)
}

// GetMaxDelay returns the parsed max delay duration.
func (rc *RetryConfig) GetMaxDelay() (time.Duration, error) {
	if rc.MaxDelay == "" {
		return 0, nil
	}
	return time.ParseDuration(rc.MaxDelay)
}