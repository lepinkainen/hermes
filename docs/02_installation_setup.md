# Installation & Setup

This guide will help you install and configure Hermes to import your data from various sources.

## Prerequisites

- **Go** (version 1.18 or later)
- **Git** for cloning the repository
- **API keys** for certain data sources (e.g., OMDB, Steam)

## Installation

### Option 1: Build from Source

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/hermes.git
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

Hermes uses a configuration file (`config.yaml`) to store settings like API keys, input/output directories, and importer-specific options.

### Creating a Configuration File

Create a `config.yaml` file in the project root directory with the following structure:

```yaml
# Global settings
output:
  markdown: "./markdown" # Directory for Markdown output
  json: "./json" # Directory for JSON output
overwrite: false # Whether to overwrite existing files

# Importer-specific settings
goodreads:
  csvfile: "./path/to/goodreads_export.csv"

imdb:
  csvfile: "./path/to/imdb_ratings.csv"
  apikey: "your_omdb_api_key" # Get from http://www.omdbapi.com/apikey.aspx

letterboxd:
  csvfile: "./path/to/letterboxd_export.csv"
  apikey: "your_omdb_api_key" # Same as IMDB if using OMDB

steam:
  apikey: "your_steam_api_key" # Get from https://steamcommunity.com/dev/apikey
  steamid: "your_steam_id" # Your 64-bit Steam ID
```

### API Keys

Some importers require API keys for data enrichment:

- **OMDB API** (for IMDb and Letterboxd importers):

  - Register at [OMDb API](http://www.omdbapi.com/apikey.aspx)
  - Free tier allows 1,000 requests per day

- **Steam API** (for Steam importer):
  - Get your API key from [Steam Dev](https://steamcommunity.com/dev/apikey)
  - Find your Steam ID using [SteamID Finder](https://steamidfinder.com/)

## Basic Usage

### Running an Import

To import data from a supported source:

```bash
# Import Goodreads data
./hermes goodreads --csvfile path/to/goodreads_export.csv

# Import IMDb ratings
./hermes imdb --csvfile path/to/imdb_ratings.csv --apikey your_omdb_api_key

# Import Letterboxd diary
./hermes letterboxd --csvfile path/to/letterboxd_export.csv --apikey your_omdb_api_key

# Import Steam library
./hermes steam --apikey your_steam_api_key --steamid your_steam_id
```

### Command-Line Options

Common options available for all importers:

- `--markdown-dir`: Directory for Markdown output (overrides config)
- `--json-dir`: Directory for JSON output (overrides config)
- `--overwrite`: Overwrite existing files (overrides config)
- `--verbose`: Enable verbose logging

### Output Files

After running an import, check the output directories:

- **Markdown files**: `./markdown/{importer_name}/`
- **JSON files**: `./json/{importer_name}/`

## Optional: Datasette Setup

Hermes can export your data to a local SQLite database or a remote Datasette instance for advanced querying and sharing.

### Local Datasette
1. Install [Datasette](https://datasette.io/) (requires Python):
   ```sh
   pip install datasette
   ```
2. After running an import, serve your database:
   ```sh
   datasette serve hermes.db
   ```
3. Open the provided URL in your browser to explore your data.

### Remote Datasette
1. Set up a remote Datasette instance with the [datasette-insert](https://github.com/simonw/datasette-insert) plugin.
2. Generate an API token for your user.
3. Configure Hermes with your remote URL and token (see [datasette_integration.md](./datasette_integration.md)).

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
