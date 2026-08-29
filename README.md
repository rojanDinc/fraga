# Fraga
<p align="center"><img width="400" alt="Fraga logo" src="docs/images/full_logo.png" /></p>

<p align="center">Fraga (Fråga in Swedish) translates to "question" in English<br />is a CLI tool for asking one-shot questions to LLMs.</p>

<p align="center"><img width="800" alt="Fraga demo" src="docs/images/demo.gif" /></p>

## Installation

```bash
go install github.com/rojanDinc/fraga@latest
```

## Configuration

Run `fraga init` to create a default configuration file at `~/.config/fraga/fraga.jsonc`.

### Using OpenRouter

OpenRouter exposes an OpenAI-compatible API, so it can be configured as a provider of type `openai` by pointing the base URL at OpenRouter:

```jsonc
"providers": {
  "openrouter": {
    "type": "openai",
    "api_key": "${OPENROUTER_API_KEY}",
    "base_url": "https://openrouter.ai/api/v1"
  }
}
```

### Environment Variables

Any provider value can reference an environment variable with the `${VAR}` syntax, which is resolved when the config is loaded. This is the recommended way to keep secrets like API keys out of the config file:

```jsonc
"providers": {
  "my-openai": {
    "type": "openai",
    "api_key": "${OPENAI_API_KEY}",
    "base_url": "https://api.openai.com/v1"
  }
}
```

Missing variables expand to an empty string.

Runtime settings can also be overridden via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `FRAGA_MODEL` | Default model to use | |
| `FRAGA_PROVIDER` | Provider to use (must match a name in the providers map) | |
| `FRAGA_TEMPERATURE` | Sampling temperature | 0.3 |
| `FRAGA_MAX_TOKENS` | Maximum tokens per response | 4096 |
| `FRAGA_RENDER_MARKDOWN` | Enable markdown rendering | false |

## Usage

Ask a question:

```bash
fraga Show me a kubernetes deployment manifest example
```

Use a specific model:

```bash
fraga --model claude-haiku-4.5 "Explain goroutines in Go"
```

### Commands

- `fraga init` - Initialize configuration file
- `fraga list-tools` - List available MCP tools
- `fraga [question]` - Ask a question

## MCP Setup

MCP servers provide tools that the LLM can use to interact with external systems. Configure servers in the `mcp` section:

### Stdio Servers

```jsonc
"mcp": {
  "filesystem": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed"],
    "env": {}
  }
}
```

### Streamable HTTP Servers

Remote MCP servers using the Streamable HTTP transport:

```jsonc
"mcp": {
  "remote": {
    "url": "https://remote-mcp-server.com/mcp",
    "headers": {
      "Authorization": "Bearer your-token"
    }
  }
}
```

Values can reference environment variables with the `${VAR}` syntax, e.g. `"Authorization": "Bearer ${MCP_TOKEN}"`.
