package llm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAnthropicInputSchema(t *testing.T) {
	tests := []struct {
		name           string
		parameters     map[string]interface{}
		wantProperties interface{}
		wantRequired   []string
		wantExtra      map[string]any
	}{
		{
			name: "full schema splits into dedicated fields",
			parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"query"},
			},
			wantProperties: map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
			},
			wantRequired: []string{"query"},
		},
		{
			name: "required as string slice",
			parameters: map[string]interface{}{
				"required": []string{"a", "b"},
			},
			wantRequired: []string{"a", "b"},
		},
		{
			name: "unknown keys pass through as extra fields",
			parameters: map[string]interface{}{
				"$schema":           "http://json-schema.org/draft-07/schema#",
				"additionalPropers": true,
			},
			wantExtra: map[string]any{
				"$schema":           "http://json-schema.org/draft-07/schema#",
				"additionalPropers": true,
			},
		},
		{
			name:         "empty schema",
			parameters:   map[string]interface{}{},
			wantExtra:    nil,
			wantRequired: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := toAnthropicInputSchema(tt.parameters)
			assert.Equal(t, tt.wantProperties, schema.Properties)
			assert.Equal(t, tt.wantRequired, schema.Required)
			assert.Equal(t, tt.wantExtra, schema.ExtraFields)

			// Sanity: the schema must serialize to a flat JSON object, with
			// "properties" at the top level (not double-nested).
			data, err := json.Marshal(schema)
			require.NoError(t, err)
			var obj map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &obj))
			if tt.wantProperties != nil {
				assert.Equal(t, tt.wantProperties, obj["properties"])
			}
			if tt.wantRequired != nil {
				assert.NotNil(t, obj["required"])
			}
		})
	}
}

func TestToAnthropicMessages(t *testing.T) {
	t.Run("system prompt returns empty when no system messages", func(t *testing.T) {
		msgs, system, err := toAnthropicMessages([]Message{
			{Role: "user", Content: "hi"},
		})
		require.NoError(t, err)
		assert.Len(t, msgs, 1)
		assert.Empty(t, system)
	})

	t.Run("assistant with empty content and no tool calls errors", func(t *testing.T) {
		_, _, err := toAnthropicMessages([]Message{
			{Role: "assistant", Content: ""},
		})
		assert.Error(t, err)
	})

	t.Run("unknown role errors", func(t *testing.T) {
		_, _, err := toAnthropicMessages([]Message{
			{Role: "weird", Content: "hi"},
		})
		assert.Error(t, err)
	})

	t.Run("tool result becomes user message with tool_result block", func(t *testing.T) {
		msgs, _, err := toAnthropicMessages([]Message{
			{Role: "tool", ToolCallID: "call_1", Content: `{"ok":true}`},
		})
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "user", string(msgs[0].Role))
		require.Len(t, msgs[0].Content, 1)
		block := msgs[0].Content[0]
		require.NotNil(t, block.OfToolResult)
		assert.Equal(t, "call_1", block.OfToolResult.ToolUseID)
	})

	t.Run("assistant tool calls become tool_use blocks", func(t *testing.T) {
		msgs, _, err := toAnthropicMessages([]Message{
			{
				Role:    "assistant",
				Content: "let me check",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "read_file", Arguments: `{"path":"/tmp/x"}`},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "assistant", string(msgs[0].Role))
		require.Len(t, msgs[0].Content, 2)
		assert.NotNil(t, msgs[0].Content[0].OfText)
		require.NotNil(t, msgs[0].Content[1].OfToolUse)
		assert.Equal(t, "read_file", msgs[0].Content[1].OfToolUse.Name)
		assert.Equal(t, "call_1", msgs[0].Content[1].OfToolUse.ID)
	})
}
