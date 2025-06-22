# Hermes Project Rules

## Tech Stack Reference

See `llm-shared/project_tech_stack.md` for core technology choices, build system configuration, and library preferences.

## Project-Specific Guidelines

- **Unit Tests:** Write basic unit tests for new functionality. Always check the tests pass after finishing changes.
- **Purpose:** See README.md for project definition

## Code Structure

- All import processors go under the `cmd/` directory
- The internal/ directory contains common utility functions, use it when possible. Add to it when necessary.
- Each processor has its own subdirectory and Go package

## Implementation Requirements

- Maintain consistent Go style and idiomatic patterns
- Follow the existing architectural patterns
- Each data source processor should be implemented as a separate command

## Error Handling

- Use standard Go error handling (`errors.New`, `fmt.Errorf`). Return errors up the call stack for handling by Cobra's `RunE`.
- Wrap errors with context where appropriate to aid debugging (e.g., `fmt.Errorf("failed to process item %s: %w", itemID, err)`).
- Utilize custom error types from `internal/errors` for specific conditions like API rate limits when applicable.
- Log significant errors within the command logic but generally return them to let the top level (`main.go` or Cobra) handle exit codes.

## Configuration Management

- Configuration is managed via Viper, reading `config.yaml` by default.
- Global settings (like output directories, overwrite flag) are defined in `root.go` and accessed via `internal/config` or Viper directly.
- Command-specific configuration (e.g., input file paths, API keys) should use keys namespaced by the command name in `config.yaml` (e.g., `goodreads.csvfile`, `steam.apikey`).
- Prioritize command-line flags over config file values when both are provided.

## Logging

- Log informational messages about progress (e.g., starting import, items processed) at `InfoLevel`.
- Use `DebugLevel` for verbose information useful for debugging (e.g., detailed API request/response info, cache hits/misses).
- Use `WarnLevel` for recoverable issues (e.g., skipping an item due to missing data but continuing the import).
- Use `ErrorLevel` for significant problems encountered within functions, often just before returning an error.

## Caching

- Use the `cache/` directory for caching external API responses.
- Organize cache files into subdirectories named after the data source (e.g., `cache/goodreads/`, `cache/omdb/`).
- Implement caching logic within the specific command package (e.g., `cmd/goodreads/cache.go`).
- Respect API rate limits using appropriate delays or by handling specific rate limit errors.

## Output Formats & Handling

- Default output directories (`markdown/`, `json/`) are set in `root.go` and configurable via `config.yaml`.
- Commands should allow specifying subdirectories for their output via flags/config (e.g., `markdown/goodreads/`).
- Use `internal/fileutil` for writing Markdown and JSON files, ensuring consistent formatting and handling the `overwrite` flag logic.
- Follow existing patterns for Markdown frontmatter and JSON structure for each data type.

## External API Interaction

- Implement API client logic within the relevant command package (e.g., `cmd/goodreads/openlibrary.go`).
- Respect API rate limits using appropriate delays (`time.Sleep`) or by handling specific rate limit errors.
- Utilize caching (`cache/`) to minimize redundant API calls.
- Handle common API errors gracefully (e.g., log a warning for 404 Not Found, retry or fail on persistent errors).

## Testing

- Write unit tests for parsing logic, API interaction (using mocks/stubs), and output generation.
- Place tests in `_test.go` files within the same package.
- Use the `testdata/` subdirectory within each command package for input fixtures and expected output files.
- Employ table-driven tests for validating multiple input cases efficiently.

## Utilities

- Use shared utility functions from `internal/` packages (e.g., `cmdutil` for command setup, `fileutil` for file operations).
- Contribute reusable logic back to these `internal/` packages when appropriate.

## Documentation

- Write Go doc comments for all exported functions, types, and constants.
- Keep command help messages (`Short`, `Long` fields in `cobra.Command`) clear and up-to-date.
- Update `README.md` and any relevant files in `docs/` when adding new commands or changing functionality significantly.
