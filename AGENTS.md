# Fraga Agent Guidelines

This document provides guidelines for AI agents working on the Fraga codebase.

## Project Overview

Fraga is a CLI tool for asking one-shot questions to LLMs with MCP tool support. It's written in Go 1.25.4 and uses environment-based configuration via envconfig.

## Build Commands

```bash
# Build the CLI binary
go build -o fraga ./cmd/fraga

# Build and run
go run ./cmd/fraga

# Install dependencies
go mod tidy
go mod download

# Verify module
go mod verify
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/config
go test ./internal/llm
go test ./internal/mcp
go test ./internal/cli

# Run a single test
go test -run TestFunctionName ./package

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run race detector
go test -race ./...
```

## Lint Commands

```bash
# Format code (always run before committing)
gofmt -w .

# Or use goimports for import management
goimports -w .

# Vet for common issues
go vet ./...

# Static analysis (if golangci-lint is available)
golangci-lint run
```

## Code Style Guidelines

### General
- Use `gofmt` for formatting
- Line length: aim for 100-120 characters
- No trailing whitespace
- End files with a newline
- Use tabs for indentation (Go standard)

### Imports
Order matters. Group imports as:
```go
import (
    // Standard library packages (alphabetical)
    "context"
    "encoding/json"
    "fmt"
    "os"
    
    // Third-party packages (alphabetical)
    "github.com/kelseyhightower/envconfig"
    "github.com/openai/openai-go"
    "github.com/spf13/cobra"
    
    // Internal/local packages
    "github.com/rojanDinc/fraga/internal/config"
    "github.com/rojanDinc/fraga/internal/llm"
)
```

### Naming Conventions

**Types:**
- Exported types: `PascalCase` (e.g., `Provider`, `StreamChunk`)
- Unexported types: `camelCase` (e.g., `openaiProvider` if needed)
- Interface names: noun describing behavior (e.g., `Provider`)

**Functions:**
- Exported functions: `PascalCase` (e.g., `NewProvider`, `Load`)
- Unexported functions: `camelCase` (e.g., `parseModels`, `loadMCPServers`)
- Constructor pattern: `NewXXX` (e.g., `NewOpenAIProvider`)
- Getter pattern: direct field access preferred, not `GetXXX()`

**Variables:**
- Short names in small scopes: `cfg`, `err`, `ctx`
- Descriptive names in larger scopes: `configDir`, `providerName`
- Constants: `PascalCase` for exported, `camelCase` for unexported
- Acronyms: all caps (e.g., `APIKey`, `MCP`, `URL`)

**Files:**
- Lowercase, no underscores (e.g., `config.go`, `openai.go`)
- One main concept per file
- Test files: `*_test.go`

### Types and Structs

```go
// Use struct tags for configuration
type Config struct {
    DefaultModel string               `envconfig:"default_model" default:"gpt-4o"`
    Providers    Providers            `envconfig:"-"`
}

// Group related fields
type ProviderConfig struct {
    APIKey  string   `envconfig:"api_key"`
    BaseURL string   `envconfig:"base_url"`
    Models  []string `envconfig:"models"`
}
```

### Error Handling

Always wrap errors with context:
```go
// Good
if err != nil {
    return fmt.Errorf("failed to load config from environment: %w", err)
}

// Good - specific context
return fmt.Errorf("failed to call tool %s: %w", toolName, err)

// Avoid - no context
return err
```

Use sentinel errors sparingly. Prefer error wrapping.

### Context Usage

Always pass `context.Context` as first parameter:
```go
func (p *Provider) Chat(ctx context.Context, messages []Message) error {
    // Use ctx for all operations
}
```

### Interfaces

Define interfaces where needed for testability:
```go
type Provider interface {
    Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (<-chan StreamChunk, error)
}
```

### Comments

- Document all exported types, functions, and methods
- Use complete sentences with periods
- Start with the name of the thing being documented

```go
// Provider defines the interface for LLM providers.
type Provider interface {
    // Chat streams a chat completion from the LLM.
    Chat(ctx context.Context, messages []Message, tools []Tool, settings config.Settings) (<-chan StreamChunk, error)
}

// NewProvider creates a new LLM provider based on the provider name.
func NewProvider(cfg *config.Config, providerName string) (Provider, error)
```

### Testing

- Test file naming: `package_test.go`
- Use table-driven tests
- Test error cases explicitly
- Mock interfaces for unit tests

Example:
```go
func TestLoad(t *testing.T) {
    tests := []struct {
        name    string
        envVars map[string]string
        wantErr bool
    }{
        {
            name: "valid config",
            envVars: map[string]string{
                "FRAGA_OPENAI_API_KEY": "test-key",
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Project Structure

```
cmd/
  fraga/
    main.go           # Entry point
internal/
  cli/                # Cobra CLI commands
    root.go
    commands.go
  config/             # Configuration management
    config.go
  llm/                # LLM providers
    provider.go       # Interface definition
    openai.go
    anthropic.go
    openrouter.go
  mcp/                # MCP client
    client.go
go.mod
go.sum
```

## Dependencies

Key dependencies to be familiar with:
- `github.com/spf13/cobra` - CLI framework
- `github.com/kelseyhightower/envconfig` - Environment configuration
- `github.com/openai/openai-go` - OpenAI SDK
- `github.com/anthropics/anthropic-sdk-go` - Anthropic SDK
- `github.com/mark3labs/mcp-go` - MCP client

## Configuration

Configuration is environment-based using envconfig:
- Prefix: `FRAGA_`
- Format: `FRAGA_SECTION_FIELD` (e.g., `FRAGA_OPENAI_API_KEY`)
- MCP servers configured in `~/.config/fraga/mcp.json`

### Custom Headers

Each provider supports custom HTTP headers via the `headers` field in the provider configuration:

```json
"openai": {
  "api_key": "sk-your-openai-api-key",
  "base_url": "https://api.openai.com/v1",
  "headers": {
    "X-Custom-Header": "custom-value",
    "OpenAI-Beta": "assistants=v2"
  }
}
```

Custom headers are applied to every outgoing HTTP request for that provider. This is useful for:
- Setting provider-specific beta flags
- Adding authentication headers for proxies
- Passing custom metadata (e.g., `HTTP-Referer` for OpenRouter)


## Common Tasks

Adding a new LLM provider:
1. Create provider file in `internal/llm/`
2. Implement `Provider` interface
3. Add to `NewProvider()` switch statement
4. Add config struct to `config.Providers`

Adding a new command:
1. Add command function in `internal/cli/commands.go`
2. Register in `NewRootCmd()` in `root.go`

## Pre-commit Checklist

- [ ] Code formatted with `gofmt -w .`
- [ ] All tests pass: `go test ./...`
- [ ] No vet issues: `go vet ./...`
- [ ] Module tidy: `go mod tidy`
- [ ] Binary builds: `go build -o fraga ./cmd/fraga`
- [ ] Comments added for exported items

## Build Tags

None currently used. If needed:
```go
//go:build integration
// +build integration
```

## Environment for Development

Required environment variables for testing:
```bash
export FRAGA_OPENAI_API_KEY="your-key"
# or
export FRAGA_ANTHROPIC_API_KEY="your-key"
# or
export FRAGA_OPENROUTER_API_KEY="your-key"
```
