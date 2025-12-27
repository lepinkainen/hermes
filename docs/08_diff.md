# Diff Reports

Hermes can generate Obsidian-friendly diff reports that compare imported data across sources.

## IMDb vs Letterboxd

`hermes diff imdb-letterboxd` compares the movie rows stored in SQLite for IMDb and Letterboxd and produces a checklist-style markdown report.

### What it reports

- **IMDb-only**: IMDb movies that are not present in the Letterboxd import
- **Letterboxd-only**: Letterboxd movies that are not present in IMDb
- **Fuzzy matches**: Title + year matches when IMDb IDs are missing on one side

TV titles are excluded from the IMDb side. Matching is done by IMDb ID first, then by normalized title + year.

### Usage

```bash
./hermes diff imdb-letterboxd
```

### Output

By default the report is written to:

```
markdown/diffs/imdb_letterboxd_diff-YYYY-MM-DD.md
```

The report is Obsidian-compatible and uses checkboxes so you can track manual reconciliation:

```markdown
## IMDb-only (missing from Letterboxd)

- [ ] The Matrix (1999) — IMDb tt0133093 — https://www.imdb.com/title/tt0133093/ — IMDb rating 9/10
  - Possible matches (title + year):
    - The Matrix (1999) — https://letterboxd.com/film/the-matrix/ — Letterboxd rating 4.5/5
```

### Configuration

- **Main DB**: Defaults to `datasette.dbfile` (usually `./hermes.db`).
- **Cache DB**: Defaults to `cache.dbfile` (usually `./cache.db`).

The diff uses `cache.db` to resolve missing IMDb IDs from the Letterboxd mapping cache.

### Flags

- `--output` (`-o`) - Output markdown file path
- `--db-file` - Path to the main SQLite database
- `--cache-db-file` - Path to the cache SQLite database (global flag)
