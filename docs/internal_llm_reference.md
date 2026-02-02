# Internal Package Guide for LLMs

Hermes already ships battle-tested helpers for nearly every importer concern. This guide walks through the `internal/` packages so LLM agents can reach for existing building blocks instead of re‑implementing them. When in doubt, search these packages first.

## Cache (`internal/cache`)

- `CacheDB` wraps the SQLite cache with R/W locks and helpers such as `CreateTable`, `Get`, `Set`, `ClearExpired`, and `ClearAll`. Always obtain it through `GetGlobalCache()` so schemas from `schema.go` are created once per run.
- `GetOrFetch` / `GetOrFetchWithPolicy` accept a table name, key, and fetch function. They parse TTL from `cache.ttl`, transparently log failures, and return `(value, fromCache, err)`. Use the `shouldCache` callback to skip writes for empty responses (e.g., 404 lookups).
- `ValidCacheTableNames` and the whitelist enforced by `validateTableName` prevent SQL injection. Never format table names manually.

These helpers power the OMDB/TMDB caches—leverage them for other API clients instead of hand-rolled caching.

## Command Utilities (`internal/cmdutil`)

- `BaseCommandConfig` captures the output + JSON paths, overwrite behaviour, and CLI flag bindings.
- `SetupOutputDir` folds together CLI flags, Viper config (`markdownoutputdir`, `jsonoutputdir`), and sensible defaults. It also precreates directories.

Every importer command should call `SetupOutputDir` before writing files so directory logic stays consistent.

## Config (`internal/config`)

- Call `InitConfig()` once (typically during CLI bootstrapping) to set defaults and populate globals from Viper.
- Global variables (`OverwriteFiles`, `UpdateCovers`, `TMDBAPIKey`, `OMDBAPIKey`) are read by file writers and enrichment flows. Toggle them via `SetOverwriteFiles`/`SetUpdateCovers` when parsing flags rather than inventing new switches.

## Content Builders (`internal/content`)

- `BuildTMDBContent(details, mediaType, sections)` emits structured Markdown from TMDB detail maps. Sections like `overview`, `info`, and `seasons` are already implemented.
- `BuildCoverImageEmbed`, `WrapWithMarkers`, `HasTMDBContentMarkers`, `GetTMDBContent`, and `ReplaceTMDBContent` standardise how TMDB-enriched blocks are embedded and updated in notes.

If you need TMDB-specific Markdown, call into this package—do *not* duplicate the Markdown templates elsewhere.

## CSV Processing (`internal/csvutil`)

- `ProcessCSV(filename, parser, opts)` handles opening, header skipping, record validation, and optional `SkipInvalid` behaviour. Provide a parser function that converts `[]string` into your struct.

This utility removes the need for per-importer CSV loops; simply define a parser and reuse the helper.

## Datastore (`internal/datastore`)

- `Store` interface abstracts local SQLite vs. remote Datasette writers (`Connect`, `CreateTable`, `BatchInsert`, `Close`).
- `SQLiteStore` implements the interface with prepared statement batching and transaction management—use `NewSQLiteStore(path)` for local persistence.

If an importer needs to emit structured data, depend on `Store` rather than touching `database/sql` directly.

## TMDB/Media Enrichment (`internal/enrichment`)

- `EnrichFromTMDB(ctx, title, year, imdbID, existingTMDBID, opts)` handles lookups, TUI selection, cover downloads (with optional cache mirroring), and Markdown generation. It returns a `TMDBEnrichment` struct you can merge into notes.
- `TMDBEnrichmentOptions` toggles cover download, force refresh, attachments directories, interactive selection, etc.
- `MergeTags` and `TagsFromAny` handle frontmatter tag deduplication and YAML decoding.

Whenever you need TMDB data, pipe through this module to stay consistent with cover naming, caching, and error handling.

## Error Helpers (`internal/errors`)

- `RateLimitError` + `IsRateLimitError` model API back-off conditions; OMDB/TMDB code already uses them to short-circuit.
- `StopProcessingError` + `IsStopProcessingError` signal user-driven cancellation from the TUI layer.

Surface new terminal conditions through these types instead of inventing bespoke sentinel errors.

## File Utilities (`internal/fileutil`)

- Path helpers: `SanitizeFilename`, `GetMarkdownFilePath`, `RelativeTo`, `FileExists`.
- Writers: `WriteFileWithOverwrite`, `WriteMarkdownFile`, `WriteJSONFile` honour `config.OverwriteFiles`.
- Cover pipeline: `BuildCoverFilename`, `DownloadCover`, and `AddCoverToMarkdown` wrap download/reuse logic (including TMDB-preferred covers). Respect `CoverDownloadOptions.UpdateCovers` rather than deleting files manually.
- Markdown composition: `MarkdownBuilder` + `TagCollector` add titles, arrays, dates, callouts, cover embeds, TMDB enrichment blocks (`AddTMDBEnrichmentFields`), and Obsidian-specific syntax.

Use the builder instead of string concatenation to keep notes consistent and to benefit from helpers like automatic tag dedupe and duration formatting.

## Frontmatter Parsing (`internal/frontmatter`)

- `ParseMarkdown([]byte)` splits YAML frontmatter from the body and returns `ParsedNote`.
- `ParsedNote.GetInt` / `GetString`, along with `IntFromAny` and `StringFromAny`, cope with YAML’s mixed numeric types.
- `DetectMediaType` inspects `tmdb_type` first, then falls back to tags (movie vs TV).

Any code that needs to inspect an existing note should go through this parser instead of ad-hoc regexes.

## Importer Enrichment Flow (`internal/importer/enrich`)

- `Options[T,O]` describes how OMDB + TMDB enrichment should run for a given importer. Provide hooks (`FetchOMDB`, `ApplyOMDB`, `OnOMDBError`, `FetchTMDB`, `ApplyTMDB`, etc.) and the shared `Enrich` function orchestrates the workflow.
- `Result` reports provider-specific errors so callers can log granular status.

Leverage this when writing new importers: it already handles rate limits, skip flags, and failure aggregation.

## Media ID Extraction (`internal/importer/mediaids`)

- `MediaIDs` struct captures TMDB, IMDB, and Letterboxd IDs.
- `FromFrontmatter` and `FromFile` parse IDs from an existing note.
- `HasAny` and `Summary` report what was found; great for logs like “skipping movie (tmdb:12345)”.

Call these helpers before hitting external APIs—they let you reuse stored IDs and avoid duplicate lookups.

## OMDB Helpers (`internal/omdb`)

- `GetCached(cacheKey, fetcher)` wraps OMDB API calls with the shared cache and rate-limit tracking (`MarkRateLimitReached`, `RequestsAllowed`, `ResetRateLimit`). It seeds TTL data in `omdb_cache`.
- `SeedCacheByID` writes a secondary IMDb-ID entry so later lookups by ID hit the cache even if the first fetch used title/year.

New OMDB consumers should always go through `GetCached` so the global rate limit gate works.

## TMDB Client & Cache (`internal/tmdb`)

- `Client` exposes searching (`SearchMovies`, `SearchMulti`), `FindByIMDBID`, metadata fetchers (`GetMovieDetails`, `GetTVDetails`, `GetFull*`), and cover helpers (`GetCoverURLByID`, `DownloadAndResizeImage`, `GetCoverAndMetadataByID`).
- Cached variants (`CachedSearchMovies`, `CachedSearchMulti`, `CachedFindByIMDBID`, `CachedGetFullMovieDetails`, etc.) automatically store JSON blobs via `internal/cache`. Use them when repeated lookups are possible; pass `force=true` to bypass cache and refresh.
- The client handles retries, poster URL assembly (`ImageURL`), metadata normalization, and genre tag generation. Errors like `ErrInvalidMediaType`/`ErrNoPoster` are exported.

Whenever your code needs TMDB data, instantiate `tmdb.NewClient(apiKey)` (optionally overriding HTTP/timeouts via functional options) and call the cached helpers first.

## TUI (`internal/tui`)

- `Select(title, results)` launches the Bubble Tea interface for picking a TMDB result. The returned `SelectionResult.Action` distinguishes between `ActionSelected`, `ActionSkipped`, and `ActionStopped` (user aborted the run).

Use this helper instead of crafting new TUIs—`EnrichFromTMDB` already wires it up, but other interactive flows can reuse it too.

## Testing Utilities (`internal/testutil`)

- `TestEnv` sandboxes filesystem interactions—`Path`, `WriteFile`, `ReadFile`, `MkdirAll`, `Symlink`, `TempFile`, and assertions keep tests confined to `t.TempDir()`.
- Config helpers (`ResetConfig`, `SetTestConfig`, `SetTestConfigWithOptions`, `SetViperValue`, `SetupTestCache`) snapshot/restore `internal/config` globals and Viper state.
- `GoldenHelper` streamlines golden-file comparisons with opt-in update mode.

All new tests should rely on these helpers for isolation instead of duplicating temp-dir + config reset logic.

---

**TL;DR:** Before inventing a new helper, skim this document and the corresponding package. There’s likely already a function that reads CSVs, writes Markdown, downloads covers, parses frontmatter, or mediates API rate limits. Reuse them to stay aligned with the rest of Hermes.


---

*Document created: 2025-11-20*
*Last reviewed: 2025-11-20*