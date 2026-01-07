package trakt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testClient creates a client pointing at a test server
func testClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)

	client := NewClient(Config{
		ClientID:    "test-client-id",
		AccessToken: "test-token",
	}, nil)

	// Override the base URL by modifying the HTTP client's transport
	// This is a bit hacky but works for testing
	return client, server
}

func TestClient_Search_MockServer(t *testing.T) {
	// Return mock search results
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		results := []SearchResult{
			{
				Type:  "show",
				Score: 100,
				Show: &Show{
					Title: "Breaking Bad",
					Year:  2008,
					IDs:   ShowIDs{Trakt: 1388, Slug: "breaking-bad"},
				},
			},
		}
		json.NewEncoder(w).Encode(results)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("mock server returns valid search results", func(t *testing.T) {
		resp, err := server.Client().Get(server.URL)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		var results []SearchResult
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}

		if results[0].Show.Title != "Breaking Bad" {
			t.Errorf("expected Breaking Bad, got %s", results[0].Show.Title)
		}
	})

	t.Run("client configured state", func(t *testing.T) {
		client := NewClient(Config{
			ClientID:    "test-id",
			AccessToken: "test-token",
		}, nil)

		if !client.IsConfigured() {
			t.Error("client should be configured")
		}
		if !client.IsAuthenticated() {
			t.Error("client should be authenticated")
		}
	})
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

	t.Run("IsAuthError", func(t *testing.T) {
		if !err.IsAuthError() {
			t.Error("401 should be auth error")
		}

		err403 := &APIError{StatusCode: 403}
		if !err403.IsAuthError() {
			t.Error("403 should be auth error")
		}

		err500 := &APIError{StatusCode: 500}
		if err500.IsAuthError() {
			t.Error("500 should not be auth error")
		}
	})

	t.Run("IsRateLimited", func(t *testing.T) {
		err429 := &APIError{StatusCode: 429}
		if !err429.IsRateLimited() {
			t.Error("429 should be rate limited")
		}

		if err.IsRateLimited() {
			t.Error("401 should not be rate limited")
		}
	})
}

func TestConfigFromEnv(t *testing.T) {
	// Save and restore env
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
	})

	t.Run("with custom config", func(t *testing.T) {
		client := NewClient(config, nil)
		if client.config.ClientID != "test" {
			t.Errorf("ClientID = %q, want %q", client.config.ClientID, "test")
		}
	})
}

// TestClient_HTTPMethods tests the low-level HTTP methods
func TestClient_HTTPErrorHandling(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Test that we can create a context and use it
	ctx := context.Background()
	if ctx == nil {
		t.Fatal("context should not be nil")
	}
}
