# Caching

This document describes the caching mechanism used by Hermes to store API responses and other data that is expensive to fetch repeatedly.

## Purpose of Caching

Hermes implements caching for several important reasons:

1. **API Rate Limits**: Many external APIs used for data enrichment have rate limits. Caching helps respect these limits by avoiding redundant requests.
2. **Performance**: Fetching data from external APIs can be slow. Caching improves performance by storing previously fetched data locally.
3. **Offline Operation**: Caching allows Hermes to work with previously fetched data even when internet connectivity is unavailable.
4. **Resumability**: If an import process is interrupted, caching allows it to resume without re-fetching already processed data.

## Cache Directory Structure

By default, Hermes stores cache files in the `cache/` directory at the root of the project. Each importer has its own subdirectory within the cache directory:

```
cache/
├── goodreads/       # Goodreads API cache
├── imdb/            # OMDB API cache for IMDb
├── letterboxd/      # OMDB API cache for Letterboxd
└── steam/           # Steam API cache
```

## Cache Implementation

Each importer implements its own caching logic in a `cache.go` file. The caching implementation typically follows these patterns:

### Cache Keys

Cache keys are derived from the request parameters to ensure uniqueness. For example:

- For OMDB API (used by IMDb and Letterboxd importers), the cache key might be the IMDb ID or a combination of title and year.
- For OpenLibrary API (used by Goodreads importer), the cache key might be the ISBN or a combination of title and author.
- For Steam API, the cache key might be the Steam app ID.

### Cache File Format

Cache files are typically stored as JSON files, with filenames derived from the cache key. For example:

- `cache/imdb/tt0111161.json` for The Shawshank Redemption
- `cache/goodreads/9780451524935.json` for 1984 by George Orwell
- `cache/steam/70.json` for Half-Life

### Cache Operations

The typical cache operations include:

1. **Check Cache**: Before making an API request, check if the data is already cached.
2. **Read Cache**: If the data is cached, read it from the cache file.
3. **Write Cache**: If the data is not cached, fetch it from the API and write it to the cache.

Example pseudocode for a typical caching implementation:

```go
func fetchWithCache(key string, fetchFunc func() (Data, error)) (Data, error) {
    // Check if data is in cache
    cacheFile := filepath.Join("cache", "importer", key+".json")
    if fileExists(cacheFile) {
        // Read from cache
        data, err := readFromCache(cacheFile)
        if err == nil {
            return data, nil
        }
        // If reading from cache fails, fall back to API
    }

    // Fetch from API
    data, err := fetchFunc()
    if err != nil {
        return Data{}, err
    }

    // Write to cache
    writeToCache(cacheFile, data)

    return data, nil
}
```

## Cache Configuration

Caching behavior can be controlled through configuration options:

### Global Cache Settings

```yaml
cache:
  enabled: true # Enable or disable caching globally
  directory: "./cache" # Cache directory path
  ttl: 2592000 # Cache time-to-live in seconds (30 days)
```

### Importer-Specific Cache Settings

```yaml
imdb:
  cache:
    enabled: true # Enable or disable caching for this importer
    ttl: 604800 # Cache time-to-live in seconds (7 days)
```

### Command-Line Flags

- `--no-cache`: Disable caching for the current operation
- `--clear-cache`: Clear the cache before starting the operation
- `--cache-ttl`: Set the cache time-to-live in seconds

## Cache Invalidation

Cache invalidation strategies include:

1. **Time-Based Invalidation**: Cache entries older than a specified TTL (time-to-live) are considered stale and will be refreshed.
2. **Manual Invalidation**: Users can manually clear the cache using the `--clear-cache` flag.
3. **Selective Invalidation**: Some importers may implement selective cache invalidation based on specific criteria.

## API-Specific Caching Considerations

### OMDB API (IMDb and Letterboxd)

- OMDB has a limit of 1,000 requests per day for the free tier.
- Cache files are stored in `cache/imdb/` and `cache/letterboxd/`.
- Cache keys are typically IMDb IDs (e.g., `tt0111161`).

### OpenLibrary API (Goodreads)

- OpenLibrary doesn't have strict rate limits but encourages responsible use.
- Cache files are stored in `cache/goodreads/`.
- Cache keys are typically ISBNs or a combination of title and author.

### Steam API

- Steam API has rate limits that can vary based on the endpoint.
- Cache files are stored in `cache/steam/`.
- Cache keys are typically Steam app IDs.

## Troubleshooting

### Stale Cache Data

If you suspect the cache contains stale data, you can:

1. Clear the entire cache:

   ```bash
   rm -rf cache/*
   ```

2. Clear the cache for a specific importer:

   ```bash
   rm -rf cache/imdb/*
   ```

3. Use the `--clear-cache` flag:
   ```bash
   ./hermes imdb --clear-cache
   ```

### Cache Corruption

If cache files become corrupted, Hermes will typically log an error and fall back to fetching data from the API. You can manually delete corrupted cache files or clear the entire cache.

## Next Steps

- See [Logging & Error Handling](07_logging_error_handling.md) for information about logging and error handling
- See the importer-specific documentation for details on the caching implementation for each importer


---

*Document created: 2025-04-30*
*Last reviewed: 2025-04-30*