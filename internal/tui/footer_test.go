package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want string
	}{
		{name: "zero", in: 0, want: "0 tokens"},
		{name: "below one thousand", in: 999, want: "999 tokens"},
		{name: "exactly one thousand", in: 1000, want: "1.0k tokens"},
		{name: "kilo range", in: 1234, want: "1.2k tokens"},
		{name: "exactly one million", in: 1_000_000, want: "1.0m tokens"},
		{name: "mega range", in: 2_345_000, want: "2.3m tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTokens(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
