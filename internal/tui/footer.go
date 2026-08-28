package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func Footer(elapsed time.Duration, inputTokens, outputTokens int, provider, model string) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Muted)

	separator := style.Render(strings.Repeat("─", terminalWidth()))
	metadata := style.Render(fmt.Sprintf(
		"%s • Ctx: %s • Provider: %s • Model: %s",
		elapsed.Round(time.Millisecond),
		FormatTokens(inputTokens+outputTokens),
		provider,
		model,
	))

	fmt.Fprintf(os.Stdout, "\n%s\n%s\n", separator, metadata)
}

func FormatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d tokens", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk tokens", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fm tokens", float64(n)/1000000)
}
