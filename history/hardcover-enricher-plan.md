# Plan: Add Hardcover.app Book Enrichment

## Overview

Implement a Hardcover enricher for the book enrichment system, following the same pattern as ISBNdb and OpenLibrary enrichers. Hardcover provides metadata not available in other sources: series information, content warnings, moods, genres, and edition data.

---

## What is Hardcover?

Hardcover.app is a book tracking app focused on providing rich metadata about books:
- **Series Information**: Series name, book number in series, series order
- **Content Warnings**: Spoiler-free warnings (violence, sexual content, abuse, etc.)
- **Moods**: Community-tagged moods (cozy, dark, romantic, etc.)
- **Genres**: Detailed genre classifications
- **Edition Data**: Edition details, physical specs, ISBNs
- **API**: GraphQL-based API (no REST API available)
- **Authentication**: API token required (free account available)

**Limitations:**
- GraphQL API requires building queries instead of REST endpoints
- Rate limiting: ~1 request per second (observed from librario)
- ISBN matching may not always work (searches by title + author sometimes needed)

---

## Priority and Data Strategy

### Priority Assignment
- **Priority: 3** (Lower priority than ISBNdb, OpenLibrary, Google Books)
- Rationale: Data is enrichment/supplementary, not core metadata
- Best used as fallback for additional context

### Data Provided
```go
type HardcoverEnrichmentData struct {
    // From core metadata
    Rating          *float64
    ReviewCount     *int
    Description     *string // Community description

    // Hardcover-specific
    Series          *SeriesInfo
    ContentWarnings []string
    Moods           []string
    Genres          []string

    // Edition details
    PhysicalFormat  *string
    Publisher       *string
    PublishedDate   *string
}

type SeriesInfo struct {
    Name     string
    Number   int
    Position string // "Book 1 in series" format
}
```

---

## Implementation Strategy

### Phase 1: GraphQL Client Setup

Create `cmd/goodreads/enrichers/hardcover.go` with:

1. **GraphQL Query Building**
   ```go
   const hardcoverGraphQL = `
   query SearchBooks($query: String!) {
       books(query: $query, first: 1) {
           edges {
               node {
                   id
                   title
                   description
                   isbn
                   rating {
                       average
                       count
                   }
                   series {
                       name
                       position
                   }
                   contentWarnings {
                       name
                   }
                   moods {
                       name
                   }
                   genres {
                       name
                   }
                   // ... other fields
               }
           }
       }
   }
   `
   ```

2. **HTTP Client with GraphQL**
   - Use `*http.Client` like other enrichers
   - POST requests to `https://api.hardcover.app/graphql`
   - Bearer token in Authorization header
   - Request body: `{"query": "...", "variables": {...}}`

3. **Response Unmarshaling**
   ```go
   type hardcoverResponse struct {
       Data struct {
           Books struct {
               Edges []struct {
                   Node hardcoverBook
               }
           }
       }
       Errors []struct {
           Message string
       }
   }
   ```

### Phase 2: Search Strategy

**Search by ISBN (primary):**
```graphql
query SearchByISBN($isbn: String!) {
    booksByIsbn(isbn: $isbn) { ... }
}
```

**Fallback to title + author search:**
```graphql
query SearchByTitle($title: String!, $author: String!) {
    books(query: "$title by $author", first: 1) { ... }
}
```

### Phase 3: Enricher Implementation

```go
type HardcoverEnricher struct {
    httpClient  *http.Client
    apiToken    string
    rateLimiter *ratelimit.Limiter
    clientOnce  sync.Once
    limiterOnce sync.Once
}

func (e *HardcoverEnricher) Name() string {
    return "Hardcover"
}

func (e *HardcoverEnricher) Priority() int {
    return hardcoverPriority // 3
}

func (e *HardcoverEnricher) Ping(ctx context.Context) error {
    // Test query: simple book search that should return results
    // e.g., search for "The Great Gatsby"
}

func (e *HardcoverEnricher) Enrich(ctx context.Context, isbn string) (*book.EnrichmentData, error) {
    // 1. Try ISBN search
    // 2. If not found, try fallback search (requires title+author from input)
    // 3. Cache results
    // 4. Map to EnrichmentData
}
```

### Phase 4: Caching

Add to `internal/cache/schema.go`:
```go
type HardcoverCacheSchema struct {
    ISBN       string `json:"isbn"`
    Data       json.RawMessage `json:"data"`
    NotFound   bool `json:"not_found"`
    CreatedAt  time.Time `json:"created_at"`
}
```

Register table:
- Table name: `"hardcover_cache"`
- TTL: 720h (30 days) for successful results, 168h (7 days) for "not found"
- Use `cache.GetOrFetchWithTTL()` with `SelectNegativeCacheTTL()`

### Phase 5: Configuration

**Config file additions** (`config.yaml`):
```yaml
hardcover:
  api_token: "your_hardcover_api_token"  # Optional
```

**Environment variable**:
```bash
HERMES_HARDCOVER_API_TOKEN=your_token
```

**Default behavior**: Skip silently if no API token configured (like ISBNdb)

### Phase 6: Integration

Update `cmd/goodreads/enrich.go`:
```go
func getDefaultEnrichers() []book.Enricher {
    enrichers := []book.Enricher{
        isbndb.NewISBNdbEnricher(apiKey),
        openlibrary.NewOpenLibraryEnricher(),
        googlebooks.NewGoogleBooksEnricher(),
    }

    // Add Hardcover if token configured
    if token := os.Getenv("HERMES_HARDCOVER_API_TOKEN"); token != "" {
        enrichers = append(enrichers, hardcover.NewHardcoverEnricher(token))
    }

    return enrichers
}
```

---

## Implementation Challenges & Solutions

### Challenge 1: ISBN Search May Fail
**Solution**: Implement fallback search using title + author
- Requires passing title/author to Enrich() method OR
- Modify EnrichmentData to include source fields for reference

### Challenge 2: GraphQL API Complexity
**Solution**:
- Start with minimal query (just the data we need)
- Use code generation (optional): `github.com/99designs/gqlgen`
- Or just unmarshals into JSON and extract fields

### Challenge 3: Limited Rate Limit Info
**Solution**:
- Conservative 1 req/sec rate limiter (same as ISBNdb)
- Monitor for 429 (Too Many Requests) responses
- Implement exponential backoff if needed

### Challenge 4: Account Required
**Solution**:
- Document that free Hardcover account is sufficient
- Link to signup: https://hardcover.app/signup
- API token available in account settings

---

## Testing Strategy

### Unit Tests
- Test GraphQL query building
- Test response parsing (valid response, empty results, errors)
- Test caching behavior
- Test fallback search logic

### Integration Tests (optional, requires Hardcover account)
- Test live API calls with real ISBN
- Test rate limiting doesn't exceed 1 req/sec
- Test cache hit/miss behavior

### Example Test Data
```json
{
  "data": {
    "books": {
      "edges": [
        {
          "node": {
            "id": "12345",
            "title": "The Great Gatsby",
            "isbn": "9780743273565",
            "description": "...",
            "rating": {
              "average": 3.9,
              "count": 85000
            },
            "series": null,
            "contentWarnings": [],
            "moods": ["romantic", "tragic"],
            "genres": ["fiction", "classic"]
          }
        }
      ]
    }
  }
}
```

---

## File Changes Summary

### New Files
- `cmd/goodreads/enrichers/hardcover.go` - Hardcover enricher implementation
- `cmd/goodreads/enrichers/hardcover_test.go` - Unit tests

### Modified Files
- `cmd/goodreads/enrich.go` - Add Hardcover enricher to defaults
- `internal/cache/schema.go` - Add HardcoverCacheSchema
- `cmd/root.go` - Add Hardcover API token flag/config option
- `docs/04_configuration.md` - Document Hardcover configuration

### Testing Requirements
- Test Hardcover enricher independently
- Integration test with real ISBN (requires API token)
- Test priority merging with Hardcover as lowest priority

---

## Configuration Example

**`config.yaml`:**
```yaml
hardcover:
  api_token: "your_hardcover_api_token"
```

**Command line:**
```bash
# Not typically set via CLI, use config or env var
HERMES_HARDCOVER_API_TOKEN=token ./build/hermes import goodreads -f export.csv
```

**CLI Flag** (optional addition):
```bash
./build/hermes import goodreads --hardcover-token="your_token" -f export.csv
```

---

## Verification Steps

1. **Linting**: `task lint` must pass
2. **Tests**: `task test` must pass with new tests
3. **Build**: `task build` succeeds
4. **Manual test**:
   ```bash
   HERMES_HARDCOVER_API_TOKEN=test_token ./build/hermes import goodreads \
     --csvfile test_data.csv --verbose
   ```
5. **Data verification**: Check generated markdown files for Hardcover data (moods, series, warnings)

---

## Risks & Mitigation

| Risk | Mitigation |
|------|-----------|
| API token management | Document in config, support env var, make optional |
| GraphQL complexity | Start simple, add fields incrementally |
| ISBN matching failures | Implement title+author fallback search |
| Rate limit unknowns | Conservative 1/sec limiter, monitor errors |
| Account signup friction | Provide signup link, note free tier is sufficient |

---

## Future Enhancements

Once basic implementation works:
1. **Auto-detection of title+author** - Extract from Book struct instead of requiring separate params
2. **Content warnings expansion** - Additional fields (explicit content, violence level, etc.)
3. **Series data enrichment** - Full series metadata including series author
4. **Mood-based recommendations** - Use moods to recommend similar books
5. **GraphQL code generation** - Use gqlgen for type-safe queries

---

## References

- Hardcover.app: https://hardcover.app
- Hardcover GraphQL API: https://api.hardcover.app/graphql
- Librario Hardcover implementation: Check librario source for patterns
- Current enrichers: `cmd/goodreads/enrichers/` for implementation examples

---

**Status**: Plan created, ready for implementation discussion

*Created: 2025-01-15*
