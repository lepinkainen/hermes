package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lepinkainen/hermes/internal/cache"
)

// CachedSearchResults wraps SearchResult slice for caching.
type CachedSearchResults struct {
	Results []SearchResult `json:"results"`
}

// CachedMovieDetails wraps movie details map for caching.
type CachedMovieDetails struct {
	Details map[string]any `json:"details"`
}

// CachedTVDetails wraps TV details map for caching.
type CachedTVDetails struct {
	Details map[string]any `json:"details"`
}

// CachedFindResult wraps the result of a find-by-external-ID operation.
type CachedFindResult struct {
	TMDBID    int    `json:"tmdb_id"`
	MediaType string `json:"media_type"`
	Found     bool   `json:"found"`
}

// CachedMetadata wraps Metadata for caching.
type CachedMetadata struct {
	Metadata *Metadata `json:"metadata"`
}

// CachedSearchMovies performs a cached movie-specific search on TMDB.
// Cache key format: movies_{normalized_query}_{year}_{limit}
func (c *Client) CachedSearchMovies(ctx context.Context, query string, year int, limit int) ([]SearchResult, bool, error) {
	cacheKey := fmt.Sprintf("movies_%s_%d_%d", normalizeQuery(query), year, limit)

	result, fromCache, err := cache.GetOrFetchWithPolicy("tmdb_cache", cacheKey, func() (*CachedSearchResults, error) {
		results, searchErr := c.SearchMovies(ctx, query, year, limit)
		if searchErr != nil {
			return nil, searchErr
		}
		return &CachedSearchResults{Results: results}, nil
	}, func(result *CachedSearchResults) bool {
		return result != nil && len(result.Results) > 0
	})

	if err != nil {
		return nil, false, err
	}

	return result.Results, fromCache, nil
}

// CachedSearchMulti performs a cached multi-search on TMDB.
// Cache key format: search_{normalized_query}_{year}_{limit}
func (c *Client) CachedSearchMulti(ctx context.Context, query string, year int, limit int) ([]SearchResult, bool, error) {
	cacheKey := fmt.Sprintf("search_%s_%d_%d", normalizeQuery(query), year, limit)

	result, fromCache, err := cache.GetOrFetchWithPolicy("tmdb_cache", cacheKey, func() (*CachedSearchResults, error) {
		results, searchErr := c.SearchMulti(ctx, query, year, limit)
		if searchErr != nil {
			return nil, searchErr
		}
		return &CachedSearchResults{Results: results}, nil
	}, func(result *CachedSearchResults) bool {
		return result != nil && len(result.Results) > 0
	})

	if err != nil {
		return nil, false, err
	}

	return result.Results, fromCache, nil
}

// CachedGetMovieDetails fetches movie details with caching.
// Cache key format: movie_{tmdb_id}
func (c *Client) CachedGetMovieDetails(ctx context.Context, movieID int) (map[string]any, bool, error) {
	cacheKey := fmt.Sprintf("movie_%d", movieID)

	result, fromCache, err := cache.GetOrFetch("tmdb_cache", cacheKey, func() (*CachedMovieDetails, error) {
		details, fetchErr := c.GetMovieDetails(ctx, movieID)
		if fetchErr != nil {
			return nil, fetchErr
		}
		return &CachedMovieDetails{Details: details}, nil
	})

	if err != nil {
		return nil, false, err
	}

	return result.Details, fromCache, nil
}

// CachedGetTVDetails fetches TV details with caching.
// Cache key format: tv_{tmdb_id} or tv_{tmdb_id}_{append_to_response}
func (c *Client) CachedGetTVDetails(ctx context.Context, tvID int, appendToResponse string) (map[string]any, bool, error) {
	cacheKey := fmt.Sprintf("tv_%d", tvID)
	if appendToResponse != "" {
		cacheKey = fmt.Sprintf("tv_%d_%s", tvID, normalizeQuery(appendToResponse))
	}

	result, fromCache, err := cache.GetOrFetch("tmdb_cache", cacheKey, func() (*CachedTVDetails, error) {
		details, fetchErr := c.GetTVDetails(ctx, tvID, appendToResponse)
		if fetchErr != nil {
			return nil, fetchErr
		}
		return &CachedTVDetails{Details: details}, nil
	})

	if err != nil {
		return nil, false, err
	}

	return result.Details, fromCache, nil
}

// CachedGetFullMovieDetails fetches full movie details with caching.
// Cache key format: movie_full_{tmdb_id}
func (c *Client) CachedGetFullMovieDetails(ctx context.Context, movieID int, force bool) (map[string]any, bool, error) {
	cacheKey := fmt.Sprintf("movie_full_%d", movieID)

	if force {
		details, err := c.GetFullMovieDetails(ctx, movieID)
		if err != nil {
			return nil, false, err
		}
		c.cacheTMDBValue(cacheKey, &CachedMovieDetails{Details: details})
		return details, false, nil
	}

	result, fromCache, err := cache.GetOrFetch("tmdb_cache", cacheKey, func() (*CachedMovieDetails, error) {
		details, fetchErr := c.GetFullMovieDetails(ctx, movieID)
		if fetchErr != nil {
			return nil, fetchErr
		}
		return &CachedMovieDetails{Details: details}, nil
	})

	if err != nil {
		return nil, false, err
	}

	return result.Details, fromCache, nil
}

// CachedGetFullTVDetails fetches full TV details with caching.
// Cache key format: tv_full_{tmdb_id}
func (c *Client) CachedGetFullTVDetails(ctx context.Context, tvID int, force bool) (map[string]any, bool, error) {
	cacheKey := fmt.Sprintf("tv_full_%d", tvID)

	if force {
		details, err := c.GetFullTVDetails(ctx, tvID)
		if err != nil {
			return nil, false, err
		}
		c.cacheTMDBValue(cacheKey, &CachedTVDetails{Details: details})
		return details, false, nil
	}

	result, fromCache, err := cache.GetOrFetch("tmdb_cache", cacheKey, func() (*CachedTVDetails, error) {
		details, fetchErr := c.GetFullTVDetails(ctx, tvID)
		if fetchErr != nil {
			return nil, fetchErr
		}
		return &CachedTVDetails{Details: details}, nil
	})

	if err != nil {
		return nil, false, err
	}

	return result.Details, fromCache, nil
}

// CachedGetMetadataByID fetches metadata by TMDB ID with caching.
// Cache key format: metadata_{media_type}_{tmdb_id}
func (c *Client) CachedGetMetadataByID(ctx context.Context, mediaID int, mediaType string, force bool) (*Metadata, bool, error) {
	cacheKey := fmt.Sprintf("metadata_%s_%d", mediaType, mediaID)

	if force {
		metadata, err := c.GetMetadataByID(ctx, mediaID, mediaType)
		if err != nil {
			return nil, false, err
		}
		c.cacheTMDBValue(cacheKey, &CachedMetadata{Metadata: metadata})
		return metadata, false, nil
	}

	result, fromCache, err := cache.GetOrFetch("tmdb_cache", cacheKey, func() (*CachedMetadata, error) {
		metadata, fetchErr := c.GetMetadataByID(ctx, mediaID, mediaType)
		if fetchErr != nil {
			return nil, fetchErr
		}
		return &CachedMetadata{Metadata: metadata}, nil
	})

	if err != nil {
		return nil, false, err
	}

	return result.Metadata, fromCache, nil
}

// CachedFindByIMDBID finds TMDB ID by IMDB ID with caching.
// Cache key format: find_imdb_{imdb_id}
func (c *Client) CachedFindByIMDBID(ctx context.Context, imdbID string) (int, string, bool, error) {
	cacheKey := fmt.Sprintf("find_imdb_%s", imdbID)

	result, fromCache, err := cache.GetOrFetchWithPolicy("tmdb_cache", cacheKey, func() (*CachedFindResult, error) {
		tmdbID, mediaType, findErr := c.FindByIMDBID(ctx, imdbID)
		if findErr != nil {
			return nil, findErr
		}
		return &CachedFindResult{
			TMDBID:    tmdbID,
			MediaType: mediaType,
			Found:     tmdbID > 0,
		}, nil
	}, func(result *CachedFindResult) bool {
		return result != nil && result.Found
	})

	if err != nil {
		return 0, "", false, err
	}

	return result.TMDBID, result.MediaType, fromCache, nil
}

func (c *Client) cacheTMDBValue(cacheKey string, payload any) {
	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		slog.Warn("Failed to initialize cache for TMDB force refresh", "error", err)
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("Failed to marshal TMDB payload for cache refresh", "key", cacheKey, "error", err)
		return
	}

	if err := cacheDB.Set("tmdb_cache", cacheKey, string(data), 0); err != nil {
		slog.Warn("Failed to update TMDB cache entry", "key", cacheKey, "error", err)
	}
}

// normalizeQuery normalizes a query string for use as a cache key.
func normalizeQuery(query string) string {
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
