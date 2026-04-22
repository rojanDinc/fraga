package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		expectedErr error
	}{
		{
			name:     "valid openai provider",
			provider: "openai",
		},
		{
			name:     "valid anthropic provider",
			provider: "anthropic",
		},
		{
			name:     "valid openrouter provider",
			provider: "openrouter",
		},
		{
			name:        "invalid provider",
			provider:    "unknown",
			expectedErr: ErrUnknownProvider,
		},
		{
			name:        "empty provider",
			provider:    "",
			expectedErr: ErrUnknownProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidProvider(tt.provider)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
