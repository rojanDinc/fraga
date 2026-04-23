package llm

import (
	"context"
	"fmt"

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

	client := anthropic.NewClient(opts...)
	return &AnthropicProvider{client: client, model: model}
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (<-chan StreamChunk, error) {
	stream := make(chan StreamChunk)

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

	go func() {
		defer close(stream)

		msgStream := p.client.Messages.NewStreaming(ctx, req)

		var toolCalls []ToolCall
		var currentToolCall *ToolCall

		for msgStream.Next() {
			event := msgStream.Current()

			if text := event.Delta.Text; text != "" {
				stream <- StreamChunk{Content: text}
			}

			if event.Delta.PartialJSON != "" {
				if currentToolCall != nil {
					currentToolCall.Arguments += event.Delta.PartialJSON
				}
			}

			if event.Type == "content_block_start" {
				toolUse := event.ContentBlock.AsToolUse()
				if toolUse.ID != "" {
					currentToolCall = &ToolCall{
						ID:   toolUse.ID,
						Name: toolUse.Name,
					}
					toolCalls = append(toolCalls, *currentToolCall)
				}
			}

			if event.Type == "content_block_stop" {
				currentToolCall = nil
			}
		}

		if err := msgStream.Err(); err != nil {
			stream <- StreamChunk{Error: fmt.Errorf("streaming error: %w", err)}
			return
		}

		stream <- StreamChunk{ToolCalls: toolCalls, Done: true}
	}()

	return stream, nil
}
