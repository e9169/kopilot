package copilot

import (
	"testing"

	sdk "github.com/github/copilot-sdk/go"
)

func TestParseMCPServersWithTypedMap(t *testing.T) {
	extra := map[string]any{
		"MCPServers": map[string]sdk.MCPServerConfig{
			"a": {"type": "http", "url": "http://localhost:3030/mcp"},
		},
	}

	got := parseMCPServers(extra)
	if got == nil {
		t.Fatal("parseMCPServers() returned nil")
	}
	if got["a"]["type"] != "http" {
		t.Fatalf("type mismatch: got %q", got["a"]["type"])
	}
	if got["a"]["url"] != "http://localhost:3030/mcp" {
		t.Fatalf("url mismatch: got %q", got["a"]["url"])
	}
}

func TestParseMCPServersWithGenericMap(t *testing.T) {
	extra := map[string]any{
		"MCPServers": map[string]any{
			"b": map[string]any{"type": "http", "url": "https://example.com/mcp"},
		},
	}

	got := parseMCPServers(extra)
	if got == nil {
		t.Fatal("parseMCPServers() returned nil")
	}
	if got["b"]["type"] != "http" {
		t.Fatalf("type mismatch: got %q", got["b"]["type"])
	}
	if got["b"]["url"] != "https://example.com/mcp" {
		t.Fatalf("url mismatch: got %q", got["b"]["url"])
	}
}

func TestParseMCPServersWithInvalidShape(t *testing.T) {
	extra := map[string]any{
		"MCPServers": "not-a-map",
	}

	got := parseMCPServers(extra)
	if got != nil {
		t.Fatalf("expected nil for invalid shape, got %#v", got)
	}
}
