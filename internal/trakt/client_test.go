package trakt

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient creates a client with a mock server
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := NewClient(Config{
		ClientID:    "test-client-id",
		AccessToken: "test-token",
	}, nil)
	client.baseURL = server.URL

	return client
}

func TestClient_Search(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("trakt-api-key") != "test-client-id" {
			t.Error("missing or wrong API key header")
		}

		results := []SearchResult{
			{
				Type:  "show",
				Score: 1000,
				Show: &Show{
					Title: "Breaking Bad",
					Year:  2008,
					IDs:   ShowIDs{Trakt: 1388, Slug: "breaking-bad"},
				},
			},
			{
				Type:  "movie",
				Score: 500,
				Movie: &Movie{
					Title: "Breaking Bad Movie",
					Year:  2019,
					IDs:   MovieIDs{Trakt: 12345, Slug: "el-camino"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	})

	client := newTestClient(t, handler)

	t.Run("search shows and movies", func(t *testing.T) {
		results, err := client.Search(context.Background(), "breaking bad", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		if results[0].Show.Title != "Breaking Bad" {
			t.Errorf("expected Breaking Bad, got %s", results[0].Show.Title)
		}
	})

	t.Run("search with type filter", func(t *testing.T) {
		results, err := client.Search(context.Background(), "breaking", "show")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if results == nil {
			t.Error("expected results, got nil")
		}
	})
}

func TestClient_GetHistory(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing or wrong Authorization header")
		}

		history := []HistoryItem{
			{
				Type: "episode",
				Show: &Show{Title: "Breaking Bad", Year: 2008},
				Episode: &Episode{
					Title:  "Pilot",
					Season: 1,
					Number: 1,
					IDs:    EpisodeIDs{Trakt: 62085},
				},
			},
			{
				Type:  "movie",
				Movie: &Movie{Title: "Inception", Year: 2010},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(history)
	})

	client := newTestClient(t, handler)

	t.Run("get all history", func(t *testing.T) {
		history, err := client.GetHistory(context.Background(), "", 10)
		if err != nil {
			t.Fatalf("GetHistory failed: %v", err)
		}

		if len(history) != 2 {
			t.Errorf("expected 2 items, got %d", len(history))
		}
	})

	t.Run("get shows history", func(t *testing.T) {
		history, err := client.GetHistory(context.Background(), "shows", 5)
		if err != nil {
			t.Fatalf("GetHistory failed: %v", err)
		}
		if history == nil {
			t.Error("expected history, got nil")
		}
	})
}

func TestClient_AddToHistory(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Read and verify request body
		body, _ := io.ReadAll(r.Body)
		var item WatchedItem
		if err := json.Unmarshal(body, &item); err != nil {
			t.Errorf("failed to parse request body: %v", err)
		}

		resp := SyncResponse{
			Added: struct {
				Movies   int `json:"movies"`
				Episodes int `json:"episodes"`
			}{Episodes: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	client := newTestClient(t, handler)

	item := WatchedItem{
		Episodes: []Episode{{IDs: EpisodeIDs{Trakt: 12345}}},
	}

	resp, err := client.AddToHistory(context.Background(), item)
	if err != nil {
		t.Fatalf("AddToHistory failed: %v", err)
	}

	if resp.Added.Episodes != 1 {
		t.Errorf("expected 1 episode added, got %d", resp.Added.Episodes)
	}
}

func TestClient_RemoveFromHistory(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sync/history/remove" {
			t.Errorf("expected /sync/history/remove, got %s", r.URL.Path)
		}

		resp := SyncResponse{
			Deleted: struct {
				Movies   int `json:"movies"`
				Episodes int `json:"episodes"`
			}{Episodes: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	client := newTestClient(t, handler)

	item := WatchedItem{
		Episodes: []Episode{{IDs: EpisodeIDs{Trakt: 12345}}},
	}

	resp, err := client.RemoveFromHistory(context.Background(), item)
	if err != nil {
		t.Fatalf("RemoveFromHistory failed: %v", err)
	}

	if resp.Deleted.Episodes != 1 {
		t.Errorf("expected 1 episode deleted, got %d", resp.Deleted.Episodes)
	}
}

func TestClient_GetShow(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shows/1388" {
			t.Errorf("expected /shows/1388, got %s", r.URL.Path)
		}

		show := Show{
			Title: "Breaking Bad",
			Year:  2008,
			IDs:   ShowIDs{Trakt: 1388, Slug: "breaking-bad", IMDB: "tt0903747"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(show)
	})

	client := newTestClient(t, handler)

	show, err := client.GetShow(context.Background(), "1388")
	if err != nil {
		t.Fatalf("GetShow failed: %v", err)
	}

	if show.Title != "Breaking Bad" {
		t.Errorf("expected Breaking Bad, got %s", show.Title)
	}
	if show.IDs.Trakt != 1388 {
		t.Errorf("expected Trakt ID 1388, got %d", show.IDs.Trakt)
	}
}

func TestClient_GetEpisode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shows/1388/seasons/1/episodes/1" {
			t.Errorf("expected /shows/1388/seasons/1/episodes/1, got %s", r.URL.Path)
		}

		episode := Episode{
			Title:  "Pilot",
			Season: 1,
			Number: 1,
			IDs:    EpisodeIDs{Trakt: 62085, IMDB: "tt0959621"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(episode)
	})

	client := newTestClient(t, handler)

	ep, err := client.GetEpisode(context.Background(), "1388", 1, 1)
	if err != nil {
		t.Fatalf("GetEpisode failed: %v", err)
	}

	if ep.Title != "Pilot" {
		t.Errorf("expected Pilot, got %s", ep.Title)
	}
	if ep.Season != 1 || ep.Number != 1 {
		t.Errorf("expected S01E01, got S%02dE%02d", ep.Season, ep.Number)
	}
}

func TestClient_GetMovie(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movies/inception-2010" {
			t.Errorf("expected /movies/inception-2010, got %s", r.URL.Path)
		}

		movie := Movie{
			Title: "Inception",
			Year:  2010,
			IDs:   MovieIDs{Trakt: 16662, Slug: "inception-2010", IMDB: "tt1375666"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(movie)
	})

	client := newTestClient(t, handler)

	movie, err := client.GetMovie(context.Background(), "inception-2010")
	if err != nil {
		t.Fatalf("GetMovie failed: %v", err)
	}

	if movie.Title != "Inception" {
		t.Errorf("expected Inception, got %s", movie.Title)
	}
}

func TestClient_GetDeviceCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/oauth/device/code" {
			t.Errorf("expected /oauth/device/code, got %s", r.URL.Path)
		}

		code := DeviceCode{
			DeviceCode:      "device123",
			UserCode:        "ABCD1234",
			VerificationURL: "https://trakt.tv/activate",
			ExpiresIn:       600,
			Interval:        5,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(code)
	})

	client := newTestClient(t, handler)

	code, err := client.GetDeviceCode(context.Background())
	if err != nil {
		t.Fatalf("GetDeviceCode failed: %v", err)
	}

	if code.UserCode != "ABCD1234" {
		t.Errorf("expected ABCD1234, got %s", code.UserCode)
	}
	if code.ExpiresIn != 600 {
		t.Errorf("expected 600, got %d", code.ExpiresIn)
	}
}

func TestClient_PollForToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/device/token" {
			t.Errorf("expected /oauth/device/token, got %s", r.URL.Path)
		}

		token := Token{
			AccessToken:  "access123",
			RefreshToken: "refresh456",
			TokenType:    "Bearer",
			ExpiresIn:    7776000,
			CreatedAt:    1704067200,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(token)
	})

	client := newTestClient(t, handler)

	token, err := client.PollForToken(context.Background(), "device123")
	if err != nil {
		t.Fatalf("PollForToken failed: %v", err)
	}

	if token.AccessToken != "access123" {
		t.Errorf("expected access123, got %s", token.AccessToken)
	}
}

func TestClient_HTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantAuth   bool
		wantRate   bool
	}{
		{"unauthorized", 401, true, false},
		{"forbidden", 403, true, false},
		{"rate limited", 429, false, true},
		{"server error", 500, false, false},
		{"not found", 404, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"error":"test error"}`))
			})

			client := newTestClient(t, handler)

			_, err := client.Search(context.Background(), "test", "")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected APIError, got %T", err)
			}

			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, apiErr.StatusCode)
			}

			if apiErr.IsAuthError() != tt.wantAuth {
				t.Errorf("IsAuthError() = %v, want %v", apiErr.IsAuthError(), tt.wantAuth)
			}

			if apiErr.IsRateLimited() != tt.wantRate {
				t.Errorf("IsRateLimited() = %v, want %v", apiErr.IsRateLimited(), tt.wantRate)
			}
		})
	}
}

func TestClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name:     "configured with client ID",
			config:   Config{ClientID: "test-id"},
			expected: true,
		},
		{
			name:     "not configured without client ID",
			config:   Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config, nil)
			if got := client.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClient_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name:     "authenticated with access token",
			config:   Config{AccessToken: "test-token"},
			expected: true,
		},
		{
			name:     "not authenticated without access token",
			config:   Config{ClientID: "test-id"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config, nil)
			if got := client.IsAuthenticated(); got != tt.expected {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 401,
		Method:     "GET",
		Path:       "/sync/history",
	}

	t.Run("Error message format", func(t *testing.T) {
		expected := "trakt API error: GET /sync/history returned status 401"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("TRAKT_CLIENT_ID", "test-id")
	t.Setenv("TRAKT_CLIENT_SECRET", "test-secret")
	t.Setenv("TRAKT_ACCESS_TOKEN", "test-access")
	t.Setenv("TRAKT_REFRESH_TOKEN", "test-refresh")

	config := ConfigFromEnv()

	if config.ClientID != "test-id" {
		t.Errorf("ClientID = %q, want %q", config.ClientID, "test-id")
	}
	if config.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %q, want %q", config.ClientSecret, "test-secret")
	}
	if config.AccessToken != "test-access" {
		t.Errorf("AccessToken = %q, want %q", config.AccessToken, "test-access")
	}
	if config.RefreshToken != "test-refresh" {
		t.Errorf("RefreshToken = %q, want %q", config.RefreshToken, "test-refresh")
	}
}

func TestClient_NewClient(t *testing.T) {
	config := Config{ClientID: "test"}

	t.Run("with nil logger", func(t *testing.T) {
		client := NewClient(config, nil)
		if client == nil {
			t.Fatal("NewClient returned nil")
		}
		if client.logger == nil {
			t.Error("logger should be set to default")
		}
		if client.baseURL != BaseURL {
			t.Errorf("baseURL = %q, want %q", client.baseURL, BaseURL)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		client := NewClient(config, nil)
		if client.config.ClientID != "test" {
			t.Errorf("ClientID = %q, want %q", client.config.ClientID, "test")
		}
	})
}
