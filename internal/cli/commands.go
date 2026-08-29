package cli

import (
	"errors"
	"fmt"
	"log/slog"

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
				if !errors.Is(err, config.ErrConfigExists) {
					return err
				}
				cfgDir, err := config.GetConfigDir()
				if err != nil {
					return err
				}

				slog.Warn("Config already exists", "path", cfgDir)
				return nil
			}

			fmt.Println(initMessage)

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
			cfgDir, err := config.GetConfigDir()
			if err != nil {
				return err
			}

			cfg, err := config.LoadWithoutValidation(cfgDir)
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
					fmt.Printf("  - %s: %s (streamable HTTP)\n", name, server.URL)
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
