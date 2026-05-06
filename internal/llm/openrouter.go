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

func NewOpenRouterProvider(cfg *config.ProviderConfig, model string) *OpenRouterProvider {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	// Use default base URL if not provided
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = config.DefaultOpenRouterBaseURL
	}
	opts = append(opts, option.WithBaseURL(baseURL))

	for k, v := range cfg.Headers {
		opts = append(opts, option.WithHeader(k, v))
	}

	client := openai.NewClient(opts...)

	return &OpenRouterProvider{client: client, model: model}
}

func (p *OpenRouterProvider) Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (ChatResult, error) {
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

	completion, err := p.client.Chat.Completions.New(ctx, req)
	if err != nil {
		return ChatResult{}, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(completion.Choices) == 0 {
		return ChatResult{}, fmt.Errorf("no choices in completion response")
	}

	msg := completion.Choices[0].Message
	var toolCalls []ToolCall
	for _, tc := range msg.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return ChatResult{
		Content:   msg.Content,
		ToolCalls: toolCalls,
	}, nil
}
