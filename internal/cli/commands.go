package cli

import (
	"log/slog"
	"strings"

	"github.com/rojanDinc/fraga/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration",
		Long:  `Creates a default configuration file at ~/.config/fraga/fraga.jsonc (with comments)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config file
			if err := config.InitDefault(); err != nil {
				// If it already exists, that's OK
				if !strings.HasPrefix(err.Error(), "config already exists") {
					return err
				}
				slog.Info("config already exists at ~/.config/fraga/fraga.json or ~/.config/fraga/fraga.jsonc")
				return nil
			}

			slog.Info("configuration initialized")
			slog.Info("config file created at: ~/.config/fraga/fraga.jsonc")
			slog.Info("next steps: edit config file to add LLM providers, set default_model, add API keys")
			slog.Info("example configuration", "config", config.GetExampleConfig())
			slog.Info("environment variable overrides available",
				"FRAGA_DEFAULT_MODEL", "",
				"FRAGA_OPENAI_API_KEY", "",
				"FRAGA_ANTHROPIC_API_KEY", "",
				"FRAGA_OPENROUTER_API_KEY", "",
				"FRAGA_TEMPERATURE", "",
				"FRAGA_MAX_TOKENS", "",
				"FRAGA_RENDER_MARKDOWN", "")

			return nil
		},
	}
}

func newListToolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-tools",
		Short: "List available MCP tools",
		Long:  `List all available tools from configured MCP servers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWithoutValidation()
			if err != nil {
				return err
			}

			if len(cfg.MCP) == 0 {
				slog.Info("no MCP servers configured")
				slog.Info("add them to ~/.config/fraga/fraga.json or ~/.config/fraga/fraga.jsonc under the 'mcp' key")
				return nil
			}

			slog.Info("configured MCP servers")
			for name, server := range cfg.MCP {
				if server.URL != "" {
					slog.Info("MCP server", "name", name, "url", server.URL, "transport", "SSE")
				} else {
					slog.Info("MCP server", "name", name, "command", server.Command, "args", server.Args, "transport", "stdio")
				}
			}

			slog.Info("to see available tools, ensure MCP servers are configured correctly")
			slog.Info("tools are discovered dynamically when running queries")

			return nil
		},
	}
}
