package gemini

import (
	"context"
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

func jsonTypeToGenAI(t string) genai.Type {
	switch t {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	default:
		return genai.TypeObject
	}
}

func parseProperties(raw map[string]any) map[string]*genai.Schema {
	result := make(map[string]*genai.Schema)
	for k, v := range raw {
		if vMap, ok := v.(map[string]any); ok {
			result[k] = convertJSONSchemaToType(vMap)
		}
	}
	return result
}

func parseRequired(raw []any) []string {
	var result []string
	for _, req := range raw {
		if s, ok := req.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func convertJSONSchemaToType(schema map[string]any) *genai.Schema {
	if schema == nil {
		return nil
	}
	s := &genai.Schema{}
	if t, ok := schema["type"].(string); ok {
		s.Type = jsonTypeToGenAI(t)
	} else {
		s.Type = genai.TypeObject
	}
	if desc, ok := schema["description"].(string); ok {
		s.Description = desc
	}
	if items, ok := schema["items"].(map[string]any); ok {
		s.Items = convertJSONSchemaToType(items)
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		s.Properties = parseProperties(properties)
	}
	if required, ok := schema["required"].([]any); ok {
		s.Required = parseRequired(required)
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
		Tools:             toolDeclarations,
		SystemInstruction: sysInstructions,
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
	}

	return s, nil
}

// Session implements llm.Session for Gemini.
type Session struct {
	chat      *genai.Chat
	model     string
	streaming bool
	toolMap   map[string]llm.Tool
	emitter   llm.EventEmitter
	cancel    context.CancelFunc
}

func (s *Session) Disconnect() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *Session) emit(event llm.Event) {
	s.emitter.Emit(event)
}

func (s *Session) On(handler func(llm.Event)) {
	s.emitter.On(handler)
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
	var parts []genai.Part
	if prompt != "" {
		parts = append(parts, *genai.NewPartFromText(prompt))
	}
	for {
		var (
			next []genai.Part
			cont bool
		)
		if s.streaming {
			next, cont = s.runStreamingStep(ctx, parts)
		} else {
			next, cont = s.runNonStreamingStep(ctx, parts)
		}
		if !cont {
			return
		}
		parts = next
	}
}

func processCandidateParts(parts []*genai.Part, fullContent *strings.Builder, functionCalls *[]*genai.FunctionCall, emit func(llm.Event)) {
	for _, part := range parts {
		if part.Text != "" {
			fullContent.WriteString(part.Text)
			emit(llm.Event{Type: llm.EventDelta, Data: &llm.DeltaData{Content: part.Text}})
		}
		if part.FunctionCall != nil {
			*functionCalls = append(*functionCalls, part.FunctionCall)
		}
	}
}

func (s *Session) runStreamingStep(ctx context.Context, parts []genai.Part) ([]genai.Part, bool) {
	stream := s.chat.SendMessageStream(ctx, parts...)
	var fullContent strings.Builder
	var functionCalls []*genai.FunctionCall
	for resp, err := range stream {
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				s.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: err.Error()}})
			}
			return nil, false
		}
		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			processCandidateParts(resp.Candidates[0].Content.Parts, &fullContent, &functionCalls, s.emit)
		}
	}
	if fullContent.Len() > 0 {
		s.emit(llm.Event{Type: llm.EventMessage, Data: &llm.MessageData{Content: fullContent.String()}})
	}
	if len(functionCalls) > 0 {
		return s.dispatchToolCalls(functionCalls), true
	}
	return nil, false
}

func (s *Session) runNonStreamingStep(ctx context.Context, parts []genai.Part) ([]genai.Part, bool) {
	resp, err := s.chat.SendMessage(ctx, parts...)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			s.emit(llm.Event{Type: llm.EventError, Data: &llm.ErrorData{Message: err.Error()}})
		}
		return nil, false
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, false
	}
	var functionCalls []*genai.FunctionCall
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			s.emit(llm.Event{Type: llm.EventMessage, Data: &llm.MessageData{Content: part.Text}})
		}
		if part.FunctionCall != nil {
			functionCalls = append(functionCalls, part.FunctionCall)
		}
	}
	if len(functionCalls) > 0 {
		return s.dispatchToolCalls(functionCalls), true
	}
	return nil, false
}

func (s *Session) dispatchToolCalls(functionCalls []*genai.FunctionCall) []genai.Part {
	var parts []genai.Part
	for _, fc := range functionCalls {
		parts = append(parts, s.handleToolCall(fc))
	}
	return parts
}

func (s *Session) handleToolCall(tc *genai.FunctionCall) genai.Part {
	var result map[string]any

	params, argsStr := llm.NormalizeToolArguments(tc.Args)
	resAny, err := llm.InvokeTool(s.toolMap, params, llm.ToolInvocation{
		ID:        tc.Name, // Gemini doesn't use unique call IDs natively in the same way.
		Name:      tc.Name,
		Arguments: argsStr,
	})

	if err != nil {
		if _, ok := s.toolMap[tc.Name]; !ok {
			result = map[string]any{"error": fmt.Sprintf("Unknown tool %s", tc.Name)}
		} else {
			result = map[string]any{"error": fmt.Sprintf("Error executing tool: %v", err)}
		}
	} else {
		result = llm.ResultMap(resAny)
	}

	if result == nil {
		result = map[string]any{"error": fmt.Sprintf("Unknown tool %s", tc.Name)}
	}

	return *genai.NewPartFromFunctionResponse(tc.Name, result)
}
