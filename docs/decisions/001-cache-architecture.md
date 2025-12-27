# ADR 001: Cache Architecture Unification

## Status

Accepted - Implemented 2025-12-27

## Context

Hermes had multiple cache implementations across the codebase:
- `internal/cache/cache.go` - Generic SQLite-based caching infrastructure
- `internal/tmdb/cache.go` - TMDB-specific caching with 9 operations
- `internal/omdb/cache.go` - OMDB caching with rate limiting
- `cmd/steam/cache.go` - Thin wrapper around generic cache
- `cmd/imdb/cache.go` - Ultra-thin wrapper around OMDB cache
- `cmd/letterboxd/cache.go` - Thin wrapper around OMDB cache
- `cmd/letterboxd/mapping_cache.go` - Custom bidirectional mapping cache
- `cmd/goodreads/cache.go` - Negative caching for OpenLibrary
- `cmd/goodreads/googlebooks_cache.go` - Negative caching for Google Books

**Problems:**
1. **Inconsistent patterns**: Different approaches to negative caching (inline TTL selectors vs no caching)
2. **Code duplication**: Thin wrapper files that added little value
3. **No standardization**: Each importer implemented caching differently
4. **Undocumented decisions**: Why some caches were specialized wasn't clear

## Decision

We decided to:

### 1. Keep Single Unified SQLite Database

**Rationale:**
- Already working well across all importers
- Single `cache.db` file simplifies management
- Table-per-source provides isolation without database proliferation
- Easy to delete entire cache or invalidate specific sources

### 2. Consolidate Thin Wrapper Files

**What we removed:**
- `cmd/steam/cache.go` (39 lines)
- `cmd/imdb/cache.go` (13 lines)
- `cmd/letterboxd/cache.go` (23 lines)

**Rationale:**
- These files provided no domain-specific logic
- Direct inline calls to `cache.GetOrFetch()` or `omdb.GetCached()` are clearer
- Reduces indirection and makes cache usage more explicit at call sites
- 75 lines of boilerplate eliminated

### 3. Keep Specialized Wrappers for Domain Logic

**What we kept:**
- `internal/tmdb/cache.go` - Query normalization, force refresh, 9 operations
- `internal/omdb/cache.go` - Rate limiting, cache seeding
- `cmd/letterboxd/mapping_cache.go` - Bidirectional lookups via direct SQL
- `cmd/goodreads/cache.go` - Negative caching example
- `cmd/goodreads/googlebooks_cache.go` - Negative caching example

**Rationale:**
- TMDB cache: Provides query normalization (essential for consistent cache keys), force refresh (needed by enhance command), and wraps 9 different TMDB operations
- OMDB cache: Integrates with rate limiting system (OMDB-specific concern), provides cache seeding by IMDb ID (cross-importer optimization)
- Letterboxd mapping: Uses direct SQL for bidirectional lookups and structured data (not simple key-value), stores user-confirmed TMDB selections
- Goodreads caches: Demonstrate negative caching pattern, serve as reference implementations

### 4. Standardize Negative Caching Pattern

**Implementation:**
```go
// Added to internal/cache/cache.go
const (
    DefaultCacheTTL  = 720 * time.Hour  // 30 days
    NegativeCacheTTL = 168 * time.Hour  // 7 days
)

func SelectNegativeCacheTTL[T any](isNotFound func(T) bool) func(T) time.Duration {
    return func(result T) time.Duration {
        if isNotFound(result) {
            return NegativeCacheTTL
        }
        return DefaultCacheTTL
    }
}
```

**Rationale:**
- Centralizes TTL configuration (7d/30d split)
- Makes negative caching pattern reusable
- Clearer intent than inline TTL selectors
- Easier to maintain and update TTL values globally

### 5. Three Core Caching Strategies

Maintained three distinct strategies in `internal/cache/cache.go`:

1. **GetOrFetch()** - Basic caching with global TTL
2. **GetOrFetchWithPolicy()** - Conditional caching (e.g., don't cache empty results)
3. **GetOrFetchWithTTL()** - Variable TTL for negative caching

**Rationale:**
- Covers all use cases in the codebase
- Clear semantic differences between strategies
- Type-safe generic API prevents errors
- Well-tested and proven in production

## Alternatives Considered

### Alternative 1: Per-Table TTL Configuration

**Approach:** Allow configuring different TTLs for each cache table

```yaml
cache:
  ttl:
    tmdb: 720h
    omdb: 168h
    steam: 1440h
```

**Rejected because:**
- Adds complexity to configuration
- Most tables can use the same TTL
- Negative caching pattern (different TTLs for different *result types* within a table) is more useful
- Can achieve same result with `GetOrFetchWithTTL()` if needed

### Alternative 2: Complete Consolidation

**Approach:** Remove all specialized wrappers, including TMDB and OMDB

**Rejected because:**
- TMDB query normalization is essential and complex
- OMDB rate limiting is a real concern and needs integration
- Would push complexity into each call site
- Specialized wrappers provide valuable abstractions

### Alternative 3: Separate Databases per Source

**Approach:** `tmdb.db`, `omdb.db`, `steam.db`, etc.

**Rejected because:**
- Multiple database files harder to manage
- No benefit over table-per-source in single database
- Cache invalidation becomes more complex
- Backup/restore/deletion more complicated

### Alternative 4: In-Memory Cache

**Approach:** Use in-memory caching (map, sync.Map, etc.)

**Rejected because:**
- Cache lost on restart (significant for 30-day TTL)
- Memory pressure for large datasets
- SQLite provides persistence and proven reliability
- Already have working SQLite implementation

## Consequences

### Positive

✅ **Reduced code duplication**: 75 lines of wrapper code removed
✅ **Standardized patterns**: Negative caching now has a clear, reusable pattern
✅ **Better documentation**: Comprehensive dev guide and user docs
✅ **Clearer intent**: Cache usage more explicit at call sites
✅ **Easier to maintain**: Centralized TTL constants and helper functions
✅ **Reference implementations**: Goodreads caches demonstrate best practices

### Negative

⚠️ **Slightly more verbose**: Inline calls are a few lines longer than wrapper function calls
⚠️ **No per-table TTL**: If needed in future, would require refactoring (though `GetOrFetchWithTTL` provides workaround)

### Neutral

ℹ️ **Specialized wrappers remain**: Still have TMDB, OMDB, and mapping caches (intentional, provides value)
ℹ️ **Three strategies**: Continue to maintain three caching functions (worth it for clarity)

## Implementation

Implemented in three phases:

### Phase 1: Consolidation (hermes-qylv)
- Removed `cmd/steam/cache.go`, inlined calls to `cache.GetOrFetch()`
- Removed `cmd/imdb/cache.go`, inlined calls to `omdb.GetCached()`
- Removed `cmd/letterboxd/cache.go`, inlined calls to `omdb.GetCached()`
- All tests passed, no functional changes

### Phase 2: TTL Standardization (hermes-b43n)
- Added `SelectNegativeCacheTTL` helper to `internal/cache/cache.go`
- Added `DefaultCacheTTL` and `NegativeCacheTTL` constants
- Updated `cmd/goodreads/cache.go` to use new helper
- Updated `cmd/goodreads/googlebooks_cache.go` to use new helper
- Added comprehensive tests for helper function
- All tests passed, cache coverage improved (51.7% from 50.7%)

### Phase 3: Documentation (hermes-ngsx)
- Updated `docs/caching.md` with negative caching section
- Created `docs/cache-architecture.md` developer guide
- Created this ADR (`docs/decisions/001-cache-architecture.md`)
- Updated `CLAUDE.md` with cache guidance for AI assistants

## Compliance

This decision aligns with:
- **Go idioms**: Simple, explicit code over abstraction
- **Project principles**: Single-user private use, pragmatic over perfect
- **DRY principle**: Eliminate duplication where it adds no value
- **Clear over clever**: Explicit cache calls vs hidden wrapper magic

## Future Considerations

### Potential Enhancements (Not Needed Now)

1. **Per-table TTL configuration**: If specific tables need different default TTLs
2. **Cache versioning**: If schema changes invalidate old entries
3. **Background cleanup**: Automated removal of expired entries (currently lazy)
4. **Cache metrics**: Track hit rates, sizes, etc. (currently just debug logs)

### When to Revisit

- If multiple sources need per-table TTL (hasn't happened yet)
- If cache database grows too large (lazy expiration not keeping up)
- If we add many more sources (current pattern scales well)
- If users request more granular cache control

## Related Decisions

- None (this is the first ADR)

## References

- Implementation: Commits 77ffdd13 (consolidation) and 31202e4a (TTL standardization)
- Developer Guide: `docs/cache-architecture.md`
- User Guide: `docs/caching.md`
- Epic: hermes-r1lc - Cache Architecture Unification
  - hermes-qylv: Evaluate and consolidate cache implementations
  - hermes-b43n: Standardize cache TTL behavior
  - hermes-ngsx: Document cache architecture decisions

---

*Date: 2025-12-27*
*Author: Claude Opus 4.5 via Claude Code*
*Status: Implemented*
