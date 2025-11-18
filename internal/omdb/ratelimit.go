package omdb

import (
	"log/slog"
	"sync/atomic"
)

var rateLimitReached atomic.Bool

// MarkRateLimitReached marks the OMDB API rate limit as reached.
// It logs a warning on the first call and subsequent calls are no-ops.
func MarkRateLimitReached() {
	if rateLimitReached.CompareAndSwap(false, true) {
		slog.Warn("OMDB API rate limit reached; skipping further OMDB requests for this run")
	}
}

// RequestsAllowed returns true if OMDB requests are still allowed.
func RequestsAllowed() bool {
	return !rateLimitReached.Load()
}

// ResetRateLimit resets the rate limit flag. Useful for testing.
func ResetRateLimit() {
	rateLimitReached.Store(false)
}
