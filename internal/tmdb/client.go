// Package tmdb provides a client for TheMovieDB API.
package tmdb

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/ratelimit"
)

const (
	defaultBaseURL       = "https://api.themoviedb.org/3"
	defaultImageBaseURL  = "https://image.tmdb.org/t/p/original"
	defaultMaxAttempts   = 3
	defaultMaxWidth      = 1000
	defaultRatePerSecond = 4 // TMDB allows ~40 requests per 10 seconds
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
	rateLimiter   *ratelimit.Limiter
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
		rateLimiter:   ratelimit.New("TMDB", defaultRatePerSecond),
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

// WithRateLimiter sets a custom rate limiter for the client.
func WithRateLimiter(limiter *ratelimit.Limiter) Option {
	return func(client *Client) {
		if limiter != nil {
			client.rateLimiter = limiter
		}
	}
}
