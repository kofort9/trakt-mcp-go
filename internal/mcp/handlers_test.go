package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kofifort/trakt-mcp-go/internal/trakt"
)

// mockTraktClient is a test double for the trakt.Client
// Since we can't easily mock the real client without interfaces,
// we test the handler logic with a real but unconfigured client

func TestRegisterTools(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{}, nil)

	RegisterTools(server, client)

	// Verify all expected tools are registered
	expectedTools := []string{"authenticate", "search_show", "get_history", "log_watch"}

	server.mu.RLock()
	defer server.mu.RUnlock()

	for _, name := range expectedTools {
		if _, ok := server.tools[name]; !ok {
			t.Errorf("tool %q not registered", name)
		}
		if _, ok := server.handlers[name]; !ok {
			t.Errorf("handler for %q not registered", name)
		}
	}

	if len(server.tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(server.tools))
	}
}

func TestAuthenticateHandler_NotConfigured(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{}, nil) // No client ID

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["authenticate"]
	server.mu.RUnlock()

	result, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for unconfigured client")
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	if result.Content[0].Text == "" {
		t.Error("expected error message")
	}
}

func TestSearchHandler_EmptyQuery(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test"}, nil)

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["search_show"]
	server.mu.RUnlock()

	// Empty query should return error
	result, err := handler(context.Background(), json.RawMessage(`{"query":""}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for empty query")
	}
}

func TestSearchHandler_InvalidArgs(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test"}, nil)

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["search_show"]
	server.mu.RUnlock()

	// Invalid JSON should return error
	result, err := handler(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid JSON")
	}
}

func TestGetHistoryHandler_NotAuthenticated(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test"}, nil) // No access token

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["get_history"]
	server.mu.RUnlock()

	result, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for unauthenticated client")
	}

	if len(result.Content) == 0 || result.Content[0].Text == "" {
		t.Fatal("expected error message about authentication")
	}
}

func TestLogWatchHandler_NotAuthenticated(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test"}, nil)

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["log_watch"]
	server.mu.RUnlock()

	result, err := handler(context.Background(), json.RawMessage(`{"type":"episode"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for unauthenticated client")
	}
}

func TestLogWatchHandler_InvalidType(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test", AccessToken: "token"}, nil)

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["log_watch"]
	server.mu.RUnlock()

	result, err := handler(context.Background(), json.RawMessage(`{"type":"invalid"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid type")
	}
}

func TestLogWatchHandler_MissingShowName(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test", AccessToken: "token"}, nil)

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["log_watch"]
	server.mu.RUnlock()

	// Episode without showName
	result, err := handler(context.Background(), json.RawMessage(`{"type":"episode","season":1,"episode":1}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing showName")
	}
}

func TestLogWatchHandler_MissingMovieName(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test", AccessToken: "token"}, nil)

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["log_watch"]
	server.mu.RUnlock()

	// Movie without movieName
	result, err := handler(context.Background(), json.RawMessage(`{"type":"movie"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing movieName")
	}
}

func TestLogWatchHandler_InvalidSeasonEpisode(t *testing.T) {
	server := NewServer(nil)
	client := trakt.NewClient(trakt.Config{ClientID: "test", AccessToken: "token"}, nil)

	RegisterTools(server, client)

	server.mu.RLock()
	handler := server.handlers["log_watch"]
	server.mu.RUnlock()

	// Test cases for invalid season/episode validation
	// Note: season 0 is valid (specials), but episode must be > 0
	tests := []struct {
		name string
		args string
	}{
		{"negative season", `{"type":"episode","showName":"Test","season":-1,"episode":1}`},
		{"zero episode", `{"type":"episode","showName":"Test","season":1,"episode":0}`},
		{"negative episode", `{"type":"episode","showName":"Test","season":1,"episode":-1}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := handler(context.Background(), json.RawMessage(tc.args))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.IsError {
				t.Error("expected error result for invalid season/episode")
			}
		})
	}
}

func TestTextContent(t *testing.T) {
	content := TextContent("test message")

	if content.Type != "text" {
		t.Errorf("Type = %q, want %q", content.Type, "text")
	}

	if content.Text != "test message" {
		t.Errorf("Text = %q, want %q", content.Text, "test message")
	}
}

func TestErrorContent(t *testing.T) {
	result := ErrorContent(context.Canceled)

	if !result.IsError {
		t.Error("IsError should be true")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Error("content type should be text")
	}
}

// Integration tests with mock Trakt server

func newMockTraktServer(t *testing.T, handler http.Handler) (*httptest.Server, *trakt.Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := trakt.NewClient(trakt.Config{
		ClientID:    "test-client-id",
		AccessToken: "test-token",
	}, nil)
	client.SetBaseURL(server.URL)

	return server, client
}

func TestSearchHandler_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		results := []trakt.SearchResult{
			{
				Type:  "show",
				Score: 1000,
				Show: &trakt.Show{
					Title: "Breaking Bad",
					Year:  2008,
					IDs:   trakt.ShowIDs{Trakt: 1388},
				},
			},
			{
				Type:  "movie",
				Score: 500,
				Movie: &trakt.Movie{
					Title: "Breaking Bad Movie",
					Year:  2019,
					IDs:   trakt.MovieIDs{Trakt: 12345},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	searchHandler := server.handlers["search_show"]
	server.mu.RUnlock()

	result, err := searchHandler(context.Background(), json.RawMessage(`{"query":"breaking bad"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Breaking Bad") {
		t.Errorf("expected 'Breaking Bad' in result, got: %s", text)
	}
	if !strings.Contains(text, "1388") {
		t.Errorf("expected Trakt ID '1388' in result, got: %s", text)
	}
}

func TestSearchHandler_NoResults(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]trakt.SearchResult{})
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	searchHandler := server.handlers["search_show"]
	server.mu.RUnlock()

	result, err := searchHandler(context.Background(), json.RawMessage(`{"query":"nonexistent show xyz"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("no results should not be an error")
	}

	if !strings.Contains(result.Content[0].Text, "No results found") {
		t.Errorf("expected 'No results found', got: %s", result.Content[0].Text)
	}
}

func TestGetHistoryHandler_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		history := []trakt.HistoryItem{
			{
				Type: "episode",
				Show: &trakt.Show{Title: "Breaking Bad"},
				Episode: &trakt.Episode{
					Title:  "Pilot",
					Season: 1,
					Number: 1,
				},
			},
			{
				Type:  "movie",
				Movie: &trakt.Movie{Title: "Inception"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(history)
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	historyHandler := server.handlers["get_history"]
	server.mu.RUnlock()

	result, err := historyHandler(context.Background(), json.RawMessage(`{"limit":10}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Breaking Bad") {
		t.Errorf("expected 'Breaking Bad' in result, got: %s", text)
	}
	if !strings.Contains(text, "S01E01") {
		t.Errorf("expected 'S01E01' in result, got: %s", text)
	}
}

func TestGetHistoryHandler_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]trakt.HistoryItem{})
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	historyHandler := server.handlers["get_history"]
	server.mu.RUnlock()

	result, err := historyHandler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("empty history should not be an error")
	}

	if !strings.Contains(result.Content[0].Text, "No watch history") {
		t.Errorf("expected 'No watch history', got: %s", result.Content[0].Text)
	}
}

func TestGetHistoryHandler_InvalidArgs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	historyHandler := server.handlers["get_history"]
	server.mu.RUnlock()

	result, err := historyHandler(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestAuthenticateHandler_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := trakt.DeviceCode{
			DeviceCode:      "device123",
			UserCode:        "ABCD1234",
			VerificationURL: "https://trakt.tv/activate",
			ExpiresIn:       600,
			Interval:        5,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(code)
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	authHandler := server.handlers["authenticate"]
	server.mu.RUnlock()

	result, err := authHandler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "ABCD1234") {
		t.Errorf("expected user code 'ABCD1234' in result, got: %s", text)
	}
	if !strings.Contains(text, "trakt.tv/activate") {
		t.Errorf("expected verification URL in result, got: %s", text)
	}
}

func TestLogWatchHandler_EpisodeSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasPrefix(r.URL.Path, "/search"):
			// Search for show
			results := []trakt.SearchResult{
				{
					Type:  "show",
					Score: 1000,
					Show: &trakt.Show{
						Title: "Breaking Bad",
						Year:  2008,
						IDs:   trakt.ShowIDs{Trakt: 1388},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(results)

		case strings.Contains(r.URL.Path, "/episodes/"):
			// Get episode
			ep := trakt.Episode{
				Title:  "Pilot",
				Season: 1,
				Number: 1,
				IDs:    trakt.EpisodeIDs{Trakt: 62085},
			}
			_ = json.NewEncoder(w).Encode(ep)

		case r.URL.Path == "/sync/history":
			// Add to history
			resp := trakt.SyncResponse{
				Added: struct {
					Movies   int `json:"movies"`
					Episodes int `json:"episodes"`
				}{Episodes: 1},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	logHandler := server.handlers["log_watch"]
	server.mu.RUnlock()

	result, err := logHandler(context.Background(), json.RawMessage(`{
		"type": "episode",
		"showName": "Breaking Bad",
		"season": 1,
		"episode": 1
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Logged") || !strings.Contains(text, "Breaking Bad") {
		t.Errorf("expected success message with show name, got: %s", text)
	}
}

func TestLogWatchHandler_MovieSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasPrefix(r.URL.Path, "/search"):
			results := []trakt.SearchResult{
				{
					Type:  "movie",
					Score: 1000,
					Movie: &trakt.Movie{
						Title: "Inception",
						Year:  2010,
						IDs:   trakt.MovieIDs{Trakt: 16662},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(results)

		case r.URL.Path == "/sync/history":
			resp := trakt.SyncResponse{
				Added: struct {
					Movies   int `json:"movies"`
					Episodes int `json:"episodes"`
				}{Movies: 1},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	logHandler := server.handlers["log_watch"]
	server.mu.RUnlock()

	result, err := logHandler(context.Background(), json.RawMessage(`{
		"type": "movie",
		"movieName": "Inception"
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Logged") || !strings.Contains(text, "Inception") {
		t.Errorf("expected success message with movie name, got: %s", text)
	}
}

func TestLogWatchHandler_EpisodeAlreadyWatched(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasPrefix(r.URL.Path, "/search"):
			results := []trakt.SearchResult{
				{
					Type:  "show",
					Score: 1000,
					Show: &trakt.Show{
						Title: "Breaking Bad",
						Year:  2008,
						IDs:   trakt.ShowIDs{Trakt: 1388},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(results)

		case strings.Contains(r.URL.Path, "/episodes/"):
			ep := trakt.Episode{
				Title:  "Pilot",
				Season: 1,
				Number: 1,
				IDs:    trakt.EpisodeIDs{Trakt: 62085},
			}
			_ = json.NewEncoder(w).Encode(ep)

		case r.URL.Path == "/sync/history":
			// Already watched - existing count > 0
			resp := trakt.SyncResponse{
				Existing: struct {
					Movies   int `json:"movies"`
					Episodes int `json:"episodes"`
				}{Episodes: 1},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	logHandler := server.handlers["log_watch"]
	server.mu.RUnlock()

	result, err := logHandler(context.Background(), json.RawMessage(`{
		"type": "episode",
		"showName": "Breaking Bad",
		"season": 1,
		"episode": 1
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("already watched should not be an error")
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Already watched") {
		t.Errorf("expected 'Already watched' message, got: %s", text)
	}
}

func TestLogWatchHandler_ShowNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]trakt.SearchResult{})
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	logHandler := server.handlers["log_watch"]
	server.mu.RUnlock()

	result, err := logHandler(context.Background(), json.RawMessage(`{
		"type": "episode",
		"showName": "Nonexistent Show XYZ",
		"season": 1,
		"episode": 1
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error for show not found")
	}

	if !strings.Contains(result.Content[0].Text, "No show found") {
		t.Errorf("expected 'No show found' message, got: %s", result.Content[0].Text)
	}
}

func TestLogWatchHandler_AmbiguousShow(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Multiple results with low scores - ambiguous
		results := []trakt.SearchResult{
			{
				Type:  "show",
				Score: 500,
				Show: &trakt.Show{
					Title: "Lost",
					Year:  2004,
					IDs:   trakt.ShowIDs{Trakt: 73},
				},
			},
			{
				Type:  "show",
				Score: 450,
				Show: &trakt.Show{
					Title: "Lost in Space",
					Year:  2018,
					IDs:   trakt.ShowIDs{Trakt: 117523},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(results)
	})

	_, client := newMockTraktServer(t, handler)

	server := NewServer(nil)
	RegisterTools(server, client)

	server.mu.RLock()
	logHandler := server.handlers["log_watch"]
	server.mu.RUnlock()

	result, err := logHandler(context.Background(), json.RawMessage(`{
		"type": "episode",
		"showName": "Lost",
		"season": 1,
		"episode": 1
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error for ambiguous show")
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Multiple shows found") {
		t.Errorf("expected disambiguation message, got: %s", text)
	}
}
