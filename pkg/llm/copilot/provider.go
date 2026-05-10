package copilot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/e9169/kopilot/pkg/llm"
	sdk "github.com/github/copilot-sdk/go"
)

// Provider implements llm.Provider using the GitHub Copilot SDK.
type Provider struct {
	client *sdk.Client
}

// NewProvider creates a new Copilot provider.
func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "GitHub Copilot"
}

func (p *Provider) Start(ctx context.Context) error {
	p.client = sdk.NewClient(&sdk.ClientOptions{
		LogLevel: "error",
	})
	if err := p.client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start copilot client: %w", err)
	}
	return nil
}

func (p *Provider) Stop() error {
	if p.client != nil {
		return p.client.Stop()
	}
	return nil
}

func parseMCPServers(extra map[string]any) map[string]sdk.MCPServerConfig {
	if extra == nil {
		return nil
	}
	raw, ok := extra["MCPServers"]
	if !ok || raw == nil {
		return nil
	}
	if typed, ok := raw.(map[string]sdk.MCPServerConfig); ok {
		return typed
	}
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]sdk.MCPServerConfig, len(rawMap))
	for name, cfgAny := range rawMap {
		if entry, ok := parseMCPServerEntry(cfgAny); ok {
			out[name] = entry
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseMCPServerEntry(cfgAny any) (sdk.MCPServerConfig, bool) {
	switch cfg := cfgAny.(type) {
	case sdk.MCPServerConfig:
		return cfg, true
	case map[string]string:
		entry := sdk.MCPServerConfig{}
		if v := cfg["type"]; v != "" {
			entry["type"] = v
		}
		if v := cfg["url"]; v != "" {
			entry["url"] = v
		}
		return entry, len(entry) > 0
	case map[string]any:
		entry := sdk.MCPServerConfig{}
		if v, ok := cfg["type"].(string); ok && v != "" {
			entry["type"] = v
		}
		if v, ok := cfg["url"].(string); ok && v != "" {
			entry["url"] = v
		}
		return entry, len(entry) > 0
	}
	return nil, false
}

func (p *Provider) CreateSession(ctx context.Context, config *llm.SessionConfig) (llm.Session, error) {
	sdkTools := make([]sdk.Tool, len(config.Tools))
	for i, t := range config.Tools {
		handler := t.Handler
		sdkTools[i] = sdk.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
			Handler: func(inv sdk.ToolInvocation) (sdk.ToolResult, error) {
				argsStr := "{}"
				if b, err := json.Marshal(inv.Arguments); err == nil {
					argsStr = string(b)
				}

				var params map[string]any
				if argsStr != "{}" {
					_ = json.Unmarshal([]byte(argsStr), &params)
				}

				resAny, err := handler(params, llm.ToolInvocation{
					ID:        inv.ToolCallID,
					Name:      inv.ToolName,
					Arguments: argsStr,
				})

				var textResult string
				if resBytes, err := json.Marshal(resAny); err == nil {
					textResult = string(resBytes)
				} else {
					textResult = fmt.Sprintf("%v", resAny)
				}

				return sdk.ToolResult{
					TextResultForLLM: textResult,
					ResultType:       "json",
				}, err
			},
		}
	}

	mcpServers := parseMCPServers(config.ExtraConfig)

	var customAgents []sdk.CustomAgentConfig
	if config.ExtraConfig != nil {
		if agents, ok := config.ExtraConfig["CustomAgents"].([]sdk.CustomAgentConfig); ok {
			customAgents = agents
		}
	}

	session, err := p.client.CreateSession(ctx, &sdk.SessionConfig{
		Model:               config.Model,
		Streaming:           config.Streaming,
		Tools:               sdkTools,
		SystemMessage:       &sdk.SystemMessageConfig{Mode: "replace", Content: config.SystemMessage},
		OnPermissionRequest: sdk.PermissionHandler.ApproveAll,
		MCPServers:          mcpServers,
		CustomAgents:        customAgents,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create copilot session: %w", err)
	}

	return &Session{session: session}, nil
}

// Session implements llm.Session for Copilot.
type Session struct {
	session *sdk.Session
}

func (s *Session) Disconnect() error {
	return s.session.Disconnect()
}

func (s *Session) SendPrompt(ctx context.Context, prompt string) error {
	_, err := s.session.Send(ctx, sdk.MessageOptions{Prompt: prompt})
	return err
}

func (s *Session) On(handler func(llm.Event)) {
	s.session.On(func(sdkEvent sdk.SessionEvent) {
		event := llm.Event{}
		switch sdkEvent.Type {
		case "assistant.message":
			event.Type = llm.EventMessage
			if d, ok := sdkEvent.Data.(*sdk.AssistantMessageData); ok {
				event.Data = &llm.MessageData{Content: d.Content}
			}
		case "assistant.message_delta":
			event.Type = llm.EventDelta
			if d, ok := sdkEvent.Data.(*sdk.AssistantMessageDeltaData); ok {
				event.Data = &llm.DeltaData{Content: d.DeltaContent}
			}
		case "session.error":
			event.Type = llm.EventError
			if d, ok := sdkEvent.Data.(*sdk.SessionErrorData); ok {
				event.Data = &llm.ErrorData{Message: d.Message}
			}
		case "session.idle":
			event.Type = llm.EventIdle
			event.Data = nil
		case "assistant.usage":
			event.Type = llm.EventUsage
			if d, ok := sdkEvent.Data.(*sdk.AssistantUsageData); ok && d.QuotaSnapshots != nil {
				if snapshot, exists := d.QuotaSnapshots["premium_interactions"]; exists {
					event.Data = &llm.UsageData{
						QuotaPercentage: snapshot.RemainingPercentage,
						QuotaUnlimited:  snapshot.IsUnlimitedEntitlement,
						QuotaUsed:       snapshot.UsedRequests,
						QuotaTotal:      snapshot.EntitlementRequests,
					}
				}
			}
		default:
			// Unhandled event
			return
		}
		handler(event)
	})
}
