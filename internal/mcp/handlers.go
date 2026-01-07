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

		switch a.Type {
		case "episode":
			return logEpisode(ctx, client, a.ShowName, a.Season, a.Episode, a.WatchedAt)
		case "movie":
			return logMovie(ctx, client, a.MovieName, a.WatchedAt)
		default:
			return ToolCallResult{
				Content: []Content{TextContent("Error: type must be 'episode' or 'movie'")},
				IsError: true,
			}, nil
		}
	}
}

func logEpisode(ctx context.Context, client *trakt.Client, showName string, season, episode int, watchedAt string) (ToolCallResult, error) {
	if showName == "" {
		return ToolCallResult{
			Content: []Content{TextContent("Error: showName is required for episodes")},
			IsError: true,
		}, nil
	}
	// Season 0 is valid (specials), but episode must be positive
	if season < 0 || episode <= 0 {
		return ToolCallResult{
			Content: []Content{TextContent("Error: season must be >= 0 and episode must be positive")},
			IsError: true,
		}, nil
	}

	// Search for the show
	results, err := client.Search(ctx, showName, "show")
	if err != nil {
		return ErrorContent(err), nil
	}
	if len(results) == 0 || results[0].Show == nil {
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("No show found for: %s", showName))},
			IsError: true,
		}, nil
	}

	// Check for ambiguous results
	if len(results) > 1 && results[0].Score < 1000 {
		// Multiple matches with no clear winner - ask user to disambiguate
		var msg string
		msg = fmt.Sprintf("Multiple shows found for '%s'. Please be more specific or use the year:\n", showName)
		for i, r := range results {
			if i >= 5 {
				msg += fmt.Sprintf("... and %d more\n", len(results)-5)
				break
			}
			if r.Show != nil {
				msg += fmt.Sprintf("• %s (%d) - Trakt ID: %d\n", r.Show.Title, r.Show.Year, r.Show.IDs.Trakt)
			}
		}
		return ToolCallResult{
			Content: []Content{TextContent(msg)},
			IsError: true,
		}, nil
	}

	show := results[0].Show

	// Get the episode to verify it exists and get its ID
	ep, err := client.GetEpisode(ctx, fmt.Sprintf("%d", show.IDs.Trakt), season, episode)
	if err != nil {
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("Episode S%02dE%02d not found for %s", season, episode, show.Title))},
			IsError: true,
		}, nil
	}

	// Sync to history
	item := trakt.WatchedItem{
		WatchedAt: watchedAt,
		Episodes: []trakt.Episode{
			{
				IDs: trakt.EpisodeIDs{Trakt: ep.IDs.Trakt},
			},
		},
	}

	resp, err := client.AddToHistory(ctx, item)
	if err != nil {
		return ErrorContent(err), nil
	}

	if resp.Added.Episodes > 0 {
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("✅ Logged: **%s** S%02dE%02d - %s",
				show.Title, season, episode, ep.Title))},
		}, nil
	}

	if resp.Existing.Episodes > 0 {
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("ℹ️ Already watched: **%s** S%02dE%02d - %s",
				show.Title, season, episode, ep.Title))},
		}, nil
	}

	return ToolCallResult{
		Content: []Content{TextContent("⚠️ Episode was not added (unknown reason)")},
	}, nil
}

func logMovie(ctx context.Context, client *trakt.Client, movieName string, watchedAt string) (ToolCallResult, error) {
	if movieName == "" {
		return ToolCallResult{
			Content: []Content{TextContent("Error: movieName is required for movies")},
			IsError: true,
		}, nil
	}

	// Search for the movie
	results, err := client.Search(ctx, movieName, "movie")
	if err != nil {
		return ErrorContent(err), nil
	}
	if len(results) == 0 || results[0].Movie == nil {
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("No movie found for: %s", movieName))},
			IsError: true,
		}, nil
	}

	// Check for ambiguous results
	if len(results) > 1 && results[0].Score < 1000 {
		var msg string
		msg = fmt.Sprintf("Multiple movies found for '%s'. Please be more specific or use the year:\n", movieName)
		for i, r := range results {
			if i >= 5 {
				msg += fmt.Sprintf("... and %d more\n", len(results)-5)
				break
			}
			if r.Movie != nil {
				msg += fmt.Sprintf("• %s (%d) - Trakt ID: %d\n", r.Movie.Title, r.Movie.Year, r.Movie.IDs.Trakt)
			}
		}
		return ToolCallResult{
			Content: []Content{TextContent(msg)},
			IsError: true,
		}, nil
	}

	movie := results[0].Movie

	// Sync to history
	item := trakt.WatchedItem{
		WatchedAt: watchedAt,
		Movies: []trakt.Movie{
			{
				IDs: trakt.MovieIDs{Trakt: movie.IDs.Trakt},
			},
		},
	}

	resp, err := client.AddToHistory(ctx, item)
	if err != nil {
		return ErrorContent(err), nil
	}

	if resp.Added.Movies > 0 {
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("✅ Logged: **%s** (%d)",
				movie.Title, movie.Year))},
		}, nil
	}

	if resp.Existing.Movies > 0 {
		return ToolCallResult{
			Content: []Content{TextContent(fmt.Sprintf("ℹ️ Already watched: **%s** (%d)",
				movie.Title, movie.Year))},
		}, nil
	}

	return ToolCallResult{
		Content: []Content{TextContent("⚠️ Movie was not added (unknown reason)")},
	}, nil
}
