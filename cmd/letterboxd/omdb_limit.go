package letterboxd

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

func omdbRequestsAllowed() bool {
	return !omdbRateLimitReached.Load()
}
