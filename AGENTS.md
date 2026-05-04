# Repository Guidelines

### Managing AI-Generated Planning Documents

AI assistants often create planning and design documents during development:

- PLAN.md, IMPLEMENTATION.md, ARCHITECTURE.md
- DESIGN.md, CODEBASE_SUMMARY.md, INTEGRATION_PLAN.md
- TESTING_GUIDE.md, TECHNICAL_DESIGN.md, and similar files

**Best Practice: Use a dedicated directory for these ephemeral files**

**Recommended approach:**

- Create a `history/` directory in the project root
- Store ALL AI-generated planning/design docs in `history/`
- Keep the repository root clean and focused on permanent project files
- Only access `history/` when explicitly asked to review past planning

**Example .gitignore entry (optional):**

```
# AI planning documents (ephemeral)
history/
```

**Benefits:**

- ✅ Clean repository root
- ✅ Clear separation between ephemeral and permanent documentation
- ✅ Easy to exclude from version control if desired
- ✅ Preserves planning history for archeological research
- ✅ Reduces noise when browsing the project

### Important Rules

- ✅ Store AI planning docs in `history/` directory
- ✅ Always run `task build` before claiming work is done
- ❌ Do NOT clutter repo root with planning documents

### LLM Reference

Need a quick tour of the shared helpers under `internal/`? Read `docs/internal_llm_reference.md` for package-by-package guidance before writing new utilities.

## Project Structure & Module Organization

- `main.go` wires the CLI and dispatches importer subcommands.
- `cmd/` hosts CLI entrypoints per provider (e.g. `cmd/goodreads`, `cmd/steam`).
- `internal/` contains shared services: `cache` for local stores, `datastore` for SQLite/JSON writers, `config` for settings.
- `docs/` is the canonical reference; update it alongside behaviour changes and new flags.
- Generated build and coverage artifacts live in `build/` and `coverage/`; sample exports under `exports/` and `json/` support local runs but keep large fixtures out of commits.

## Caching

- Hermes caches provider responses in `cache.db` (SQLite) in the repo root; it is safe to delete and is separate from `hermes.db`.
- Default TTL is `720h` (30 days); override with `--cache-db-file`, `--cache-ttl`, or env vars `CACHE_DBFILE`/`CACHE_TTL`.
- Tables are created automatically per provider (`omdb_cache`, `openlibrary_cache`, `steam_cache`, `letterboxd_cache`, `tmdb_cache`); entries past TTL refresh on next use and malformed entries are retried.
- Warm caches by running the relevant importer once; invalidate selectively with `hermes cache invalidate tmdb|omdb|steam|letterboxd|openlibrary` or delete `cache.db` to clear everything.
- Legacy JSON caches under `cache/` are deprecated and can be removed; negative TMDB results are intentionally not cached to allow future discoveries.

## Build, Test, and Development Commands

- `task build` runs lint, tests, and produces `build/hermes` with the current Git SHA embedded.
- `task test` executes `go test -race -coverprofile=coverage/coverage.out ./...` and emits `coverage/coverage.html` for review.
- `task lint` wraps `golangci-lint run ./...`; resolve findings before opening a PR.
- `go run ./cmd/root.go --help` is a quick sanity check for new flags; swap in a provider folder (e.g. `./cmd/goodreads`) to exercise importer flows.

## Coding Style & Naming Conventions

- Format Go sources with `gofmt` or goimports integrations; Go defaults to tab-indentation, so avoid manual overrides.
- Keep package names lowercase and singular; exported identifiers use UpperCamelCase, unexported ones use lowerCamelCase.
- Prefer context-aware logging through the `humanlog` helpers and centralize config lookups in `internal/config` to keep importer packages lean.

## Testing Guidelines

- Co-locate `_test.go` files with the code under test; favour table-driven cases and `testify` assertions for clarity.
- Run `task test` (or `go test ./...` when iterating) before pushing; inspect `coverage/coverage.html` for critical paths such as `internal/datastore` or importer pipelines.
- Store lightweight fixtures under package-level `testdata/` directories and avoid reusing the large exports shipped at the repo root.

## Commit & Pull Request Guidelines

- Follow the existing Title-Case, imperative commit style (`Refactor caching`, `Add Steam importer config`) and keep each commit focused.
- PRs should explain the motivation, list manual verification steps, and link issues; attach screenshots or sample output when behaviour is user-visible.
- Before requesting review, ensure lint/tests pass, docs in `docs/` reflect the change, and configuration updates reference `config.yml` or `.env` expectations.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **Note remaining work** - Capture anything that needs follow-up in the handoff
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
4. **Clean up** - Clear stashes, prune remote branches
5. **Verify** - All changes committed AND pushed
6. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
