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
)

//go:embed examples/template.jsonc
var defaultConfigTemplate []byte

//go:embed examples/config.jsonc
var exampleConfig []byte

const (
	ConfigDirName       = "fraga"
	ConfigFileName      = "fraga.json"
	ConfigFileNameJSONC = "fraga.jsonc"

	// Default provider URLs
	DefaultOpenAIBaseURL     = "https://api.openai.com/v1"
	DefaultAnthropicBaseURL  = "https://api.anthropic.com"
	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
)

var ErrUnknownProvider = errors.New("unknown provider")

// Config holds the complete Fraga configuration
type Config struct {
	Model     string               `json:"model"`
	Provider  string               `json:"provider"`
	Providers Providers            `json:"providers"`
	Settings  Settings             `json:"settings"`
	MCP       map[string]MCPServer `json:"mcp,omitempty"`
}

// Providers holds configuration for all LLM providers
type Providers struct {
	OpenAI     ProviderConfig `json:"openai"`
	Anthropic  ProviderConfig `json:"anthropic"`
	OpenRouter ProviderConfig `json:"openrouter"`
}

// ProviderConfig holds configuration for a single provider
type ProviderConfig struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

// Settings holds application settings
type Settings struct {
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"max_tokens"`
	SystemPrompt   string  `json:"system_prompt"`
	RenderMarkdown bool    `json:"render_markdown"`
}

// MCPServer holds configuration for an MCP server
type MCPServer struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// Load loads configuration from JSON/JSONC file with environment variable overrides
func Load(configDir string) (*Config, error) {
	cfg, err := loadFromFile(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at ~/.config/fraga/fraga.json or ~/.config/fraga/fraga.jsonc. Run 'fraga init' to create one")
		}
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Validate that at least one provider is configured
	if !cfg.hasConfiguredProvider() {
		return nil, fmt.Errorf("no LLM provider configured. Please add at least one provider to ~/.config/fraga/fraga.json")
	}

	// Validate that model is set
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is not set in config. Please set it in ~/.config/fraga/fraga.json")
	}

	// Validate that provider is set
	if cfg.Provider == "" {
		return nil, fmt.Errorf("provider is not set in config. Please set it in ~/.config/fraga/fraga.json")
	}

	// Validate provider name
	if err := isValidProvider(cfg.Provider); err != nil {
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
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

	// Override providers
	if val := os.Getenv("FRAGA_OPENAI_API_KEY"); val != "" {
		c.Providers.OpenAI.APIKey = val
	}
	if val := os.Getenv("FRAGA_OPENAI_BASE_URL"); val != "" {
		c.Providers.OpenAI.BaseURL = val
	}

	if val := os.Getenv("FRAGA_ANTHROPIC_API_KEY"); val != "" {
		c.Providers.Anthropic.APIKey = val
	}
	if val := os.Getenv("FRAGA_ANTHROPIC_BASE_URL"); val != "" {
		c.Providers.Anthropic.BaseURL = val
	}

	if val := os.Getenv("FRAGA_OPENROUTER_API_KEY"); val != "" {
		c.Providers.OpenRouter.APIKey = val
	}
	if val := os.Getenv("FRAGA_OPENROUTER_BASE_URL"); val != "" {
		c.Providers.OpenRouter.BaseURL = val
	}

	// Override settings
	if val := os.Getenv("FRAGA_TEMPERATURE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			c.Settings.Temperature = f
		}
	}
	if val := os.Getenv("FRAGA_MAX_TOKENS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			c.Settings.MaxTokens = i
		}
	}
	if val := os.Getenv("FRAGA_RENDER_MARKDOWN"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			c.Settings.RenderMarkdown = b
		}
	}
}

func (c *Config) hasConfiguredProvider() bool {
	return c.Providers.OpenAI.APIKey != "" ||
		c.Providers.Anthropic.APIKey != "" ||
		c.Providers.OpenRouter.APIKey != ""
}

func isValidProvider(provider string) error {
	switch provider {
	case "openai", "anthropic", "openrouter":
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
	}
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", ConfigDirName), nil
}

func getConfigPath(configDir string) (string, error) {
	// Check for .jsonc first (preferred for new configs with comments)
	jsoncPath := filepath.Join(configDir, ConfigFileNameJSONC)
	if _, err := os.Stat(jsoncPath); err == nil {
		return jsoncPath, nil
	}

	// Fall back to .json
	jsonPath := filepath.Join(configDir, ConfigFileName)
	return jsonPath, nil
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
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config already exists at %s", configPath)
	}
	jsonPath := filepath.Join(configDir, ConfigFileName)
	if _, err := os.Stat(jsonPath); err == nil {
		return fmt.Errorf("config already exists at %s", jsonPath)
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

// GetExampleConfig returns an example configuration string with comments
func GetExampleConfig() string {
	return string(exampleConfig)
}
