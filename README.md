# trakt-mcp-go

[![CI](https://github.com/kofort9/trakt-mcp-go/actions/workflows/ci.yml/badge.svg)](https://github.com/kofort9/trakt-mcp-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kofifort/trakt-mcp-go)](https://goreportcard.com/report/github.com/kofifort/trakt-mcp-go)

Trakt.tv MCP server for Claude - High-performance Go implementation, single binary deployment.

## Features

- 🚀 **Single binary** - No runtime dependencies, instant startup
- 🔐 **OAuth device flow** - Secure Trakt.tv authentication
- 📺 **Watch tracking** - Log episodes and movies as watched
- 🔍 **Search** - Find shows and movies by title
- 📊 **History** - View and analyze your watch history

## Installation

### Using Go

```bash
go install github.com/kofifort/trakt-mcp-go/cmd/trakt-mcp@latest
```

### From Source

```bash
git clone https://github.com/kofort9/trakt-mcp-go.git
cd trakt-mcp-go
make build
```

## Configuration

Set environment variables for Trakt.tv API access:

```bash
export TRAKT_CLIENT_ID="your-client-id"
export TRAKT_CLIENT_SECRET="your-client-secret"
export TRAKT_ACCESS_TOKEN="your-access-token"  # After authentication
```

Get your API credentials at [Trakt.tv API](https://trakt.tv/oauth/applications).

## Usage with Claude Code

Add the server to your Claude Code configuration:

```bash
claude mcp add trakt-mcp-go -- /path/to/trakt-mcp
```

Or with environment variables:

```bash
claude mcp add trakt-mcp-go \
  -e TRAKT_CLIENT_ID="your-client-id" \
  -e TRAKT_CLIENT_SECRET="your-client-secret" \
  -- /path/to/trakt-mcp
```

## Available Tools

| Tool | Description |
|------|-------------|
| `authenticate` | Start OAuth device flow authentication |
| `search_show` | Search for TV shows and movies |
| `get_history` | Retrieve watch history |
| `log_watch` | Log a watch (coming soon) |

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Run locally
make run
```

## Architecture

```
trakt-mcp-go/
├── cmd/trakt-mcp/        # Entry point
├── internal/
│   ├── mcp/              # MCP JSON-RPC server
│   │   ├── server.go     # Server implementation
│   │   ├── handlers.go   # Tool handlers
│   │   └── types.go      # MCP protocol types
│   └── trakt/            # Trakt API client
│       ├── client.go     # HTTP client
│       └── types.go      # API types
```

## Why Go?

This is a Go port of [trakt-mcp](https://github.com/kofort9/trakt-mcp) (TypeScript). Benefits:

- **Single binary deployment** - No node_modules
- **Faster startup** - Important for MCP servers invoked on demand
- **Strong typing** - Catches bugs at compile time
- **Cross-platform builds** - Easy distribution

## License

MIT
