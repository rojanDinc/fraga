package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/rojanDinc/fraga/internal/config"
	"github.com/rojanDinc/fraga/internal/llm"
	"github.com/rojanDinc/fraga/internal/mcp"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const grayColor = "#808080"

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
			return runAsk(strings.Join(args, " "))
		},
	}

	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Override the default model to use")
	cmd.Flags().StringVarP(&providerFlag, "provider", "p", "", "Override the default provider to use")
	cmd.Flags().StringVar(&systemPromptFlag, "system-prompt", "", "Use a custom system prompt from ~/.config/fraga/system_prompts/")
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newListToolsCmd())

	return cmd
}

func runAsk(question string) error {
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

	if providerName != "openai" && providerName != "anthropic" && providerName != "openrouter" {
		return fmt.Errorf("invalid provider: %q (must be openai, anthropic, or openrouter)", providerName)
	}

	provider, err := llm.NewProvider(cfg, providerName, model)
	if err != nil {
		return err
	}

	var mcpClient *mcp.Client
	var llmTools []llm.Tool
	if len(cfg.MCP) > 0 {
		mcpClient, err = mcp.New(cfg.MCP)
		if err != nil {
			return fmt.Errorf("failed to initialize MCP: %w", err)
		}
		defer mcpClient.Close()

		tools, err := mcpClient.ListTools(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list MCP tools: %w", err)
		}

		for _, tool := range tools {
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
		settings.Temperature = sp.Temperature
		settings.MaxTokens = sp.MaxTokens
	} else if cfg.Settings.SystemPrompt != "" {
		sp, err := config.LoadSystemPrompt(cfg.Settings.SystemPrompt)
		if err != nil {
			return err
		}
		systemPromptContent = sp.Content
		settings.Temperature = sp.Temperature
		settings.MaxTokens = sp.MaxTokens
	} else {
		sp, err := config.GetDefaultSystemPrompt()
		if err != nil {
			return err
		}
		systemPromptContent = sp.Content
		settings.Temperature = sp.Temperature
		settings.MaxTokens = sp.MaxTokens
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPromptContent},
		{Role: "user", Content: question},
	}

	var assistantContent strings.Builder
	var toolCalls []llm.ToolCall
	var totalInputTokens int
	var totalOutputTokens int

	// Phase 1: Get the first response from the LLM
	err = spinner.New().
		Title("Preparing an answer...").
		Style(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ff8b42")),
		).
		ActionWithErr(func(ctx context.Context) error {
			result, err := provider.Chat(ctx, messages, llmTools, settings)
			if err != nil {
				return fmt.Errorf("failed to start chat: %w", err)
			}

			assistantContent.WriteString(result.Content)
			toolCalls = result.ToolCalls
			totalInputTokens += result.InputTokens
			totalOutputTokens += result.OutputTokens

			return nil
		}).
		Run()
	if err != nil {
		return err
	}

	// Render markdown for the first response if no tool calls
	if len(toolCalls) == 0 {
		if err := printAnswer(assistantContent.String(), settings.ShouldRenderMarkdown()); err != nil {
			slog.Error("failed to print answer", "err", err)
		}
		printResponseFooter(startTime, totalInputTokens, totalOutputTokens, providerName, model)
	}

	if len(toolCalls) > 0 && mcpClient != nil {
		toolResults := make(map[string]*mcp.ToolResult)

		// Phase 2: Call MCP tools
		err = spinner.New().
			Title("Preparing an answer...").
			ActionWithErr(func(ctx context.Context) error {
				for _, tc := range toolCalls {
					var args map[string]any
					if tc.Arguments != "" {
						if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
							return fmt.Errorf("failed to parse tool arguments: %w", err)
						}
					}

					result, err := mcpClient.CallTool(ctx, findServerForTool(cfg.MCP, tc.Name), tc.Name, args)
					if err != nil {
						return fmt.Errorf("failed to call tool %s: %w", tc.Name, err)
					}
					toolResults[tc.Name] = result
				}

				return nil
			}).
			Run()
		if err != nil {
			return err
		}

		messages = append(messages, llm.Message{Role: "assistant", Content: assistantContent.String()})

		resultJSON, err := json.Marshal(toolResults)
		if err != nil {
			return err
		}
		messages = append(messages, llm.Message{Role: "user", Content: string(resultJSON)})

		var secondContent strings.Builder

		// Phase 3: Get the second response after tool results
		err = spinner.New().
			Title("Preparing an answer...").
			ActionWithErr(func(ctx context.Context) error {
				result, err := provider.Chat(ctx, messages, llmTools, settings)
				if err != nil {
					return fmt.Errorf("failed to continue chat: %w", err)
				}

				secondContent.WriteString(result.Content)
				totalInputTokens += result.InputTokens
				totalOutputTokens += result.OutputTokens

				return nil
			}).
			Run()
		if err != nil {
			return err
		}

		// Render markdown for the second response
		if err := printAnswer(secondContent.String(), settings.ShouldRenderMarkdown()); err != nil {
			slog.Error("failed to print answer", "err", err)
		}
		printResponseFooter(startTime, totalInputTokens, totalOutputTokens, providerName, model)
	}

	return nil
}

// TODO: Fix this function
func findServerForTool(servers map[string]config.MCPServer, toolName string) string {
	for name := range servers {
		return name
	}
	return ""
}

func printAnswer(content string, printPretty bool) error {
	content = strings.TrimSpace(content)
	if printPretty {
		return printPrettyAnswer(content)
	}

	return printPlainAnswer(content)
}

func printPrettyAnswer(content string) error {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	out, err := renderer.Render(content)
	if err != nil {
		return err
	}

	if _, err := os.Stdout.WriteString(out); err != nil {
		slog.Error("failed to write output", "error", err)
	}

	return nil
}

func printPlainAnswer(content string) error {
	if _, err := os.Stdout.WriteString(content); err != nil {
		return err
	}

	return nil
}

func printResponseFooter(startTime time.Time, inputTokens int, outputTokens int, provider string, model string) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}

	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	metadataStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	separator := separatorStyle.Render(strings.Repeat("─", width))

	duration := time.Since(startTime)
	timing := "Took " + duration.Round(time.Millisecond).String()
	totalTokens := inputTokens + outputTokens
	contextStr := formatTokens(totalTokens)

	metadata := metadataStyle.Render(fmt.Sprintf("%s • Ctx: %s • Model: %s/%s", timing, contextStr, provider, model))

	os.Stdout.WriteString("\n" + separator + "\n")
	os.Stdout.WriteString(metadata + "\n")
}

func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d tokens", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk tokens", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fm tokens", float64(n)/1000000)
}
