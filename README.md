# Fraga

<p align="center">Fraga (Fråga in Swedish) translates to "ask" in English<br />is a CLI tool for asking one-shot questions to LLMs, all in your terminal.</p>

<p align="center"><img width="800" alt="Fraga demo" src="docs/images/demo.gif" /></p>

## Features

- One-shot questions for quick answers.
- MCP tool support.
- OpenAI & Anthropic compatible, use any provider that supports these APIs.

## Quickstart

### Installation

```bash
go install github.com/rojanDinc/fraga@latest
```

### Configuration

Run `fraga init` to create a default configuration file at `~/.config/fraga/fraga.jsonc`.

Check the example [config](./internal/config/examples/config.jsonc) file for a full example.

#### Adding a Provider

Each provider has a `type` of `openai` or `anthropic`. Add as many named providers as you need; the `provider` setting selects which one is used.

##### Configuring Providers

```jsonc
{
  "provider": "openrouter",
  "model": "google/gemma-4-31b-it",
  "providers": {
    "openrouter": {
      "type": "openai",
      "api_key": "sk-your-openai-api-key",
      "base_url": "https://openrouter.ai/api/v1"
    },
    "my-anthropic": {
      "type": "anthropic",
      "api_key": "sk-ant-your-anthropic-api-key",
      "base_url": "https://api.anthropic.com",
      "headers": {
        "header-1": "value-1"
      }
    }
    "local": {
      "type": "openai",
      "base_url": "https://localhost:1234/v1"
    }
  }
}
```

##### Environment Variables

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

`api_key` is optional: omit it to use the standard `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` environment variable, or to talk to keyless local servers. Missing variables expand to an empty string.

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

### Commands and Flags

- `fraga init`, initializes a fraga config file in the system home dir if it doesn't exist.
- `fraga list-tools`, lists available MCP tools configured.
- `fraga [question]`, ask a question.
  - `-p, --provider`, override the configured provider.
  - `-m, --model string`, override the configured model.

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
