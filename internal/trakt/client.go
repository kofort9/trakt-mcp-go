package trakt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	BaseURL        = "https://api.trakt.tv"
	APIVersion     = "2"
	DefaultTimeout = 30 * time.Second
)

// APIError represents an error from the Trakt API.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("trakt API error: %s %s returned status %d", e.Method, e.Path, e.StatusCode)
}

// IsAuthError returns true if this is an authentication error.
func (e *APIError) IsAuthError() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}

// IsRateLimited returns true if this is a rate limit error.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// Config holds the Trakt API configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
}

// ConfigFromEnv creates a Config from environment variables.
func ConfigFromEnv() Config {
	return Config{
		ClientID:     os.Getenv("TRAKT_CLIENT_ID"),
		ClientSecret: os.Getenv("TRAKT_CLIENT_SECRET"),
		AccessToken:  os.Getenv("TRAKT_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("TRAKT_REFRESH_TOKEN"),
	}
}

// Client is a Trakt API client.
type Client struct {
	config     Config
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new Trakt API client.
func NewClient(config Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		logger: logger,
	}
}

// IsConfigured returns true if the client has API credentials.
func (c *Client) IsConfigured() bool {
	return c.config.ClientID != ""
}

// IsAuthenticated returns true if the client has an access token.
func (c *Client) IsAuthenticated() bool {
	return c.config.AccessToken != ""
}

// Search searches for shows or movies.
func (c *Client) Search(ctx context.Context, query string, searchType string) ([]SearchResult, error) {
	if searchType == "" {
		searchType = "show,movie"
	}

	params := url.Values{}
	params.Set("query", query)

	path := fmt.Sprintf("/search/%s?%s", searchType, params.Encode())

	var results []SearchResult
	if err := c.get(ctx, path, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// GetHistory retrieves watch history.
func (c *Client) GetHistory(ctx context.Context, historyType string, limit int) ([]HistoryItem, error) {
	path := "/sync/history"
	if historyType != "" {
		path = fmt.Sprintf("/sync/history/%s", historyType)
	}

	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	if len(params) > 0 {
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}

	var history []HistoryItem
	if err := c.get(ctx, path, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// AddToHistory adds items to watch history.
func (c *Client) AddToHistory(ctx context.Context, item WatchedItem) (*SyncResponse, error) {
	var resp SyncResponse
	if err := c.post(ctx, "/sync/history", item, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RemoveFromHistory removes items from watch history.
func (c *Client) RemoveFromHistory(ctx context.Context, item WatchedItem) (*SyncResponse, error) {
	var resp SyncResponse
	if err := c.post(ctx, "/sync/history/remove", item, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDeviceCode initiates device authentication.
func (c *Client) GetDeviceCode(ctx context.Context) (*DeviceCode, error) {
	body := map[string]string{
		"client_id": c.config.ClientID,
	}

	var code DeviceCode
	if err := c.post(ctx, "/oauth/device/code", body, &code); err != nil {
		return nil, err
	}

	return &code, nil
}

// PollForToken polls for OAuth token after device code authorization.
func (c *Client) PollForToken(ctx context.Context, deviceCode string) (*Token, error) {
	body := map[string]string{
		"code":          deviceCode,
		"client_id":     c.config.ClientID,
		"client_secret": c.config.ClientSecret,
	}

	var token Token
	if err := c.post(ctx, "/oauth/device/token", body, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// HTTP helpers

func (c *Client) get(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodGet, path, nil, result)
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trakt-api-version", APIVersion)
	req.Header.Set("trakt-api-key", c.config.ClientID)

	if c.config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	}

	c.logger.Debug("trakt request", "method", method, "path", path)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Log error without sensitive response body details
		c.logger.Error("trakt API error",
			"status", resp.StatusCode,
			"method", method,
			"path", path,
		)
		// Return sanitized error - don't leak response body which may contain tokens
		return &APIError{StatusCode: resp.StatusCode, Method: method, Path: path}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}
