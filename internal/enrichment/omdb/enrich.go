package omdb

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/cache"
)

// EnrichFromOMDB fetches movie ratings from OMDB using the IMDb ID
func EnrichFromOMDB(ctx context.Context, imdbID string) (*RatingsEnrichment, error) {
	if imdbID == "" {
		return nil, fmt.Errorf("IMDb ID is required")
	}

	// Try to get from cache first, or fetch if not cached
	omdbResp, _, err := cache.GetOrFetch("omdb_cache", imdbID, func() (*OMDBResponse, error) {
		return FetchByIMDBID(ctx, imdbID)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch OMDB data: %w", err)
	}

	// If movie not found, return nil without error
	if omdbResp == nil {
		return nil, nil
	}

	// Parse ratings from the response
	ratings := parseRatings(omdbResp.Ratings, omdbResp.ImdbRating)

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
