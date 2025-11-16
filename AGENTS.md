# Repository Guidelines

## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**
```bash
bd ready --json
```

**Create new issues:**
```bash
bd create "Issue title" -t bug|feature|task -p 0-4 --json
bd create "Issue title" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**
```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**
```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`
6. **Commit together**: Always commit the `.beads/issues.jsonl` file together with the code changes so issue state stays in sync with code state

### Auto-Sync

bd automatically syncs with git:
- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### MCP Server (Recommended)

If using Claude or MCP-compatible clients, install the beads MCP server:

```bash
pip install beads-mcp
```

Add to MCP config (e.g., `~/.config/claude/config.json`):
```json
{
  "beads": {
    "command": "beads-mcp",
    "args": []
  }
}
```

Then use `mcp__beads__*` functions instead of CLI commands.

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

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `history/` directory
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems
- ❌ Do NOT clutter repo root with planning documents

For more details, see README.md and QUICKSTART.md.

## Project Structure & Module Organization
- `main.go` wires the CLI and dispatches importer subcommands.
- `cmd/` hosts CLI entrypoints per provider (e.g. `cmd/goodreads`, `cmd/steam`).
- `internal/` contains shared services: `cache` for local stores, `datastore` for SQLite/JSON writers, `config` for settings.
- `docs/` is the canonical reference; update it alongside behaviour changes and new flags.
- Generated build and coverage artifacts live in `build/` and `coverage/`; sample exports under `exports/` and `json/` support local runs but keep large fixtures out of commits.

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
