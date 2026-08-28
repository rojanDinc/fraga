package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/rojanDinc/fraga/internal/config"
)

// defaultAnthropicMaxTokens is the fallback for MaxTokens when not configured.
// The Anthropic API requires max_tokens to be a positive integer.
const defaultAnthropicMaxTokens = 4096

type AnthropicProvider struct {
	client anthropic.Client
	model  string
}

func NewAnthropicProvider(cfg *config.ProviderConfig, model string) *AnthropicProvider {
	var opts []option.RequestOption

	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
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
	anthropicMessages, systemPromptMessages, err := toAnthropicMessages(messages)
	if err != nil {
		return ChatResult{}, err
	}

	var anthropicTools []anthropic.ToolUnionParam
	for _, tool := range tools {
		toolParam := anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
			InputSchema: toAnthropicInputSchema(tool.Parameters),
		}
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{OfTool: &toolParam})
	}

	maxTokens := settings.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultAnthropicMaxTokens
	}

	req := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		Messages:  anthropicMessages,
		MaxTokens: int64(maxTokens),
		System:    systemPromptMessages,
	}

	if len(anthropicTools) > 0 {
		req.Tools = anthropicTools
	}

	if settings.Temperature != nil {
		req.Temperature = anthropic.Float(*settings.Temperature)
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
		Content:      content.String(),
		ToolCalls:    toolCalls,
		InputTokens:  int(resp.Usage.InputTokens),
		OutputTokens: int(resp.Usage.OutputTokens),
	}, nil
}

// toAnthropicMessages splits generic messages into Anthropic conversation
// messages and system blocks. Assistant messages become text plus tool_use
// content blocks; tool results become tool_result blocks in a user message,
// preserving the request/response association by tool call ID.
func toAnthropicMessages(messages []Message) ([]anthropic.MessageParam, []anthropic.TextBlockParam, error) {
	anthropicMessages := make([]anthropic.MessageParam, 0)
	systemPromptMessages := make([]anthropic.TextBlockParam, 0)

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemPromptMessages = append(systemPromptMessages, anthropic.TextBlockParam{Text: msg.Content})
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case "assistant":
			blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.ToolCalls)+1)
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, json.RawMessage(tc.Arguments), tc.Name))
			}
			if len(blocks) == 0 {
				return nil, nil, fmt.Errorf("assistant message with neither content nor tool calls")
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(blocks...))
		case "tool":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
			))
		default:
			return nil, nil, fmt.Errorf("unknown message role %q", msg.Role)
		}
	}
	return anthropicMessages, systemPromptMessages, nil
}

// toAnthropicInputSchema converts a generic JSON-schema map (as exposed by
// MCP tools) to the Anthropic input schema fields. The MCP schema's "type",
// "properties" and "required" keys are mapped to their dedicated SDK fields;
// any remaining keys are passed through as extra fields.
func toAnthropicInputSchema(parameters map[string]interface{}) anthropic.ToolInputSchemaParam {
	schema := anthropic.ToolInputSchemaParam{}
	extra := make(map[string]any)

	for k, v := range parameters {
		switch k {
		case "properties":
			schema.Properties = v
		case "required":
			if s, ok := toStringSlice(v); ok {
				schema.Required = s
			} else {
				extra[k] = v
			}
		case "type":
			// Handled implicitly; ToolInputSchemaParam marshals as "object".
		default:
			extra[k] = v
		}
	}

	if len(extra) > 0 {
		schema.ExtraFields = extra
	}
	return schema
}

func toStringSlice(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return t, true
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}
