package copilot

import (
	"testing"

	"github.com/e9169/kopilot/pkg/llm"
	sdk "github.com/github/copilot-sdk/go"
	"github.com/github/copilot-sdk/go/rpc"
)

func httpURL(t *testing.T, cfg sdk.MCPServerConfig) string {
	t.Helper()
	http, ok := cfg.(sdk.MCPHTTPServerConfig)
	if !ok {
		t.Fatalf("expected MCPHTTPServerConfig, got %T", cfg)
	}
	return http.URL
}

func TestParseMCPServersWithTypedMap(t *testing.T) {
	extra := map[string]any{
		"MCPServers": map[string]sdk.MCPServerConfig{
			"a": sdk.MCPHTTPServerConfig{URL: "http://localhost:3030/mcp"},
		},
	}

	got := parseMCPServers(extra)
	if got == nil {
		t.Fatal("parseMCPServers() returned nil")
	}
	if url := httpURL(t, got["a"]); url != "http://localhost:3030/mcp" {
		t.Fatalf("url mismatch: got %q", url)
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
	if url := httpURL(t, got["b"]); url != "https://example.com/mcp" {
		t.Fatalf("url mismatch: got %q", url)
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

func TestParseMCPServerEntryWithStringMap(t *testing.T) {
	got, ok := parseMCPServerEntry(map[string]string{"type": "http", "url": "https://example.com/mcp"})
	if !ok {
		t.Fatal("parseMCPServerEntry should accept map[string]string")
	}
	if url := httpURL(t, got); url != "https://example.com/mcp" {
		t.Fatalf("url mismatch: got %q", url)
	}
}

func TestProviderNameAndStop(t *testing.T) {
	p := NewProvider()
	if got := p.Name(); got != "GitHub Copilot" {
		t.Fatalf("Name() = %q, want %q", got, "GitHub Copilot")
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestConvertSDKEventKnownTypes(t *testing.T) {
	cases := []struct {
		name  string
		event sdk.SessionEvent
		want  llm.EventType
	}{
		{
			name:  "assistant message",
			event: sdk.SessionEvent{Data: &rpc.AssistantMessageData{Content: "hello"}},
			want:  llm.EventMessage,
		},
		{
			name:  "assistant delta",
			event: sdk.SessionEvent{Data: &rpc.AssistantMessageDeltaData{DeltaContent: "h"}},
			want:  llm.EventDelta,
		},
		{
			name:  "session error",
			event: sdk.SessionEvent{Data: &rpc.SessionErrorData{Message: "boom"}},
			want:  llm.EventError,
		},
		{
			name:  "session idle",
			event: sdk.SessionEvent{Data: &rpc.SessionIdleData{}},
			want:  llm.EventIdle,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := convertSDKEvent(tc.event)
			if !ok {
				t.Fatal("convertSDKEvent returned ok=false for known type")
			}
			if got.Type != tc.want {
				t.Fatalf("event type = %q, want %q", got.Type, tc.want)
			}
		})
	}
}

func TestConvertSDKEventUnknownType(t *testing.T) {
	got, ok := convertSDKEvent(sdk.SessionEvent{Data: rpc.RawSessionEventData{EventType: "unknown.type"}})
	if ok {
		t.Fatalf("unknown event should return ok=false, got %#v", got)
	}
}
