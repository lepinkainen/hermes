# Hermes Caching

Hermes now uses a SQLite-backed cache (`cache.db`) to store responses from external providers. The cache is separate from the main Datasette/export database (`hermes.db`) and can be safely deleted at any time without affecting your exported notes.

## Default layout

- Cache database: `cache.db` in the project root (config key `cache.dbfile`)
- TTL: `720h` (30 days) for all cached entries (config key `cache.ttl`)
- Cache tables (all share the same shape: `cache_key TEXT PRIMARY KEY`, `data TEXT NOT NULL`, `cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`):
  - `omdb_cache` (IMDb/OMDB lookups)
  - `openlibrary_cache` (Goodreads/OpenLibrary lookups)
  - `steam_cache` (Steam app lookups)
  - `letterboxd_cache` (Letterboxd TMDB lookups)
  - `tmdb_cache` (TMDB metadata and search results)

Tables and indexes are created automatically on first use; no manual migration steps are required.

## Configuration

You can change cache settings via CLI flags, config file entries (`cache.dbfile`, `cache.ttl`), or environment variables (e.g., `CACHE_DBFILE`, `CACHE_TTL` when using `viper.AutomaticEnv`):

```
hermes --cache-db-file /tmp/cache.db --cache-ttl 168h import imdb --input imdb.csv
```

TTL accepts any Go duration string (e.g., `24h`, `7h30m`, `30m`).

## TTL behaviour

- Entries older than the configured TTL are treated as misses and refreshed on the next request.
- Cached lookups that fail to unmarshal are retried and replaced with fresh data.
- Negative TMDB results (empty searches or missing IMDb IDs) are not cached, so subsequent runs can discover newly added titles.

## Cache warming

The cache warms itself on demand—whenever a provider fetches data, the response is cached. If you want warm caches before a larger run, execute the relevant importer once (e.g., `hermes import imdb --input imdb.csv`) to seed entries; subsequent runs will reuse them until TTL expiry.

## Migration from JSON caches

Legacy file-based caches stored under `cache/` have been removed. All providers now use `cache.db`. If you still have a stale `cache/` directory, it can be deleted safely.

## Troubleshooting

- Stale results: delete `cache.db` or lower `cache.ttl` and rerun; tables recreate automatically.
- Need a one-off refresh: use flags that trigger the provider’s force/re-enrich behaviour (e.g., `--force` for enhance) to bypass cached TMDB data.
- Wrong cache location: set `CACHE_DBFILE=/custom/path/cache.db` or pass `--cache-db-file` to point Hermes at the right database.


---

*Document created: 2025-11-18*
*Last reviewed: 2025-11-18*