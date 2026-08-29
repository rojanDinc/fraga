package mcp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rojanDinc/fraga/internal/config"
)

const connectTimeout = 10 * time.Second

type Client struct {
	clients map[string]*mcp.ClientSession
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

type headerTransport struct {
	headers map[string]string
	base    http.RoundTripper
}

func New(cfg map[string]config.MCPServer) (*Client, error) {
	clients := make(map[string]*mcp.ClientSession)

	// Sort server names so initialization failures are deterministic and the
	// already-started clients can be cleaned up reliably.

	names := slices.Sorted(maps.Keys(cfg))

	for _, name := range names {
		serverCfg := cfg[name]
		// Bound the connection handshake with a per-server timeout, so a
		// dead or unresponsive server cannot hang startup forever.

		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)

		var transport mcp.Transport
		switch {
		case serverCfg.URL != "":
			httpClient := &http.Client{}
			if len(serverCfg.Headers) > 0 {
				httpClient.Transport = &headerTransport{headers: serverCfg.Headers, base: http.DefaultTransport}
			}
			transport = &mcp.StreamableClientTransport{Endpoint: serverCfg.URL, HTTPClient: httpClient}
		case serverCfg.Command != "":
			cmd := exec.Command(serverCfg.Command, serverCfg.Args...)
			if env := envList(serverCfg.Env); len(env) > 0 {
				cmd.Env = append(os.Environ(), env...)
			}
			transport = &mcp.CommandTransport{Command: cmd}
		default:
			cancel()
			continue
		}

		client := mcp.NewClient(&mcp.Implementation{Name: "fraga", Version: "0.1.0"}, nil)
		session, err := client.Connect(ctx, transport, nil)
		cancel()
		if err != nil {
			closeAll(clients)
			return nil, fmt.Errorf("failed to initialize MCP client for %s: %w", name, err)
		}

		clients[name] = session
	}

	return &Client{clients: clients}, nil
}

func (c *Client) Close() error {
	var errs []error
	for name, client := range c.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MCP client %s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var allTools []Tool

	for serverName, client := range c.clients {
		toolsResult, err := client.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			return nil, fmt.Errorf("failed to list tools from %s: %w", serverName, err)
		}

		for _, tool := range toolsResult.Tools {
			// Preserve the full input schema (type, properties, required, and
			// any extra keys like additionalProperties or $schema), so providers
			// receive complete tool semantics.

			schema, _ := tool.InputSchema.(map[string]interface{})

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

	result, err := client.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
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

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return t.base.RoundTrip(req)
}

// envList converts a server env map to KEY=VALUE pairs. The pairs are
// appended to the inherited process environment by the command transport, so
// configured variables take precedence.

func envList(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	list := make([]string, 0, len(env))
	for k, v := range env {
		list = append(list, k+"="+v)
	}
	sort.Strings(list)
	return list
}

func closeAll(clients map[string]*mcp.ClientSession) {
	for _, c := range clients {
		c.Close()
	}
}
