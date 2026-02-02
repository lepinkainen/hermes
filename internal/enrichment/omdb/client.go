package omdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/ratelimit"
	"github.com/spf13/viper"
)

const (
	omdbBaseURL = "http://www.omdbapi.com"
)

var (
	omdbRateLimiter *ratelimit.Limiter
	omdbLimiterOnce sync.Once
)

// getOMDBRateLimiter returns a singleton rate limiter for OMDB.
// OMDB free tier allows 1000 requests/day; we use 1 req/sec to be conservative.
func getOMDBRateLimiter() *ratelimit.Limiter {
	omdbLimiterOnce.Do(func() {
		omdbRateLimiter = ratelimit.New("OMDB", 1)
	})
	return omdbRateLimiter
}

// GetAPIKey retrieves the OMDB API key from config
// It checks multiple config keys in order of preference
func GetAPIKey() (string, error) {
	// Try omdb.api_key first (global key)
	apiKey := viper.GetString("omdb.api_key")
	if apiKey != "" {
		return apiKey, nil
	}

	// Fallback to imdb.omdb_api_key
	apiKey = viper.GetString("imdb.omdb_api_key")
	if apiKey != "" {
		return apiKey, nil
	}

	// Fallback to letterboxd.omdb_api_key
	apiKey = viper.GetString("letterboxd.omdb_api_key")
	if apiKey != "" {
		return apiKey, nil
	}

	return "", fmt.Errorf("OMDB API key not found in config (tried omdb.api_key, imdb.omdb_api_key, letterboxd.omdb_api_key)")
}

// FetchByIMDBID retrieves movie data from OMDB API using an IMDb ID
func FetchByIMDBID(ctx context.Context, imdbID string) (*OMDBResponse, error) {
	apiKey, err := GetAPIKey()
	if err != nil {
		return nil, err
	}

	// Wait for rate limiter
	limiter := getOMDBRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	slog.Debug("Fetching OMDB data by IMDb ID", "imdb_id", imdbID)

	url := fmt.Sprintf("%s/?i=%s&apikey=%s", omdbBaseURL, imdbID, apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Warn("Failed to read error response body", "error", err)
		} else {
			var errorResp struct {
				Response string `json:"Response"`
				Error    string `json:"Error"`
			}
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if errorResp.Error == "Request limit reached!" {
					return nil, errors.NewRateLimitError("OMDB API request limit reached")
				}
				slog.Warn("OMDB API error", "error", errorResp.Error)
			}
		}
		return nil, fmt.Errorf("OMDB API returned non-200 status code: %d for ID: %s", resp.StatusCode, imdbID)
	}

	var omdbResp OMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&omdbResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if the API returned an error response
	if omdbResp.Response == "False" {
		if strings.Contains(omdbResp.Error, "not found") || strings.Contains(omdbResp.Error, "not found!") {
			slog.Debug("Movie not found in OMDB", "imdb_id", imdbID)
			return nil, nil // Not an error, just not found
		}
		return nil, fmt.Errorf("OMDB API error: %s", omdbResp.Error)
	}

	// Check if we got valid data
	if omdbResp.ImdbID == "" || omdbResp.Title == "" {
		return nil, fmt.Errorf("invalid or empty response from OMDB API for ID: %s", imdbID)
	}

	return &omdbResp, nil
}

// FetchByTitleYear retrieves movie data from OMDB API using title and year
func FetchByTitleYear(ctx context.Context, title string, year int) (*OMDBResponse, error) {
	apiKey, err := GetAPIKey()
	if err != nil {
		return nil, err
	}

	// Wait for rate limiter
	limiter := getOMDBRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	slog.Debug("Fetching OMDB data by title and year", "title", title, "year", year)

	// Encode the title for URL
	escapedTitle := strings.ReplaceAll(title, " ", "+")

	url := fmt.Sprintf("%s/?t=%s&y=%d&apikey=%s", omdbBaseURL, escapedTitle, year, apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Warn("Failed to read error response body", "error", err)
		} else {
			var errorResp struct {
				Response string `json:"Response"`
				Error    string `json:"Error"`
			}
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if errorResp.Error == "Request limit reached!" {
					return nil, errors.NewRateLimitError("OMDB API request limit reached")
				}
				slog.Warn("OMDB API error", "error", errorResp.Error)
			}
		}
		return nil, fmt.Errorf("OMDB API returned non-200 status code: %d for title: %s (%d)", resp.StatusCode, title, year)
	}

	var omdbResp OMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&omdbResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if the API returned an error response
	if omdbResp.Response == "False" {
		if strings.Contains(omdbResp.Error, "not found") || strings.Contains(omdbResp.Error, "not found!") {
			slog.Debug("Movie not found in OMDB", "title", title, "year", year)
			return nil, nil // Not an error, just not found
		}
		return nil, fmt.Errorf("OMDB API error: %s", omdbResp.Error)
	}

	// Check if we got valid data
	if omdbResp.Title == "" {
		return nil, fmt.Errorf("invalid or empty response from OMDB API for title: %s (%d)", title, year)
	}

	return &omdbResp, nil
}
