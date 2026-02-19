# Importer Deduplication Plan

## Overview

This plan addresses the five high-priority code duplication issues identified across the four importers (goodreads, imdb, letterboxd, steam). Each issue is tracked as a separate GitHub issue.

## Issue 1: Eliminate Redundant JSON Wrapper Functions

**Problem:** All four importers have identical single-line wrapper functions around `fileutil.WriteJSONFile()`:

- `cmd/goodreads/json.go:8` — `writeBookToJson()`
- `cmd/imdb/json.go:8` — `writeMovieToJson()`
- `cmd/letterboxd/json.go:9` — `writeJSONFile()`
- `cmd/steam/json.go:8` — `writeGameToJson()`

Each is a trivial `func(items, filename) error` that calls `fileutil.WriteJSONFile(items, filename, config.OverwriteFiles)`.

**Plan:**

1. Find all call sites for each wrapper function.
2. Replace each call with a direct `fileutil.WriteJSONFile()` call.
3. Delete the four `json.go` wrapper files (keeping only test files).
4. Update any tests that reference the wrapper functions.
5. Note: Letterboxd has an extra `writeMoviesToJSON()` in `parser.go:157-179` that does enrichment pre-processing before calling `writeJSONFile()` — this one stays but calls `fileutil.WriteJSONFile()` directly.

**Files changed:** 8 files (4 deleted, 4 call sites updated)

---

## Issue 2: Extract Shared Cover Download Handler

**Problem:** All four importers have near-identical cover download logic:

- `cmd/goodreads/markdown.go:119-154` — `buildCoverContent()`
- `cmd/imdb/markdown.go:111-129` — inline in `writeMovieToMarkdown()`
- `cmd/letterboxd/markdown.go:91-109` — inline in `writeMovieToMarkdown()`
- `cmd/steam/markdown.go:59-72` — inline in `CreateMarkdownFile()`

The pattern is: check TMDB enrichment cover → fall back to source URL → call `fileutil.DownloadCover()` → set frontmatter `cover` field → return filename for body embed.

**Plan:**

1. Create `internal/obsidian/cover.go` with a shared `ResolveCover()` function:
   ```go
   type CoverSource struct {
       TMDBCoverPath     string // from enrichment, if available
       TMDBCoverFilename string // from enrichment, if available
       FallbackURL       string // source-specific poster/header URL
   }

   type CoverResult struct {
       Filename     string // for body embed
       RelativePath string // for frontmatter "cover" field
   }

   func ResolveCover(source CoverSource, title, directory string) *CoverResult
   ```
2. The function handles the TMDB-first priority logic and `fileutil.DownloadCover()` fallback.
3. Replace inline cover logic in all four importers with `obsidian.ResolveCover()`.
4. Goodreads has a unique fallback (OpenLibrary cover URL construction) — pass the constructed URL as `FallbackURL`.
5. Apply the `CoverResult` to frontmatter and body in each importer (2-3 lines each vs 15-35 lines today).
6. Remove the `coverContent` struct from goodreads (no longer needed).

**Files changed:** 5 files (1 new, 4 modified)

---

## Issue 3: Extract Shared Tag Helper Functions

**Problem:** Three tag patterns are duplicated across importers:

**Decade tags** (3 implementations, 2 inconsistent):
- `cmd/goodreads/markdown.go:106-109` — arithmetic: `(year/10)*10`
- `cmd/imdb/markdown.go:82-85` — arithmetic: `(year/10)*10`
- `cmd/letterboxd/markdown.go:66,172-193` — switch-case `getDecadeTag()` (returns `"year/pre-1950s"` for old years, others don't)
- `internal/fileutil/markdown.go:96-117` — `MarkdownBuilder.GetDecadeTag()` (same switch-case, unused standalone)

**Rating tags** (3 implementations):
- `cmd/goodreads/markdown.go:103` — `rating/%.0f` (float formatting)
- `cmd/imdb/markdown.go:77-79` — `rating/%d` (int formatting)
- `cmd/letterboxd/markdown.go:61-63` — `rating/%d` with `math.Round()`

**Genre tags** (2 implementations):
- `cmd/imdb/markdown.go:88-90` — `genre/%s` loop
- `cmd/letterboxd/markdown.go:69-71` — `genre/%s` loop (identical)

**Plan:**

1. Add helper methods to `internal/obsidian/tags.go` on `TagSet`:
   ```go
   func (ts *TagSet) AddDecadeTag(year int)       // uses arithmetic approach: (year/10)*10, handles year <= 0
   func (ts *TagSet) AddRatingTag(rating float64)  // rating/N with rounding, skips if <= 0
   func (ts *TagSet) AddGenreTags(genres []string)  // genre/X for each genre
   ```
2. `AddDecadeTag` uses the arithmetic approach `(year/10)*10` which is simpler and handles all decades correctly (including future ones). Add `year/pre-1950s` for years < 1950 to match Letterboxd's existing behavior if year > 0.
3. Replace inline tag logic in goodreads, imdb, and letterboxd with these helpers.
4. Remove `getDecadeTag()` from `cmd/letterboxd/markdown.go:172-193`.
5. Deprecate/remove `MarkdownBuilder.GetDecadeTag()` from `internal/fileutil/markdown.go:96-117` if no other callers exist.

**Files changed:** 4 files (1 modified in internal, 3 importers modified)

---

## Issue 4: Extract Standard Enrichment Error Handlers

**Problem:** IMDb and Letterboxd have identical error handler closures for OMDB/TMDB enrichment:

`cmd/imdb/parser.go:216-232`:
```go
OnOMDBError: func(err error) {
    slog.Warn("Failed to enrich from OMDB", "title", movie.Title, "error", err)
},
OnOMDBRateLimit: func(error) {
    omdb.MarkRateLimitReached()
    slog.Warn("Skipping OMDB enrichment after rate limit", "title", movie.Title)
},
OnTMDBError: func(err error) {
    slog.Warn("Failed to enrich from TMDB", "title", movie.Title, "error", err)
},
```

`cmd/letterboxd/parser.go:219-243` — identical except `movie.Name` instead of `movie.Title`.

**Plan:**

1. Add factory functions to `internal/importer/enrich/handlers.go`:
   ```go
   func OMDBErrorHandler(title string) func(error)
   func OMDBRateLimitHandler(title string, markRateLimit func()) func(error)
   func TMDBErrorHandler(title string) func(error)
   ```
2. `OMDBRateLimitHandler` takes a `markRateLimit func()` parameter so it's not coupled to any specific OMDB client instance.
3. Replace the inline closures in both importers.
4. Future importers with OMDB/TMDB enrichment can reuse these directly.

**Files changed:** 3 files (1 new, 2 modified)

---

## Issue 5: Extract Shared BuildAndWriteMarkdown Function

**Problem:** All four importers repeat the same build-and-write sequence:

```go
markdown, err := obsidian.BuildNoteMarkdown(fm, body)
if err != nil {
    return fmt.Errorf("failed to build markdown: %w", err)
}
return fileutil.WriteMarkdownFile(filePath, string(markdown), config.OverwriteFiles)
```

Some also append trailing newlines (`"\n\n\n"`) before writing.

- `cmd/goodreads/markdown.go:241-255` — uses `WriteFileWithOverwrite` + `LogFileWriteResult` + appends `\n\n\n`
- `cmd/imdb/markdown.go:185-191` — uses `WriteMarkdownFile`
- `cmd/letterboxd/markdown.go:162-168` — uses `WriteMarkdownFile`
- `cmd/steam/markdown.go:180-189` — uses `WriteMarkdownFile` + appends `\n\n\n`

**Plan:**

1. Add `BuildAndWriteNote()` to `internal/obsidian/note.go` (or existing file):
   ```go
   func BuildAndWriteNote(filePath string, fm *Frontmatter, body string, overwrite bool) error
   ```
2. This function calls `BuildNoteMarkdown()`, then `fileutil.WriteMarkdownFile()`.
3. Replace the 4-6 line sequences in each importer with a single function call.
4. For the trailing newlines in goodreads/steam: standardize behavior. Either always add trailing newlines in `BuildAndWriteNote` or remove the inconsistency. The trailing `\n\n\n` appears to be for Obsidian formatting — investigate whether it's needed and standardize.
5. Goodreads currently uses `WriteFileWithOverwrite` + `LogFileWriteResult` instead of `WriteMarkdownFile` — unify to use the same path.

**Files changed:** 5 files (1 modified in internal, 4 importers modified)

---

## Implementation Order

1. **Issue 1 (JSON wrappers)** — Simplest, no new code needed, just deletions and inline replacements.
2. **Issue 3 (Tag helpers)** — Self-contained addition to existing `obsidian/tags.go`, no cross-cutting concerns.
3. **Issue 4 (Enrichment handlers)** — Small new file, focused changes in 2 importers.
4. **Issue 5 (BuildAndWriteMarkdown)** — Small addition to obsidian package, touches all 4 importers.
5. **Issue 2 (Cover handler)** — Most complex, involves new abstraction and different cover source patterns.

Each issue should be a separate PR to keep reviews focused.

## Estimated Impact

- ~100 lines of code deleted
- ~50 lines of new shared utility code
- ~280 lines of duplicated code replaced with shared function calls
- Net reduction: ~230 lines
