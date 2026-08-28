package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRouterHeaders(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		headers  map[string]string
		expected map[string]string
	}{
		{
			name:    "openrouter base url adds attribution headers",
			baseURL: "https://openrouter.ai/api/v1",
			expected: map[string]string{
				"HTTP-Referer":            openAIProviderAppURL,
				"X-OpenRouter-Title":      openAIProviderAppName,
				"X-OpenRouter-Categories": "cli-agent",
			},
		},
		{
			name:    "custom openrouter subdomain adds attribution headers",
			baseURL: "https://my-proxy.openrouter.ai/api/v1",
			expected: map[string]string{
				"HTTP-Referer":            openAIProviderAppURL,
				"X-OpenRouter-Title":      openAIProviderAppName,
				"X-OpenRouter-Categories": "cli-agent",
			},
		},
		{
			name:    "non-openrouter base url leaves headers unchanged",
			baseURL: "https://api.openai.com/v1",
			headers: map[string]string{
				"X-Custom": "value",
			},
			expected: map[string]string{
				"X-Custom": "value",
			},
		},
		{
			name:    "explicit headers override openrouter defaults",
			baseURL: "https://openrouter.ai/api/v1",
			headers: map[string]string{
				"HTTP-Referer": "https://example.com",
			},
			expected: map[string]string{
				"HTTP-Referer":            "https://example.com",
				"X-OpenRouter-Title":      openAIProviderAppName,
				"X-OpenRouter-Categories": "cli-agent",
			},
		},
		{
			name:     "empty base url treated as openai",
			baseURL:  "",
			headers:  map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, openRouterHeaders(tt.baseURL, tt.headers))
		})
	}
}

func TestToOpenAIMessages(t *testing.T) {
	t.Run("converts all roles", func(t *testing.T) {
		msgs, err := toOpenAIMessages([]Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello"},
			{
				Role:    "assistant",
				Content: "let me check",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "read_file", Arguments: `{"path":"/tmp/x"}`},
				},
			},
			{Role: "tool", ToolCallID: "call_1", Content: `{"ok":true}`},
		})
		require.NoError(t, err)
		require.Len(t, msgs, 5)

		assert.NotNil(t, msgs[0].OfSystem)
		assert.NotNil(t, msgs[1].OfUser)

		// Plain assistant message should not carry tool calls.
		require.NotNil(t, msgs[2].OfAssistant)
		assert.Empty(t, msgs[2].OfAssistant.ToolCalls)

		// Assistant with tool calls must keep the IDs for response mapping.
		require.NotNil(t, msgs[3].OfAssistant)
		require.Len(t, msgs[3].OfAssistant.ToolCalls, 1)
		assert.Equal(t, "call_1", msgs[3].OfAssistant.ToolCalls[0].ID)
		assert.Equal(t, "read_file", msgs[3].OfAssistant.ToolCalls[0].Function.Name)
		assert.Equal(t, "let me check", msgs[3].OfAssistant.Content.OfString.Value)

		// Tool result must reference the tool call ID.
		require.NotNil(t, msgs[4].OfTool)
		assert.Equal(t, "call_1", msgs[4].OfTool.ToolCallID)
	})

	t.Run("unknown role errors", func(t *testing.T) {
		_, err := toOpenAIMessages([]Message{{Role: "weird", Content: "hi"}})
		assert.Error(t, err)
	})
}
