package enrich

import "log/slog"

// OMDBErrorHandler returns a standard error handler for OMDB enrichment failures.
func OMDBErrorHandler(title string) func(error) {
	return func(err error) {
		slog.Warn("Failed to enrich from OMDB", "title", title, "error", err)
	}
}

// OMDBRateLimitHandler returns a handler for OMDB rate limit events.
// markRateLimit is called first to mark the rate limit state (e.g., omdb.MarkRateLimitReached).
func OMDBRateLimitHandler(title string, markRateLimit func()) func(error) {
	return func(_ error) {
		markRateLimit()
		slog.Warn("Skipping OMDB enrichment after rate limit", "title", title)
	}
}

// TMDBErrorHandler returns a standard error handler for TMDB enrichment failures.
func TMDBErrorHandler(title string) func(error) {
	return func(err error) {
		slog.Warn("Failed to enrich from TMDB", "title", title, "error", err)
	}
}
