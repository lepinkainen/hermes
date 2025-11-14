// Package tmdb provides a client for TheMovieDB API.
package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

const (
	defaultBaseURL      = "https://api.themoviedb.org/3"
	defaultImageBaseURL = "https://image.tmdb.org/t/p/original"
	defaultMaxAttempts  = 3
	defaultMaxWidth     = 1000
)

var (
	// ErrInvalidMediaType is returned when an unsupported media type is provided.
	ErrInvalidMediaType = errors.New("invalid media type")
	// ErrNoPoster is returned when no poster is available for the media.
	ErrNoPoster = errors.New("poster not available")
)

// HTTPDoer is an interface for making HTTP requests.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client is a TMDB API client.
type Client struct {
	apiKey        string
	baseURL       string
	imageBaseURL  string
	httpClient    HTTPDoer
	mu            sync.RWMutex
	genreCache    map[string]map[int]string
	retryAttempts int
}

// NewClient creates a new TMDB API client.
func NewClient(apiKey string, opts ...Option) *Client {
	client := &Client{
		apiKey:        apiKey,
		baseURL:       defaultBaseURL,
		imageBaseURL:  defaultImageBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		genreCache:    make(map[string]map[int]string),
		retryAttempts: defaultMaxAttempts,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c HTTPDoer) Option {
	return func(client *Client) {
		if c != nil {
			client.httpClient = c
		}
	}
}

// WithBaseURL sets a custom base URL for the TMDB API.
func WithBaseURL(base string) Option {
	return func(client *Client) {
		if base != "" {
			client.baseURL = strings.TrimSuffix(base, "/")
		}
	}
}

// WithImageBaseURL sets a custom base URL for TMDB images.
func WithImageBaseURL(base string) Option {
	return func(client *Client) {
		if base != "" {
			client.imageBaseURL = strings.TrimSuffix(base, "/")
		}
	}
}

// WithRetryAttempts sets the number of retry attempts for failed requests.
func WithRetryAttempts(attempts int) Option {
	return func(client *Client) {
		if attempts > 0 {
			client.retryAttempts = attempts
		}
	}
}

// SearchResult represents a single search result from TMDB.
type SearchResult struct {
	ID           int
	MediaType    string
	Title        string
	Name         string
	PosterPath   string
	Overview     string
	ReleaseDate  string
	FirstAirDate string
	VoteAverage  float64
}

// DisplayTitle returns the appropriate title for the search result.
func (r SearchResult) DisplayTitle() string {
	if r.Title != "" {
		return r.Title
	}
	return r.Name
}

// Year extracts the year from the release or air date.
func (r SearchResult) Year() string {
	source := r.ReleaseDate
	if r.MediaType == "tv" {
		source = r.FirstAirDate
	}
	if source == "" {
		return "Unknown"
	}
	if len(source) >= 4 {
		return source[:4]
	}
	return source
}

// Metadata holds TMDB metadata for a movie or TV show.
type Metadata struct {
	TMDBID        int
	TMDBType      string
	Runtime       *int
	TotalEpisodes *int
	GenreTags     []string
}

// SearchMulti performs a multi-search on TMDB for movies and TV shows.
func (c *Client) SearchMulti(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 1
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", query)
	params.Set("include_adult", "false")

	endpoint := fmt.Sprintf("%s/search/multi?%s", c.baseURL, params.Encode())

	var response struct {
		Results []struct {
			ID           int     `json:"id"`
			MediaType    string  `json:"media_type"`
			Title        string  `json:"title"`
			Name         string  `json:"name"`
			PosterPath   string  `json:"poster_path"`
			Overview     string  `json:"overview"`
			ReleaseDate  string  `json:"release_date"`
			FirstAirDate string  `json:"first_air_date"`
			VoteAverage  float64 `json:"vote_average"`
		} `json:"results"`
	}

	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, limit)
	for _, item := range response.Results {
		if len(results) >= limit {
			break
		}
		if item.MediaType != "movie" && item.MediaType != "tv" {
			continue
		}
		if item.PosterPath == "" {
			continue
		}

		results = append(results, SearchResult{
			ID:           item.ID,
			MediaType:    item.MediaType,
			Title:        item.Title,
			Name:         item.Name,
			PosterPath:   item.PosterPath,
			Overview:     item.Overview,
			ReleaseDate:  item.ReleaseDate,
			FirstAirDate: item.FirstAirDate,
			VoteAverage:  item.VoteAverage,
		})
	}

	return results, nil
}

// GetMovieDetails fetches detailed information for a movie by ID.
func (c *Client) GetMovieDetails(ctx context.Context, movieID int) (map[string]any, error) {
	endpoint := fmt.Sprintf("%s/movie/%d?api_key=%s", c.baseURL, movieID, url.QueryEscape(c.apiKey))
	return c.getJSONMap(ctx, endpoint)
}

// GetTVDetails fetches detailed information for a TV show by ID.
func (c *Client) GetTVDetails(ctx context.Context, tvID int, appendToResponse string) (map[string]any, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	if appendToResponse != "" {
		params.Set("append_to_response", appendToResponse)
	}
	endpoint := fmt.Sprintf("%s/tv/%d?%s", c.baseURL, tvID, params.Encode())
	return c.getJSONMap(ctx, endpoint)
}

// GetFullTVDetails fetches full TV show details including external IDs and keywords.
func (c *Client) GetFullTVDetails(ctx context.Context, tvID int) (map[string]any, error) {
	return c.GetTVDetails(ctx, tvID, "external_ids,keywords,content_ratings")
}

// GetFullMovieDetails fetches full movie details including external IDs and keywords.
func (c *Client) GetFullMovieDetails(ctx context.Context, movieID int) (map[string]any, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("append_to_response", "external_ids,keywords")
	endpoint := fmt.Sprintf("%s/movie/%d?%s", c.baseURL, movieID, params.Encode())
	return c.getJSONMap(ctx, endpoint)
}

// GetMetadataByResult fetches metadata for a search result.
func (c *Client) GetMetadataByResult(ctx context.Context, result SearchResult) (*Metadata, error) {
	switch result.MediaType {
	case "movie":
		return c.getMetadataByMovieID(ctx, result.ID)
	case "tv":
		return c.getMetadataByTVID(ctx, result.ID)
	default:
		return nil, ErrInvalidMediaType
	}
}

// GetMetadataByID fetches metadata by TMDB ID and media type.
func (c *Client) GetMetadataByID(ctx context.Context, mediaID int, mediaType string) (*Metadata, error) {
	switch mediaType {
	case "movie":
		return c.getMetadataByMovieID(ctx, mediaID)
	case "tv":
		return c.getMetadataByTVID(ctx, mediaID)
	default:
		return nil, ErrInvalidMediaType
	}
}

func (c *Client) getMetadataByMovieID(ctx context.Context, movieID int) (*Metadata, error) {
	details, err := c.GetMovieDetails(ctx, movieID)
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{
		TMDBID:   movieID,
		TMDBType: "movie",
	}

	if runtime, ok := getInt(details, "runtime"); ok {
		metadata.Runtime = &runtime
	}

	if tags, err := c.buildGenreTags(ctx, "movie", details); err == nil {
		metadata.GenreTags = tags
	}

	return metadata, nil
}

func (c *Client) getMetadataByTVID(ctx context.Context, tvID int) (*Metadata, error) {
	details, err := c.GetTVDetails(ctx, tvID, "")
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{
		TMDBID:   tvID,
		TMDBType: "tv",
	}

	if runtime, ok := getEpisodeRuntime(details); ok {
		metadata.Runtime = &runtime
	}
	if episodes, ok := getInt(details, "number_of_episodes"); ok {
		metadata.TotalEpisodes = &episodes
	}

	if tags, err := c.buildGenreTags(ctx, "tv", details); err == nil {
		metadata.GenreTags = tags
	}

	return metadata, nil
}

// GetCoverURLByID fetches the cover image URL by TMDB ID and media type.
func (c *Client) GetCoverURLByID(ctx context.Context, mediaID int, mediaType string) (string, error) {
	var details map[string]any
	var err error

	switch mediaType {
	case "movie":
		details, err = c.GetMovieDetails(ctx, mediaID)
	case "tv":
		details, err = c.GetTVDetails(ctx, mediaID, "")
	default:
		return "", ErrInvalidMediaType
	}
	if err != nil {
		return "", err
	}

	posterPath, _ := getString(details, "poster_path")
	if posterPath == "" {
		return "", ErrNoPoster
	}
	return c.ImageURL(posterPath), nil
}

// ImageURL constructs the full image URL from a poster path.
func (c *Client) ImageURL(posterPath string) string {
	return c.imageBaseURL + posterPath
}

// GetCoverAndMetadataByID fetches both cover URL and metadata by ID.
func (c *Client) GetCoverAndMetadataByID(ctx context.Context, mediaID int, mediaType string) (string, *Metadata, error) {
	cover, err := c.GetCoverURLByID(ctx, mediaID, mediaType)
	if err != nil {
		if errors.Is(err, ErrNoPoster) {
			// still return metadata even without a poster
			meta, metaErr := c.GetMetadataByID(ctx, mediaID, mediaType)
			return "", meta, metaErr
		}
		return "", nil, err
	}
	meta, err := c.GetMetadataByID(ctx, mediaID, mediaType)
	if err != nil {
		return cover, nil, err
	}
	return cover, meta, nil
}

// GetCoverAndMetadataByResult fetches both cover URL and metadata from a search result.
func (c *Client) GetCoverAndMetadataByResult(ctx context.Context, result SearchResult) (string, *Metadata, error) {
	cover := c.ImageURL(result.PosterPath)
	meta, err := c.GetMetadataByResult(ctx, result)
	if err != nil {
		return cover, nil, err
	}
	return cover, meta, nil
}

// DownloadAndResizeImage downloads an image and resizes it to the specified width.
func (c *Client) DownloadAndResizeImage(ctx context.Context, imageURL, savePath string, maxWidth int) error {
	if maxWidth <= 0 {
		maxWidth = defaultMaxWidth
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d downloading image", resp.StatusCode)
	}

	img, err := imaging.Decode(resp.Body, imaging.AutoOrientation(true))
	if err != nil {
		return err
	}

	width := img.Bounds().Dx()
	if width > maxWidth {
		img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
	}

	if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err != nil {
		return err
	}

	return imaging.Save(img, savePath, imaging.JPEGQuality(85))
}

func (c *Client) buildGenreTags(ctx context.Context, mediaType string, details map[string]any) ([]string, error) {
	rawGenres, ok := details["genres"].([]any)
	if !ok || len(rawGenres) == 0 {
		return nil, nil
	}

	genres, err := c.getGenres(ctx, mediaType)
	if err != nil {
		return nil, err
	}

	tags := make([]string, 0, len(rawGenres))
	for _, raw := range rawGenres {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, ok := getInt(m, "id")
		if !ok {
			continue
		}
		name, ok := genres[id]
		if !ok {
			continue
		}
		tags = append(tags, fmt.Sprintf("%s/%s", mediaType, sanitizeGenreName(name)))
	}

	return tags, nil
}

func (c *Client) getGenres(ctx context.Context, mediaType string) (map[int]string, error) {
	c.mu.RLock()
	if genres, ok := c.genreCache[mediaType]; ok {
		c.mu.RUnlock()
		return genres, nil
	}
	c.mu.RUnlock()

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	endpoint := fmt.Sprintf("%s/genre/%s/list?%s", c.baseURL, mediaType, params.Encode())

	var response struct {
		Genres []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
	}

	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return nil, err
	}

	result := make(map[int]string, len(response.Genres))
	for _, g := range response.Genres {
		result[g.ID] = g.Name
	}

	c.mu.Lock()
	c.genreCache[mediaType] = result
	c.mu.Unlock()

	return result, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	var lastErr error
	for attempt := 1; attempt <= c.retryAttempts; attempt++ {
		if err := c.doJSONRequest(ctx, endpoint, target); err != nil {
			lastErr = err
			if !isRetryable(err) || attempt == c.retryAttempts {
				return err
			}
			time.Sleep(backoffDelay(attempt))
			continue
		}
		return nil
	}
	return lastErr
}

func (c *Client) getJSONMap(ctx context.Context, endpoint string) (map[string]any, error) {
	var data map[string]any
	if err := c.getJSON(ctx, endpoint, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) doJSONRequest(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("tmdb: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func isRetryable(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		// Network errors (connection resets etc.)
		if strings.Contains(urlErr.Error(), "connection") {
			return true
		}
	}
	return false
}

func backoffDelay(attempt int) time.Duration {
	// exponential backoff capped at 10 seconds
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	if delay > 10*time.Second {
		return 10 * time.Second
	}
	return delay
}

func sanitizeGenreName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "&", "and")
	name = strings.ReplaceAll(name, "#", "")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return strings.Trim(name, "-")
}

func getInt(m map[string]any, key string) (int, bool) {
	val, ok := m[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case json.Number:
		i, err := strconv.Atoi(v.String())
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func getString(m map[string]any, key string) (string, bool) {
	val, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

func getEpisodeRuntime(details map[string]any) (int, bool) {
	val, ok := details["episode_run_time"]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case []any:
		if len(v) == 0 {
			return 0, false
		}
		switch first := v[0].(type) {
		case float64:
			return int(first), true
		case int:
			return first, true
		case string:
			i, err := strconv.Atoi(first)
			if err != nil {
				return 0, false
			}
			return i, true
		default:
			return 0, false
		}
	case []int:
		if len(v) == 0 {
			return 0, false
		}
		return v[0], true
	default:
		return 0, false
	}
}
