package llm

import (
	"context"
	"fmt"

	"github.com/rojanDinc/fraga/internal/config"
)

type Message struct {
	Role    string
	Content string
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
	switch providerName {
	case "openai":
		if cfg.Providers.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("the OpenAI provider not configured")
		}
		return NewOpenAIProvider(&cfg.Providers.OpenAI, model), nil
	case "anthropic":
		if cfg.Providers.Anthropic.APIKey == "" {
			return nil, fmt.Errorf("the Anthropic provider not configured")
		}
		return NewAnthropicProvider(&cfg.Providers.Anthropic, model), nil
	case "openrouter":
		if cfg.Providers.OpenRouter.APIKey == "" {
			return nil, fmt.Errorf("the OpenRouter provider not configured")
		}
		return NewOpenRouterProvider(&cfg.Providers.OpenRouter, model), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}
