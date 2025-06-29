# Architecture

This document describes the high-level architecture of Hermes, explaining how the different components work together to import, enrich, and format data from various sources.

## Project Structure

Hermes follows a modular architecture with clear separation of concerns:

```
hermes/
├── cmd/                  # Command implementations
│   ├── root.go           # Root command and global flags
│   ├── import.go         # Parent import command
│   ├── goodreads/        # Goodreads importer
│   ├── imdb/             # IMDb importer
│   ├── letterboxd/       # Letterboxd importer
│   └── steam/            # Steam importer
├── internal/             # Internal shared packages
│   ├── cmdutil/          # Command utilities
│   ├── config/           # Configuration handling
│   ├── datastore/        # Local SQLite and remote Datasette integration
│   ├── errors/           # Custom error types
│   └── fileutil/         # File operations utilities
├── build/                # Build artifacts
├── cache/                # API response cache
├── json/                 # JSON output directory
├── markdown/             # Markdown output directory
├── main.go               # Application entry point
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── config.yaml           # Configuration file
└── Taskfile.yml          # Task runner configuration
```

## Core Components

### Command-Line Interface (CLI)

Hermes uses the Cobra library to implement its command-line interface:

- **Root Command** (`cmd/root.go`): Defines global flags and configuration
- **Import Command** (`cmd/import.go`): Parent command for all importers
- **Importer Commands** (`cmd/{source}/cmd.go`): Source-specific import commands

### Configuration Management

Configuration is handled using Viper:

- Reads from `config.yaml` by default
- Supports environment variables
- Command-line flags override config file values
- Global settings are defined in `root.go`
- Source-specific settings are namespaced (e.g., `goodreads.csvfile`)

### Importer Architecture

Each importer follows a similar structure:

```
cmd/{source}/
├── cmd.go          # Command registration and execution
├── parser.go       # Data parsing logic
├── types.go        # Data models
├── api.go          # External API integration
├── cache.go        # Caching implementation
├── json.go         # JSON output formatter
├── markdown.go     # Markdown output formatter
└── testdata/       # Test files
```

The typical data flow within an importer is:

1. **Command Execution** (`cmd.go`): Parses flags, reads config, and orchestrates the import process
2. **Data Parsing** (`parser.go`): Reads and parses the input data (CSV, API response)
3. **Data Enrichment** (e.g., `openlibrary.go`, `omdb.go`): Fetches additional metadata from external APIs
4. **Output Generation** (`json.go`, `markdown.go`): Formats the enriched data for output

### Shared Utilities

The `internal/` directory contains shared utilities used across importers:

- **cmdutil**: Helper functions for setting up commands
- **config**: Configuration handling utilities
- **errors**: Custom error types (e.g., rate limit errors)
- **fileutil**: File operations, including Markdown and JSON formatting

## Datastore & Datasette Integration

Hermes supports exporting data to a local SQLite database or a remote Datasette instance. The `internal/datastore` package abstracts both storage backends, and importers can write to either based on configuration. This enables advanced querying and sharing of your imported data.

- **Local Mode:** Data is written to a SQLite file (default: `hermes.db`).
- **Remote Mode:** Data is sent to a remote Datasette instance using the `datasette-insert` plugin and API token.

## Data Flow

The general data flow in Hermes follows these steps:

```
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
- Custom human-readable formatter in `internal/humanlog`
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

- **Kong**: Command-line interface
- **Viper**: Configuration management
- **slog**: Structured logging (Go standard library)
- **Various API clients**: For data enrichment

## Next Steps

- See [Configuration](04_configuration.md) for detailed configuration options
- See [Output Formats](05_output_formats.md) for information about the output formats
- See [Caching](06_caching.md) for details on the caching implementation
