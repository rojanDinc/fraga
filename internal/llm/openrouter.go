package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/rojanDinc/fraga/internal/config"
)

type OpenRouterProvider struct {
	client openai.Client
	model  string
}

func NewOpenRouterProvider(cfg *config.ProviderConfig) *OpenRouterProvider {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	// Use default base URL if not provided
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = config.DefaultOpenRouterBaseURL
	}
	opts = append(opts, option.WithBaseURL(baseURL))

	client := openai.NewClient(opts...)
	model := "openai/gpt-4o"
	if len(cfg.Models) > 0 {
		model = cfg.Models[0]
	}

	return &OpenRouterProvider{client: client, model: model}
}

func (p *OpenRouterProvider) Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (<-chan StreamChunk, error) {
	stream := make(chan StreamChunk)

	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case "user":
			openaiMessages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			openaiMessages[i] = openai.AssistantMessage(msg.Content)
		case "system":
			openaiMessages[i] = openai.SystemMessage(msg.Content)
		}
	}

	var openaiTools []openai.ChatCompletionToolParam
	for _, tool := range tools {
		params := openai.FunctionDefinitionParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
		}
		if tool.Parameters != nil {
			params.Parameters = openai.FunctionParameters(tool.Parameters)
		}
		openaiTools = append(openaiTools, openai.ChatCompletionToolParam{
			Function: params,
		})
	}

	req := openai.ChatCompletionNewParams{
		Model:     openai.ChatModel(p.model),
		Messages:  openaiMessages,
		Tools:     openaiTools,
		MaxTokens: openai.Int(int64(settings.MaxTokens)),
	}

	if settings.Temperature > 0 {
		req.Temperature = openai.Float(settings.Temperature)
	}

	go func() {
		defer close(stream)

		chatStream := p.client.Chat.Completions.NewStreaming(ctx, req)
		defer chatStream.Close()

		var toolCalls []ToolCall

		for chatStream.Next() {
			chunk := chatStream.Current()

			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta

				if delta.Content != "" {
					stream <- StreamChunk{Content: delta.Content}
				}

				for _, tc := range delta.ToolCalls {
					if int(tc.Index) >= len(toolCalls) {
						toolCalls = append(toolCalls, ToolCall{})
					}
					if tc.ID != "" {
						toolCalls[tc.Index].ID = tc.ID
					}
					if tc.Function.Name != "" {
						toolCalls[tc.Index].Name = tc.Function.Name
					}
					toolCalls[tc.Index].Arguments += tc.Function.Arguments
				}
			}
		}

		if err := chatStream.Err(); err != nil {
			stream <- StreamChunk{Error: fmt.Errorf("streaming error: %w", err)}
			return
		}

		stream <- StreamChunk{ToolCalls: toolCalls, Done: true}
	}()

	return stream, nil
}
