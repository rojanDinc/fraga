package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidProvider(t *testing.T) {
	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"my-openai": {
				Type:   "openai",
				APIKey: "test-key",
			},
			"my-anthropic": {
				Type:   "anthropic",
				APIKey: "test-key",
			},
			"bad-type": {
				Type:   "unknown",
				APIKey: "test-key",
			},
			"no-type": {
				APIKey: "test-key",
			},
		},
	}

	tests := []struct {
		name        string
		provider    string
		expectedErr error
	}{
		{
			name:     "valid custom openai provider",
			provider: "my-openai",
		},
		{
			name:     "valid custom anthropic provider",
			provider: "my-anthropic",
		},
		{
			name:        "unknown provider",
			provider:    "unknown",
			expectedErr: ErrUnknownProvider,
		},
		{
			name:        "empty provider",
			provider:    "",
			expectedErr: ErrUnknownProvider,
		},
		{
			name:        "invalid type",
			provider:    "bad-type",
			expectedErr: ErrUnknownProvider,
		},
		{
			name:        "missing type",
			provider:    "no-type",
			expectedErr: ErrUnknownProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cfg.isValidProvider(tt.provider)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestHasConfiguredProvider(t *testing.T) {
	tests := []struct {
		name      string
		providers map[string]ProviderConfig
		want      bool
	}{
		{
			name: "provider with api key",
			providers: map[string]ProviderConfig{
				"openai": {Type: "openai", APIKey: "test-key"},
			},
			want: true,
		},
		{
			name: "keyless provider counts as configured",
			providers: map[string]ProviderConfig{
				"local": {Type: "openai", BaseURL: "http://localhost:11434/v1"},
			},
			want: true,
		},
		{
			name:      "no providers",
			providers: map[string]ProviderConfig{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Providers: tt.providers}
			assert.Equal(t, tt.want, cfg.hasConfiguredProvider())
		})
	}
}

func TestExpandEnvVars(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-secret")
	t.Setenv("BASE_URL", "https://openrouter.ai/api/v1")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expands env var",
			input:    "${OPENAI_API_KEY}",
			expected: "sk-secret",
		},
		{
			name:     "expands env var in string",
			input:    "https://${BASE_URL}",
			expected: "https://https://openrouter.ai/api/v1",
		},
		{
			name:     "expands multiple env vars",
			input:    "${OPENAI_API_KEY}-${BASE_URL}",
			expected: "sk-secret-https://openrouter.ai/api/v1",
		},
		{
			name:     "missing env var expands to empty",
			input:    "${MISSING_VAR}",
			expected: "",
		},
		{
			name:     "no env var reference passes through",
			input:    "sk-literal-key",
			expected: "sk-literal-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, expandEnvString(tt.input))
		})
	}
}

func TestConfigExpandEnvVars(t *testing.T) {
	t.Setenv("CLAUDE_API_KEY", "sk-ant-secret")

	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"claude": {
				APIKey:  "${CLAUDE_API_KEY}",
				BaseURL: "https://api.anthropic.com",
				Headers: map[string]string{
					"X-Title": "Fraga",
					"X-Key":   "${CLAUDE_API_KEY}",
				},
			},
		},
	}

	cfg.expandEnvVars()

	provider := cfg.Providers["claude"]
	assert.Equal(t, "sk-ant-secret", provider.APIKey)
	assert.Equal(t, "https://api.anthropic.com", provider.BaseURL)
	assert.Equal(t, "sk-ant-secret", provider.Headers["X-Key"])
	assert.Equal(t, "Fraga", provider.Headers["X-Title"])
}

func TestConfigExpandEnvVarsMCPHeaders(t *testing.T) {
	t.Setenv("MCP_TOKEN", "sk-mcp-secret")

	cfg := &Config{
		MCP: map[string]MCPServer{
			"remote": {
				URL: "https://remote-mcp-server.com/mcp",
				Headers: map[string]string{
					"Authorization": "Bearer ${MCP_TOKEN}",
					"X-Custom":      "literal",
				},
			},
		},
	}

	cfg.expandEnvVars()

	server := cfg.MCP["remote"]
	assert.Equal(t, "Bearer sk-mcp-secret", server.Headers["Authorization"])
	assert.Equal(t, "literal", server.Headers["X-Custom"])
}

func TestLoadSystemPromptRejectsTraversal(t *testing.T) {
	tests := []string{
		"../secret",
		"../../etc/passwd",
		"foo/bar",
		`foo\bar`,
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := LoadSystemPrompt(name)
			assert.Error(t, err)
		})
	}
}

func TestEnvOverrideTemperature(t *testing.T) {
	t.Setenv("FRAGA_TEMPERATURE", "0.5")

	cfg := &Config{}
	cfg.applyEnvOverrides()

	require.NotNil(t, cfg.Settings.Temperature)
	assert.Equal(t, 0.5, *cfg.Settings.Temperature)

	// An explicit zero temperature must be preserved, not treated as unset.
	t.Setenv("FRAGA_TEMPERATURE", "0")
	cfg = &Config{}
	cfg.applyEnvOverrides()
	require.NotNil(t, cfg.Settings.Temperature)
	assert.Equal(t, 0.0, *cfg.Settings.Temperature)
}
