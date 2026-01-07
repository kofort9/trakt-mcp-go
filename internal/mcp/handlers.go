package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kofifort/trakt-mcp-go/internal/trakt"
)

// RegisterTools registers all Trakt tools with the MCP server.
func RegisterTools(s *Server, client *trakt.Client) {
	// authenticate - OAuth device flow
	s.RegisterTool(Tool{
		Name:        "authenticate",
		Description: "Authenticate with Trakt.tv using OAuth device flow. Returns a verification URL and code for the user to authorize.",
		InputSchema: JSONSchema{
			Type:       "object",
			Properties: map[string]JSONSchema{},
		},
	}, makeAuthenticateHandler(client))

	// search_show - search for content
	s.RegisterTool(Tool{
		Name:        "search_show",
		Description: "Search for TV shows, movies, or anime by title. Returns matching content with IDs and metadata.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"query": {
					Type:        "string",
					Description: "Search query (title or keywords)",
				},
				"type": {
					Type:        "string",
					Description: "Content type filter (optional)",
					Enum:        []string{"show", "movie"},
				},
			},
			Required: []string{"query"},
		},
	}, makeSearchHandler(client))

	// get_history - retrieve watch history
	s.RegisterTool(Tool{
		Name:        "get_history",
		Description: "Retrieve watch history with optional filters. Supports content type filtering.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"type": {
					Type:        "string",
					Description: "Filter by content type (optional)",
					Enum:        []string{"shows", "movies"},
				},
				"limit": {
					Type:        "number",
					Description: "Maximum number of items to return",
				},
			},
		},
	}, makeGetHistoryHandler(client))

	// log_watch - log a watch
	s.RegisterTool(Tool{
		Name:        "log_watch",
		Description: "Log a single episode or movie as watched. Accepts ISO 8601 dates. If no date provided, uses current time.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"type": {
					Type:        "string",
					Description: "Content type",
					Enum:        []string{"episode", "movie"},
				},
				"showName": {
					Type:        "string",
					Description: "Show name (required for episodes)",
				},
				"season": {
					Type:        "number",
					Description: "Season number (required for episodes)",
				},
				"episode": {
					Type:        "number",
					Description: "Episode number (required for episodes)",
				},
				"movieName": {
					Type:        "string",
					Description: "Movie name (required for movies)",
				},
				"watchedAt": {
					Type:        "string",
					Description: "When it was watched. ISO 8601 format",
				},
			},
			Required: []string{"type"},
		},
	}, makeLogWatchHandler(client))
}

// Handler factories

func makeAuthenticateHandler(client *trakt.Client) ToolHandler {
	return func(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
		if !client.IsConfigured() {
			return ToolCallResult{
				Content: []Content{TextContent("Error: TRAKT_CLIENT_ID and TRAKT_CLIENT_SECRET environment variables must be set")},
				IsError: true,
			}, nil
		}

		code, err := client.GetDeviceCode(ctx)
		if err != nil {
			return ErrorContent(err), nil
		}

		msg := fmt.Sprintf(`🔐 **Trakt Authentication**

Please visit: %s
Enter code: **%s**

The code expires in %d seconds.

After authorizing, the access token will be displayed. Set it as TRAKT_ACCESS_TOKEN environment variable.`,
			code.VerificationURL, code.UserCode, code.ExpiresIn)

		return ToolCallResult{
			Content: []Content{TextContent(msg)},
		}, nil
	}
}

func makeSearchHandler(client *trakt.Client) ToolHandler {
	type searchArgs struct {
		Query string `json:"query"`
		Type  string `json:"type"`
	}

	return func(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
		var a searchArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return ErrorContent(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if a.Query == "" {
			return ToolCallResult{
				Content: []Content{TextContent("Error: query is required")},
				IsError: true,
			}, nil
		}

		results, err := client.Search(ctx, a.Query, a.Type)
		if err != nil {
			return ErrorContent(err), nil
		}

		if len(results) == 0 {
			return ToolCallResult{
				Content: []Content{TextContent(fmt.Sprintf("No results found for: %s", a.Query))},
			}, nil
		}

		// Format results
		var output string
		for i, r := range results {
			if i >= 10 {
				output += fmt.Sprintf("\n... and %d more results", len(results)-10)
				break
			}

			switch r.Type {
			case "show":
				if r.Show != nil {
					output += fmt.Sprintf("📺 **%s** (%d) - Trakt ID: %d\n",
						r.Show.Title, r.Show.Year, r.Show.IDs.Trakt)
				}
			case "movie":
				if r.Movie != nil {
					output += fmt.Sprintf("🎬 **%s** (%d) - Trakt ID: %d\n",
						r.Movie.Title, r.Movie.Year, r.Movie.IDs.Trakt)
				}
			}
		}

		return ToolCallResult{
			Content: []Content{TextContent(output)},
		}, nil
	}
}

func makeGetHistoryHandler(client *trakt.Client) ToolHandler {
	type historyArgs struct {
		Type  string `json:"type"`
		Limit int    `json:"limit"`
	}

	return func(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
		if !client.IsAuthenticated() {
			return ToolCallResult{
				Content: []Content{TextContent("Error: Not authenticated. Use the authenticate tool first.")},
				IsError: true,
			}, nil
		}

		var a historyArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return ErrorContent(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if a.Limit <= 0 {
			a.Limit = 10
		}

		history, err := client.GetHistory(ctx, a.Type, a.Limit)
		if err != nil {
			return ErrorContent(err), nil
		}

		if len(history) == 0 {
			return ToolCallResult{
				Content: []Content{TextContent("No watch history found.")},
			}, nil
		}

		var output string
		for _, h := range history {
			switch h.Type {
			case "episode":
				if h.Show != nil && h.Episode != nil {
					output += fmt.Sprintf("📺 %s S%02dE%02d - %s (%s)\n",
						h.Show.Title, h.Episode.Season, h.Episode.Number,
						h.Episode.Title, h.WatchedAt.Format("2006-01-02"))
				}
			case "movie":
				if h.Movie != nil {
					output += fmt.Sprintf("🎬 %s (%s)\n",
						h.Movie.Title, h.WatchedAt.Format("2006-01-02"))
				}
			}
		}

		return ToolCallResult{
			Content: []Content{TextContent(output)},
		}, nil
	}
}

func makeLogWatchHandler(client *trakt.Client) ToolHandler {
	type logWatchArgs struct {
		Type      string `json:"type"`
		ShowName  string `json:"showName"`
		Season    int    `json:"season"`
		Episode   int    `json:"episode"`
		MovieName string `json:"movieName"`
		WatchedAt string `json:"watchedAt"`
	}

	return func(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
		if !client.IsAuthenticated() {
			return ToolCallResult{
				Content: []Content{TextContent("Error: Not authenticated. Use the authenticate tool first.")},
				IsError: true,
			}, nil
		}

		var a logWatchArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return ErrorContent(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// For now, return a stub - full implementation requires search + sync
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("🚧 log_watch for %s not yet fully implemented", a.Type))},
		}, nil
	}
}
