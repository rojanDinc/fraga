package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/rojanDinc/fraga/internal/config"
)

type AnthropicProvider struct {
	client anthropic.Client
	model  string
}

func NewAnthropicProvider(cfg *config.ProviderConfig, model string) *AnthropicProvider {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	// Use default base URL if not provided
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = config.DefaultAnthropicBaseURL
	}
	opts = append(opts, option.WithBaseURL(baseURL))

	for k, v := range cfg.Headers {
		opts = append(opts, option.WithHeader(k, v))
	}

	client := anthropic.NewClient(opts...)

	return &AnthropicProvider{client: client, model: model}
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (ChatResult, error) {
	anthropicMessages := make([]anthropic.MessageParam, 0)
	systemPromptMessages := make([]anthropic.TextBlockParam, 0)

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemPromptMessages = append(systemPromptMessages, anthropic.TextBlockParam{Text: msg.Content})
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case "assistant":
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	var anthropicTools []anthropic.ToolUnionParam
	for _, tool := range tools {
		toolParam := anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
		}
		if tool.Parameters != nil {
			toolParam.InputSchema = anthropic.ToolInputSchemaParam{
				Properties: tool.Parameters,
			}
		}
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{OfTool: &toolParam})
	}

	req := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		Messages:  anthropicMessages,
		MaxTokens: int64(settings.MaxTokens),
		System:    systemPromptMessages,
	}

	if len(anthropicTools) > 0 {
		req.Tools = anthropicTools
	}

	if settings.Temperature > 0 {
		req.Temperature = anthropic.Float(settings.Temperature)
	}

	resp, err := p.client.Messages.New(ctx, req)
	if err != nil {
		return ChatResult{}, fmt.Errorf("message request failed: %w", err)
	}

	var content strings.Builder
	var toolCalls []ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content.WriteString(block.Text)
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}

	return ChatResult{
		Content:   content.String(),
		ToolCalls: toolCalls,
	}, nil
}
