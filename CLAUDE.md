# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Note**: This project uses [bd (beads)](https://github.com/steveyegge/beads) for issue tracking. Use `bd` commands instead of markdown TODOs. See AGENTS.md for complete workflow details.

## Issue Tracking with bd (beads)

**Quick Reference:**

```bash
bd ready --json              # Check for unblocked work
bd create "Title" -t bug|feature|task -p 0-4 --json  # Create issue
bd update bd-42 --status in_progress --json          # Claim task
bd close bd-42 --reason "Completed" --json           # Complete task

# Complex create with all common flags
bd create "Add unit tests for TagCollector" -t task -p 2 -d "Test deduplication, AddIf conditions, GetSorted" -l "testing,fileutil" --deps "discovered-from:hermes-5sq"
```

**Issue Types:** `bug`, `feature`, `task`, `epic`, `chore`

**Priorities:** `0` (Critical) → `4` (Backlog), default is `2` (Medium)

**Workflow:**

1. Check ready work: `bd ready`
2. Claim task: `bd update <id> --status in_progress`
3. Work on it: Implement, test, document
4. Discover new work? Create linked issue with `--deps discovered-from:<parent-id>`
5. Complete: `bd close <id> --reason "Done"`
6. Commit: Always commit `.beads/issues.jsonl` with code changes

**Important Rules:**

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `history/` directory
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers

Refer to llm-shared/project_tech_stack.md for core technology choices, build system configuration, and library preferences.

## Key Commands

- `task` or `task build` - Build the application (runs tests and lint first)
- `task test` - Run tests with coverage report (generates coverage/coverage.html)
- `task lint` - Run golangci-lint
- `task clean` - Clean build artifacts
- `task upgrade-deps` - Update all dependencies
- `task build-linux` - Cross-compile for Linux
- `task test-ci` - Run tests for CI (with ci build tag)

**Running the application:**

- `./build/hermes --help` - View available commands
- `./build/hermes import goodreads --help` - View importer-specific options
- `./build/hermes enhance --help` - View enhance command options
- `go run . import goodreads -f goodreads_library_export.csv` - Run directly without building
- `go run . import goodreads --automated --headful` - Automated Goodreads export download (Chrome required)
- `go run . import goodreads --automated --dry-run` - Test automation without import
- `go run . enhance -d markdown/imdb --tmdb-generate-content` - Enhance existing notes with TMDB data

**Development workflow:**

- Tests must pass before builds (enforced by Taskfile dependencies)
- Always run `goimports -w .` after modifying Go files
- Use `go test ./...` for quick test runs during development

## Architecture Overview

Hermes is a data import/export tool that converts exports from various sources (Goodreads, IMDb, Letterboxd, Steam) into unified formats (JSON, Markdown, SQLite/Datasette). It also provides an `enhance` command to enrich existing markdown notes with TMDB data.

**Key Components:**

- `cmd/root.go` - Kong-based CLI structure with nested commands
- `cmd/{source}/` - Each data source has its own package (goodreads, imdb, letterboxd, steam)
- `cmd/enhance/` - Enhance existing markdown notes with TMDB data
- `internal/` - Shared utilities and packages
  - `cache/` - API response caching
  - `cmdutil/` - Command setup helpers
  - `config/` - Global configuration management
  - `datastore/` - SQLite/Datasette integration
  - `enrichment/` - TMDB enrichment functionality
  - `errors/` - Custom error types
  - `fileutil/` - File operations, markdown/JSON utilities
  - `tmdb/` - TMDB API client
  - `tui/` - Interactive terminal UI for TMDB selection

**Standard Importer Structure:**

```plain
cmd/{source}/
├── cmd.go          # Command setup and execution
├── parser.go       # Input data parsing
├── types.go        # Data models
├── {api}.go        # External API integration (e.g., omdb.go, openlibrary.go)
├── cache.go        # API response caching
├── json.go         # JSON output formatting
├── markdown.go     # Markdown output formatting
├── testdata/       # Test fixtures and expected outputs
└── *_test.go       # Unit tests
```

## CLI Framework (Kong)

- Uses Kong for command-line parsing (defined in `cmd/root.go`)
- CLI structure: `CLI` struct contains global flags and `ImportCmd` with subcommands
- Each importer command (GoodreadsCmd, IMDBCmd, etc.) implements a `Run() error` method
- Kong automatically handles help text, usage messages, and error handling
- Commands read from config file if CLI flags not provided (CLI flags take precedence)

**Adding a new importer command:**

1. Add command struct to `cmd/root.go` (e.g., `type NewSourceCmd struct`)
2. Add struct field to `ImportCmd` with cmd and help tags
3. Implement `Run() error` method that calls the importer package
4. Follow pattern of reading from config with CLI flag override

## Enhance Command

The `enhance` command enriches existing markdown notes with TMDB data without re-importing from original sources.

**Usage:**

```bash
# Basic usage - enhance all notes in a directory
./build/hermes enhance -d markdown/imdb

# Recursive scan with TMDB content generation
./build/hermes enhance -d markdown/letterboxd -r --tmdb-generate-content

# Download cover images and generate content
./build/hermes enhance -d markdown/imdb --tmdb-download-cover --tmdb-generate-content

# Interactive mode for manual TMDB selection
./build/hermes enhance -d markdown/letterboxd --tmdb-interactive

# Dry run to see what would be enhanced
./build/hermes enhance -d markdown/imdb --dry-run
```

**Key Features:**

- Scans directory for markdown files with YAML frontmatter
- Extracts title, year, and IMDB ID from existing notes
- Searches TMDB for matching content
- Updates frontmatter with TMDB ID, runtime, genres, etc.
- Optionally downloads cover images
- Optionally generates TMDB content sections (cast, crew, similar titles, etc.)
- Supports interactive TUI for selecting from multiple TMDB matches
- Skips notes that already have TMDB data (unless `--overwrite` flag is used)
- Dry-run mode to preview changes without modifying files

**Implementation:**

- `cmd/enhance/cmd.go` - Command logic, file discovery, and processing
- `cmd/enhance/parser.go` - YAML frontmatter parsing and markdown rebuilding
- Uses `internal/enrichment` for TMDB API integration
- Leverages existing TMDB client and TUI components

## Configuration

- Primary config: `config.yaml` (YAML format, auto-generated on first run)
- CLI flags override config file values
- Global settings in `cmd/root.go`: output directories, overwrite flag, datasette config
- Command-specific config uses namespaced keys (e.g., `goodreads.csvfile`, `steam.apikey`)
- Viper manages config loading with defaults in `initConfig()`

## Development Patterns

**Adding New Importers:**

1. Create new package under `cmd/{source}/`
2. Implement standard structure (cmd.go, parser.go, types.go, etc.)
3. Add command struct and Run() method to `cmd/root.go`
4. Add struct field to `ImportCmd` in `cmd/root.go`
5. Follow existing patterns for API integration, caching, and output formatting
6. Add tests in `testdata/` subdirectory

**Common Utilities:**

- `internal/cmdutil` - Command setup helpers
- `internal/fileutil` - File operations, markdown/JSON utilities (MarkdownBuilder pattern)
- `internal/config` - Global configuration management
- `internal/datastore` - SQLite/Datasette integration
- `internal/cache` - API response caching
- `internal/errors` - Custom error types (e.g., RateLimitError)

**API Integration:**

- All API responses cached in SQLite database (`cache.db` in project root, configurable via `cache.dbfile`)
- Cache TTL defaults to 720h (30 days), configurable via `cache.ttl`
- Implement API client logic within relevant command package (e.g., `cmd/goodreads/openlibrary.go`)
- Respect API rate limits using delays (`time.Sleep`) or by handling specific rate limit errors
- Handle common API errors gracefully (log warnings for 404s, retry or fail on persistent errors)
- Use existing patterns from OMDB or OpenLibrary integration

**Error Handling:**

- Use standard Go error handling (`errors.New`, `fmt.Errorf`)
- Return errors up the call stack for handling by Kong's command execution
- Wrap errors with context: `fmt.Errorf("failed to process item %s: %w", itemID, err)`
- Use custom error types from `internal/errors/` for specific conditions
- Log significant errors but return them to let top level handle exit codes

## Caching Patterns

Use the appropriate caching strategy from `internal/cache`:

- **`cache.GetOrFetch()`** - Cache all responses with global TTL (default 30 days)
  - Use for: Steam games, TMDB details by ID
  - Example: `cache.GetOrFetch("steam_cache", appID, fetchFunc)`

- **`cache.GetOrFetchWithPolicy()`** - Cache only certain responses (conditional)
  - Use for: TMDB searches (don't cache empty results)
  - Example: `cache.GetOrFetchWithPolicy("tmdb_cache", key, fetchFunc, shouldCache)`

- **`cache.GetOrFetchWithTTL()`** - Different TTLs for different result types (negative caching)
  - Use for: Goodreads books (7 days for "not found", 30 days for successful)
  - Helper: `cache.SelectNegativeCacheTTL(func(r *CachedResult) bool { return r.NotFound })`
  - Example: See `cmd/goodreads/cache.go` for reference implementation

**When adding new cached operations:**
1. Choose appropriate strategy (GetOrFetch, GetOrFetchWithPolicy, or GetOrFetchWithTTL)
2. Design deterministic cache keys (normalize if needed)
3. Add table schema to `internal/cache/schema.go`
4. Add table name to `ValidCacheTableNames` map
5. Add source to cache invalidation in `internal/cache/cmd.go`

**For detailed guidance:**
- Developer guide: `docs/cache-architecture.md`
- User documentation: `docs/caching.md`
- Architecture decisions: `docs/decisions/001-cache-architecture.md`

## Testing Requirements

- Write unit tests for parsing logic, API interaction, and output generation
- Test files in same package with `_test.go` suffix
- Tests must pass before builds (enforced by Taskfile)
- Use `testdata/` subdirectory within each command package for fixtures
- Employ table-driven tests for validating multiple input cases
- Use `//go:build !ci` to skip tests in CI that require external dependencies

## Datasette Integration

- Supports both local SQLite and remote Datasette storage
- `internal/datastore` provides unified interface
- Configuration under `datasette:` key in config (enabled, mode, dbfile, remote_url, api_token)
- Local mode writes to `hermes.db`, remote mode uses API
- Enable via `--datasette` flag or config file

## Logging

- Uses `log/slog` with custom handler from `github.com/lepinkainen/humanlog`
- Initialized in `cmd/root.go` with `slog.LevelInfo` default
- `InfoLevel` for progress messages (starting import, items processed)
- `DebugLevel` for verbose debugging info (API request/response details, cache hits/misses)
- `WarnLevel` for recoverable issues (skipping items but continuing)
- `ErrorLevel` for significant problems before returning an error

## Output Formats & Handling

- Default output directories (`markdown/`, `json/`) set in `cmd/root.go`, configurable via `config.yaml`
- Commands allow specifying subdirectories for output via `-o` flag (e.g., `markdown/goodreads/`)
- Use `internal/fileutil` for writing files (MarkdownBuilder for Markdown, WriteJSON for JSON)
- Markdown files use YAML frontmatter (Obsidian-compatible)
- Follow existing patterns for Markdown frontmatter and JSON structure for each data type
- Respect `--overwrite` flag logic

## Important Notes

- Go-only project (Go 1.24+)
- Follows standard Go project layout and idioms
- Each importer handles its own data enrichment and API integration
- Use `goimports -w .` after making changes (not gofmt)
- Use shared utility functions from `internal/` packages
- Build artifacts go to `build/` directory
- Cache stored in SQLite database (`cache.db`, safe to delete for cache invalidation)
