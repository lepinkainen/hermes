# Gemini Agent Guide for Hermes

This guide provides essential information for developing in the Hermes codebase.

## Project Overview & Architecture

Hermes is a Go-based CLI tool for importing data from sources like Goodreads, IMDb, and Steam, and exporting it into Markdown, JSON, or SQLite/Datasette formats.

- **Entrypoint**: `main.go` calls `cmd.Execute()` to start the Kong CLI application.
- **Commands (`cmd/`)**: Each data importer is a self-contained package within a subdirectory (e.g., `cmd/goodreads/`, `cmd/steam/`). This is the primary location for adding or modifying importer logic.
- **Shared Logic (`internal/`)**: Contains reusable packages for common functionality:
    - `config`: Viper-based configuration management.
    - `fileutil`: Helpers for writing Markdown and JSON files.
    - `datastore`: SQLite and Datasette integration.
    - `errors`: Custom error types (e.g., for rate limiting).
- **Configuration**: Managed via a `config.yaml` file. CLI flags take precedence over config file settings.
- **Output**: Data is written to `json/` and `markdown/` directories by default.
- **Caching**: API responses are cached in the `cache/` directory, with subdirectories for each importer, to minimize external calls.

## Developer Workflow

The project uses `Taskfile.yml` for task automation.

- **Build & Test**: `task build` - This is the primary command for development. It automatically runs tests, lints the code, and compiles the binary to `build/hermes`.
- **Run Tests**: `task test` - Runs all tests and generates a coverage report in `coverage/`.
- **Lint Code**: `task lint` - Runs `golangci-lint`.
- **Run the CLI**: For development, use `go run . <command> [flags]`. For example: `go run . import goodreads -f path/to/export.csv`.

## Key Development Patterns

- **Adding a New Importer**:
    1. Create a new package under `cmd/`.
    2. Mimic the structure of an existing importer (e.g., `cmd/goodreads`):
        - `cmd.go`: Kong command definition.
        - `parser.go`: Logic for parsing the source data file.
        - `types.go`: Structs for the data models.
        - `api.go` (or similar): Client for external APIs (e.g., OMDB, OpenLibrary).
        - `cache.go`, `json.go`, `markdown.go`: Handlers for caching and output formats.
    3. Add the new command to `cmd/root.go`.

- **Error Handling**:
    - Return errors up the call stack.
    - Wrap errors with context using `fmt.Errorf("...: %w", err)` to provide a clear trace.
    - Use custom error types from `internal/errors` where applicable.

- **Utilities**:
    - Always use helpers from `internal/` for common tasks like file writing (`fileutil`) and configuration (`config`).
    - Contribute new, reusable logic back to the `internal/` packages.

- **Dependencies**:
    - The project uses Go modules. Key libraries include `kong` for the CLI, `viper` for configuration, and `modernc.org/sqlite` for the database. Add new dependencies to `go.mod` only when necessary.

## Gemini Agent Specific Notes
- Always use `task build` to build the project.