package mcp

import (
	"context"
	"encoding/json"
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

	// Episode with invalid season/episode numbers
	result, err := handler(context.Background(), json.RawMessage(`{"type":"episode","showName":"Test","season":0,"episode":0}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid season/episode")
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
