package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
)

func PrintAnswer(content string, pretty bool) error {
	content = strings.TrimSpace(content)
	if pretty {
		return renderMarkdown(content)
	}
	return printPlain(content)
}

func renderMarkdown(content string) error {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(terminalWidth()),
	)
	if err != nil {
		return fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	out, err := renderer.Render(content)
	if err != nil {
		return err
	}

	if _, err := os.Stdout.WriteString(out); err != nil {
		fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func printPlain(content string) error {
	if _, err := os.Stdout.WriteString(content + "\n"); err != nil {
		return err
	}

	return nil
}
