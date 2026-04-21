package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/rojanDinc/fraga/internal/config"
	"github.com/rojanDinc/fraga/internal/llm"
	"github.com/rojanDinc/fraga/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	modelFlag        string
	systemPromptFlag string
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fraga [question]",
		Short: "Ask one-shot questions to an LLM",
		Long:  `Fraga is a CLI tool for asking one-shot questions to LLMs with MCP tool support.`,
		Args:  cobra.ArbitraryArgs,
		RunE:  runAsk,
	}

	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Override the default model to use")
	cmd.Flags().StringVar(&systemPromptFlag, "system-prompt", "", "Use a custom system prompt from ~/.config/fraga/system_prompts/")
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newListToolsCmd())

	return cmd
}

func runAsk(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide a question to ask")
	}

	question := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	model := cfg.DefaultModel
	if modelFlag != "" {
		model = modelFlag
	}

	providerName, err := cfg.GetProviderForModel(model)
	if err != nil {
		return err
	}

	provider, err := llm.NewProvider(cfg, providerName)
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
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPromptContent},
		{Role: "user", Content: question},
	}

	stream, err := provider.Chat(context.Background(), messages, llmTools, settings)
	if err != nil {
		return fmt.Errorf("failed to start chat: %w", err)
	}

	var toolCalls []llm.ToolCall
	var assistantContent strings.Builder

	// Buffer all content without printing
	for chunk := range stream {
		if chunk.Error != nil {
			return fmt.Errorf("stream error: %w", chunk.Error)
		}

		if chunk.Content != "" {
			assistantContent.WriteString(chunk.Content)
		}

		if len(chunk.ToolCalls) > 0 {
			toolCalls = chunk.ToolCalls
		}

		if chunk.Done {
			break
		}
	}

	// Render markdown for the first response if no tool calls
	if len(toolCalls) == 0 {
		renderMarkdown(assistantContent.String(), settings.RenderMarkdown)
	}

	if len(toolCalls) > 0 && mcpClient != nil {
		toolResults := make(map[string]*mcp.ToolResult)
		for _, tc := range toolCalls {
			var args map[string]interface{}
			if tc.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
					return fmt.Errorf("failed to parse tool arguments: %w", err)
				}
			}

			result, err := mcpClient.CallTool(context.Background(), findServerForTool(cfg.MCP, tc.Name), tc.Name, args)
			if err != nil {
				return fmt.Errorf("failed to call tool %s: %w", tc.Name, err)
			}
			toolResults[tc.Name] = result
		}

		messages = append(messages, llm.Message{Role: "assistant", Content: assistantContent.String()})

		resultJSON, err := json.Marshal(toolResults)
		if err != nil {
			return err
		}
		messages = append(messages, llm.Message{Role: "user", Content: string(resultJSON)})

		stream, err = provider.Chat(context.Background(), messages, llmTools, settings)
		if err != nil {
			return fmt.Errorf("failed to continue chat: %w", err)
		}

		var secondContent strings.Builder
		for chunk := range stream {
			if chunk.Error != nil {
				return fmt.Errorf("stream error: %w", chunk.Error)
			}

			if chunk.Content != "" {
				secondContent.WriteString(chunk.Content)
			}

			if chunk.Done {
				break
			}
		}

		// Render markdown for the second response
		renderMarkdown(secondContent.String(), settings.RenderMarkdown)
	}

	if _, err := os.Stdout.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}
	return nil
}

func findServerForTool(servers map[string]config.MCPServer, toolName string) string {
	for name := range servers {
		return name
	}
	return ""
}

func renderMarkdown(content string, renderMarkdown bool) {
	if !renderMarkdown || content == "" {
		if content != "" {
			if _, err := os.Stdout.WriteString(content); err != nil {
				slog.Error("failed to write output", "error", err)
			}
		}
		return
	}

	out, err := glamour.Render(content, "auto")
	if err != nil {
		slog.Error("markdown rendering failed", "error", err)
		if _, err := os.Stdout.WriteString(content); err != nil {
			slog.Error("failed to write output", "error", err)
		}
		return
	}

	if _, err := os.Stdout.WriteString(out); err != nil {
		slog.Error("failed to write output", "error", err)
	}
}
