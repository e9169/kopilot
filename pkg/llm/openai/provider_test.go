package openai

import (
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
)

func intPtr(i int) *int { return &i }

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
