package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/tailscale/hujson"
	"k8s.io/utils/ptr"
)

//go:embed examples/config.jsonc
var defaultConfigTemplate []byte

const (
	ConfigDirName       = "fraga"
	ConfigFileName      = "fraga.json"
	ConfigFileNameJSONC = "fraga.jsonc"

	// Default provider URLs
	DefaultOpenAIBaseURL    = "https://api.openai.com/v1"
	DefaultAnthropicBaseURL = "https://api.anthropic.com"

	// configPathHint is the user-facing location of the config file, referenced
	// in validation error messages.
	configPathHint = "~/.config/fraga/fraga.json"
)

var (
	ErrUnknownProvider = errors.New("unknown provider")
	ErrConfigExists    = errors.New("config already exists")
)

// Config holds the complete Fraga configuration
type Config struct {
	Model     string                    `json:"model"`
	Provider  string                    `json:"provider"`
	Providers map[string]ProviderConfig `json:"providers"`
	Settings  Settings                  `json:"settings"`
	MCP       map[string]MCPServer      `json:"mcp,omitempty"`
}

// ProviderConfig holds configuration for a single provider.
// The Type field determines which backend is used: "openai" or "anthropic".
type ProviderConfig struct {
	Type    string            `json:"type"`
	APIKey  string            `json:"api_key,omitempty"`
	BaseURL string            `json:"base_url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Settings holds application settings
type Settings struct {
	Temperature    *float64 `json:"temperature,omitempty"`
	MaxTokens      int      `json:"max_tokens"`
	SystemPrompt   string   `json:"system_prompt"`
	RenderMarkdown *bool    `json:"render_markdown,omitempty"`
}

// ShouldRenderMarkdown returns true if markdown rendering should be enabled.
// It defaults to true when the field is omitted from config.
func (s Settings) ShouldRenderMarkdown() bool {
	return ptr.Deref(s.RenderMarkdown, true)
}

// MCPServer holds configuration for an MCP server
// Headers are sent with every request when using the Streamable HTTP transport
// (i.e. when URL is set).
type MCPServer struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Load loads config with requiring proper configuration set
func Load(configDir string) (*Config, error) {
	cfg, err := LoadWithoutValidation(configDir)
	if err != nil {
		return nil, err
	}

	// Validate that at least one provider is configured
	if !cfg.hasConfiguredProvider() {
		return nil, fmt.Errorf("no LLM provider configured. Please add at least one provider to %s", configPathHint)
	}

	// Validate that model is set
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is not set in config. Please set it in %s", configPathHint)
	}

	// Validate that provider is set
	if cfg.Provider == "" {
		return nil, fmt.Errorf("provider is not set in config. Please set it in %s", configPathHint)
	}

	// Validate provider name and type
	if err := cfg.isValidProvider(cfg.Provider); err != nil {
		return nil, fmt.Errorf("invalid provider: %s", err)
	}

	return cfg, nil
}

// LoadWithoutValidation loads config without requiring provider configuration
func LoadWithoutValidation(configDir string) (*Config, error) {
	cfg, err := loadFromFile(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at ~/.config/fraga/fraga.json or ~/.config/fraga/fraga.jsonc. Run 'fraga init' to create one")
		}
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

func loadFromFile(configDir string) (*Config, error) {
	configPath, err := getConfigPath(configDir)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Standardize HuJSON/JSONC to standard JSON (removes comments and trailing commas)
	standardized, err := hujson.Standardize(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(standardized, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Resolve ${VAR} references to environment variables in provider values
	cfg.expandEnvVars()

	return &cfg, nil
}

func (c *Config) applyEnvOverrides() {
	// Override model
	if val := os.Getenv("FRAGA_MODEL"); val != "" {
		c.Model = val
	}

	// Override provider
	if val := os.Getenv("FRAGA_PROVIDER"); val != "" {
		c.Provider = val
	}

	// Override settings
	if val := os.Getenv("FRAGA_TEMPERATURE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			c.Settings.Temperature = &f
		}
	}
	if val := os.Getenv("FRAGA_MAX_TOKENS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			c.Settings.MaxTokens = i
		}
	}
	if val := os.Getenv("FRAGA_RENDER_MARKDOWN"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			c.Settings.RenderMarkdown = &b
		}
	}
}

func (c *Config) hasConfiguredProvider() bool {
	return len(c.Providers) > 0
}

func (c *Config) isValidProvider(name string) error {
	provider, ok := c.Providers[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownProvider, name)
	}

	switch provider.Type {
	case "openai", "anthropic":
		return nil
	default:
		return fmt.Errorf("%w: provider %q has invalid type %q (must be openai or anthropic)", ErrUnknownProvider, name, provider.Type)
	}
}

// expandEnvVars resolves ${VAR} references to environment variables in
// provider configuration values. Missing variables expand to an empty string.
func (c *Config) expandEnvVars() {
	for name, provider := range c.Providers {
		provider.APIKey = expandEnvString(provider.APIKey)
		provider.BaseURL = expandEnvString(provider.BaseURL)
		for k, v := range provider.Headers {
			provider.Headers[k] = expandEnvString(v)
		}
		c.Providers[name] = provider
	}

	for name, server := range c.MCP {
		for k, v := range server.Headers {
			server.Headers[k] = expandEnvString(v)
		}
		c.MCP[name] = server
	}
}

func expandEnvString(s string) string {
	return os.Expand(s, os.Getenv)
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", ConfigDirName), nil
}

func getConfigPath(configDir string) (string, error) {
	// Prefer .jsonc for new configs with comments, falling back to .json.
	if ok, path := existingConfigPath(configDir); ok {
		return path, nil
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

func existingConfigPath(configDir string) (bool, string) {
	for _, name := range []string{ConfigFileNameJSONC, ConfigFileName} {
		path := filepath.Join(configDir, name)
		if _, err := os.Stat(path); err == nil {
			return true, path
		}
	}
	return false, ""
}

// InitDefault creates a default config file with helpful comments
func InitDefault() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Prefer .jsonc for new configs to support comments
	configPath := filepath.Join(configDir, ConfigFileNameJSONC)

	// Check if config already exists (check both .json and .jsonc)
	if ok, path := existingConfigPath(configDir); ok {
		return fmt.Errorf("%w at %s", ErrConfigExists, path)
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, defaultConfigTemplate, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
