package gemini

import (
	"testing"

	"google.golang.org/genai"
)

func TestJsonTypeToGenAI(t *testing.T) {
	cases := []struct {
		in   string
		want genai.Type
	}{
		{"string", genai.TypeString},
		{"number", genai.TypeNumber},
		{"integer", genai.TypeInteger},
		{"boolean", genai.TypeBoolean},
		{"array", genai.TypeArray},
		{"object", genai.TypeObject},
		{"unknown", genai.TypeObject},
		{"", genai.TypeObject},
	}
	for _, c := range cases {
		got := jsonTypeToGenAI(c.in)
		if got != c.want {
			t.Errorf("jsonTypeToGenAI(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestConvertJSONSchemaToType_Nil(t *testing.T) {
	if got := convertJSONSchemaToType(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %+v", got)
	}
}

func TestConvertJSONSchemaToType_BasicFields(t *testing.T) {
	schema := map[string]any{
		"type":        "string",
		"description": "a string field",
	}
	got := convertJSONSchemaToType(schema)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Type != genai.TypeString {
		t.Errorf("Type = %v, want TypeString", got.Type)
	}
	if got.Description != "a string field" {
		t.Errorf("Description = %q, want %q", got.Description, "a string field")
	}
}

func TestConvertJSONSchemaToType_ObjectWithProperties(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "integer"},
		},
		"required": []any{"name"},
	}
	got := convertJSONSchemaToType(schema)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Type != genai.TypeObject {
		t.Errorf("Type = %v, want TypeObject", got.Type)
	}
	if len(got.Properties) != 2 {
		t.Errorf("Properties count = %d, want 2", len(got.Properties))
	}
	if got.Properties["name"].Type != genai.TypeString {
		t.Errorf("name property type = %v, want TypeString", got.Properties["name"].Type)
	}
	if got.Properties["age"].Type != genai.TypeInteger {
		t.Errorf("age property type = %v, want TypeInteger", got.Properties["age"].Type)
	}
	if len(got.Required) != 1 || got.Required[0] != "name" {
		t.Errorf("Required = %v, want [name]", got.Required)
	}
}

func TestConvertJSONSchemaToType_ArrayWithItems(t *testing.T) {
	schema := map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "number"},
	}
	got := convertJSONSchemaToType(schema)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Type != genai.TypeArray {
		t.Errorf("Type = %v, want TypeArray", got.Type)
	}
	if got.Items == nil {
		t.Fatal("expected Items to be non-nil")
	}
	if got.Items.Type != genai.TypeNumber {
		t.Errorf("Items.Type = %v, want TypeNumber", got.Items.Type)
	}
}

func TestConvertJSONSchemaToType_DefaultsToObject(t *testing.T) {
	schema := map[string]any{}
	got := convertJSONSchemaToType(schema)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Type != genai.TypeObject {
		t.Errorf("Type = %v, want TypeObject (default)", got.Type)
	}
}
