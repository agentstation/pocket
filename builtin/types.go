package builtin

// NodeMetadata describes a node type.
type NodeMetadata struct {
	Type         string                 `json:"type"`
	Category     string                 `json:"category"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"inputSchema,omitempty"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
	ConfigSchema map[string]interface{} `json:"configSchema"`
	Examples     []Example              `json:"examples,omitempty"`
	Since        string                 `json:"since,omitempty"`
}

// Example shows how to use a node.
type Example struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Input       interface{}            `json:"input,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
}
