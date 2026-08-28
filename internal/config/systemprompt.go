package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
)

const systemPromptsDirName = "system_prompts"

type systemPromptMatter struct {
	Temperature *float64 `yaml:"temperature"`
	MaxTokens   int      `yaml:"max_tokens"`
}

//go:embed default_system_prompt.md
var defaultSystemPromptContent string

type SystemPrompt struct {
	Content     string
	Temperature *float64
	MaxTokens   int
}

func GetDefaultSystemPrompt() (SystemPrompt, error) {
	return parseSystemPrompt([]byte(defaultSystemPromptContent))
}

func LoadSystemPrompt(name string) (SystemPrompt, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return SystemPrompt{}, fmt.Errorf("failed to get config directory: %w", err)
	}

	promptPath := filepath.Join(configDir, systemPromptsDirName, name+".md")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return SystemPrompt{}, fmt.Errorf("system prompt %q not found at %s: %w", name, promptPath, err)
	}

	systemPrompt, err := parseSystemPrompt(data)
	if err != nil {
		return SystemPrompt{}, fmt.Errorf("failed to parse system prompt %q: %w", name, err)
	}

	return systemPrompt, nil
}

func parseSystemPrompt(data []byte) (SystemPrompt, error) {
	var matter systemPromptMatter

	content, err := frontmatter.Parse(strings.NewReader(string(data)), &matter)
	if err != nil {
		return SystemPrompt{}, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return SystemPrompt{
		Content:     strings.TrimSpace(string(content)),
		Temperature: matter.Temperature,
		MaxTokens:   matter.MaxTokens,
	}, nil
}

func GetSystemPromptsDir() string {
	configDir, err := GetConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, systemPromptsDirName)
}

func InitSystemPromptsDir() error {
	dir := GetSystemPromptsDir()
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}
