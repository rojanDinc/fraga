# Fraga

Fraga (Fråga in swedish) translates to "question" in english is a CLI tool for asking one-shot questions to LLMs with MCP (Model Context Protocol) tool support.

## Installation

```bash
go install github.com/rojanDinc/fraga@latest
```

## Configuration

Run `fraga init` to create a default configuration file at `~/.config/fraga/fraga.jsonc`.

### Configuration File

```jsonc
{
  // Default model to use for all requests
  // For one-shot questions, smaller models like gpt-5.4-nano, claude-haiku, or
  // gemini-3.1-flash-lite are often sufficient and significantly faster than
  // larger models.
  "default_model": "openai/gpt-5.4-nano",

  // Provider configuration
  "providers": {
    "openai": {
      "api_key": "",
      "base_url": "https://api.openai.com/v1",
      "models": ["gpt-5.4-nano"]
    },
    "anthropic": {
      "api_key": "",
      "base_url": "https://api.anthropic.com",
      "models": ["claude-haiku-4-5"]
    },
    "openrouter": {
      "api_key": "",
      "base_url": "https://openrouter.ai/api/v1",
      "models": ["openai/gpt-5.4-nano", "anthropic/claude-haiku-4-5"]
    }
  },

  "settings": {
    "temperature": 0.3,
    "max_tokens": 4096,
    "system_prompt": "",
    "render_markdown": true
  },

  "mcp": {}
}
```

### Environment Variables

Any configuration value can be overridden with environment variables:

| Variable | Description |
|----------|-------------|
| `FRAGA_DEFAULT_MODEL` | Default model to use |
| `FRAGA_OPENAI_API_KEY` | OpenAI API key |
| `FRAGA_ANTHROPIC_API_KEY` | Anthropic API key |
| `FRAGA_OPENROUTER_API_KEY` | OpenRouter API key |
| `FRAGA_TEMPERATURE` | Sampling temperature |
| `FRAGA_MAX_TOKENS` | Maximum tokens per response |
| `FRAGA_RENDER_MARKDOWN` | Enable markdown rendering |

## Usage

Ask a question:

```bash
fraga "Show me a kubernetes deployment manifest example"
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

### SSE Servers

```jsonc
"mcp": {
  "remote": {
    "url": "https://remote-mcp-server.com/sse"
  }
}
```
