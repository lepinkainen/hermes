# Configuration

This guide covers the configuration options currently supported by Hermes. When in doubt, `hermes --help` is the source of truth for flags.

## Configuration File

Hermes reads `config.yml` (or `config.yaml`) from the current working directory. There is no CLI flag to change the config path. If no config file exists, Hermes writes a default config file and exits so you can edit it before running commands again.

## Global Configuration

```yaml
markdownoutputdir: "./markdown/"
jsonoutputdir: "./json/"
overwritefiles: false
updatecovers: false

datasette:
  enabled: true
  dbfile: "./hermes.db"

cache:
  dbfile: "./cache.db"
  ttl: "720h"

tmdb:
  cover_cache:
    enabled: false
    path: "tmdb-cover-cache"
```

### Output Directories

- `markdownoutputdir`: Root folder for Markdown output.
- `jsonoutputdir`: Root folder for JSON output.

Each importer writes into a subdirectory under these roots (for example, `markdown/imdb/`). The importers accept an `output` value to override the subdirectory name.

### Overwrite and Cover Updates

- `overwritefiles`: When true, importers overwrite existing markdown files.
- `updatecovers`: When true, importers re-download cover images even if they already exist.

### Datasette

- `datasette.enabled`: Toggle writing to SQLite for Datasette.
- `datasette.dbfile`: Path to the SQLite database (default `./hermes.db`).

### Cache Settings

- `cache.dbfile`: Path to the shared cache database (default `./cache.db`).
- `cache.ttl`: TTL string (for example, `720h`), applied to TMDB/OMDB/OpenLibrary cache entries.

### TMDB Cover Cache

- `tmdb.cover_cache.enabled`: Enable the local TMDB cover cache.
- `tmdb.cover_cache.path`: Directory for cached TMDB cover images.

## Importer-Specific Configuration

### Goodreads

```yaml
goodreads:
  csvfile: "./path/to/goodreads_export.csv"
  output: "goodreads"
  automation:
    email: "your_goodreads_email"
    password: "your_goodreads_password"
    headful: false
    download_dir: "exports"
    timeout: "3m"

isbndb:
  api_key: "your_isbndb_api_key"
```

### IMDb

```yaml
imdb:
  csvfile: "./path/to/imdb_ratings.csv"
  output: "imdb"
  omdb_api_key: "your_omdb_api_key"
```

### Letterboxd

```yaml
letterboxd:
  csvfile: "./path/to/letterboxd_export.csv"
  output: "letterboxd"
  omdb_api_key: "your_omdb_api_key"
  automation:
    username: "your_letterboxd_username"
    password: "your_letterboxd_password"
    headful: false
    download_dir: "exports"
    timeout: "3m"
```

### Steam

```yaml
steam:
  steamid: "your_steam_id"
  apikey: "your_steam_api_key"
  output: "steam"
```

## Enhance and TMDB

Enhance and TMDB enrichment use a TMDB API key. The preferred source is the `TMDB_API_KEY` environment variable. You can also place the key in the config file using `TMDBAPIKey` if you need a file-based option.

## Command-Line Flags

Flags override config values. Global flags live at the root command, and importer-specific flags live under each subcommand.

### Global Flags

- `--overwrite`: Overwrite existing markdown files.
- `--update-covers`: Re-download cover images even if they already exist.
- `--datasette`: Enable Datasette output (default true).
- `--datasette-db`: SQLite database file path.
- `--cache-db-file`: Cache SQLite database file path.
- `--cache-ttl`: Cache TTL duration string.
- `--use-tmdb-cover-cache`: Use the TMDB cover cache.
- `--tmdb-cover-cache-path`: Path to the TMDB cover cache directory.

### Importer Flags (selection)

Goodreads:
- `--input`
- `--automated`
- `--goodreads-email`
- `--goodreads-password`
- `--headful`
- `--download-dir`
- `--automation-timeout`
- `--dry-run`

IMDb:
- `--input`
- `--tmdb-generate-content`
- `--tmdb-no-interactive`
- `--tmdb-content-sections`

Letterboxd:
- `--input`
- `--automated`
- `--letterboxd-username`
- `--letterboxd-password`
- `--headful`
- `--download-dir`
- `--automation-timeout`
- `--dry-run`

Steam:
- `--steam-id`
- `--api-key`

Enhance:
- `--input-dirs`
- `--recursive`
- `--dry-run`
- `--regenerate-data`
- `--force`
- `--refresh-cache`
- `--tmdb-no-interactive`
- `--tmdb-content-sections`
- `--omdb-no-enrich`

Diff:
- `--output`
- `--html`
- `--db-file`

> Note: there are no `--config`, `--markdown-dir`, or `--json-dir` flags today. Use the config file for output directory defaults.

## Environment Variables

Hermes binds a small set of environment variables explicitly:

- `TMDB_API_KEY`
- `GOODREADS_HEADFUL`
- `GOODREADS_DOWNLOAD_DIR`
- `GOODREADS_AUTOMATION_TIMEOUT`
- `LETTERBOXD_USERNAME`
- `LETTERBOXD_PASSWORD`
- `LETTERBOXD_HEADFUL`
- `LETTERBOXD_DOWNLOAD_DIR`
- `LETTERBOXD_AUTOMATION_TIMEOUT`
- `HERMES_LOG_LEVEL`

Other configuration is expected to come from the config file or CLI flags.

## Configuration Precedence

1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

## Next Steps

- See [Output Formats](05_output_formats.md)
- See [Caching](06_caching.md)
- See [Logging & Error Handling](07_logging_error_handling.md)

---

*Document created: 2025-11-19*
*Last reviewed: 2026-02-02*
