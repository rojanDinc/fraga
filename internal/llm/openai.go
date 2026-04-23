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
	model  string
}

func NewOpenAIProvider(cfg *config.ProviderConfig, model string) *OpenAIProvider {
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
	return &OpenAIProvider{client: client, model: model}
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (ChatResult, error) {
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
