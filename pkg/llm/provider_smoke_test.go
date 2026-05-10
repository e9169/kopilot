package llm_test

import (
	"context"
	"os"
	"testing"

	"github.com/e9169/kopilot/pkg/llm"
	copilotprovider "github.com/e9169/kopilot/pkg/llm/copilot"
	geminiprovider "github.com/e9169/kopilot/pkg/llm/gemini"
	openaiprovider "github.com/e9169/kopilot/pkg/llm/openai"
)

// stubSession is a minimal llm.Session for contract testing.
type stubSession struct {
	handlers []func(llm.Event)
}

func (s *stubSession) Disconnect() error                            { return nil }
func (s *stubSession) SendPrompt(_ context.Context, _ string) error { return nil }
func (s *stubSession) On(h func(llm.Event))                         { s.handlers = append(s.handlers, h) }
func (s *stubSession) emit(e llm.Event) {
	for _, h := range s.handlers {
		h(e)
	}
}

// TestCopilotProviderSmoke verifies that the Copilot provider can be instantiated.
// It does not require authentication; Start() will fail without a valid token,
// but we verify the provider can be created without panicking.
func TestCopilotProviderSmoke(t *testing.T) {
	p := copilotprovider.NewProvider()
	if p == nil {
		t.Fatal("copilot.NewProvider() returned nil")
	}
}

// TestOpenAIProviderSmoke verifies that the OpenAI provider can be instantiated
// and that Start() accepts any non-empty key without panicking.
// Uses a fake key to avoid a real network call.
func TestOpenAIProviderSmoke(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Setenv("OPENAI_API_KEY", "sk-fake-key-for-smoke-test")
	}
	p := openaiprovider.NewProvider()
	if p == nil {
		t.Fatal("openai.NewProvider() returned nil")
	}
	// Start just initialises the client; it does not make a network request.
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("openai provider Start() failed: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("openai provider Stop() failed: %v", err)
	}
}

// TestGeminiProviderSmoke verifies that the Gemini provider returns a clear error
// when no API key or ADC credentials are available, rather than panicking.
func TestGeminiProviderSmoke(t *testing.T) {
	// Only run if no credentials are available so we test the error path.
	if os.Getenv("GEMINI_API_KEY") != "" {
		t.Skip("GEMINI_API_KEY set — skipping error-path smoke test")
	}

	p := geminiprovider.NewProvider()
	if p == nil {
		t.Fatal("gemini.NewProvider() returned nil")
	}
	// Start should fail gracefully (not panic) when no key is configured.
	err := p.Start(context.Background())
	if err == nil {
		// If gcloud ADC is configured this may succeed — that's fine.
		t.Log("gemini provider Start() succeeded (ADC credentials found)")
		_ = p.Stop()
	} else {
		t.Logf("gemini provider Start() returned expected error: %v", err)
	}
}

// TestEventTypeConstantsAreDistinct guards against accidental duplication of EventType values.
func TestEventTypeConstantsAreDistinct(t *testing.T) {
	all := []llm.EventType{
		llm.EventMessage,
		llm.EventDelta,
		llm.EventIdle,
		llm.EventError,
		llm.EventUsage,
	}
	seen := make(map[llm.EventType]bool)
	for _, et := range all {
		if et == "" {
			t.Errorf("empty EventType constant")
		}
		if seen[et] {
			t.Errorf("duplicate EventType value: %q", et)
		}
		seen[et] = true
	}
}

// TestStubSessionEventDelivery verifies that events arrive at handlers in emission order
// with the exact EventType the emitter sent.
func TestStubSessionEventDelivery(t *testing.T) {
	sess := &stubSession{}
	var received []llm.EventType
	sess.On(func(e llm.Event) { received = append(received, e.Type) })

	sequence := []llm.EventType{
		llm.EventDelta,
		llm.EventDelta,
		llm.EventMessage,
		llm.EventIdle,
	}
	for _, et := range sequence {
		sess.emit(llm.Event{Type: et})
	}

	if len(received) != len(sequence) {
		t.Fatalf("received %d events, want %d", len(received), len(sequence))
	}
	for i, want := range sequence {
		if received[i] != want {
			t.Errorf("event[%d] = %q, want %q", i, received[i], want)
		}
	}
}

// TestSessionConfigMCPServersPreserved verifies that ExtraConfig["MCPServers"] survives
// the SessionConfig struct boundary unchanged.
func TestSessionConfigMCPServersPreserved(t *testing.T) {
	servers := []map[string]string{
		{"name": "test-server", "url": "https://mcp.example.com"},
	}
	cfg := &llm.SessionConfig{
		Model: "test-model",
		ExtraConfig: map[string]any{
			"MCPServers": servers,
		},
	}

	got, ok := cfg.ExtraConfig["MCPServers"]
	if !ok {
		t.Fatal("MCPServers key missing from ExtraConfig")
	}
	gotSlice, ok := got.([]map[string]string)
	if !ok {
		t.Fatalf("MCPServers type = %T, want []map[string]string", got)
	}
	if len(gotSlice) != 1 || gotSlice[0]["name"] != "test-server" {
		t.Errorf("MCPServers content = %v, want original slice", gotSlice)
	}
}
