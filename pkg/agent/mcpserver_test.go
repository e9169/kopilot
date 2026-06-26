package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/e9169/kopilot/pkg/llm"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestDefineK8sToolsCount(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputJSON}
	tools := defineK8sTools(provider, state)
	if len(tools) != 6 {
		t.Errorf("defineK8sTools returned %d tools, want 6", len(tools))
	}
}

func TestDefineToolsCountUnchanged(t *testing.T) {
	provider := createMockProvider(t)
	state := &agentState{mode: ModeReadOnly, outputFormat: OutputJSON}
	tools := defineTools(provider, state)
	if len(tools) != 9 {
		t.Errorf("defineTools returned %d tools, want 9", len(tools))
	}
}

func TestBridgeTool_SuccessResult(t *testing.T) {
	type result struct {
		Value string `json:"value"`
	}
	tool := llm.Tool{
		Name:        "test_tool",
		Description: "a test tool",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(params any, inv llm.ToolInvocation) (any, error) {
			return result{Value: "ok"}, nil
		},
	}

	mcpTool, handler := bridgeTool(tool)

	if mcpTool.Name != "test_tool" {
		t.Errorf("tool name = %q, want test_tool", mcpTool.Name)
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "test_tool"
	req.Params.Arguments = map[string]any{}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success result, got error: %v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	var got result
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if got.Value != "ok" {
		t.Errorf("value = %q, want ok", got.Value)
	}
}

func TestBridgeTool_HandlerError(t *testing.T) {
	tool := llm.Tool{
		Name:        "fail_tool",
		Description: "always fails",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(params any, inv llm.ToolInvocation) (any, error) {
			return nil, fmt.Errorf("something went wrong")
		},
	}

	_, handler := bridgeTool(tool)

	req := mcp.CallToolRequest{}
	req.Params.Name = "fail_tool"
	req.Params.Arguments = map[string]any{}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for handler error")
	}
}

func TestToolResultToMCP_String(t *testing.T) {
	res := toolResultToMCP("hello")
	if res.IsError {
		t.Fatal("unexpected error result")
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	if text.Text != "hello" {
		t.Errorf("text = %q, want hello", text.Text)
	}
}

func TestToolResultToMCP_Nil(t *testing.T) {
	res := toolResultToMCP(nil)
	if res.IsError {
		t.Fatal("unexpected error result")
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	if text.Text != "" {
		t.Errorf("text = %q, want empty string", text.Text)
	}
}

func TestToolResultToMCP_Struct(t *testing.T) {
	res := toolResultToMCP(map[string]any{"key": "val"})
	if res.IsError {
		t.Fatal("unexpected error result")
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if got["key"] != "val" {
		t.Errorf("key = %v, want val", got["key"])
	}
}
