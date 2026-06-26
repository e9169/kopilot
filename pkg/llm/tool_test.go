package llm

import (
	"testing"
)

func TestDefineTool(t *testing.T) {
	type Params struct {
		Name string `json:"name"`
	}

	tool := DefineTool("greet", "says hello", func(p Params, inv ToolInvocation) (any, error) {
		return map[string]any{"greeting": "hello " + p.Name}, nil
	})

	if tool.Name != "greet" {
		t.Errorf("Name = %q, want greet", tool.Name)
	}
	if tool.Description != "says hello" {
		t.Errorf("Description = %q, want 'says hello'", tool.Description)
	}
	if tool.Parameters == nil {
		t.Fatal("Parameters should not be nil")
	}
	if tool.Handler == nil {
		t.Fatal("Handler should not be nil")
	}

	result, err := tool.Handler(map[string]any{"name": "world"}, ToolInvocation{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if m["greeting"] != "hello world" {
		t.Errorf("greeting = %q, want 'hello world'", m["greeting"])
	}
}

func TestDefineTool_NilParams(t *testing.T) {
	type Params struct{ X int }
	tool := DefineTool("noop", "does nothing", func(p Params, inv ToolInvocation) (any, error) {
		return p.X, nil
	})

	result, err := tool.Handler(nil, ToolInvocation{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Errorf("result = %v, want 0 (zero value)", result)
	}
}

func TestParseToolArgumentsString(t *testing.T) {
	got := ParseToolArgumentsString(`{"name":"world"}`)
	if got["name"] != "world" {
		t.Fatalf("parsed name = %v, want world", got["name"])
	}

	raw := ParseToolArgumentsString(`{invalid`)
	if raw["raw"] != "{invalid" {
		t.Fatalf("raw fallback = %v, want original string", raw["raw"])
	}
}

func TestNormalizeToolArguments(t *testing.T) {
	params, raw := NormalizeToolArguments(map[string]any{"x": 1})
	if raw == "" {
		t.Fatal("raw should not be empty")
	}
	if params["x"] != float64(1) {
		t.Fatalf("params[x] = %v, want 1", params["x"])
	}

	params, raw = NormalizeToolArguments(nil)
	if raw != "{}" {
		t.Fatalf("raw for nil = %q, want {}", raw)
	}
	if len(params) != 0 {
		t.Fatalf("params for nil should be empty, got %#v", params)
	}
}

func TestResultHelpers(t *testing.T) {
	if got := ResultString(map[string]any{"ok": true}); got == "" {
		t.Fatal("ResultString should return a non-empty JSON string")
	}
	m := ResultMap(map[string]any{"ok": true})
	if m["ok"] != true {
		t.Fatalf("ResultMap[ok] = %v, want true", m["ok"])
	}
}

func TestInvokeTool(t *testing.T) {
	tools := map[string]Tool{
		"echo": {
			Name: "echo",
			Handler: func(params any, inv ToolInvocation) (any, error) {
				return params, nil
			},
		},
	}
	got, err := InvokeTool(tools, map[string]any{"k": "v"}, ToolInvocation{Name: "echo"})
	if err != nil {
		t.Fatalf("InvokeTool() error = %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["k"] != "v" {
		t.Fatalf("InvokeTool() result = %#v, want map with k=v", got)
	}

	if _, err := InvokeTool(tools, map[string]any{}, ToolInvocation{Name: "missing"}); err == nil {
		t.Fatal("InvokeTool() should fail for unknown tools")
	}
}
