package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/e9169/kopilot/pkg/k8s"
	"github.com/e9169/kopilot/pkg/llm"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RunMCPServer starts kopilot as a stdio MCP server, exposing the 6 Kubernetes
// tools to any MCP client (e.g. Claude Code). No LLM provider is instantiated.
//
// Write operations via kubectl_exec are blocked (ModeReadOnly) because
// interactive confirmation prompts would corrupt the JSON-RPC stream on stdio.
// Blocks until the client closes stdin.
func RunMCPServer(k8sProvider *k8s.Provider) error {
	log.SetOutput(os.Stderr)

	state := &agentState{
		mode:         ModeReadOnly,
		outputFormat: OutputJSON,
	}

	tools := defineK8sTools(k8sProvider, state)

	s := server.NewMCPServer("kopilot", AppVersion)
	for _, t := range tools {
		mcpTool, handler := bridgeTool(t)
		s.AddTool(mcpTool, handler)
	}

	log.Printf("kopilot MCP server ready (%d tools, read-only)", len(tools))
	return server.NewStdioServer(s).Listen(context.Background(), os.Stdin, os.Stdout)
}

// bridgeTool converts a kopilot llm.Tool into an mcp.Tool + handler pair.
//
// Schema: the existing map[string]any is marshalled to json.RawMessage and
// passed via NewToolWithRawSchema, preserving required/items/nested schemas.
// Top-level $schema/$id/title keys are stripped — strict MCP clients reject them.
//
// Handler: req.GetArguments() returns map[string]any which is passed directly to
// the llm.Tool handler; DefineTool's generic wrapper re-marshals it to the typed
// struct internally, so no conversion is needed here.
func bridgeTool(t llm.Tool) (mcp.Tool, server.ToolHandlerFunc) {
	params := t.Parameters
	delete(params, "$schema")
	delete(params, "$id")
	delete(params, "title")

	rawSchema, err := json.Marshal(params)
	if err != nil {
		rawSchema = json.RawMessage(`{"type":"object","properties":{}}`)
	}

	mcpTool := mcp.NewToolWithRawSchema(t.Name, t.Description, rawSchema)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := t.Handler(req.GetArguments(), llm.ToolInvocation{Name: req.Params.Name})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return toolResultToMCP(result), nil
	}

	return mcpTool, handler
}

// toolResultToMCP wraps an llm.Tool handler result in an MCP text content item.
// Strings are used as-is; other types are JSON-marshalled.
func toolResultToMCP(result any) *mcp.CallToolResult {
	switch v := result.(type) {
	case string:
		return mcp.NewToolResultText(v)
	case nil:
		return mcp.NewToolResultText("")
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to serialize result: %v", err))
		}
		return mcp.NewToolResultText(string(b))
	}
}
