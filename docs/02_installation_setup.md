# Installation & Setup

This guide will help you install and configure Hermes to import your data from various sources.

## Prerequisites

- **Go** (version 1.24 or later)
- **Git** for cloning the repository
- **API keys** for enrichment:
  - OMDB API key (IMDb and Letterboxd importers)
  - Steam Web API key (Steam importer)
  - TMDB API key (enhance command and optional TMDB enrichment for IMDb/Letterboxd)

## Installation

### Option 1: Build from Source

1. Clone the repository:

   ```bash
   git clone <repository_url>
   cd hermes
   ```

2. Build the project using Taskfile:

   ```bash
   # Install Task if you don't have it already
   go install github.com/go-task/task/v3/cmd/task@latest

   # Build the project
   task build
   ```

3. Verify the installation:
   ```bash
   ./hermes --version
   ```

## Configuration

Hermes uses a configuration file (`config.yml` or `config.yaml`) to store settings like API keys, input/output directories, and importer-specific options. A sample `config.yml` is included in the repository root—copy and edit it or set the relevant fields via CLI flags/environment variables.

### Creating a Configuration File

Create a `config.yml` file in the project root directory with the following structure:

```yaml
markdownoutputdir: "./markdown/"
jsonoutputdir: "./json/"
overwrite: false

goodreads:
  csvfile: "./path/to/goodreads_export.csv"
imdb:
  csvfile: "./path/to/imdb_ratings.csv"
  omdb_api_key: "your_omdb_api_key"
letterboxd:
  csvfile: "./path/to/letterboxd_export.csv"
steam:
  steamid: "your_steam_id"
  apikey: "your_steam_api_key"

datasette:
  enabled: true
  dbfile: "./hermes.db"

cache:
  dbfile: "./cache.db"
  ttl: "720h"
```

### API Keys

Some importers require API keys for data enrichment:

- **OMDB API** (for IMDb and Letterboxd importers):

  - Register at [OMDb API](http://www.omdbapi.com/apikey.aspx)
  - Free tier allows 1,000 requests per day

- **Steam API** (for Steam importer):
  - Get your API key from [Steam Dev](https://steamcommunity.com/dev/apikey)
  - Find your Steam ID using [SteamID Finder](https://steamidfinder.com/)
- **TMDB API** (for the `enhance` command and optional importer enrichment):
  - Register at [themoviedb.org](https://www.themoviedb.org/settings/api)
  - Export your key as `TMDB_API_KEY` or add `TMDBAPIKey` to the config file

## Basic Usage

### Running an Import

To import data from a supported source:

```bash
# Enhance existing Markdown notes with TMDB metadata/content
TMDB_API_KEY=your_tmdb_key ./hermes enhance --dir /path/to/notes --recursive --overwrite-tmdb

# Import Goodreads data
./hermes import goodreads --csvfile path/to/goodreads_export.csv

# Import IMDb ratings
./hermes import imdb --csvfile path/to/imdb_ratings.csv --apikey your_omdb_api_key

# Import Letterboxd diary
./hermes import letterboxd --csvfile path/to/letterboxd_export.csv --apikey your_omdb_api_key

# Import Steam library
./hermes import steam --apikey your_steam_api_key --steamid your_steam_id
```

### Command-Line Options

Common options available for all importers:

- `--markdown-dir`: Directory for Markdown output (overrides config)
- `--json-dir`: Directory for JSON output (overrides config)
- `--overwrite`: Overwrite existing files (overrides config)
- `--verbose`: Enable verbose logging

### Output Files

After running an import, check the output directories:

- **Markdown files**: `./markdown/{importer_name}/` (or the directory you set in config/flags)
- **JSON files**: `./json/{importer_name}/`

## Optional: Datasette Setup

Hermes exports to a local SQLite database (`hermes.db`) by default so you can explore data in Datasette immediately after an import.

1. Install [Datasette](https://datasette.io/) (requires Python):
   ```sh
   pip install datasette
   ```
2. Serve your database:
   ```sh
   datasette serve hermes.db
   ```
3. Open the provided URL in your browser to explore your data. See [datasette_integration.md](./datasette_integration.md) for more details.

## Troubleshooting

### API Rate Limits

- **OMDB**: Limited to 1,000 requests per day. Large collections may require multiple days to process.
- **Steam**: Has rate limits that may require restarting the import after a few hours.

### Caching

Hermes caches API responses in the `cache/` directory to:

- Respect API rate limits
- Speed up subsequent imports
- Allow for offline processing of previously fetched data

If you encounter issues with outdated data, you can clear the cache:

```bash
rm -rf cache/{importer_name}/*
```

## Next Steps

- See [Architecture](03_architecture.md) for details on how Hermes is structured
- See [Configuration](04_configuration.md) for advanced configuration options
