package cli

import (
	"fmt"
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
				fmt.Println("Config already exists at ~/.config/fraga/fraga.json or ~/.config/fraga/fraga.jsonc")
				return nil
			}

			fmt.Println("Configuration initialized!")
			fmt.Println()
			fmt.Println("Config file created at: ~/.config/fraga/fraga.jsonc")
			fmt.Println()
			fmt.Println("=== Next Steps ===")
			fmt.Println()
			fmt.Println("1. Edit the config file to add your LLM provider(s)")
			fmt.Println("2. Set default_model to your preferred model")
			fmt.Println("3. Add your API keys (can also use environment variables)")
			fmt.Println()
			fmt.Println("=== Example Configuration ===")
			fmt.Println(config.GetExampleConfig())
			fmt.Println()
			fmt.Println("=== Environment Variable Overrides ===")
			fmt.Println("You can override any config value using environment variables:")
			fmt.Println("  FRAGA_DEFAULT_MODEL")
			fmt.Println("  FRAGA_OPENAI_API_KEY")
			fmt.Println("  FRAGA_ANTHROPIC_API_KEY")
			fmt.Println("  FRAGA_OPENROUTER_API_KEY")
			fmt.Println("  FRAGA_TEMPERATURE")
			fmt.Println("  FRAGA_MAX_TOKENS")
			fmt.Println("  FRAGA_RENDER_MARKDOWN")

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
				fmt.Println("No MCP servers configured.")
				fmt.Println("Add them to ~/.config/fraga/fraga.json or ~/.config/fraga/fraga.jsonc under the 'mcp' key")
				return nil
			}

			fmt.Println("Configured MCP servers:")
			for name, server := range cfg.MCP {
				if server.URL != "" {
					fmt.Printf("  - %s: %s (SSE)\n", name, server.URL)
				} else {
					fmt.Printf("  - %s: %s %v (stdio)\n", name, server.Command, server.Args)
				}
			}

			fmt.Println("\nNote: To see available tools, ensure MCP servers are configured correctly.")
			fmt.Println("Tools are discovered dynamically when running queries.")

			return nil
		},
	}
}
