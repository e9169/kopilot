package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHandleToolCall_KnownTool(t *testing.T) {
	s := &Session{
		toolMap: map[string]llm.Tool{
			"echo": {
				Name: "echo",
				Handler: func(params any, inv llm.ToolInvocation) (any, error) {
					return map[string]any{"echoed": true}, nil
				},
			},
		},
		messages: []goopenai.ChatCompletionMessage{},
	}
	tc := goopenai.ToolCall{
		ID:       "call-1",
		Type:     goopenai.ToolTypeFunction,
		Function: goopenai.FunctionCall{Name: "echo", Arguments: `{"x":1}`},
	}
	s.handleToolCall(tc)
	if len(s.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s.messages))
	}
	if s.messages[0].Role != goopenai.ChatMessageRoleTool {
		t.Errorf("role = %q, want tool", s.messages[0].Role)
	}
	if s.messages[0].ToolCallID != "call-1" {
		t.Errorf("ToolCallID = %q, want call-1", s.messages[0].ToolCallID)
	}
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	s := &Session{
		toolMap:  map[string]llm.Tool{},
		messages: []goopenai.ChatCompletionMessage{},
	}
	tc := goopenai.ToolCall{
		ID:       "call-2",
		Type:     goopenai.ToolTypeFunction,
		Function: goopenai.FunctionCall{Name: "missing", Arguments: `{}`},
	}
	s.handleToolCall(tc)
	if len(s.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s.messages))
	}
	if !strings.Contains(s.messages[0].Content, "Unknown tool") {
		t.Errorf("expected 'Unknown tool' in content, got %q", s.messages[0].Content)
	}
}

func TestHandleToolCall_BrokenJSON(t *testing.T) {
	called := false
	s := &Session{
		toolMap: map[string]llm.Tool{
			"t": {
				Name: "t",
				Handler: func(params any, inv llm.ToolInvocation) (any, error) {
					called = true
					return "ok", nil
				},
			},
		},
		messages: []goopenai.ChatCompletionMessage{},
	}
	tc := goopenai.ToolCall{
		ID:       "call-3",
		Function: goopenai.FunctionCall{Name: "t", Arguments: `not-json`},
	}
	s.handleToolCall(tc)
	if !called {
		t.Error("handler should be called even when arguments JSON is malformed")
	}
	if len(s.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s.messages))
	}
}

func TestProviderNameAndStart(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "http://localhost:11434/v1")
	p := NewProvider()
	if got := p.Name(); !strings.Contains(got, "OpenAI-compatible") {
		t.Fatalf("Name() = %q, want OpenAI-compatible", got)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if p.client == nil {
		t.Fatal("client should be initialized after Start")
	}
}

func TestProviderCreateSessionWithSystemMessage(t *testing.T) {
	p := NewProvider()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	session, err := p.CreateSession(context.Background(), &llm.SessionConfig{
		Model:         "gpt-4o-mini",
		SystemMessage: "sys",
		Tools: []llm.Tool{
			{Name: "echo", Description: "echo", Parameters: map[string]any{"type": "object"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	s, ok := session.(*Session)
	if !ok {
		t.Fatalf("session type = %T, want *Session", session)
	}
	if len(s.messages) == 0 || s.messages[0].Role != goopenai.ChatMessageRoleSystem {
		t.Fatalf("system message not initialized: %+v", s.messages)
	}
}
