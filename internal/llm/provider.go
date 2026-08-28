package llm

import (
	"context"
	"fmt"

	"github.com/rojanDinc/fraga/internal/config"
)

// Message is a single message in a conversation. The fields used depend on
// the role: system/user messages use Content; assistant messages may carry
// Content and/or ToolCalls; tool messages carry the result of a single tool
// call identified by ToolCallID.
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ChatResult struct {
	Content      string
	ToolCalls    []ToolCall
	InputTokens  int
	OutputTokens int
}

type Provider interface {
	// TODO: Replace settings with options belonging to this package
	Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (ChatResult, error)
}

func NewProvider(cfg *config.Config, providerName string, model string) (Provider, error) {
	providerConfig, ok := cfg.Providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %q not found in config", providerName)
	}

	switch providerConfig.Type {
	case "openai":
		return NewOpenAIProvider(&providerConfig, model), nil
	case "anthropic":
		return NewAnthropicProvider(&providerConfig, model), nil
	default:
		return nil, fmt.Errorf("provider %q has unknown type %q (must be openai or anthropic)", providerName, providerConfig.Type)
	}
}
