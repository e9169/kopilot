package llm

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// DefineTool creates a typed llm.Tool from a struct type.
// It reflects on T to generate a JSON Schema for the parameters.
func DefineTool[T any](name string, description string, handler func(params T, inv ToolInvocation) (any, error)) Tool {
	var zero T
	reflector := jsonschema.Reflector{
		ExpandedStruct: true, // expand embedded structs
		DoNotReference: true, // don't use $ref, inline everything
	}
	schema := reflector.Reflect(&zero)

	// Convert jsonschema.Schema to map[string]any for the Tool.Parameters field
	b, err := json.Marshal(schema)
	var paramsMap map[string]any
	if err == nil {
		_ = json.Unmarshal(b, &paramsMap)
	}

	return Tool{
		Name:        name,
		Description: description,
		Parameters:  paramsMap,
		Handler: func(params any, inv ToolInvocation) (any, error) {
			var typedParams T
			// Convert any to struct
			if params != nil {
				b, err := json.Marshal(params)
				if err == nil {
					_ = json.Unmarshal(b, &typedParams)
				}
			}
			return handler(typedParams, inv)
		},
	}
}
