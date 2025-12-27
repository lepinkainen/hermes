# Cache Architecture

## Overview

Hermes uses a unified SQLite-based caching infrastructure (`cache.db`) that provides three distinct caching strategies through a type-safe generic API. All importers share the same database with separate tables per source.

**Core principles:**
- **Single unified database**: All caches use `cache.db` with table-per-source isolation
- **Three caching strategies**: Basic, policy-based, and variable TTL
- **Thread-safe singleton**: Global cache instance with mutex protection
- **Graceful degradation**: Falls back to direct API calls if cache fails

---

## Caching Strategies

### 1. GetOrFetch() - Basic Caching

**Use when:** You want to cache all API responses with the global TTL (30 days by default)

**Example:** Steam game details, TMDB movie details by ID

```go
details, fromCache, err := cache.GetOrFetch("steam_cache", appID, func() (*GameDetails, error) {
    return fetchGameData(appID)
})
```

**Behavior:**
- All successful fetches are cached
- Cached entries expire after global TTL (default 720h)
- Errors are not cached
- Simple and predictable

---

### 2. GetOrFetchWithPolicy() - Conditional Caching

**Use when:** You only want to cache certain responses (e.g., non-empty results)

**Example:** TMDB searches (don't cache empty results)

```go
results, fromCache, err := cache.GetOrFetchWithPolicy("tmdb_cache", cacheKey,
    func() (*SearchResults, error) {
        return searchTMDB(query, year)
    },
    func(result *SearchResults) bool {
        // Only cache if we got results
        return result != nil && len(result.Results) > 0
    })
```

**Behavior:**
- Only caches if `shouldCache` predicate returns `true`
- Allows selective caching based on result content
- Useful for avoiding pollution with empty/failed results

---

### 3. GetOrFetchWithTTL() - Variable TTL (Negative Caching)

**Use when:** You want different TTLs for different response types (e.g., shorter TTL for "not found")

**Example:** OpenLibrary lookups (7 days for "not found", 30 days for successful)

```go
type CachedBook struct {
    Book     *OpenLibraryBook `json:"book"`
    NotFound bool             `json:"not_found"`
}

cached, fromCache, err := cache.GetOrFetchWithTTL("openlibrary_cache", isbn,
    func() (*CachedBook, error) {
        book, err := fetchFromOpenLibrary(isbn)
        if err != nil && strings.Contains(err.Error(), "not found") {
            return &CachedBook{Book: nil, NotFound: true}, nil
        }
        return &CachedBook{Book: book, NotFound: false}, nil
    },
    cache.SelectNegativeCacheTTL(func(r *CachedBook) bool {
        return r.NotFound
    }))
```

**Behavior:**
- TTL determined by `ttlSelector` function after fetch
- Helper `SelectNegativeCacheTTL` provides standard pattern (7d/30d)
- Ideal for negative caching pattern

---

## Cache Key Patterns

### Simple Keys

Use natural identifiers from the source system:

```go
// Steam: App ID
cache.GetOrFetch("steam_cache", "440", ...)

// IMDb: IMDb ID
omdb.GetCached("tt0111161", ...)
```

### Compound Keys

Combine multiple fields with underscores, normalize to lowercase:

```go
// Letterboxd: title + year
cacheKey := fmt.Sprintf("%s_%d",
    strings.ToLower(strings.TrimSpace(title)),
    year)
// Example: "heat_1995"
```

### Normalized Keys (TMDB)

TMDB uses extensive normalization for consistency:

```go
// TMDB searches: type + normalized_query + year + limit
func normalizeQuery(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    s = strings.ReplaceAll(s, " ", "_")
    s = regexp.MustCompile(`[^a-z0-9_-]`).ReplaceAllString(s, "")
    return s
}
cacheKey := fmt.Sprintf("movies_%s_%d_%d",
    normalizeQuery(query), year, limit)
// Example: "movies_inception_2010_5"
```

**Normalization rules:**
- Lowercase all text
- Trim whitespace
- Replace spaces with underscores
- Remove special characters (keep: a-z, 0-9, _, -)

---

## Specialized Cache Implementations

The codebase has specialized wrappers around the core cache for domain-specific logic.

### internal/tmdb/cache.go

**Why it exists:**
- 9 different cached TMDB operations
- Query normalization for consistent cache keys
- Force refresh capability for enhance command
- Policy-based caching (don't cache empty searches)

**When to add operations here:**
- New TMDB API endpoints
- Operations requiring query normalization
- Operations needing force refresh support

### internal/omdb/cache.go

**Why it exists:**
- OMDB rate limiting integration
- Cache seeding by IMDb ID (cross-key caching)
- Rate limit state management

**When to use:**
- When you need OMDB data
- When rate limiting is a concern
- When you want to seed cache with alternate keys

### cmd/letterboxd/mapping_cache.go

**Why it exists:**
- Bidirectional lookups (Letterboxd URI ↔ TMDB/IMDb IDs)
- Direct SQL for complex queries
- User-confirmed selections persistence

**When you need direct SQL:**
- Bidirectional or multi-column lookups
- Custom schema beyond simple key-value
- No TTL expiration needed

---

## Adding Caching to New Operations

### Step 1: Choose Your Strategy

- **All responses valid?** → Use `GetOrFetch()`
- **Only cache non-empty?** → Use `GetOrFetchWithPolicy()`
- **Different TTLs?** → Use `GetOrFetchWithTTL()` with `SelectNegativeCacheTTL()`

### Step 2: Design Your Cache Key

```go
// Simple ID
cacheKey := itemID

// Compound key
cacheKey := fmt.Sprintf("%s_%d",
    strings.ToLower(strings.TrimSpace(name)),
    year)

// Normalized key (like TMDB)
cacheKey := fmt.Sprintf("%s_%s",
    prefix,
    normalizeQuery(query))
```

### Step 3: Add Table to Schema

Edit `internal/cache/schema.go`:

```go
var AllCacheSchemas = []string{
    // ... existing schemas
    newSourceCacheSchema,
}

const newSourceCacheSchema = `
    CREATE TABLE IF NOT EXISTS newsource_cache (
        cache_key TEXT PRIMARY KEY NOT NULL,
        data TEXT NOT NULL,
        cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_newsource_cache_cached_at
        ON newsource_cache(cached_at);
`
```

Add to `ValidCacheTableNames` map:

```go
var ValidCacheTableNames = map[string]bool{
    // ... existing tables
    "newsource_cache": true,
}
```

### Step 4: Implement Caching

**Basic caching:**

```go
data, fromCache, err := cache.GetOrFetch("newsource_cache", cacheKey,
    func() (*YourType, error) {
        return fetchFromAPI(params)
    })
```

**Negative caching:**

```go
type CachedResult struct {
    Data     *YourType `json:"data"`
    NotFound bool      `json:"not_found"`
}

cached, fromCache, err := cache.GetOrFetchWithTTL("newsource_cache", cacheKey,
    func() (*CachedResult, error) {
        data, err := fetchFromAPI(params)
        if err != nil && isNotFoundError(err) {
            return &CachedResult{Data: nil, NotFound: true}, nil
        }
        return &CachedResult{Data: data, NotFound: false}, nil
    },
    cache.SelectNegativeCacheTTL(func(r *CachedResult) bool {
        return r.NotFound
    }))

if cached.NotFound {
    return nil, fromCache, fmt.Errorf("not found: %s", cacheKey)
}
return cached.Data, fromCache, nil
```

### Step 5: Add Cache Invalidation

Edit `internal/cache/cmd.go` to add your source to the allowed list:

```go
var validSources = map[string]bool{
    // ... existing sources
    "newsource": true,
}
```

Users can now run: `hermes cache invalidate newsource`

---

## Constants and Configuration

### TTL Constants

Defined in `internal/cache/cache.go`:

```go
const (
    DefaultCacheTTL  = 720 * time.Hour  // 30 days
    NegativeCacheTTL = 168 * time.Hour  // 7 days
)
```

### Global Configuration

Users can configure via `config.yaml`:

```yaml
cache:
  dbfile: "./cache.db"  # Path to cache database
  ttl: "720h"           # Default TTL for all caches
```

Or via CLI flags:
```bash
hermes --cache-db-file /tmp/cache.db --cache-ttl 168h import imdb ...
```

---

## Testing Cache Implementations

### Unit Test Pattern

```go
func TestYourCacheFunction(t *testing.T) {
    testutil.SetTestConfig(t)

    // First call - cache miss
    data1, fromCache1, err := getCachedItem("test-id")
    require.NoError(t, err)
    assert.False(t, fromCache1, "Should be cache miss")

    // Second call - cache hit
    data2, fromCache2, err := getCachedItem("test-id")
    require.NoError(t, err)
    assert.True(t, fromCache2, "Should be cache hit")
    assert.Equal(t, data1, data2, "Cached data should match")
}
```

### Negative Caching Test

```go
func TestNegativeCaching(t *testing.T) {
    testutil.SetTestConfig(t)

    // Cache "not found"
    _, fromCache, err := getCachedItem("nonexistent")
    require.Error(t, err)
    assert.False(t, fromCache)

    // Should return cached "not found"
    _, fromCache, err = getCachedItem("nonexistent")
    require.Error(t, err)
    assert.True(t, fromCache, "Not found should be cached")
}
```

---

## Best Practices

### DO

✅ Use appropriate caching strategy for your use case
✅ Normalize cache keys for consistency
✅ Use `SelectNegativeCacheTTL` for negative caching
✅ Add cache invalidation support for your source
✅ Test both cache hit and miss scenarios
✅ Log cache hits/misses at debug level for troubleshooting

### DON'T

❌ Cache errors (except "not found" with negative caching)
❌ Use complex cache keys (keep them simple and deterministic)
❌ Bypass the generic cache functions without good reason
❌ Add per-table TTL configuration (use global + negative caching)
❌ Cache sensitive data (credentials, API keys, etc.)

---

## Debugging Cache Issues

### Enable Debug Logging

```bash
HERMES_LOG_LEVEL=debug ./hermes import imdb --input ratings.csv
```

Look for log messages:
- `"Cache hit"` - Entry found and not expired
- `"Cache miss, fetching data"` - Entry not found or expired
- `"Data cached successfully"` - New entry stored

### Inspect Cache Database

```bash
# View cache entries for a source
sqlite3 cache.db "SELECT cache_key, cached_at FROM tmdb_cache LIMIT 10"

# Check cache size
sqlite3 cache.db "SELECT COUNT(*) FROM omdb_cache"

# Find expired entries
sqlite3 cache.db "SELECT cache_key, cached_at FROM steam_cache
    WHERE datetime(cached_at) < datetime('now', '-720 hours')"
```

### Clear Cache

```bash
# Clear specific source
hermes cache invalidate tmdb

# Clear all (delete entire database)
rm cache.db
```

---

## Migration Guide

### From Direct API Calls to Cached

**Before:**

```go
func fetchMovie(imdbID string) (*Movie, error) {
    return omdbClient.GetByID(imdbID)
}
```

**After:**

```go
func fetchMovie(imdbID string) (*Movie, error) {
    movie, _, err := cache.GetOrFetch("omdb_cache", imdbID, func() (*Movie, error) {
        return omdbClient.GetByID(imdbID)
    })
    return movie, err
}
```

### From Custom Cache to Generic

If you have a custom cache implementation, migrate to the generic cache:

1. Remove custom cache code
2. Add table to `schema.go`
3. Replace cache calls with `cache.GetOrFetch()` or appropriate strategy
4. Update tests to use `testutil.SetTestConfig(t)`

---

## Architecture Decisions

For detailed rationale behind cache architecture decisions, see:
- `docs/decisions/001-cache-architecture.md` - ADR documenting key decisions

---

*Document created: 2025-12-27*
*Last reviewed: 2025-12-27*
