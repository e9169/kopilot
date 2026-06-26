package llm

import (
	"encoding/json"
	"fmt"
	"strings"

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

	// Convert jsonschema.Schema to map[string]any for the Tool.Parameters field.
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
			// Convert any to struct.
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

// ParseToolArgumentsString parses a JSON object argument string into a map.
// Invalid or non-object input is preserved under the "raw" key.
func ParseToolArgumentsString(arguments string) map[string]any {
	if strings.TrimSpace(arguments) == "" {
		return map[string]any{}
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(arguments), &params); err == nil && params != nil {
		return params
	}
	return map[string]any{"raw": arguments}
}

// NormalizeToolArguments converts arbitrary tool arguments into params and a raw JSON string.
func NormalizeToolArguments(arguments any) (map[string]any, string) {
	if arguments == nil {
		return map[string]any{}, "{}"
	}
	raw := ResultString(arguments)
	params := ParseToolArgumentsString(raw)
	return params, raw
}

// ResultString converts a tool result into a JSON string when possible.
func ResultString(result any) string {
	resultBytes, err := json.Marshal(result)
	if err == nil {
		return string(resultBytes)
	}
	return fmt.Sprintf("%v", result)
}

// ResultMap converts a tool result into a map for providers that require object output.
func ResultMap(result any) map[string]any {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return map[string]any{"result": fmt.Sprintf("%v", result)}
	}
	var resultMap map[string]any
	if err := json.Unmarshal(resultBytes, &resultMap); err == nil && resultMap != nil {
		return resultMap
	}
	return map[string]any{"result": string(resultBytes)}
}

// InvokeTool dispatches a tool invocation against a tool map.
func InvokeTool(toolMap map[string]Tool, params map[string]any, inv ToolInvocation) (any, error) {
	toolDef, ok := toolMap[inv.Name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %s", inv.Name)
	}
	return toolDef.Handler(params, inv)
}
