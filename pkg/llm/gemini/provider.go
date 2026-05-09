package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/e9169/kopilot/pkg/llm"
	"google.golang.org/genai"
)

// Provider implements llm.Provider using google.golang.org/genai.
type Provider struct {
	client *genai.Client
}

// NewProvider creates a new Gemini provider.
func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "Google Gemini"
}

func (p *Provider) Start(ctx context.Context) error {
	// genai.NewClient automatically uses GEMINI_API_KEY if present, 
	// or falls back to Vertex AI / Application Default Credentials
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize gemini client: %w\nTip: Set GEMINI_API_KEY or use 'gcloud auth application-default login'", err)
	}
	p.client = client
	return nil
}

func (p *Provider) Stop() error {
	// The new genai client doesn't have an explicit close method,
	// it relies on contexts and standard http.Client.
	return nil
}

func convertJSONSchemaToType(schema map[string]any) *genai.Schema {
	if schema == nil {
		return nil
	}
	
	s := &genai.Schema{}
	
	if t, ok := schema["type"].(string); ok {
		switch t {
		case "string":
			s.Type = genai.TypeString
		case "number":
			s.Type = genai.TypeNumber
		case "integer":
			s.Type = genai.TypeInteger
		case "boolean":
			s.Type = genai.TypeBoolean
		case "array":
			s.Type = genai.TypeArray
		case "object":
			s.Type = genai.TypeObject
		}
	} else {
		s.Type = genai.TypeObject // default
	}

	if desc, ok := schema["description"].(string); ok {
		s.Description = desc
	}

	if items, ok := schema["items"].(map[string]any); ok {
		s.Items = convertJSONSchemaToType(items)
	}

	if properties, ok := schema["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*genai.Schema)
		for k, v := range properties {
			if vMap, isMap := v.(map[string]any); isMap {
				s.Properties[k] = convertJSONSchemaToType(vMap)
			}
		}
	}

	if required, ok := schema["required"].([]any); ok {
		for _, req := range required {
			if reqStr, isStr := req.(string); isStr {
				s.Required = append(s.Required, reqStr)
			}
		}
	}

	return s
}

func (p *Provider) CreateSession(ctx context.Context, config *llm.SessionConfig) (llm.Session, error) {
	var toolDeclarations []*genai.Tool
	var funcDecls []*genai.FunctionDeclaration
	toolMap := make(map[string]llm.Tool)

	for _, t := range config.Tools {
		toolMap[t.Name] = t
		
		schema := convertJSONSchemaToType(t.Parameters)
		if schema == nil {
			schema = &genai.Schema{Type: genai.TypeObject}
		}

		funcDecls = append(funcDecls, &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  schema,
		})
	}

	if len(funcDecls) > 0 {
		toolDeclarations = append(toolDeclarations, &genai.Tool{
			FunctionDeclarations: funcDecls,
		})
	}

	sysInstructions := &genai.Content{
		Parts: []*genai.Part{{Text: config.SystemMessage}},
	}
	if config.SystemMessage == "" {
		sysInstructions = nil
	}

	chatConfig := &genai.GenerateContentConfig{
		Tools:              toolDeclarations,
		SystemInstruction:  sysInstructions,
	}

	chat, err := p.client.Chats.Create(ctx, config.Model, chatConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	s := &Session{
		chat:      chat,
		model:     config.Model,
		streaming: config.Streaming,
		toolMap:   toolMap,
		handlers:  []func(llm.Event){},
	}

	return s, nil
}

// Session implements llm.Session for Gemini.
type Session struct {
	chat      *genai.Chat
	model     string
	streaming bool
	toolMap   map[string]llm.Tool
	handlers  []func(llm.Event)
	cancel    context.CancelFunc
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

	go func() {
		defer cancel()
		s.runCompletionLoop(ctx, prompt)
		s.emit(llm.Event{Type: llm.EventIdle})
	}()

	return nil
}

func (s *Session) runCompletionLoop(ctx context.Context, prompt string) {
	// The initial request sends the user prompt. 
	// Subsequent loops handle tool responses.
	var parts []genai.Part
	if prompt != "" {
		parts = append(parts, *genai.NewPartFromText(prompt))
	}

	for {
		if s.streaming {
			stream := s.chat.SendMessageStream(ctx, parts...)

			var fullContent strings.Builder
			var functionCalls []*genai.FunctionCall

			for resp, err := range stream {
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						s.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: err.Error()}})
					}
					return
				}
				if len(resp.Candidates) > 0 {
					candidate := resp.Candidates[0]
					if candidate.Content != nil {
						for _, part := range candidate.Content.Parts {
							if part.Text != "" {
								fullContent.WriteString(part.Text)
								s.emit(llm.Event{Type: llm.EventDelta, Data: &llm.DeltaData{Content: part.Text}})
							}
							if part.FunctionCall != nil {
								functionCalls = append(functionCalls, part.FunctionCall)
							}
						}
					}
				}
			}

			if fullContent.Len() > 0 {
				s.emit(llm.Event{Type: llm.EventMessage, Data: &llm.MessageData{Content: fullContent.String()}})
			}

			if len(functionCalls) > 0 {
				parts = nil // clear for tool response
				for _, fc := range functionCalls {
					res := s.handleToolCall(fc)
					parts = append(parts, res)
				}
				continue
			}
			return

		} else {
			// Non-streaming
			resp, err := s.chat.SendMessage(ctx, parts...)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					s.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: err.Error()}})
				}
				return
			}
			
			if len(resp.Candidates) == 0 {
				return
			}
			
			candidate := resp.Candidates[0]
			if candidate.Content == nil {
				return
			}

			var functionCalls []*genai.FunctionCall
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					s.emit(llm.Event{Type: llm.EventMessage, Data: &llm.MessageData{Content: part.Text}})
				}
				if part.FunctionCall != nil {
					functionCalls = append(functionCalls, part.FunctionCall)
				}
			}

			if len(functionCalls) > 0 {
				parts = nil // clear for tool response
				for _, fc := range functionCalls {
					res := s.handleToolCall(fc)
					parts = append(parts, res)
				}
				continue
			}
			return
		}
	}
}

func (s *Session) handleToolCall(tc *genai.FunctionCall) genai.Part {
	toolDef, ok := s.toolMap[tc.Name]
	var result map[string]any

	if !ok {
		result = map[string]any{"error": fmt.Sprintf("Unknown tool %s", tc.Name)}
	} else {
		// Convert tc.Args (map[string]any) to JSON string for Arguments
		argsBytes, _ := json.Marshal(tc.Args)

		resAny, err := toolDef.Handler(tc.Args, llm.ToolInvocation{
			ID:        tc.Name, // Gemini doesn't use unique call IDs natively in the same way
			Name:      tc.Name,
			Arguments: string(argsBytes),
		})

		if err != nil {
			result = map[string]any{"error": fmt.Sprintf("Error executing tool: %v", err)}
		} else {
			// Convert to map[string]any for Gemini FunctionResponse
			resultBytes, _ := json.Marshal(resAny)
			var resMap map[string]any
			if err := json.Unmarshal(resultBytes, &resMap); err == nil {
				result = resMap
			} else {
				result = map[string]any{"result": string(resultBytes)}
			}
		}
	}

	return *genai.NewPartFromFunctionResponse(tc.Name, result)
}
