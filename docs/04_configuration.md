# Configuration

This document details the configuration options available in Hermes, explaining how to customize the behavior of the application and its importers.

## Configuration File

Hermes uses a YAML configuration file (`config.yml`/`config.yaml`) to store settings. By default, the application looks for this file in the current working directory, but you can specify a different location using the `--config` flag.

## Global Configuration

These settings apply to all importers:

```yaml
markdownoutputdir: "./markdown"
jsonoutputdir: "./json"
overwrite: false

datasette:
  enabled: true
  dbfile: "./hermes.db"

cache:
  dbfile: "./cache.db"
  ttl: "720h"
```

### Output Directories

- `markdownoutputdir`: Root folder for Markdown output
- `jsonoutputdir`: Root folder for JSON output

Each importer creates a subdirectory under these roots (e.g., `markdown/imdb/`), and the enhance command rewrites files in place under the provided directory.

### Overwrite Flag

- `overwrite`: When set to `true`, importers overwrite existing files; otherwise they skip files that already exist.

### Datasette

- `datasette.enabled`: Toggle writing to SQLite for Datasette
- `datasette.dbfile`: Path to the SQLite database (default `./hermes.db`)

### Cache Settings

- `cache.dbfile`: Path to the shared cache database (default `./cache.db`)
- `cache.ttl`: TTL string (e.g., `720h`), applied to TMDB/OMDB/OpenLibrary cache entries

## Importer-Specific Configuration

Each importer has its own configuration section:

### Goodreads

```yaml
goodreads:
  # Path to the Goodreads CSV export file
  csvfile: "./path/to/goodreads_export.csv"

  # Whether to enrich data with OpenLibrary API
  enrich: true

  # Custom output directories (override global settings)
  output:
    markdown: "./markdown/books"
    json: "./json/books"

  # Automation settings for automated export (optional)
  automation:
    email: "your_goodreads_email"
    password: "your_goodreads_password"

# ISBNdb API key for enhanced book enrichment (optional)
# If configured, ISBNdb data is used with highest priority for book metadata
isbndb:
  api_key: "your_isbndb_api_key"
```

#### Automated Export

Goodreads supports automated CSV export using Chrome/Chromium browser automation:

```bash
hermes import goodreads --automated
```

This requires the `automation.email` and `automation.password` fields to be set in your config.

### IMDb

```yaml
imdb:
  # Path to the IMDb CSV export file
  csvfile: "./path/to/imdb_ratings.csv"

  # OMDB API key for data enrichment
  omdb_api_key: "your_omdb_api_key"

  # Whether to enrich data with OMDB API
  enrich: true

  # Custom output directories (override global settings)
  output:
    markdown: "./markdown/movies"
    json: "./json/movies"
```

### Letterboxd

```yaml
letterboxd:
  # Path to the Letterboxd CSV export file
  csvfile: "./path/to/letterboxd_export.csv"

  # OMDB API key for data enrichment
  omdb_api_key: "your_omdb_api_key"

  # Whether to enrich data with OMDB API
  enrich: true

  # Custom output directories (override global settings)
  output:
    markdown: "./markdown/films"
    json: "./json/films"

  # Automation settings for automated export (optional)
  automation:
    username: "your_letterboxd_username"
    password: "your_letterboxd_password"
```

#### Automated Export

Letterboxd supports automated data export using Chrome/Chromium browser automation:

```bash
hermes import letterboxd --automated
```

This requires the `automation.username` and `automation.password` fields to be set in your config.

### Steam

```yaml
steam:
  # Steam API key
  apikey: "your_steam_api_key"

  # Steam user ID (64-bit format)
  steamid: "your_steam_id"

  # Whether to fetch additional game details
  fetchdetails: true

  # Custom output directories (override global settings)
  output:
    markdown: "./markdown/games"
    json: "./json/games"
```

### Enhance

The enhance command primarily relies on CLI flags/environment variables, but you must supply a TMDB API key (typically via `TMDB_API_KEY`). Optional config keys include:

```yaml
tmdbapikey: "your_tmdb_key"
```

Enhance-specific flags (`--recursive`, `--overwrite-tmdb`, `--force`, `--tmdb-content-sections`) are set per run rather than via config.

## Datasette Integration

Hermes exports to a local SQLite database (default: `hermes.db`) for Datasette. Configure this in your config file:

```yaml
datasette:
  enabled: true # Datasette output is on by default; set false to skip
  dbfile: "./hermes.db" # Path to SQLite file
```

The cache database is separate (`cache.db`) and stores API responses; it does not need to be served via Datasette.

## Command-Line Flags

Command-line flags override values from the configuration file. Common flags include:

### Global Flags

- `--config`: Path to the configuration file
- `--markdown-dir`: Directory for Markdown output
- `--json-dir`: Directory for JSON output
- `--overwrite`: Overwrite existing files
- `--verbose`: Enable verbose logging (equivalent to `--loglevel debug`)
- `--loglevel`: Set logging level

### Importer-Specific Flags

#### Goodreads

- `--csvfile`: Path to the Goodreads CSV export file
- `--no-enrich`: Skip data enrichment

#### IMDb

- `--csvfile`: Path to the IMDb CSV export file
- `--apikey`: OMDB API key
- `--no-enrich`: Skip data enrichment

#### Letterboxd

- `--csvfile`: Path to the Letterboxd CSV export file
- `--apikey`: OMDB API key
- `--no-enrich`: Skip data enrichment

#### Steam

- `--apikey`: Steam API key
- `--steamid`: Steam user ID
- `--no-details`: Skip fetching additional game details

#### Datasette

- `--datasette`: Enable Datasette output (defaults to true)
- `--datasette-dbfile`: Path to SQLite database file

## Environment Variables

Hermes also supports configuration via environment variables. Environment variables take precedence over the configuration file but are overridden by command-line flags.

Environment variables are prefixed with `HERMES_` and use underscores instead of dots for nested properties:

- `HERMES_OUTPUT_MARKDOWN`: Directory for Markdown output
- `HERMES_OUTPUT_JSON`: Directory for JSON output
- `HERMES_OVERWRITE`: Whether to overwrite existing files
- `HERMES_LOGLEVEL`: Logging level
- `TMDB_API_KEY`: TMDB API key (used by the enhance command and TMDB enrichment)

Importer-specific environment variables:

- `HERMES_GOODREADS_CSVFILE`: Path to the Goodreads CSV export file
- `HERMES_IMDB_CSVFILE`: Path to the IMDb CSV export file
- `HERMES_IMDB_OMDB_API_KEY`: OMDB API key
- `HERMES_LETTERBOXD_CSVFILE`: Path to the Letterboxd CSV export file
- `HERMES_LETTERBOXD_OMDB_API_KEY`: OMDB API key
- `HERMES_STEAM_APIKEY`: Steam API key
- `HERMES_STEAM_STEAMID`: Steam user ID
- `HERMES_DATASETTE_ENABLED`: Enable Datasette output
- `HERMES_DATASETTE_MODE`: Datasette mode (local/remote)
- `HERMES_DATASETTE_DBFILE`: SQLite database file
- `HERMES_DATASETTE_URL`: Remote Datasette URL
- `HERMES_DATASETTE_TOKEN`: API token for remote insert

## Configuration Precedence

Hermes resolves configuration values in the following order (highest to lowest precedence):

1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

## Example Configuration File

```yaml
markdownoutputdir: "./markdown/"
jsonoutputdir: "./json/"
overwrite: false

datasette:
  enabled: true
  dbfile: "./hermes.db"

cache:
  dbfile: "./cache.db"
  ttl: "720h"

goodreads:
  csvfile: "./data/goodreads_library_export.csv"

# Optional: ISBNdb API key for enhanced book enrichment
isbndb:
  api_key: "your_isbndb_api_key"

imdb:
  csvfile: "./data/ratings.csv"
  omdb_api_key: "your_omdb_api_key"

letterboxd:
  csvfile: "./data/letterboxd_export.csv"
  omdb_api_key: "your_omdb_api_key"

steam:
  steamid: "your_steam_id"
  apikey: "your_steam_api_key"
```

## Next Steps

- See [Output Formats](05_output_formats.md) for information about the output formats
- See [Caching](06_caching.md) for details on the caching implementation
- See [Logging & Error Handling](07_logging_error_handling.md) for information about logging and error handling


---

*Document created: 2025-11-19*
*Last reviewed: 2025-11-19*