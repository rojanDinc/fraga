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

type StreamChunk struct {
	Content   string
	ToolCalls []ToolCall
	Done      bool
	Error     error
}

type Provider interface {
	Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (<-chan StreamChunk, error)
}

func NewProvider(cfg *config.Config, providerName string) (Provider, error) {
	switch providerName {
	case "openai":
		if cfg.Providers.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("OpenAI provider not configured")
		}
		return NewOpenAIProvider(&cfg.Providers.OpenAI), nil
	case "anthropic":
		if cfg.Providers.Anthropic.APIKey == "" {
			return nil, fmt.Errorf("Anthropic provider not configured")
		}
		return NewAnthropicProvider(&cfg.Providers.Anthropic), nil
	case "openrouter":
		if cfg.Providers.OpenRouter.APIKey == "" {
			return nil, fmt.Errorf("OpenRouter provider not configured")
		}
		return NewOpenRouterProvider(&cfg.Providers.OpenRouter), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}
