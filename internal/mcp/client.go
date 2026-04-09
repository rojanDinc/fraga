package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rojanDinc/fraga/internal/config"
)

type Client struct {
	clients map[string]*client.Client
}

func New(cfg map[string]config.MCPServer) (*Client, error) {
	clients := make(map[string]*client.Client)

	for name, serverCfg := range cfg {
		var c *client.Client
		var err error

		if serverCfg.URL != "" {
			c, err = client.NewSSEMCPClient(serverCfg.URL)
		} else if serverCfg.Command != "" {
			c, err = client.NewStdioMCPClient(serverCfg.Command, nil, serverCfg.Args...)
		} else {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create MCP client for %s: %w", name, err)
		}

		ctx := context.Background()
		if err := c.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start MCP client for %s: %w", name, err)
		}

		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "fraga",
			Version: "0.1.0",
		}

		if _, err := c.Initialize(ctx, initRequest); err != nil {
			return nil, fmt.Errorf("failed to initialize MCP client for %s: %w", name, err)
		}

		clients[name] = c
	}

	return &Client{clients: clients}, nil
}

func (c *Client) Close() error {
	for name, client := range c.clients {
		if err := client.Close(); err != nil {
			return fmt.Errorf("failed to close MCP client %s: %w", name, err)
		}
	}
	return nil
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var allTools []Tool

	for serverName, client := range c.clients {
		toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to list tools from %s: %w", serverName, err)
		}

		for _, tool := range toolsResult.Tools {
			schema := make(map[string]interface{})
			if tool.InputSchema.Properties != nil {
				schema["properties"] = tool.InputSchema.Properties
			}
			if tool.InputSchema.Required != nil {
				schema["required"] = tool.InputSchema.Required
			}
			if tool.InputSchema.Type != "" {
				schema["type"] = tool.InputSchema.Type
			}

			allTools = append(allTools, Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: schema,
				Server:      serverName,
			})
		}
	}

	return allTools, nil
}

func (c *Client) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	client, ok := c.clients[serverName]
	if !ok {
		return nil, fmt.Errorf("MCP server not found: %s", serverName)
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName

	if arguments != nil {
		req.Params.Arguments = arguments
	}

	result, err := client.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", toolName, err)
	}

	content := make([]interface{}, len(result.Content))
	for i, c := range result.Content {
		content[i] = c
	}

	return &ToolResult{
		Content: content,
		IsError: result.IsError,
	}, nil
}

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Server      string
}

type ToolResult struct {
	Content []interface{}
	IsError bool
}

func (t Tool) ToLLMTool() map[string]interface{} {
	return map[string]interface{}{
		"name":        t.Name,
		"description": t.Description,
		"parameters":  t.InputSchema,
	}
}

func ToolResultsToJSON(results map[string]*ToolResult) (map[string]interface{}, error) {
	jsonResults := make(map[string]interface{})
	for name, result := range results {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		var obj interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, err
		}
		jsonResults[name] = obj
	}
	return jsonResults, nil
}
