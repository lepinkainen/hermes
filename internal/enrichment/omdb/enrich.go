package omdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
)

// OMDBCacheTTL is the cache TTL for OMDB responses (24 hours).
// This allows daily re-checks for new releases that may get ratings.
const OMDBCacheTTL = 24 * time.Hour

// OMDBNotFoundCacheTTL is the cache TTL for "not found" or "invalid ID" responses (7 days).
// This prevents repeated API calls for movies not yet in OMDB.
const OMDBNotFoundCacheTTL = 7 * 24 * time.Hour

// CacheStatus represents the state of an OMDB cache entry.
type CacheStatus int

const (
	// CacheStatusNotCached means there's no cache entry for this ID.
	CacheStatusNotCached CacheStatus = iota
	// CacheStatusNotFound means the ID was checked but not found in OMDB.
	CacheStatusNotFound
	// CacheStatusNoRatings means the movie exists but has no ratings yet.
	CacheStatusNoRatings
	// CacheStatusHasRatings means the movie exists and has ratings.
	CacheStatusHasRatings
)

// CheckCacheStatus checks the OMDB cache status for an IMDb ID.
// This is used to determine whether to skip OMDB enrichment.
func CheckCacheStatus(imdbID string) CacheStatus {
	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		return CacheStatusNotCached
	}

	// Use longer TTL for lookup to catch both normal and negative cache entries
	data, found, err := cacheDB.Get("omdb_cache", imdbID, OMDBNotFoundCacheTTL)
	if err != nil || !found {
		return CacheStatusNotCached
	}

	// Try to unmarshal as the new wrapper type first
	var cached CachedOMDBResponse
	if err := json.Unmarshal([]byte(data), &cached); err == nil {
		if cached.NotFound {
			return CacheStatusNotFound
		}
		if cached.Response != nil {
			hasRatings := (cached.Response.ImdbRating != "" && cached.Response.ImdbRating != "N/A") || len(cached.Response.Ratings) > 0
			if hasRatings {
				return CacheStatusHasRatings
			}
			return CacheStatusNoRatings
		}
	}

	// Fall back to old format (direct OMDBResponse)
	var resp OMDBResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return CacheStatusNotCached
	}

	hasRatings := (resp.ImdbRating != "" && resp.ImdbRating != "N/A") || len(resp.Ratings) > 0
	if hasRatings {
		return CacheStatusHasRatings
	}
	return CacheStatusNoRatings
}

// HasCachedRatings checks if OMDB cache has a response with actual ratings.
// Returns (hasCachedResponse, hasActualRatings).
// This is used to avoid reprocessing files where OMDB has no ratings for the title.
// Deprecated: Use CheckCacheStatus for more detailed status information.
func HasCachedRatings(imdbID string) (bool, bool) {
	status := CheckCacheStatus(imdbID)
	switch status {
	case CacheStatusNotCached:
		return false, false
	case CacheStatusNotFound:
		return true, false
	case CacheStatusNoRatings:
		return true, false
	case CacheStatusHasRatings:
		return true, true
	default:
		return false, false
	}
}

// EnrichFromOMDB fetches movie ratings from OMDB using the IMDb ID
func EnrichFromOMDB(ctx context.Context, imdbID string) (*RatingsEnrichment, error) {
	if imdbID == "" {
		return nil, fmt.Errorf("IMDb ID is required")
	}

	// Try to get from cache first, or fetch if not cached
	// Use negative caching: 24h TTL for valid responses, 7 days for "not found"
	cached, _, err := cache.GetOrFetchWithTTL("omdb_cache", imdbID,
		func() (*CachedOMDBResponse, error) {
			resp, fetchErr := FetchByIMDBID(ctx, imdbID)
			if fetchErr != nil {
				// Check if this is a "not found" or "invalid ID" error that should be cached
				errStr := fetchErr.Error()
				if strings.Contains(errStr, "Incorrect IMDb ID") ||
					strings.Contains(errStr, "not found") ||
					strings.Contains(errStr, "Invalid IMDb ID") {
					// Cache as "not found" so we don't keep hitting the API
					return &CachedOMDBResponse{
						Response: nil,
						NotFound: true,
					}, nil
				}
				// Other errors (network, rate limit, etc.) should not be cached
				return nil, fetchErr
			}
			// Successful response (or nil for movies not in OMDB)
			return &CachedOMDBResponse{
				Response: resp,
				NotFound: resp == nil,
			}, nil
		},
		func(r *CachedOMDBResponse) time.Duration {
			if r.NotFound {
				return OMDBNotFoundCacheTTL // 7 days for "not found"
			}
			return OMDBCacheTTL // 24 hours for valid responses
		})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch OMDB data: %w", err)
	}

	// If movie not found or invalid ID, return nil without error
	if cached.NotFound || cached.Response == nil {
		return nil, nil
	}

	// Parse ratings from the response
	ratings := parseRatings(cached.Response.Ratings, cached.Response.ImdbRating)

	return ratings, nil
}

// parseRatings extracts and normalizes ratings from OMDB response
func parseRatings(ratings []Rating, imdbRating string) *RatingsEnrichment {
	enrichment := &RatingsEnrichment{}

	// Parse IMDb rating
	if imdbRating != "" && imdbRating != "N/A" {
		if rating, err := parseIMDbRating(imdbRating); err == nil {
			enrichment.IMDbRating = rating
		} else {
			slog.Warn("Failed to parse IMDb rating", "value", imdbRating, "error", err)
		}
	}

	// Parse ratings from the Ratings array
	for _, rating := range ratings {
		switch rating.Source {
		case "Internet Movie Database":
			// We already have this from ImdbRating field, but use it as fallback
			if enrichment.IMDbRating == 0 {
				if r, err := parseIMDbRating(rating.Value); err == nil {
					enrichment.IMDbRating = r
				}
			}
		case "Rotten Tomatoes":
			if rt, tomatometer, err := parseRottenTomatoes(rating.Value); err == nil {
				enrichment.RottenTomatoes = rt
				enrichment.RTTomatometer = tomatometer
			} else {
				slog.Warn("Failed to parse Rotten Tomatoes rating", "value", rating.Value, "error", err)
			}
		case "Metacritic":
			if mc, err := parseMetacritic(rating.Value); err == nil {
				enrichment.Metacritic = mc
			} else {
				slog.Warn("Failed to parse Metacritic rating", "value", rating.Value, "error", err)
			}
		}
	}

	return enrichment
}

// parseIMDbRating parses IMDb rating from "8.8/10" format
func parseIMDbRating(value string) (float64, error) {
	if value == "" || value == "N/A" {
		return 0, fmt.Errorf("empty or N/A value")
	}

	// Handle "8.8/10" format
	parts := strings.Split(value, "/")
	if len(parts) > 0 {
		rating, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse rating: %w", err)
		}
		return rating, nil
	}

	return 0, fmt.Errorf("invalid format")
}

// parseRottenTomatoes parses Rotten Tomatoes rating from "94%" format
// Returns the percentage string and the numeric tomatometer value (0-100)
func parseRottenTomatoes(value string) (string, int, error) {
	if value == "" || value == "N/A" {
		return "", 0, fmt.Errorf("empty or N/A value")
	}

	// Remove % and parse
	percentStr := strings.TrimSuffix(strings.TrimSpace(value), "%")
	percent, err := strconv.Atoi(percentStr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse percentage: %w", err)
	}

	return value, percent, nil
}

// parseMetacritic parses Metacritic rating from "85/100" format
// Returns the numeric score (0-100)
func parseMetacritic(value string) (int, error) {
	if value == "" || value == "N/A" {
		return 0, fmt.Errorf("empty or N/A value")
	}

	// Handle "85/100" format
	parts := strings.Split(value, "/")
	if len(parts) > 0 {
		score, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("failed to parse score: %w", err)
		}
		return score, nil
	}

	return 0, fmt.Errorf("invalid format")
}
