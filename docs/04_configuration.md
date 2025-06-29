# Configuration

This document details the configuration options available in Hermes, explaining how to customize the behavior of the application and its importers.

## Configuration File

Hermes uses a YAML configuration file (`config.yaml`) to store settings. By default, the application looks for this file in the current working directory, but you can specify a different location using the `--config` flag.

## Global Configuration

These settings apply to all importers:

```yaml
# Output directories
output:
  markdown: "./markdown" # Directory for Markdown output
  json: "./json" # Directory for JSON output

# Whether to overwrite existing files
overwrite: false

# Logging level (debug, info, warn, error)
loglevel: "info"
```

### Output Directories

- `output.markdown`: Directory where Markdown files will be written
- `output.json`: Directory where JSON files will be written

These directories will be created if they don't exist. Each importer will create its own subdirectory within these directories (e.g., `markdown/goodreads/`, `json/imdb/`).

### Overwrite Flag

- `overwrite`: When set to `true`, existing files will be overwritten. When `false`, the importer will skip files that already exist.

### Log Level

- `loglevel`: Controls the verbosity of logging using the standard Go `slog` library. Valid values are:
  - `debug`: Detailed debugging information
  - `info`: General information about progress
  - `warn`: Warning messages
  - `error`: Error messages only

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
```

### IMDb

```yaml
imdb:
  # Path to the IMDb CSV export file
  csvfile: "./path/to/imdb_ratings.csv"

  # OMDB API key for data enrichment
  apikey: "your_omdb_api_key"

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
  apikey: "your_omdb_api_key"

  # Whether to enrich data with OMDB API
  enrich: true

  # Custom output directories (override global settings)
  output:
    markdown: "./markdown/films"
    json: "./json/films"
```

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

## Datasette Integration

Hermes can export data to a local SQLite database or a remote Datasette instance. Configure this in your `config.yaml`:

```yaml
datasette:
  enabled: false # Enable Datasette output
  mode: "local" # "local" or "remote"
  dbfile: "./hermes.db" # Path to SQLite file (for local mode)
  remote_url: "" # Remote Datasette URL (for remote mode)
  api_token: "" # API token for remote insert (for remote mode)
```

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

- `--datasette`: Enable Datasette output
- `--datasette-mode`: Set mode to "local" or "remote"
- `--datasette-dbfile`: Path to SQLite database file
- `--datasette-url`: Remote Datasette URL
- `--datasette-token`: API token for remote insert

## Environment Variables

Hermes also supports configuration via environment variables. Environment variables take precedence over the configuration file but are overridden by command-line flags.

Environment variables are prefixed with `HERMES_` and use underscores instead of dots for nested properties:

- `HERMES_OUTPUT_MARKDOWN`: Directory for Markdown output
- `HERMES_OUTPUT_JSON`: Directory for JSON output
- `HERMES_OVERWRITE`: Whether to overwrite existing files
- `HERMES_LOGLEVEL`: Logging level

Importer-specific environment variables:

- `HERMES_GOODREADS_CSVFILE`: Path to the Goodreads CSV export file
- `HERMES_IMDB_CSVFILE`: Path to the IMDb CSV export file
- `HERMES_IMDB_APIKEY`: OMDB API key
- `HERMES_LETTERBOXD_CSVFILE`: Path to the Letterboxd CSV export file
- `HERMES_LETTERBOXD_APIKEY`: OMDB API key
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

Here's a complete example of a `config.yaml` file:

```yaml
# Global settings
output:
  markdown: "./markdown"
  json: "./json"
overwrite: false
loglevel: "info"

# Goodreads settings
goodreads:
  csvfile: "./data/goodreads_library_export.csv"
  enrich: true

# IMDb settings
imdb:
  csvfile: "./data/ratings.csv"
  apikey: "your_omdb_api_key"
  enrich: true

# Letterboxd settings
letterboxd:
  csvfile: "./data/letterboxd_export.csv"
  apikey: "your_omdb_api_key"
  enrich: true

# Steam settings
steam:
  apikey: "your_steam_api_key"
  steamid: "your_steam_id"
  fetchdetails: true

# Datasette settings
datasette:
  enabled: true
  mode: "local"
  dbfile: "./hermes.db"
```

## Next Steps

- See [Output Formats](05_output_formats.md) for information about the output formats
- See [Caching](06_caching.md) for details on the caching implementation
- See [Logging & Error Handling](07_logging_error_handling.md) for information about logging and error handling
