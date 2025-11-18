package enrich

import (
	"fmt"

	"github.com/lepinkainen/hermes/internal/enrichment"
	hermeserrors "github.com/lepinkainen/hermes/internal/errors"
)

// Options defines the shared enrichment flow for OMDB + TMDB across importers.
// T is the target record type (e.g., MovieSeen or letterboxd.Movie).
// O is the OMDB payload type returned by the fetcher (typically a pointer to T or a related struct).
type Options[T any, O any] struct {
	// SkipOMDB short-circuits OMDB enrichment (e.g., when plot/description already exists).
	SkipOMDB bool
	// FetchOMDB retrieves OMDB data (may return nil data); still used even when rate limited.
	FetchOMDB func() (O, error)
	// ApplyOMDB merges OMDB data into the target.
	ApplyOMDB func(*T, O)
	// OnOMDBError handles non-rate-limit OMDB errors (logging, metrics, etc).
	OnOMDBError func(error)
	// OnOMDBRateLimit handles OMDB rate limit events (e.g., cache marker, logging).
	OnOMDBRateLimit func(error)

	// TMDBEnabled controls whether TMDB enrichment should run.
	TMDBEnabled bool
	// FetchTMDB retrieves TMDB enrichment payload (may return nil data).
	FetchTMDB func() (*enrichment.TMDBEnrichment, error)
	// ApplyTMDB merges TMDB data into the target.
	ApplyTMDB func(*T, *enrichment.TMDBEnrichment)
	// OnTMDBError handles TMDB errors (except StopProcessingError which is bubbled).
	OnTMDBError func(error)
}

// Result captures per-provider errors for callers who want to inspect both outcomes.
type Result struct {
	OMDBErr error
	TMDBErr error
}

// Enrich runs OMDB + TMDB enrichment in a consistent order:
// - OMDB fetch (unless skipped). On rate limit, continues to TMDB.
// - TMDB fetch (when enabled). StopProcessingError is bubbled immediately.
// Returns a combined error only when both providers fail.
func Enrich[T any, O any](target *T, opts Options[T, O]) (*Result, error) {
	res := &Result{}

	if target == nil {
		return res, nil
	}

	// OMDB
	if opts.FetchOMDB != nil && !opts.SkipOMDB {
		omdbData, err := opts.FetchOMDB()
		if err != nil {
			res.OMDBErr = err
			if hermeserrors.IsRateLimitError(err) {
				if opts.OnOMDBRateLimit != nil {
					opts.OnOMDBRateLimit(err)
				}
			} else if opts.OnOMDBError != nil {
				opts.OnOMDBError(err)
			}
		} else if opts.ApplyOMDB != nil {
			opts.ApplyOMDB(target, omdbData)
		}
	}

	// TMDB
	if opts.TMDBEnabled && opts.FetchTMDB != nil {
		tmdbData, err := opts.FetchTMDB()
		if err != nil {
			if hermeserrors.IsStopProcessingError(err) {
				return res, err
			}
			res.TMDBErr = err
			if opts.OnTMDBError != nil {
				opts.OnTMDBError(err)
			}
		} else if tmdbData != nil && opts.ApplyTMDB != nil {
			opts.ApplyTMDB(target, tmdbData)
		}
	}

	if res.OMDBErr != nil && res.TMDBErr != nil {
		return res, fmt.Errorf("movie enrichment failed; omdb: %w; tmdb: %v", res.OMDBErr, res.TMDBErr)
	}

	return res, nil
}
