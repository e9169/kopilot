package copilot

import (
	"context"
	"fmt"

	"github.com/e9169/kopilot/pkg/llm"
	sdk "github.com/github/copilot-sdk/go"
	"github.com/github/copilot-sdk/go/rpc"
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
		if url := cfg["url"]; url != "" {
			return sdk.MCPHTTPServerConfig{URL: url}, true
		}
	case map[string]any:
		if url, ok := cfg["url"].(string); ok && url != "" {
			return sdk.MCPHTTPServerConfig{URL: url}, true
		}
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
				params, argsStr := llm.NormalizeToolArguments(inv.Arguments)
				resAny, err := handler(params, llm.ToolInvocation{
					ID:        inv.ToolCallID,
					Name:      inv.ToolName,
					Arguments: argsStr,
				})
				textResult := llm.ResultString(resAny)
				if err != nil {
					textResult = fmt.Sprintf("Error executing tool: %v", err)
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
		Streaming:           sdk.Bool(config.Streaming),
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

func convertSDKEvent(sdkEvent sdk.SessionEvent) (llm.Event, bool) {
	event := llm.Event{}
	switch sdkEvent.Type() {
	case sdk.SessionEventTypeAssistantMessage:
		event.Type = llm.EventMessage
		if d, ok := sdkEvent.Data.(*rpc.AssistantMessageData); ok {
			event.Data = &llm.MessageData{Content: d.Content}
		}
	case sdk.SessionEventTypeAssistantMessageDelta:
		event.Type = llm.EventDelta
		if d, ok := sdkEvent.Data.(*rpc.AssistantMessageDeltaData); ok {
			event.Data = &llm.DeltaData{Content: d.DeltaContent}
		}
	case sdk.SessionEventTypeSessionError:
		event.Type = llm.EventError
		if d, ok := sdkEvent.Data.(*rpc.SessionErrorData); ok {
			event.Data = &llm.ErrorData{Message: d.Message}
		}
	case sdk.SessionEventTypeSessionIdle:
		event.Type = llm.EventIdle
	case sdk.SessionEventTypeAssistantUsage:
		event.Type = llm.EventUsage
		if d, ok := sdkEvent.Data.(*rpc.AssistantUsageData); ok && d.QuotaSnapshots != nil {
			if snapshot, exists := d.QuotaSnapshots["premium_interactions"]; exists {
				event.Data = &llm.UsageData{
					QuotaPercentage: snapshot.RemainingPercentage,
					QuotaUnlimited:  snapshot.IsUnlimitedEntitlement,
					QuotaUsed:       float64(snapshot.UsedRequests),
					QuotaTotal:      float64(snapshot.EntitlementRequests),
				}
			}
		}
	default:
		return event, false
	}
	return event, true
}

func (s *Session) On(handler func(llm.Event)) {
	s.session.On(func(sdkEvent sdk.SessionEvent) {
		if event, ok := convertSDKEvent(sdkEvent); ok {
			handler(event)
		}
	})
}
