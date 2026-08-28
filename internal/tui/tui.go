package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type Theme struct {
	Accent lipgloss.Color
	Tool   lipgloss.Color
	Muted  lipgloss.Color
}

var DefaultTheme = Theme{
	Accent: lipgloss.Color("#FF8B42"),
	Tool:   lipgloss.Color("#D00DFC"),
	Muted:  lipgloss.Color("241"),
}

func terminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}
