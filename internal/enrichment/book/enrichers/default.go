package enrichers

import (
	"sync"

	"github.com/lepinkainen/hermes/internal/enrichment/book"
)

var (
	defaultEnrichers []book.Enricher
	initEnrichers    = sync.OnceFunc(func() {
		defaultEnrichers = []book.Enricher{
			NewISBNdbEnricher(),      // Priority 0 - highest (skips if no API key)
			NewOpenLibraryEnricher(), // Priority 1
			NewGoogleBooksEnricher(), // Priority 2
			NewBookBrainzEnricher(),  // Priority 3
			NewFinnaEnricher(),       // Priority 4 - Finnish library coverage
		}
	})
)

// Default returns the list of configured book enrichers.
// ISBNdb is included only if the API key is configured.
// The list is initialized once and reused across calls.
func Default() []book.Enricher {
	initEnrichers()
	return defaultEnrichers
}
