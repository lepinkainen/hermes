# Diff Reports

Hermes can generate Obsidian-friendly diff reports that compare imported data across sources.

## IMDb vs Letterboxd

`hermes diff imdb-letterboxd` compares the movie rows stored in SQLite for IMDb and Letterboxd and produces a checklist-style markdown report.

### What it reports

- **IMDb-only**: IMDb movies that are not present in the Letterboxd import
- **Letterboxd-only**: Letterboxd movies that are not present in IMDb
- **Auto-resolved matches**: Exact title + year matches are treated as definite and removed from the mismatch lists
- **Fuzzy matches**: Additional title + year candidates are shown when there are multiple possibilities

TV titles are excluded from the IMDb side. Matching is done by IMDb ID first, then auto-resolved title + year matches, with fuzzy suggestions shown for ambiguous cases.

### Usage

```bash
# Generate markdown report (default)
./hermes diff imdb-letterboxd

# Generate HTML report
./hermes diff imdb-letterboxd --html diff.html

# Generate both markdown and HTML
./hermes diff imdb-letterboxd -o diff.md --html diff.html
```

### Output Formats

#### Markdown (default)

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

#### HTML

The HTML output creates a self-contained, interactive web page designed for tracking cross-adding movies between platforms:

```bash
./hermes diff imdb-letterboxd --html markdown/diffs/diff.html
```

**Features:**

- **Two-column layout**: IMDb-only movies on the left, Letterboxd-only on the right
- **Interactive checkboxes**: Track which movies you've added
- **Persistent state**: Checkbox state is saved in localStorage (survives browser restarts)
- **Search links**: Each item has an "Add to Letterboxd" or "Add to IMDb" button that opens the relevant search page in a new tab
- **Fuzzy matches**: Possible matches are shown inline for manual verification
- **Progress indicators**: Shows how many items you've completed in each section
- **Filter options**: Show all items or only unchecked ones

**Workflow:**

1. Generate the HTML report
2. Open in a browser and work through the checklist
3. Click search links to add movies to the other platform
4. Check items off as you complete them
5. After adding movies, re-import from both sources and regenerate the report
6. The new report will be smaller, and previously checked items remain checked

### Configuration

- **Main DB**: Defaults to `datasette.dbfile` (usually `./hermes.db`).
- **Cache DB**: Defaults to `cache.dbfile` (usually `./cache.db`).

The diff uses `cache.db` to resolve missing IMDb IDs from the Letterboxd mapping cache.

### Flags

- `--output` (`-o`) - Output markdown file path
- `--html` (`-H`) - Output HTML file path
- `--db-file` - Path to the main SQLite database
- `--cache-db-file` - Path to the cache SQLite database (global flag)
