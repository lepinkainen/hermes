# Architecture

This document describes the high-level architecture of Hermes, explaining how the different components work together to import, enrich, and format data from various sources.

## Project Structure

Hermes follows a modular architecture with clear separation of concerns:

```
hermes/
├── cmd/                      # CLI entrypoints
│   ├── root.go               # Root command + global flags
│   ├── enhance/              # Enhance command implementation
│   ├── goodreads/            # Goodreads importer
│   ├── imdb/                 # IMDb importer
│   ├── letterboxd/           # Letterboxd importer
│   └── steam/                # Steam importer
├── internal/                 # Shared services
│   ├── cmdutil/              # Command helpers (output dirs, Kong wiring)
│   ├── config/               # Global configuration state
│   ├── content/              # Markdown content builders (TMDB sections)
│   ├── enrichment/           # TMDB orchestrator used by enhance/importers
│   ├── importer/             # Reusable importer helpers (media IDs, enrichment glue)
│   ├── datastore/            # SQLite/Datasette writer
│   ├── errors/, fileutil/, … # Support packages
│   └── tmdb/                 # Low-level TMDB API client + caching
├── llm-shared/               # Utility CLIs/tests shared with AI tooling
├── docs/                     # Project documentation (this folder)
├── history/                  # Scratch space for planning documents
├── build/, json/, markdown/  # Generated outputs
├── config.yml                # Sample configuration
├── Taskfile.yml              # Task runner recipes
└── main.go                   # CLI entry point
```

## Core Components

### Command-Line Interface (CLI)

Hermes uses the Kong library to implement its command-line interface:

- **Root Command** (`cmd/root.go`): Defines global flags, configuration bootstrap, and shared options (datasette and cache settings)
- **Import Command Group** (`hermes import ...`): Contains source-specific commands (`goodreads`, `imdb`, `letterboxd`, `steam`), each of which wires CLI flags into its package-level parser
- **Enhance Command** (`hermes enhance ...`): Walks existing Markdown notes and invokes the TMDB enrichment pipeline to backfill metadata/content
- Each importer exposes a `Parse{Source}WithParams` function that the Kong bindings call, which keeps CLI plumbing decoupled from parser logic and now exposes test hooks for coverage

### Configuration Management

Configuration is handled using Kong's built-in configuration system:

- Command-line flags define all configuration options
- Global settings are defined in `root.go`
- Source-specific settings are defined in their respective command structures
- Datasette configuration options are available globally

### Importer & Enhance Architecture

Each importer follows a similar structure:

```plain
cmd/{source}/
├── cmd.go              # Command registration and execution (Kong wrapper)
├── parser.go           # Data parsing logic
├── api/openlibrary.go  # External API integration (where applicable)
├── cache.go            # Cache lookups using internal/cache helpers
├── json.go / markdown.go
├── cmd_test.go, *_test.go
└── testdata/           # CSV or fixture files
```

The typical data flow within an importer is:

1. **Command Execution** (`cmd.go`): Parses flags, reads config, and orchestrates the import process
2. **Data Parsing** (`parser.go`): Reads and parses the input data (CSV, API response)
3. **Data Enrichment** (e.g., `openlibrary.go`, `omdb.go`): Fetches additional metadata from external APIs
4. **Output Generation** (`json.go`, `markdown.go`): Formats the enriched data for output

The `enhance` command reuses the same enrichment pipeline but starts from parsed Markdown files, leveraging `internal/importer/mediaids` to load existing IDs, `internal/enrichment` to query TMDB, and `internal/content` to splice new sections into the body.

### Shared Utilities

The `internal/` directory contains shared utilities used across importers:

- **cmdutil/config**: Helper functions for output directories and global config state
- **content**: Build standardized TMDB content sections/markers used by both importers and enhance
- **enrichment**: Wraps TMDB lookups, cover downloads, and content generation
- **importer/**: Shared enrichment helpers (`mediaids`, `enrich` smart pipeline)
- **errors**: Custom error types (e.g., rate limit errors)
- **fileutil**: File operations, including Markdown and JSON formatting

## Datastore & Datasette Integration

Hermes exports data to a local SQLite database (default: `hermes.db`) for exploration in Datasette. The `internal/datastore` package provides the SQLite store used by all importers, and datasette flags/config values live at the root CLI/config layer.

## Data Flow

The general data flow in Hermes follows these steps:

```plain
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│             │     │             │     │             │     │             │
│  Input Data │────►│    Parser   │────►│  Enrichment │────►│  Formatter  │
│             │     │             │     │             │     │             │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                                              │                    │
                                              ▼                    ▼
                                        ┌─────────────┐     ┌─────────────┐
                                        │             │     │             │
                                        │    Cache    │     │   Output    │
                                        │             │     │             │
                                        └─────────────┘     └─────────────┘
```

1. **Input Data**: Source-specific data (CSV files, API responses)
2. **Parser**: Converts raw data into internal data structures
3. **Enrichment**: Fetches additional metadata from external APIs
4. **Cache**: Stores API responses to respect rate limits and improve performance
5. **Formatter**: Converts enriched data to JSON and Markdown formats
6. **Output**: Writes formatted data to the specified output directories

## Error Handling

Hermes follows standard Go error handling practices:

- Errors are returned up the call stack
- Errors are wrapped with context using `fmt.Errorf("context: %w", err)`
- Custom error types are used for specific conditions (e.g., API rate limits)
- Significant errors are logged before being returned

## Logging

Logging is implemented using Go's standard `log/slog` package:

- Configured globally in `cmd/root.go`
- Different log levels are used for different types of messages:
  - `InfoLevel`: Progress information
  - `DebugLevel`: Detailed debugging information
  - `WarnLevel`: Recoverable issues
  - `ErrorLevel`: Significant problems

## Caching

To respect API rate limits and improve performance, Hermes implements caching:

- API responses are cached in the `cache/` directory
- Cache files are organized by data source
- Cache implementation is specific to each importer
- Cache can be bypassed using command-line flags

## Testing

Hermes includes unit tests for critical components:

- Parser tests validate the correct parsing of input data
- API client tests verify correct interaction with external APIs
- Output formatter tests ensure consistent output generation
- Test data is stored in `testdata/` directories within each importer

## Dependencies

Hermes relies on several external dependencies:

- **Kong**: Command-line interface and configuration management
- **slog**: Structured logging (Go standard library)
- **Various API clients**: For data enrichment

## Next Steps

- See [Configuration](04_configuration.md) for detailed configuration options
- See [Output Formats](05_output_formats.md) for information about the output formats
- See [Caching](06_caching.md) for details on the caching implementation
