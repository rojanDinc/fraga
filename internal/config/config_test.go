package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with openai",
			configJSON: `{
				"provider": "openai",
				"model": "gpt-4o",
				"providers": {
					"openai": {"api_key": "test-key"}
				}
			}`,
			wantErr: false,
		},
		{
			name: "valid config with anthropic",
			configJSON: `{
				"provider": "anthropic",
				"model": "claude-3-5-sonnet",
				"providers": {
					"anthropic": {"api_key": "test-key"}
				}
			}`,
			wantErr: false,
		},
		{
			name: "valid config with openrouter",
			configJSON: `{
				"provider": "openrouter",
				"model": "openai/gpt-4o",
				"providers": {
					"openrouter": {"api_key": "test-key"}
				}
			}`,
			wantErr: false,
		},
		{
			name: "missing provider",
			configJSON: `{
				"model": "gpt-4o",
				"providers": {
					"openai": {"api_key": "test-key"}
				}
			}`,
			wantErr:     true,
			errContains: "provider is not set",
		},
		{
			name: "missing model",
			configJSON: `{
				"provider": "openai",
				"providers": {
					"openai": {"api_key": "test-key"}
				}
			}`,
			wantErr:     true,
			errContains: "model is not set",
		},
		{
			name: "no provider configured",
			configJSON: `{
				"provider": "openai",
				"model": "gpt-4o",
				"providers": {
					"openai": {"api_key": ""}
				}
			}`,
			wantErr:     true,
			errContains: "no LLM provider configured",
		},
		{
			name: "invalid provider name",
			configJSON: `{
				"provider": "unknown",
				"model": "gpt-4o",
				"providers": {
					"openai": {"api_key": "test-key"}
				}
			}`,
			wantErr:     true,
			errContains: "invalid provider: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := "/tmp/fraga-test-config.json"
			err := os.WriteFile(tmpFile, []byte(tt.configJSON), 0600)
			assert.NoError(t, err)
			defer os.Remove(tmpFile)

			// Override config path for testing
			originalGetConfigPath := getConfigPath
			getConfigPath = func() (string, error) {
				return tmpFile, nil
			}
			defer func() { getConfigPath = originalGetConfigPath }()

			cfg, err := Load()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestIsValidProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{
			name:     "valid openai provider",
			provider: "openai",
			wantErr:  false,
		},
		{
			name:     "valid anthropic provider",
			provider: "anthropic",
			wantErr:  false,
		},
		{
			name:     "valid openrouter provider",
			provider: "openrouter",
			wantErr:  false,
		},
		{
			name:     "invalid provider",
			provider: "unknown",
			wantErr:  true,
		},
		{
			name:     "empty provider",
			provider: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidProvider(tt.provider)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
