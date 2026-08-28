package llm

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/rojanDinc/fraga/internal/config"
)

// OpenRouter attribution headers, set when a provider points at OpenRouter so
// the app shows up in its rankings and analytics:
// https://openrouter.ai/docs/app-attribution
const (
	openAIProviderAppURL  = "https://github.com/rojanDinc/fraga"
	openAIProviderAppName = "Fraga"
)

// openRouterHeaders returns the user-provided headers merged with default
// OpenRouter attribution headers when the base URL points at OpenRouter.
// Explicitly configured headers always win over the defaults.
func openRouterHeaders(baseURL string, headers map[string]string) map[string]string {
	if !strings.Contains(baseURL, "openrouter.ai") {
		return headers
	}

	result := map[string]string{
		"HTTP-Referer":            openAIProviderAppURL,
		"X-OpenRouter-Title":      openAIProviderAppName,
		"X-OpenRouter-Categories": "cli-agent",
	}
	maps.Copy(result, headers)
	return result
}

type OpenAIProvider struct {
	client openai.Client
	model  string
}

func NewOpenAIProvider(cfg *config.ProviderConfig, model string) *OpenAIProvider {
	var opts []option.RequestOption

	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}

	// Use default base URL if not provided
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = config.DefaultOpenAIBaseURL
	}
	opts = append(opts, option.WithBaseURL(baseURL))

	for k, v := range openRouterHeaders(baseURL, cfg.Headers) {
		opts = append(opts, option.WithHeader(k, v))
	}

	client := openai.NewClient(opts...)

	return &OpenAIProvider{client: client, model: model}
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (ChatResult, error) {
	openaiMessages, err := toOpenAIMessages(messages)
	if err != nil {
		return ChatResult{}, err
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
		Model:    openai.ChatModel(p.model),
		Messages: openaiMessages,
	}

	if len(openaiTools) > 0 {
		req.Tools = openaiTools
	}

	if settings.MaxTokens > 0 {
		req.MaxTokens = openai.Int(int64(settings.MaxTokens))
	}

	if settings.Temperature != nil {
		req.Temperature = openai.Float(*settings.Temperature)
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
		Content:      msg.Content,
		ToolCalls:    toolCalls,
		InputTokens:  int(completion.Usage.PromptTokens),
		OutputTokens: int(completion.Usage.CompletionTokens),
	}, nil
}

// toOpenAIMessages converts generic messages to OpenAI request messages.
// Assistant messages carry tool calls, tool results use the dedicated tool
// role keyed by the tool call ID.
func toOpenAIMessages(messages []Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				assistant := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: toOpenAIToolCalls(msg.ToolCalls),
				}
				if msg.Content != "" {
					assistant.Content.OfString = openai.String(msg.Content)
				}
				openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant})
			} else {
				openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
			}
		case "system":
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content))
		case "tool":
			openaiMessages = append(openaiMessages, openai.ToolMessage(msg.Content, msg.ToolCallID))
		default:
			return nil, fmt.Errorf("unknown message role %q", msg.Role)
		}
	}
	return openaiMessages, nil
}

func toOpenAIToolCalls(toolCalls []ToolCall) []openai.ChatCompletionMessageToolCallParam {
	params := make([]openai.ChatCompletionMessageToolCallParam, len(toolCalls))
	for i, tc := range toolCalls {
		params[i] = openai.ChatCompletionMessageToolCallParam{
			ID: tc.ID,
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      tc.Name,
				Arguments: tc.Arguments,
			},
		}
	}
	return params
}
