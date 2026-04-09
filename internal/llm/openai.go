package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/rojanDinc/fraga/internal/config"
)

type OpenAIProvider struct {
	client openai.Client
}

func NewOpenAIProvider(cfg *config.ProviderConfig) *OpenAIProvider {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	// Use default base URL if not provided
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = config.DefaultOpenAIBaseURL
	}
	opts = append(opts, option.WithBaseURL(baseURL))

	client := openai.NewClient(opts...)
	return &OpenAIProvider{client: client}
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (<-chan StreamChunk, error) {
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
		Model:     openai.ChatModelGPT4o,
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
