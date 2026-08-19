package book

import (
	"context"
	"log/slog"
)

// RunEnrichers runs each of the given enrichers against the provided ISBN,
// collects their results, and merges them via a PriorityMerger.
// Returns nil, nil if no enricher returned data.
func RunEnrichers(ctx context.Context, isbn string, enrichers []Enricher) (*EnrichmentData, error) {
	results := make([]EnricherResult, 0, len(enrichers))

	for _, e := range enrichers {
		data, err := e.Enrich(ctx, isbn)
		if err != nil {
			slog.Debug("Enricher failed", "enricher", e.Name(), "isbn", isbn, "error", err)
			continue
		}

		if data != nil {
			slog.Debug("Enricher returned data", "enricher", e.Name(), "isbn", isbn)
			results = append(results, EnricherResult{
				Data:     data,
				Source:   e.Name(),
				Priority: e.Priority(),
			})
		}
	}

	if len(results) == 0 {
		slog.Debug("No enrichment data found", "isbn", isbn)
		return nil, nil
	}

	merger := NewPriorityMerger()
	return merger.Merge(results), nil
}
