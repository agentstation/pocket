// Package main demonstrates YAML support in Pocket, showing how YAML
// provides better token efficiency and readability compared to JSON
// when working with LLMs.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/agentstation/pocket"
	"gopkg.in/yaml.v3"
)

// Resume represents a structured resume format
type Resume struct {
	Name       string      `yaml:"name" json:"name"`
	Email      string      `yaml:"email" json:"email"`
	Experience []JobDetail `yaml:"experience" json:"experience"`
	Skills     []string    `yaml:"skills" json:"skills"`
}

// JobDetail represents job experience
type JobDetail struct {
	Employer string `yaml:"employer" json:"employer"`
	Role     string `yaml:"role" json:"role"`
	Years    int    `yaml:"years" json:"years"`
}

func main() {
	store := pocket.NewStore()
	ctx := context.Background()

	fmt.Println("=== YAML Support Demo ===")
	fmt.Println()

	// Demo 1: Basic YAML Node
	fmt.Println("1. Basic YAML Node Output")
	fmt.Println("-------------------------")

	yamlExtractor := pocket.NewNode[any, any]("extract-data",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			text := input.(string)
			
			// Simulate extracting structured data from text
			data := map[string]interface{}{
				"source_text": text,
				"extracted": map[string]interface{}{
					"entities": []string{"Pocket", "YAML", "Go"},
					"topics":   []string{"framework", "serialization", "efficiency"},
					"metadata": map[string]interface{}{
						"word_count": len(strings.Fields(text)),
						"has_code":   strings.Contains(text, "func"),
					},
				},
			}
			
			// Convert to YAML
			yamlBytes, err := yaml.Marshal(data)
			if err != nil {
				return nil, err
			}
			
			return string(yamlBytes), nil
		}),
	)

	flow := pocket.NewFlow(yamlExtractor, store)
	result, err := flow.Run(ctx, "Pocket framework supports YAML for better token efficiency with LLMs")
	if err != nil {
		log.Fatal(err)
	}

	if yamlStr, ok := result.(string); ok {
		fmt.Println("YAML Output:")
		fmt.Println(yamlStr)
	}

	// Demo 2: Token Comparison - YAML vs JSON
	fmt.Println("\n2. Token Efficiency Comparison")
	fmt.Println("------------------------------")

	sampleData := Resume{
		Name:  "Jane Developer",
		Email: "jane@example.com",
		Experience: []JobDetail{
			{Employer: "TechCorp", Role: "Senior Engineer", Years: 3},
			{Employer: "StartupXYZ", Role: "Lead Developer", Years: 2},
		},
		Skills: []string{"Go", "Python", "Kubernetes", "AWS"},
	}

	// Convert to YAML
	yamlBytes, _ := yaml.Marshal(sampleData)
	yamlStr := string(yamlBytes)
	fmt.Println("YAML Format:")
	fmt.Println(yamlStr)

	// Convert to JSON for comparison
	jsonBytes, _ := json.MarshalIndent(sampleData, "", "  ")
	jsonStr := string(jsonBytes)
	fmt.Println("\nJSON Format:")
	fmt.Println(jsonStr)

	// Compare tokens (simplified)
	yamlChars := len(yamlStr)
	jsonChars := len(jsonStr)
	savings := float64(jsonChars-yamlChars) / float64(jsonChars) * 100

	fmt.Printf("\nCharacter Count:")
	fmt.Printf("\n  YAML: %d chars", yamlChars)
	fmt.Printf("\n  JSON: %d chars", jsonChars)
	fmt.Printf("\n  Savings: %.1f%%\n", savings)

	// Demo 3: Structured Output Node with Schema
	fmt.Println("\n3. Structured Output Node")
	fmt.Println("-------------------------")

	// Define schema and examples
	schema := Resume{}
	examples := []any{
		Resume{
			Name:  "Alice Smith",
			Email: "alice@example.com",
			Experience: []JobDetail{
				{Employer: "ExampleCorp", Role: "Engineer", Years: 3},
			},
			Skills: []string{"Go", "Docker"},
		},
	}

	structuredNode := pocket.NewNode[any, any]("resume-extractor",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// Prepare schema and examples for reference
			schemaYAML, _ := yaml.Marshal(schema)
			exampleYAML, _ := yaml.Marshal(examples[0])
			
			return map[string]interface{}{
				"input": input,
				"schemaYAML": string(schemaYAML),
				"exampleYAML": string(exampleYAML),
			}, nil
		}),
		pocket.WithExec(func(ctx context.Context, prepData any) (any, error) {
			// Extract prep data
			data := prepData.(map[string]interface{})
			input := data["input"]
			
			// Simulate structured extraction
			result := map[string]interface{}{
				"input":     input,
				"processed": true,
				"timestamp": "2024-01-01T00:00:00Z",
			}
			
			yamlBytes, err := yaml.Marshal(result)
			if err != nil {
				return nil, err
			}
			
			return map[string]interface{}{
				"yamlOutput": string(yamlBytes),
				"schemaYAML": data["schemaYAML"],
				"exampleYAML": data["exampleYAML"],
			}, nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prepData, result any) (any, string, error) {
			// Extract exec result
			execResult := result.(map[string]interface{})
			
			// Store schema and example for reference
			store.Set(ctx, "schema", execResult["schemaYAML"])
			store.Set(ctx, "example", execResult["exampleYAML"])
			
			// Return the YAML output
			return execResult["yamlOutput"], "default", nil
		}),
	)

	// Create a pipeline that processes raw text
	pipeline := pocket.NewNode[any, any]("pipeline",
		pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
			// In real use, this would prepare a prompt for an LLM
			fmt.Println("Preparing structured extraction prompt...")
			return input, nil
		}),
	)

	pipeline.Connect("default", structuredNode)

	flow = pocket.NewFlow(pipeline, store)
	result, err = flow.Run(ctx, "John Doe, john@example.com, 5 years at TechCorp as Senior Engineer")
	if err != nil {
		log.Printf("Error: %v", err)
	}

	if yamlStr, ok := result.(string); ok {
		fmt.Println("\nStructured Output (YAML):")
		fmt.Println(yamlStr)
	}

	// Demo 4: Parse YAML Response
	fmt.Println("\n4. Parsing YAML Response")
	fmt.Println("------------------------")

	// Simulate an LLM response in YAML format
	llmResponse := `
name: Bob Johnson
email: bob@techcorp.com
experience:
  - employer: TechCorp
    role: Principal Engineer
    years: 5
  - employer: StartupABC
    role: CTO
    years: 3
skills:
  - Go
  - Rust
  - System Design
  - Leadership
`

	var parsedResume Resume
	err = yaml.Unmarshal([]byte(llmResponse), &parsedResume)
	if err != nil {
		log.Printf("Parse error: %v", err)
	} else {
		fmt.Printf("Parsed Resume:\n")
		fmt.Printf("  Name: %s\n", parsedResume.Name)
		fmt.Printf("  Email: %s\n", parsedResume.Email)
		fmt.Printf("  Experience: %d positions\n", len(parsedResume.Experience))
		fmt.Printf("  Skills: %v\n", parsedResume.Skills)
	}

	// Demo 5: YAML in Workflow with Routing
	fmt.Println("\n5. YAML-based Routing Workflow")
	fmt.Println("------------------------------")

	// Create a classifier that outputs YAML with routing info
	classifier := pocket.NewNode[any, any]("classifier",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			text := input.(string)
			
			// Classify text and return structured result
			classification := map[string]interface{}{
				"text":       text,
				"category":   "technical",
				"confidence": 0.95,
				"topics":     []string{"programming", "yaml"},
				"route":      "technical-handler",
			}
			
			if strings.Contains(strings.ToLower(text), "business") {
				classification["category"] = "business"
				classification["route"] = "business-handler"
			}
			
			// Convert to YAML
			yamlBytes, err := yaml.Marshal(classification)
			if err != nil {
				return nil, err
			}
			
			return string(yamlBytes), nil
		}),
		pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, input, prep, result any) (any, string, error) {
			// Extract routing from YAML output
			if yamlStr, ok := result.(string); ok {
				// Parse YAML to get route
				var data map[string]interface{}
				if err := yaml.Unmarshal([]byte(yamlStr), &data); err == nil {
					if route, ok := data["route"].(string); ok {
						return yamlStr, route, nil
					}
				}
			}
			return result, "default", nil
		}),
	)

	// Create handlers
	techHandler := pocket.NewNode[any, any]("technical-handler",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			fmt.Println("→ Routed to Technical Handler")
			return "Technical content processed", nil
		}),
	)

	businessHandler := pocket.NewNode[any, any]("business-handler",
		pocket.WithExec(func(ctx context.Context, input any) (any, error) {
			fmt.Println("→ Routed to Business Handler")
			return "Business content processed", nil
		}),
	)

	// Connect based on classification
	classifier.Connect("technical-handler", techHandler)
	classifier.Connect("business-handler", businessHandler)

	// Test routing
	routingFlow := pocket.NewFlow(classifier, store)
	
	testInputs := []string{
		"How to implement YAML parsing in Go",
		"Business strategy for Q4 revenue growth",
	}

	for _, input := range testInputs {
		fmt.Printf("\nInput: %q\n", input)
		_, err := routingFlow.Run(ctx, input)
		if err != nil {
			log.Printf("Routing error: %v", err)
		}
	}

	// Demo 6: YAML Benefits for LLM Prompting
	fmt.Println("\n\n6. YAML Benefits for LLM Prompting")
	fmt.Println("----------------------------------")

	fmt.Println("\nWhy YAML is preferred for LLM structured output:")
	fmt.Println("1. Lower token usage (~30% fewer tokens than JSON)")
	fmt.Println("2. More forgiving syntax (quotes, escaping)")  
	fmt.Println("3. Support for comments and documentation")
	fmt.Println("4. Human-readable multiline strings")
	fmt.Println("5. Cleaner array/list syntax")

	// Example prompt template
	promptTemplate := `
You are extracting structured data.
Return ONLY valid YAML.

Schema:
%s

Example:
%s

Extract from: %s
`

	schemaBytes, _ := yaml.Marshal(schema)
	schemaYAML := string(schemaBytes)
	exampleBytes, _ := yaml.Marshal(examples[0])
	exampleYAML := string(exampleBytes)
	
	fmt.Println("\nExample LLM Prompt:")
	fmt.Printf(promptTemplate, schemaYAML, exampleYAML, "Jane Doe, Senior Engineer at TechCorp...")

	fmt.Println("\n=== Demo Complete ===")
}