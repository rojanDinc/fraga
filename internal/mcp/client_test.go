package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvList(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected []string
	}{
		{
			name:     "nil map returns nil",
			env:      nil,
			expected: nil,
		},
		{
			name:     "empty map returns nil",
			env:      map[string]string{},
			expected: nil,
		},
		{
			name: "converts to KEY=VALUE pairs sorted by key",
			env: map[string]string{
				"TOKEN": "secret",
				"DEBUG": "1",
			},
			expected: []string{"DEBUG=1", "TOKEN=secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, envList(tt.env))
		})
	}
}
