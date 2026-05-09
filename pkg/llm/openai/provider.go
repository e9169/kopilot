package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/e9169/kopilot/pkg/llm"
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// Provider implements llm.Provider using sashabaranov/go-openai.
type Provider struct {
	client *goopenai.Client
}

// NewProvider creates a new OpenAI provider.
func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		return "OpenAI-compatible (" + baseURL + ")"
	}
	return "OpenAI"
}

func (p *Provider) Start(ctx context.Context) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "dummy-key-for-local-models"
	}
	config := goopenai.DefaultConfig(apiKey)
	
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	p.client = goopenai.NewClientWithConfig(config)
	return nil
}

func (p *Provider) Stop() error {
	return nil
}

func (p *Provider) CreateSession(ctx context.Context, config *llm.SessionConfig) (llm.Session, error) {
	var tools []goopenai.Tool
	toolMap := make(map[string]llm.Tool)

	for _, t := range config.Tools {
		toolMap[t.Name] = t
		
		// Convert standard JSON schema properties to jsonschema.Definition
		paramsBytes, _ := json.Marshal(t.Parameters)
		var definition jsonschema.Definition
		if err := json.Unmarshal(paramsBytes, &definition); err != nil {
			// Fallback empty schema if parsing fails
			definition = jsonschema.Definition{
				Type: jsonschema.Object,
				Properties: map[string]jsonschema.Definition{},
			}
		}

		tools = append(tools, goopenai.Tool{
			Type: goopenai.ToolTypeFunction,
			Function: &goopenai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  definition,
			},
		})
	}

	s := &Session{
		client:        p.client,
		model:         config.Model,
		streaming:     config.Streaming,
		systemMessage: config.SystemMessage,
		tools:         tools,
		toolMap:       toolMap,
		messages:      []goopenai.ChatCompletionMessage{},
		handlers:      []func(llm.Event){},
	}

	if s.systemMessage != "" {
		s.messages = append(s.messages, goopenai.ChatCompletionMessage{
			Role:    goopenai.ChatMessageRoleSystem,
			Content: s.systemMessage,
		})
	}

	return s, nil
}

// Session implements llm.Session for OpenAI.
type Session struct {
	client        *goopenai.Client
	model         string
	streaming     bool
	systemMessage string
	tools         []goopenai.Tool
	toolMap       map[string]llm.Tool
	messages      []goopenai.ChatCompletionMessage
	handlers      []func(llm.Event)
	cancel        context.CancelFunc
}

func (s *Session) Disconnect() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *Session) emit(event llm.Event) {
	for _, h := range s.handlers {
		h(event)
	}
}

func (s *Session) On(handler func(llm.Event)) {
	s.handlers = append(s.handlers, handler)
}

func (s *Session) SendPrompt(ctx context.Context, prompt string) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.messages = append(s.messages, goopenai.ChatCompletionMessage{
		Role:    goopenai.ChatMessageRoleUser,
		Content: prompt,
	})

	go func() {
		defer cancel()
		s.runCompletionLoop(ctx)
		s.emit(llm.Event{Type: llm.EventIdle})
	}()

	return nil
}

func (s *Session) runCompletionLoop(ctx context.Context) {
	for {
		req := goopenai.ChatCompletionRequest{
			Model:    s.model,
			Messages: s.messages,
			Stream:   s.streaming,
			Tools:    s.tools,
		}

		if s.streaming {
			stream, err := s.client.CreateChatCompletionStream(ctx, req)
			if err != nil {
				s.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: err.Error()}})
				return
			}

			var fullContent strings.Builder
			var toolCalls []goopenai.ToolCall
			
			for {
				resp, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					stream.Close()
					break
				}
				if err != nil {
					stream.Close()
					if !errors.Is(err, context.Canceled) {
						s.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: err.Error()}})
					}
					return
				}

				if len(resp.Choices) > 0 {
					delta := resp.Choices[0].Delta
					if delta.Content != "" {
						fullContent.WriteString(delta.Content)
						s.emit(llm.Event{Type: llm.EventDelta, Data: &llm.DeltaData{Content: delta.Content}})
					}
					
					for _, tc := range delta.ToolCalls {
						// Expand toolCalls slice if needed
						for len(toolCalls) <= *tc.Index {
							toolCalls = append(toolCalls, goopenai.ToolCall{})
						}
						// Merge tool call chunks
						if tc.ID != "" {
							toolCalls[*tc.Index].ID = tc.ID
						}
						if tc.Type != "" {
							toolCalls[*tc.Index].Type = tc.Type
						}
						if tc.Function.Name != "" {
							if toolCalls[*tc.Index].Function.Name == "" {
								toolCalls[*tc.Index].Function.Name = tc.Function.Name
							} else {
								toolCalls[*tc.Index].Function.Name += tc.Function.Name
							}
						}
						if tc.Function.Arguments != "" {
							toolCalls[*tc.Index].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}

			s.messages = append(s.messages, goopenai.ChatCompletionMessage{
				Role:      goopenai.ChatMessageRoleAssistant,
				Content:   fullContent.String(),
				ToolCalls: toolCalls,
			})
			
			if fullContent.Len() > 0 {
				s.emit(llm.Event{Type: llm.EventMessage, Data: &llm.MessageData{Content: fullContent.String()}})
			}

			if len(toolCalls) > 0 {
				for _, tc := range toolCalls {
					s.handleToolCall(tc)
				}
				// After tool calls, continue the loop to send results back
				continue
			}
			return

		} else {
			resp, err := s.client.CreateChatCompletion(ctx, req)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					s.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: err.Error()}})
				}
				return
			}
			
			if len(resp.Choices) == 0 {
				return
			}
			
			msg := resp.Choices[0].Message
			s.messages = append(s.messages, msg)
			
			if msg.Content != "" {
				s.emit(llm.Event{Type: llm.EventMessage, Data: &llm.MessageData{Content: msg.Content}})
			}
			
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					s.handleToolCall(tc)
				}
				continue
			}
			return
		}
	}
}

func (s *Session) handleToolCall(tc goopenai.ToolCall) {
	toolDef, ok := s.toolMap[tc.Function.Name]
	var result string

	if !ok {
		result = fmt.Sprintf("Error: Unknown tool %s", tc.Function.Name)
	} else {
		// Parse arguments into a map
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			// Some LLMs might pass broken JSON, try to handle as string
			args = map[string]any{"raw": tc.Function.Arguments}
		}

		resAny, err := toolDef.Handler(args, llm.ToolInvocation{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})

		if err != nil {
			result = fmt.Sprintf("Error executing tool: %v", err)
		} else {
			if resBytes, err := json.Marshal(resAny); err == nil {
				result = string(resBytes)
			} else {
				result = fmt.Sprintf("%v", resAny)
			}
		}
	}

	s.messages = append(s.messages, goopenai.ChatCompletionMessage{
		Role:       goopenai.ChatMessageRoleTool,
		Content:    result,
		Name:       tc.Function.Name,
		ToolCallID: tc.ID,
	})
}
