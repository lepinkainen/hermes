package imdb

import (
	"log/slog"
	"sync/atomic"
)

var omdbRateLimitReached atomic.Bool

func markOmdbRateLimitReached() {
	if omdbRateLimitReached.CompareAndSwap(false, true) {
		slog.Warn("OMDB API rate limit reached; skipping further OMDB requests for this run")
	}
}

// omdbRequestsAllowed returns true if OMDB requests are still allowed
func omdbRequestsAllowed() bool {
	return !omdbRateLimitReached.Load()
}
