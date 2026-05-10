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
