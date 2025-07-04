# CLAUDE.md

Refer to llm-shared/project_tech_stack.md for core technology choices, build system configuration, and library preferences.

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Key Commands

- `task` or `task build` - Build the application
- `task test` - Run tests with coverage
- `task lint` - Run Go linters (requires golangci-lint)
- `task clean` - Clean build artifacts
- `task upgrade-deps` - Update all dependencies

**Running the application:**

- `./hermes --help` - View available commands
- `./hermes import goodreads --help` - View importer-specific options
- `go run . import goodreads -f goodreads_library_export.csv` - Run directly

**Testing:**

- Tests run automatically before builds
- Use `go test ./...` for quick test runs

## Architecture Overview

Hermes is a data import/export tool that converts exports from various sources (Goodreads, IMDb, Letterboxd, Steam) into unified formats (JSON, Markdown, SQLite/Datasette).

**Key Components:**

- `cmd/` - CLI commands
- `internal/` - Shared utilities and packages
- Each data source has its own package under `cmd/{source}/`

**Standard Importer Structure:**

```
cmd/{source}/
├── cmd.go          # Command setup and execution
├── parser.go       # Input data parsing
├── types.go        # Data models
├── {api}.go        # External API integration (e.g., omdb.go, openlibrary.go)
├── cache.go        # API response caching
├── json.go         # JSON output formatting
├── markdown.go     # Markdown output formatting
└── *_test.go       # Unit tests
```

## Configuration

- Primary config: `config.yml` (YAML format)
- CLI flags override config file values
- Global settings (output directories, overwrite flag) defined in `root.go`
- Command-specific config uses namespaced keys (e.g., `goodreads.csvfile`, `steam.apikey`)
- Config auto-generated if missing on first run

## Development Patterns

**Adding New Importers:**

1. Create new package under `cmd/{source}/`
2. Implement the standard structure (cmd.go, parser.go, types.go, etc.)
3. Register in `cmd/root.go` using Kong's command structure
4. Follow existing patterns for API integration, caching, and output formatting

**Common Utilities:**

- `internal/cmdutil` - Command setup helpers
- `internal/fileutil` - File operations, markdown/JSON utilities
- `internal/config` - Global configuration management
- `internal/datastore` - SQLite/Datasette integration

**API Integration:**

- All API responses cached under `cache/{source}/` (organized by data source)
- Implement API client logic within relevant command package (e.g., `cmd/goodreads/openlibrary.go`)
- Respect API rate limits using delays (`time.Sleep`) or by handling specific rate limit errors
- Handle common API errors gracefully (log warnings for 404s, retry or fail on persistent errors)
- Use existing patterns from OMDB or OpenLibrary integration

**Error Handling:**

- Use standard Go error handling (`errors.New`, `fmt.Errorf`). Return errors up the call stack for handling by Kong's command execution
- Wrap errors with context: `fmt.Errorf("failed to process item %s: %w", itemID, err)`
- Use custom error types from `internal/errors/` for specific conditions like API rate limits
- Log significant errors within command logic but return them to let top level handle exit codes

## Testing Requirements

- Write unit tests for parsing logic, API interaction (using mocks/stubs), and output generation
- Test files in same package with `_test.go` suffix
- Tests must pass before builds (enforced by Taskfile)
- Use `testdata/` subdirectory within each command package for input fixtures and expected output
- Employ table-driven tests for validating multiple input cases efficiently

## Datasette Integration

- Supports both local SQLite and remote Datasette storage
- `internal/datastore` provides unified interface
- Configuration under `datasette:` key in config
- Local mode writes to `hermes.db`, remote mode uses API

## Logging

- `InfoLevel` for progress messages (starting import, items processed)
- `DebugLevel` for verbose debugging info (API request/response details, cache hits/misses)
- `WarnLevel` for recoverable issues (skipping items due to missing data but continuing)
- `ErrorLevel` for significant problems, often just before returning an error

## Output Formats & Handling

- Default output directories (`markdown/`, `json/`) set in `root.go`, configurable via `config.yaml`
- Commands should allow specifying subdirectories for output via flags/config (e.g., `markdown/goodreads/`)
- Use `internal/fileutil` for writing files, ensuring consistent formatting and handling `overwrite` flag logic
- Follow existing patterns for Markdown frontmatter and JSON structure for each data type


## Documentation

- Write Go doc comments for all exported functions, types, and constants
- Keep command help messages (`Help` field in Kong commands) clear and up-to-date
- Update `README.md` and relevant files in `docs/` when adding new commands or changing functionality

## Important Notes

- Go-only project (no Python or other languages)
- Follows standard Go project layout and idioms
- Uses modern Go features (Go 1.24+)
- Each importer handles its own data enrichment and API integration
- Maintain consistent Go style and idiomatic patterns
- Use shared utility functions from `internal/` packages, contribute reusable logic back when appropriate
