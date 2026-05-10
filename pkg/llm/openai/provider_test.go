package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/e9169/kopilot/pkg/llm"
	goopenai "github.com/sashabaranov/go-openai"
)

func newTestSession(t *testing.T, handler http.HandlerFunc, streaming bool) (*Session, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	config := goopenai.DefaultConfig("test-key")
	config.BaseURL = ts.URL + "/v1"
	client := goopenai.NewClientWithConfig(config)
	s := &Session{
		client:    client,
		model:     "test-model",
		streaming: streaming,
		toolMap:   map[string]llm.Tool{},
		messages:  []goopenai.ChatCompletionMessage{},
		handlers:  []func(llm.Event){},
	}
	return s, ts
}

func collectEvents(s *Session) *[]llm.Event {
	events := &[]llm.Event{}
	s.On(func(e llm.Event) { *events = append(*events, e) })
	return events
}

func TestRunStreamingStep_ServerError(t *testing.T) {
	s, ts := newTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}, true)
	defer ts.Close()

	events := collectEvents(s)
	req := goopenai.ChatCompletionRequest{Model: s.model, Messages: s.messages, Stream: true}
	cont := s.runStreamingStep(context.Background(), req)

	if cont {
		t.Fatal("expected runStreamingStep to return false on server error")
	}
	if len(*events) == 0 {
		t.Fatal("expected an error event to be emitted")
	}
	if (*events)[0].Type != llm.EventError {
		t.Fatalf("expected EventError, got %v", (*events)[0].Type)
	}
}

func TestRunStreamingStep_TextContent(t *testing.T) {
	chunk := map[string]any{
		"id":     "chatcmpl-test",
		"object": "chat.completion.chunk",
		"choices": []map[string]any{
			{"index": 0, "delta": map[string]any{"content": "hello"}, "finish_reason": nil},
		},
	}
	chunkJSON, _ := json.Marshal(chunk)

	s, ts := newTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", chunkJSON)
	}, true)
	defer ts.Close()

	events := collectEvents(s)
	req := goopenai.ChatCompletionRequest{Model: s.model, Messages: s.messages, Stream: true}
	cont := s.runStreamingStep(context.Background(), req)

	if cont {
		t.Fatal("expected runStreamingStep to return false (no tool calls)")
	}
	var hasMessage bool
	for _, e := range *events {
		if e.Type == llm.EventMessage {
			hasMessage = true
		}
	}
	if !hasMessage {
		t.Fatal("expected an EventMessage to be emitted")
	}
}

func TestRunNonStreamingStep_ServerError(t *testing.T) {
	s, ts := newTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}, false)
	defer ts.Close()

	events := collectEvents(s)
	req := goopenai.ChatCompletionRequest{Model: s.model, Messages: s.messages}
	cont := s.runNonStreamingStep(context.Background(), req)

	if cont {
		t.Fatal("expected runNonStreamingStep to return false on server error")
	}
	if len(*events) == 0 {
		t.Fatal("expected an error event to be emitted")
	}
	if (*events)[0].Type != llm.EventError {
		t.Fatalf("expected EventError, got %v", (*events)[0].Type)
	}
}

func TestRunNonStreamingStep_EmptyChoices(t *testing.T) {
	resp := map[string]any{
		"id": "chatcmpl-test", "object": "chat.completion",
		"choices": []any{},
		"usage":   map[string]any{"prompt_tokens": 5, "completion_tokens": 0, "total_tokens": 5},
	}
	body, _ := json.Marshal(resp)

	s, ts := newTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body) //nolint:errcheck
	}, false)
	defer ts.Close()

	req := goopenai.ChatCompletionRequest{Model: s.model, Messages: s.messages}
	cont := s.runNonStreamingStep(context.Background(), req)

	if cont {
		t.Fatal("expected runNonStreamingStep to return false on empty choices")
	}
}

func TestRunNonStreamingStep_TextContent(t *testing.T) {
	resp := map[string]any{
		"id": "chatcmpl-test", "object": "chat.completion",
		"choices": []map[string]any{
			{"index": 0, "message": map[string]any{"role": "assistant", "content": "hi"}, "finish_reason": "stop"},
		},
		"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 2, "total_tokens": 7},
	}
	body, _ := json.Marshal(resp)

	s, ts := newTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body) //nolint:errcheck
	}, false)
	defer ts.Close()

	events := collectEvents(s)
	req := goopenai.ChatCompletionRequest{Model: s.model, Messages: s.messages}
	cont := s.runNonStreamingStep(context.Background(), req)

	if cont {
		t.Fatal("expected runNonStreamingStep to return false (no tool calls)")
	}
	var hasMessage bool
	for _, e := range *events {
		if e.Type == llm.EventMessage {
			hasMessage = true
		}
	}
	if !hasMessage {
		t.Fatal("expected an EventMessage to be emitted")
	}
}

func TestMergeToolCallChunk_NilIndexDropped(t *testing.T) {
	chunk := goopenai.ToolCall{Index: nil, ID: "id1"}
	got := mergeToolCallChunk(nil, chunk)
	if len(got) != 0 {
		t.Fatalf("nil-index chunk should be dropped, got %d entries", len(got))
	}
}

func TestMergeToolCallChunk_ExpandsSlice(t *testing.T) {
	idx := 2
	chunk := goopenai.ToolCall{Index: &idx, ID: "id3"}
	got := mergeToolCallChunk(nil, chunk)
	if len(got) != 3 {
		t.Fatalf("slice should expand to len 3, got %d", len(got))
	}
	if got[2].ID != "id3" {
		t.Fatalf("ID at index 2 = %q, want id3", got[2].ID)
	}
}

func TestMergeToolCallChunk_AccumulatesArguments(t *testing.T) {
	idx := 0
	var toolCalls []goopenai.ToolCall
	for _, frag := range []string{`{"k"`, `:"v"}`} {
		toolCalls = mergeToolCallChunk(toolCalls, goopenai.ToolCall{
			Index:    &idx,
			Function: goopenai.FunctionCall{Arguments: frag},
		})
	}
	want := `{"k":"v"}`
	if toolCalls[0].Function.Arguments != want {
		t.Fatalf("Arguments = %q, want %q", toolCalls[0].Function.Arguments, want)
	}
}
