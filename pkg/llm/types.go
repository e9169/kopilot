package llm

import (
	"context"
)

// Provider represents an AI provider (e.g., Copilot, OpenAI, Gemini).
type Provider interface {
	// Name returns a short human-readable name for the provider (e.g. "GitHub Copilot").
	Name() string
	// Start initializes the provider.
	Start(ctx context.Context) error
	// Stop shuts down the provider.
	Stop() error
	// CreateSession creates a new interactive session with the provider.
	CreateSession(ctx context.Context, config *SessionConfig) (Session, error)
}

// Session represents an active conversation session with the AI.
type Session interface {
	// Disconnect closes the session.
	Disconnect() error
	// SendPrompt sends a message to the AI.
	SendPrompt(ctx context.Context, prompt string) error
	// On registers an event handler for session events.
	On(handler func(Event))
}

// SessionConfig contains configuration for creating a session.
type SessionConfig struct {
	Model         string
	Streaming     bool
	Tools         []Tool
	SystemMessage string
	// Allow provider-specific configs if necessary
	ExtraConfig map[string]any
}

// Tool represents a callable tool.
type Tool struct {
	Name        string
	Description string
	// Parameters follows JSON Schema format.
	Parameters map[string]any
	// Handler is the function executed when the tool is called.
	Handler func(params any, inv ToolInvocation) (any, error)
}

// ToolInvocation contains information about the tool call.
type ToolInvocation struct {
	ID        string
	Name      string
	Arguments string
}

// EventType represents the type of session event.
type EventType string

const (
	EventMessage EventType = "message"
	EventDelta   EventType = "delta"
	EventIdle    EventType = "idle"
	EventError   EventType = "error"
	EventUsage   EventType = "usage"
)

// Event is the generic event emitted by the Session.
type Event struct {
	Type EventType
	Data any
}

// MessageData contains the complete assistant response.
type MessageData struct {
	Content string
}

// DeltaData contains an incremental chunk of the assistant response.
type DeltaData struct {
	Content string
}

// ErrorData contains error information.
type ErrorData struct {
	Message string
}

// UsageData contains token or quota usage metrics.
type UsageData struct {
	QuotaPercentage float64
	QuotaUnlimited  bool
	QuotaUsed       float64
	QuotaTotal      float64
}

// EventEmitter stores and dispatches session event handlers.
type EventEmitter struct {
	handlers []func(Event)
}

// On registers an event handler.
func (e *EventEmitter) On(handler func(Event)) {
	e.handlers = append(e.handlers, handler)
}

// Emit dispatches an event to all registered handlers.
func (e *EventEmitter) Emit(event Event) {
	for _, handler := range e.handlers {
		handler(event)
	}
}
