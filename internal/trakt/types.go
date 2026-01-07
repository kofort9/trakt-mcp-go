// Package trakt provides a client for the Trakt.tv API.
package trakt

import "time"

// Show represents a TV show from Trakt.
type Show struct {
	Title string   `json:"title"`
	Year  int      `json:"year"`
	IDs   ShowIDs  `json:"ids"`
}

// ShowIDs contains various IDs for a show.
type ShowIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	TVDB  int    `json:"tvdb"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}

// Movie represents a movie from Trakt.
type Movie struct {
	Title string   `json:"title"`
	Year  int      `json:"year"`
	IDs   MovieIDs `json:"ids"`
}

// MovieIDs contains various IDs for a movie.
type MovieIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}

// Episode represents a TV episode from Trakt.
type Episode struct {
	Season  int        `json:"season"`
	Number  int        `json:"number"`
	Title   string     `json:"title"`
	IDs     EpisodeIDs `json:"ids"`
}

// EpisodeIDs contains various IDs for an episode.
type EpisodeIDs struct {
	Trakt int    `json:"trakt"`
	TVDB  int    `json:"tvdb"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}

// SearchResult represents a search result from Trakt.
type SearchResult struct {
	Type  string  `json:"type"` // "show", "movie", "episode"
	Score float64 `json:"score"`
	Show  *Show   `json:"show,omitempty"`
	Movie *Movie  `json:"movie,omitempty"`
}

// HistoryItem represents an item in the watch history.
type HistoryItem struct {
	ID        int64     `json:"id"`
	WatchedAt time.Time `json:"watched_at"`
	Action    string    `json:"action"` // "watch", "scrobble"
	Type      string    `json:"type"`   // "episode", "movie"
	Episode   *Episode  `json:"episode,omitempty"`
	Show      *Show     `json:"show,omitempty"`
	Movie     *Movie    `json:"movie,omitempty"`
}

// WatchedItem represents an item to sync as watched.
type WatchedItem struct {
	WatchedAt string    `json:"watched_at,omitempty"` // ISO 8601
	Movies    []Movie   `json:"movies,omitempty"`
	Shows     []Show    `json:"shows,omitempty"`
	Episodes  []Episode `json:"episodes,omitempty"`
}

// SyncResponse represents the response from a sync operation.
type SyncResponse struct {
	Added    SyncStats `json:"added"`
	Deleted  SyncStats `json:"deleted"`
	Existing SyncStats `json:"existing"`
	NotFound NotFound  `json:"not_found"`
}

// SyncStats contains counts from sync operations.
type SyncStats struct {
	Movies   int `json:"movies"`
	Episodes int `json:"episodes"`
}

// NotFound contains items that weren't found during sync.
type NotFound struct {
	Movies   []Movie   `json:"movies"`
	Shows    []Show    `json:"shows"`
	Episodes []Episode `json:"episodes"`
}

// DeviceCode represents the OAuth device code response.
type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// Token represents an OAuth token.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
}

// Rating represents a rating for content.
type Rating struct {
	Rating   int       `json:"rating"` // 1-10
	RatedAt  time.Time `json:"rated_at"`
}
