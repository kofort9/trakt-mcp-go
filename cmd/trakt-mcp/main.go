// trakt-mcp is an MCP server for Trakt.tv integration with Claude.
//
// It communicates over stdio using JSON-RPC 2.0 per the MCP specification.
// Configure with environment variables:
//   - TRAKT_CLIENT_ID: Your Trakt API client ID
//   - TRAKT_CLIENT_SECRET: Your Trakt API client secret
//   - TRAKT_ACCESS_TOKEN: OAuth access token (after authentication)
//   - TRAKT_REFRESH_TOKEN: OAuth refresh token (optional)
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kofifort/trakt-mcp-go/internal/mcp"
	"github.com/kofifort/trakt-mcp-go/internal/trakt"
)

func main() {
	// Configure structured logging to stderr (stdout is for MCP protocol)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))

	// Load Trakt configuration from environment
	config := trakt.ConfigFromEnv()
	client := trakt.NewClient(config, logger)

	if !client.IsConfigured() {
		logger.Warn("TRAKT_CLIENT_ID not set - some tools will not work")
	}

	// Create MCP server and register tools
	server := mcp.NewServer(logger)
	mcp.RegisterTools(server, client)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("shutting down")
		cancel()
	}()

	// Run the server
	if err := server.Run(ctx); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

func getLogLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
