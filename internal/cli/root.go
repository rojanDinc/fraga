package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rojanDinc/fraga/internal/config"
	"github.com/rojanDinc/fraga/internal/llm"
	"github.com/rojanDinc/fraga/internal/mcp"
	"github.com/rojanDinc/fraga/internal/tui"
	"github.com/spf13/cobra"
)

// TODO: Make this configurable
// maxToolIterations bounds the chat/tool-call loop to prevent infinite
// back-and-forth when a model keeps requesting tools.
const maxToolIterations = 5

var (
	modelFlag        string
	providerFlag     string
	systemPromptFlag string
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fraga [question]",
		Short: "Ask one-shot questions to an LLM",
		Long:  `Fraga is a CLI tool for asking one-shot questions to LLMs with MCP tool support.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAsk(cmd.Context(), strings.Join(args, " "))
		},
	}

	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Override the default model to use")
	cmd.Flags().StringVarP(&providerFlag, "provider", "p", "", "Override the default provider to use")
	cmd.Flags().StringVar(&systemPromptFlag, "system-prompt", "", "Use a custom system prompt from ~/.config/fraga/system_prompts/")
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newListToolsCmd())

	return cmd
}

func runAsk(ctx context.Context, question string) error {
	startTime := time.Now()

	cfgDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgDir)
	if err != nil {
		return err
	}

	model := cfg.Model
	if modelFlag != "" {
		model = modelFlag
	}

	providerName := cfg.Provider
	if providerFlag != "" {
		providerName = providerFlag
	}

	provider, err := llm.NewProvider(cfg, providerName, model)
	if err != nil {
		return err
	}

	var mcpClient *mcp.Client
	var llmTools []llm.Tool
	toolServers := make(map[string]string)

	if len(cfg.MCP) > 0 {
		mcpClient, err = mcp.New(cfg.MCP)
		if err != nil {
			return fmt.Errorf("failed to initialize MCP: %w", err)
		}
		defer mcpClient.Close()

		tools, err := mcpClient.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list MCP tools: %w", err)
		}

		for _, tool := range tools {
			if existing, ok := toolServers[tool.Name]; ok {
				return fmt.Errorf("tool %q is exposed by both MCP servers %q and %q; rename one of them", tool.Name, existing, tool.Server)
			}
			toolServers[tool.Name] = tool.Server
			llmTools = append(llmTools, llm.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			})
		}
	}

	settings := cfg.Settings
	var systemPromptContent string

	if systemPromptFlag != "" {
		sp, err := config.LoadSystemPrompt(systemPromptFlag)
		if err != nil {
			return err
		}
		systemPromptContent = sp.Content
		if sp.Temperature != nil {
			settings.Temperature = sp.Temperature
		}
		if sp.MaxTokens > 0 {
			settings.MaxTokens = sp.MaxTokens
		}
	} else if cfg.Settings.SystemPrompt != "" {
		sp, err := config.LoadSystemPrompt(cfg.Settings.SystemPrompt)
		if err != nil {
			return err
		}
		systemPromptContent = sp.Content
		if sp.Temperature != nil {
			settings.Temperature = sp.Temperature
		}
		if sp.MaxTokens > 0 {
			settings.MaxTokens = sp.MaxTokens
		}
	} else {
		sp, err := config.GetDefaultSystemPrompt()
		if err != nil {
			return err
		}
		systemPromptContent = sp.Content
		if sp.Temperature != nil {
			settings.Temperature = sp.Temperature
		}
		if sp.MaxTokens > 0 {
			settings.MaxTokens = sp.MaxTokens
		}
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPromptContent},
		{Role: "user", Content: question},
	}

	var totalInputTokens int
	var totalOutputTokens int

	for iteration := 0; iteration < maxToolIterations; iteration++ {
		var result llm.ChatResult

		err = tui.Spinner("Preparing an answer...", func(ctx context.Context) error {
			var err error
			result, err = provider.Chat(ctx, messages, llmTools, settings)
			if err != nil {
				return fmt.Errorf("failed to chat: %w", err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		totalInputTokens += result.InputTokens
		totalOutputTokens += result.OutputTokens

		if len(result.ToolCalls) == 0 {
			if err := tui.PrintAnswer(result.Content, settings.ShouldRenderMarkdown()); err != nil {
				fmt.Errorf("failed to print answer: %w", err)
			}
			tui.Footer(time.Since(startTime), totalInputTokens, totalOutputTokens, providerName, model)
			return nil
		}

		if mcpClient == nil {
			return fmt.Errorf("model requested tool calls but no MCP servers are configured")
		}

		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		err = tui.ToolSpinner(func(ctx context.Context) error {
			for _, tc := range result.ToolCalls {
				var args map[string]any
				if tc.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
						return fmt.Errorf("failed to parse arguments for tool %s: %w", tc.Name, err)
					}
				}

				server, ok := toolServers[tc.Name]
				if !ok {
					return fmt.Errorf("no MCP server exposes tool %q", tc.Name)
				}

				toolResult, err := mcpClient.CallTool(ctx, server, tc.Name, args)
				if err != nil {
					return fmt.Errorf("failed to call tool %s: %w", tc.Name, err)
				}

				content, err := toolResultContent(toolResult)
				if err != nil {
					return fmt.Errorf("failed to serialize result of tool %s: %w", tc.Name, err)
				}

				messages = append(messages, llm.Message{
					Role:       "tool",
					Content:    content,
					ToolCallID: tc.ID,
				})
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return fmt.Errorf("exceeded %d tool call iterations", maxToolIterations)
}

// toolResultContent serializes an MCP tool result to a plain string for the
// follow-up chat message.
func toolResultContent(result *mcp.ToolResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
