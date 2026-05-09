package llm_test

import (
	"context"
	"os"
	"testing"

	copilotprovider "github.com/e9169/kopilot/pkg/llm/copilot"
	geminiprovider "github.com/e9169/kopilot/pkg/llm/gemini"
	openaiprovider "github.com/e9169/kopilot/pkg/llm/openai"
)

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
