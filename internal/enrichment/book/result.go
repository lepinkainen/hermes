package book

// EnricherResult represents the data fetched from a single Enricher.
type EnricherResult struct {
	// Data is the book metadata extracted from the source.
	// May be nil if the book was not found.
	Data *EnrichmentData

	// Source is the human-readable name of the source.
	Source string

	// Priority is the priority of the source when merging data.
	Priority int
}
