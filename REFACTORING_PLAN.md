# Refactoring Plan: Centralizing Caching and JSON Writing

This document outlines the plan to refactor duplicated code related to caching and JSON file writing into shared `internal` packages.

## Reasoning for Refactoring

Currently, the `cmd` subdirectories (`goodreads`, `imdb`, `letterboxd`, `steam`) contain highly similar, if not identical, logic for:
1.  **Caching API responses:** Each command implements its own `getCached...` function, which checks for a cached file, reads it, unmarshals it, or fetches data from an API and caches it. This leads to code duplication and makes maintenance harder.
2.  **Writing JSON files:** Functions like `writeBookToJson`, `writeMovieToJson`, and `writeGameToJson` are almost identical in their core logic (marshaling to JSON, creating/writing to file).

By centralizing this logic into the `internal` package, we achieve:
*   **Reduced code duplication:** Less code to maintain and fewer places for bugs to hide.
*   **Improved consistency:** All caching and JSON writing operations will follow a single, well-defined pattern.
*   **Easier maintenance and future development:** Changes or improvements to caching/JSON writing only need to be made in one place.
*   **Better separation of concerns:** The `cmd` packages can focus on command-specific logic, while `internal` handles common utilities.

## Step-by-Step Implementation Plan

### Step 1: Create `internal/cache` Package and Generic Caching Utility

**Objective:** Implement a reusable caching mechanism that can be adopted by all `cmd` packages.

1.  **Create Directory:**
    *   Command: `mkdir -p /Users/shrike/projects/hermes/internal/cache`
    *   Reasoning: To house the new generic caching logic.
2.  **Create `internal/cache/cache.go`:**
    *   Create a new file: `/Users/shrike/projects/hermes/internal/cache/cache.go`
    *   Content: Implement a generic function (e.g., `GetOrFetch`) that takes a cache directory, a unique identifier (e.g., filename), a function to fetch data if not cached, and a function to unmarshal cached data. This function will handle reading from/writing to disk, marshaling/unmarshaling JSON, and error handling.

### Step 2: Refactor `cmd/*/cache.go` Files

**Objective:** Replace the duplicated caching logic in each `cmd` package with calls to the new generic caching utility.

For each of `cmd/goodreads/cache.go`, `cmd/imdb/cache.go`, `cmd/letterboxd/cache.go`, and `cmd/steam/cache.go`:

1.  **Remove Duplicated Logic:** Delete the existing `getCached...` function's implementation.
2.  **Import `internal/cache`:** Add `github.com/lepinkainen/hermes/internal/cache` to the imports.
3.  **Call Generic Cache Function:** Replace the old caching logic with a call to the new generic `cache.GetOrFetch` function, adapting the parameters as needed for each specific type (Book, MovieSeen, Game, etc.).

### Step 3: Refactor `internal/fileutil/fileutil.go` for Generic JSON Writing

**Objective:** Create a single, reusable function for writing JSON data to a file.

1.  **Modify `internal/fileutil/fileutil.go`:**
    *   Add a new function, e.g., `WriteJSONFile(data interface{}, filename string, overwrite bool) error`.
    *   This function will handle `json.MarshalIndent`, `os.MkdirAll`, `os.Create`, and `os.WriteFile`. It should also incorporate the `overwrite` logic seen in `letterboxd/json.go`.

### Step 4: Refactor `cmd/*/json.go` Files

**Objective:** Replace the duplicated JSON writing logic in each `cmd` package with calls to the new generic JSON writing utility.

For each of `cmd/goodreads/json.go`, `cmd/imdb/json.go`, `cmd/letterboxd/json.go`, and `cmd/steam/json.go`:

1.  **Remove Duplicated Logic:** Delete the existing `write...ToJson` function's implementation.
2.  **Import `internal/fileutil`:** Ensure `github.com/lepinkainen/hermes/internal/fileutil` is imported.
3.  **Call Generic JSON Write Function:** Replace the old JSON writing logic with a call to the new `fileutil.WriteJSONFile` function, passing the appropriate data structure and filename.

## Verification

After implementing these changes, the following steps should be taken to verify correctness:
1.  **Run existing tests:** Execute `go test ./...` to ensure no regressions were introduced.
2.  **Manual testing:** Run each command (`goodreads`, `imdb`, `letterboxd`, `steam`) to ensure they still function as expected, both with and without existing cache files.
3.  **Code review:** Review the changes to ensure they adhere to Go best practices and project conventions.
