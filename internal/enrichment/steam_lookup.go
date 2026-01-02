package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	hermeserrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/tui"
)

// steamSearchURL is the Steam Store search API endpoint.
const steamSearchURL = "https://store.steampowered.com/api/storesearch/?term=%s&l=english&cc=US"

// SteamStoreSearchResult represents a search result from Steam Store API.
type SteamStoreSearchResult struct {
	AppID int    `json:"id"`
	Name  string `json:"name"`
	Tiny  string `json:"tiny_image"`
}

// SteamStoreSearchResponse is the API response from Steam Store search.
type SteamStoreSearchResponse struct {
	Total int                      `json:"total"`
	Items []SteamStoreSearchResult `json:"items"`
}

// CachedSteamSearchResults wraps search results for caching
type CachedSteamSearchResults struct {
	Results []tui.SteamSearchResult `json:"results"`
}

func resolveSteamAppID(ctx context.Context, title string, existingAppID int, opts SteamEnrichmentOptions) (int, error) {
	if existingAppID != 0 && !opts.Force {
		slog.Debug("Using existing Steam AppID", "appid", existingAppID, "title", title)
		return existingAppID, nil
	}

	return searchSteamAppID(ctx, title, opts)
}

func searchSteamAppID(ctx context.Context, title string, opts SteamEnrichmentOptions) (int, error) {
	results, err := searchSteamStore(ctx, title)
	if err != nil {
		return 0, fmt.Errorf("steam search failed: %w", err)
	}

	if len(results) == 0 {
		slog.Debug("No Steam results found", "title", title)
		return 0, nil
	}

	selection, err := selectSteamResult(results, title, opts.Interactive)
	if err != nil {
		return 0, err
	}
	if selection == nil {
		return 0, nil
	}

	return selection.AppID, nil
}

// searchSteamStore searches the Steam Store for games matching the query with caching.
func searchSteamStore(ctx context.Context, query string) ([]tui.SteamSearchResult, error) {
	// Normalize query for cache key
	cacheKey := normalizeSteamQuery(query)

	// Use cached search with policy to avoid caching empty results
	cached, _, err := cache.GetOrFetchWithPolicy(
		"steam_search_cache",
		cacheKey,
		func() (*CachedSteamSearchResults, error) {
			results, fetchErr := fetchSteamStoreSearch(ctx, query)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return &CachedSteamSearchResults{Results: results}, nil
		},
		func(result *CachedSteamSearchResults) bool {
			// Only cache if we got results
			return result != nil && len(result.Results) > 0
		},
	)

	if err != nil {
		return nil, err
	}

	return cached.Results, nil
}

// fetchSteamStoreSearch performs the actual Steam Store search API call
func fetchSteamStoreSearch(ctx context.Context, query string) ([]tui.SteamSearchResult, error) {
	encodedQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf(steamSearchURL, encodedQuery)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Steam search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("steam search returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp SteamStoreSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse Steam search response: %w", err)
	}

	results := make([]tui.SteamSearchResult, len(searchResp.Items))
	for i, item := range searchResp.Items {
		results[i] = tui.SteamSearchResult{
			AppID:       item.AppID,
			Name:        item.Name,
			HeaderImage: item.Tiny,
		}
	}

	return results, nil
}

// normalizeSteamQuery normalizes a query string for use as a cache key
func normalizeSteamQuery(query string) string {
	// Convert to lowercase and replace spaces with underscores
	normalized := strings.ToLower(strings.TrimSpace(query))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	// Remove special characters that might cause issues
	normalized = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, normalized)
	return normalized
}

func selectSteamResult(results []tui.SteamSearchResult, title string, interactive bool) (*tui.SteamSearchResult, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// If only one result, auto-select it
	if len(results) == 1 {
		slog.Debug("Auto-selected single Steam result", "title", title, "appid", results[0].AppID)
		return &results[0], nil
	}

	// Check for exact title match
	exact := findExactSteamMatch(results, title)

	if interactive {
		selection, err := selectSteamInteractive(title, results)
		if err != nil {
			return nil, err
		}
		return selection, nil
	}

	// Non-interactive: use exact match or first result
	if exact != nil {
		slog.Debug("Auto-selected exact Steam match", "title", title, "appid", exact.AppID)
		return exact, nil
	}

	slog.Debug("Auto-selected first Steam result", "title", title, "appid", results[0].AppID)
	return &results[0], nil
}

func findExactSteamMatch(results []tui.SteamSearchResult, title string) *tui.SteamSearchResult {
	normalizedTitle := strings.ToLower(strings.TrimSpace(title))

	var match *tui.SteamSearchResult
	matchCount := 0

	for i := range results {
		result := &results[i]
		if strings.ToLower(strings.TrimSpace(result.Name)) == normalizedTitle {
			match = result
			matchCount++
			if matchCount > 1 {
				return nil // Ambiguous
			}
		}
	}

	return match
}

// selectSteamInteractive presents a TUI for Steam game selection.
func selectSteamInteractive(title string, results []tui.SteamSearchResult) (*tui.SteamSearchResult, error) {
	selection, err := tui.SelectSteam(title, results, nil)
	if err != nil {
		return nil, fmt.Errorf("TUI selection failed: %w", err)
	}

	switch selection.Action {
	case tui.ActionSelected:
		if selection.SteamSelection != nil {
			return selection.SteamSelection, nil
		}
		return nil, nil
	case tui.ActionStopped:
		return nil, hermeserrors.NewStopProcessingError("Steam selection stopped by user")
	default:
		slog.Debug("User skipped Steam selection")
		return nil, nil
	}
}
