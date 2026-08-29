package tui

import (
	"context"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

type Action func(ctx context.Context) error

func Spinner(title string, action Action) error {
	return newSpinner(title, DefaultTheme.Accent, action)
}

func ToolSpinner(action Action) error {
	return newSpinner("Running tools...", DefaultTheme.Tool, action)
}

func newSpinner(title string, color lipgloss.Color, action Action) error {
	return spinner.New().
		Title(title).
		Style(lipgloss.NewStyle().Foreground(color)).
		ActionWithErr(action).
		Run()
}
