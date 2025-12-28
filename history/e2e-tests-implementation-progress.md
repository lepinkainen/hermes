# E2E Test Implementation Progress

**Date**: 2024-12-28
**Plan**: `/Users/shrike/.claude/plans/jaunty-dancing-micali.md`
**Context**: Implementing comprehensive E2E test coverage based on Codex suggestions

## ✅ Completed Work

### 1. Steam E2E Test (Phase 1.1) ✓
**File**: `cmd/steam/import_e2e_test.go` (208 lines)

**What was done**:
- Created complete E2E test for Steam importer
- Implemented cache population pattern using `cache.GetGlobalCache()` API
- Added `ImportSteamGamesFunc` mockable variable to `cmd/steam/steam.go:98-102`
- Updated parser to use mockable function at `cmd/steam/parser.go:67`

**Key patterns learned**:
```go
// Reset global cache after setting viper config
resetErr := cache.ResetGlobalCache()
require.NoError(t, resetErr)

// Populate cache using cache API (not direct DB writes)
globalCache, err := cache.GetGlobalCache()
err = globalCache.Set("steam_cache", cacheKey, jsonData)

// Mock the import function
prevImportFunc := ImportSteamGamesFunc
ImportSteamGamesFunc = func(sid, key string) ([]Game, error) {
    // Load fixture and return
}
defer func() { ImportSteamGamesFunc = prevImportFunc }()
```

**Test verifies**:
- 3 games imported from fixture
- Games written to database
- Details fetched from cache (no API calls)
- Markdown files generated

### 2. Goodreads Datasette Disabled Test (Phase 1.2) ✓
**File**: `cmd/goodreads/import_e2e_test.go:101-156`

**What was done**:
- Added `TestGoodreadsImportE2E_DatasetteDisabled` function
- Added helper function `fileExists()` using `os.Stat()`

**Pattern**:
```go
func TestXXXImportE2E_DatasetteDisabled(t *testing.T) {
    env := testutil.NewTestEnv(t)
    testutil.SetTestConfig(t)

    // Disable datasette
    prevDatasetteEnabled := viper.GetBool("datasette.enabled")
    viper.Set("datasette.enabled", false)
    defer viper.Set("datasette.enabled", prevDatasetteEnabled)

    // Override markdown output directory
    viper.Set("markdownoutputdir", tempDir)
    defer viper.Set("markdownoutputdir", "markdown")

    // Run importer
    err := ParseXXXWithParams(...)
    require.NoError(t, err)

    // Verify NO DB created
    require.False(t, fileExists("./hermes.db"))

    // Verify markdown files WERE created
    files, _ := filepath.Glob(filepath.Join(outputPath, "*.md"))
    require.Greater(t, len(files), 0)
}
```

---

## 🚧 In Progress - Phase 1: Foundation Tests

### 1.3 Add Datasette Disabled Tests (3 remaining)

**Status**: 1/4 complete (Goodreads ✓, IMDb ⏳, Letterboxd ⏳, Steam ⏳)

**Files to modify**:
- `cmd/imdb/import_e2e_test.go` - Add `TestImdbImportE2E_DatasetteDisabled`
- `cmd/letterboxd/import_e2e_test.go` - Add `TestLetterboxdImportE2E_DatasetteDisabled`
- `cmd/steam/import_e2e_test.go` - Add `TestSteamImportE2E_DatasetteDisabled`

**Copy the pattern from**: `cmd/goodreads/import_e2e_test.go:101-156`

**Changes needed per file**:
1. Copy `TestGoodreadsImportE2E_DatasetteDisabled` function
2. Rename to match importer (TestImdbImportE2E_DatasetteDisabled, etc.)
3. Update fixture path (e.g., `imdb_sample.csv` instead of `goodreads_sample.csv`)
4. Update expected count (IMDb: 20, Letterboxd: 20, Steam: 3)
5. Update ParseXXXWithParams call to match importer's signature
6. Copy `fileExists()` helper if not already present

**Expected time**: 30 minutes (10 min per importer)

---

## 📋 Remaining Work

### Phase 2: Output Verification (6-8 hours)

#### 2.1 Markdown Output Verification (all 4 importers)

**Files to modify**:
- `cmd/goodreads/import_e2e_test.go`
- `cmd/imdb/import_e2e_test.go`
- `cmd/letterboxd/import_e2e_test.go`
- `cmd/steam/import_e2e_test.go`

**What to add** (extend EXISTING E2E tests):
```go
func TestXXXImportE2E(t *testing.T) {
    // ... existing test setup and DB verification ...

    // NEW: Verify markdown output structure
    outputPath := filepath.Join(tempDir, "output")
    files, err := filepath.Glob(filepath.Join(outputPath, "*.md"))
    require.NoError(t, err)
    require.Greater(t, len(files), 0)

    // Sort for deterministic selection
    sort.Strings(files)

    // Read first file
    content, err := os.ReadFile(files[0])
    require.NoError(t, err)
    contentStr := string(content)

    // Verify YAML frontmatter structure
    require.Contains(t, contentStr, "---\n", "Should have YAML frontmatter")
    require.Contains(t, contentStr, "title:", "Should have title field")

    // Goodreads-specific:
    require.Contains(t, contentStr, "authors:", "Should have authors field")
    require.Contains(t, contentStr, "goodreads_id:")

    // IMDb-specific:
    require.Contains(t, contentStr, "imdb_id:")
    require.Contains(t, contentStr, "year:")

    // Letterboxd-specific:
    require.Contains(t, contentStr, "letterboxd_uri:")
    require.Contains(t, contentStr, "rating:")

    // Steam-specific:
    require.Contains(t, contentStr, "steam_appid:")
    require.Contains(t, contentStr, "playtime_forever:")

    // Verify markdown content exists (not just frontmatter)
    require.Regexp(t, `(?m)^#+ `, contentStr, "Should have markdown headers")
}
```

**Import needed**: `"sort"`

**Estimated time**: 3-4 hours

---

#### 2.2 JSON Output Verification (all 4 importers)

**Files to modify**: Same 4 files

**What to add** (new test functions):
```go
func TestXXXImportE2E_JSON(t *testing.T) {
    env := testutil.NewTestEnv(t)
    testutil.SetTestConfig(t)
    tempDir := env.RootDir()

    // Setup database
    dbPath := filepath.Join(tempDir, "test.db")
    viper.Set("datasette.enabled", true)
    viper.Set("datasette.dbfile", dbPath)
    defer viper.Set("datasette.enabled", true)
    defer viper.Set("datasette.dbfile", "./hermes.db")

    // Copy CSV fixture
    csvPath := filepath.Join(tempDir, "input.csv")
    env.CopyFile("testdata/xxx_sample.csv", "input.csv")

    // Enable JSON output
    jsonPath := filepath.Join(tempDir, "output.json")

    err := ParseXXXWithParams(
        ParseParams{
            CSVPath:    csvPath,
            OutputDir:  "output",
            WriteJSON:  true,        // ENABLE JSON
            JSONOutput: jsonPath,
        },
        // ... other params ...
    )
    require.NoError(t, err)

    // Verify JSON file exists
    require.FileExists(t, jsonPath)

    // Parse JSON
    content, err := os.ReadFile(jsonPath)
    require.NoError(t, err)

    var items []map[string]interface{}
    err = json.Unmarshal(content, &items)
    require.NoError(t, err)
    require.Len(t, items, expectedCount)

    // Verify schema - spot-check first item
    firstItem := items[0]
    require.Contains(t, firstItem, "title")
    require.NotEmpty(t, firstItem["title"])

    // Add importer-specific field checks
}
```

**Import needed**: `"encoding/json"`

**Note**: Check each importer's `ParseParams` struct - signatures vary slightly

**Estimated time**: 3-4 hours

---

### Phase 3: cmd/diff E2E Tests (6-8 hours)

**File to create**: `cmd/diff/imdb_letterboxd_e2e_test.go` (~150-200 LOC)

**Strategy**: Use **Approach B** (export schema constants) to avoid schema drift

**Step 1**: Export schema constants in production code

**Files to modify**:
```go
// cmd/imdb/parser.go:22 - Change to uppercase
const IMDbMoviesSchema = `CREATE TABLE IF NOT EXISTS imdb_movies (...)`

// cmd/letterboxd/parser.go:23 - Change to uppercase
const LetterboxdMoviesSchema = `CREATE TABLE IF NOT EXISTS letterboxd_movies (...)`
```

**Step 2**: Create test file

```go
package diff

import (
    "database/sql"
    "testing"
    "time"

    "github.com/lepinkainen/hermes/cmd/imdb"
    "github.com/lepinkainen/hermes/cmd/letterboxd"
    "github.com/lepinkainen/hermes/internal/testutil"
    "github.com/stretchr/testify/require"
    _ "modernc.org/sqlite"
)

func TestIMDbLetterboxdDiffE2E(t *testing.T) {
    env := testutil.NewTestEnv(t)

    // Create fixture databases
    mainDBPath := env.Path("hermes.db")
    createFixtureMainDB(t, mainDBPath)

    cacheDBPath := env.Path("cache.db")
    createFixtureCacheDB(t, cacheDBPath)

    // Run diff
    report, err := BuildDiffReport(mainDBPath, cacheDBPath, time.Now())
    require.NoError(t, err)

    // Verify diff results
    require.Len(t, report.ImdbOnly, 3)
    require.Len(t, report.LetterboxdOnly, 2)
    require.Len(t, report.Matched, 2)

    // Test markdown report
    note, err := BuildIMDbLetterboxdReport(mainDBPath, cacheDBPath, time.Now())
    require.NoError(t, err)
    require.Contains(t, note.Body, "## IMDb-only")

    // Test HTML report
    htmlBytes, err := renderDiffHTML(report)
    require.NoError(t, err)
    require.Contains(t, string(htmlBytes), "<html")
}

func createFixtureMainDB(t *testing.T, dbPath string) {
    t.Helper()

    db, err := sql.Open("sqlite", dbPath)
    require.NoError(t, err)
    defer db.Close()

    // Use EXPORTED production schemas - NO DRIFT!
    _, err = db.Exec(imdb.IMDbMoviesSchema)
    require.NoError(t, err)

    _, err = db.Exec(letterboxd.LetterboxdMoviesSchema)
    require.NoError(t, err)

    // Insert test data
    // Matched movies (2):
    _, err = db.Exec(`INSERT INTO imdb_movies
        (imdb_id, title, year, my_rating) VALUES
        ('tt1234567', 'The Matrix', 1999, 9),
        ('tt7654321', 'Inception', 2010, 10)`)
    require.NoError(t, err)

    _, err = db.Exec(`INSERT INTO letterboxd_movies
        (letterboxd_id, name, year, imdb_id, rating) VALUES
        ('the-matrix', 'The Matrix', 1999, 'tt1234567', 4.5),
        ('inception', 'Inception', 2010, 'tt7654321', 5.0)`)
    require.NoError(t, err)

    // IMDb-only movies (3):
    _, err = db.Exec(`INSERT INTO imdb_movies
        (imdb_id, title, year, my_rating) VALUES
        ('tt1111111', 'IMDb Only 1', 2020, 8),
        ('tt2222222', 'IMDb Only 2', 2021, 7),
        ('tt3333333', 'Fuzzy Match', 2019, 6)`)
    require.NoError(t, err)

    // Letterboxd-only movies (2):
    _, err = db.Exec(`INSERT INTO letterboxd_movies
        (letterboxd_id, name, year, rating) VALUES
        ('lb-only-1', 'Letterboxd Only 1', 2022, 4.0),
        ('fuzzy-match', 'Fuzzy Match', 2018, 3.5)`)
    require.NoError(t, err)
}

func createFixtureCacheDB(t *testing.T, dbPath string) {
    t.Helper()

    db, err := sql.Open("sqlite", dbPath)
    require.NoError(t, err)
    defer db.Close()

    // Create letterboxd_mapping_cache table
    _, err = db.Exec(`CREATE TABLE letterboxd_mapping_cache (
        letterboxd_uri TEXT,
        imdb_id TEXT
    )`)
    require.NoError(t, err)

    // Insert mappings
    _, err = db.Exec(`INSERT INTO letterboxd_mapping_cache
        (letterboxd_uri, imdb_id) VALUES
        ('https://letterboxd.com/film/the-matrix/', 'tt1234567'),
        ('https://letterboxd.com/film/inception/', 'tt7654321')`)
    require.NoError(t, err)
}
```

**Test scenarios to cover**:
- Perfect IMDb ID match ✓
- Title+year auto-resolution (add to fixture)
- Fuzzy matches ✓
- IMDb-only ✓
- Letterboxd-only ✓
- Cache mapping resolution ✓

**Estimated time**: 6-8 hours

---

### Phase 4: Cache Behavior Tests (4-5 hours)

**Files to modify**: All 4 importer E2E test files

**What to add** (new test functions):
```go
func TestXXXImportE2E_CacheHit(t *testing.T) {
    env := testutil.NewTestEnv(t)
    testutil.SetTestConfig(t)
    tempDir := env.RootDir()

    // Setup cache DB
    cacheDBPath := filepath.Join(tempDir, "cache.db")
    viper.Set("cache.dbfile", cacheDBPath)
    defer viper.Set("cache.dbfile", "./cache.db")

    // Setup datasette DB
    dbPath := filepath.Join(tempDir, "test.db")
    viper.Set("datasette.enabled", true)
    viper.Set("datasette.dbfile", dbPath)
    defer viper.Set("datasette.enabled", true)
    defer viper.Set("datasette.dbfile", "./hermes.db")

    // Override markdown output directory
    viper.Set("markdownoutputdir", tempDir)
    defer viper.Set("markdownoutputdir", "markdown")

    // Reset global cache
    resetErr := cache.ResetGlobalCache()
    require.NoError(t, resetErr)
    defer func() { _ = cache.ResetGlobalCache() }()

    // Copy CSV
    csvPath := filepath.Join(tempDir, "input.csv")
    env.CopyFile("testdata/xxx_sample.csv", "input.csv")

    // FIRST RUN: Populate cache
    err := ParseXXXWithParams(...)
    require.NoError(t, err)

    // Verify cache entries created
    cacheDB, err := sql.Open("sqlite", cacheDBPath)
    require.NoError(t, err)
    defer cacheDB.Close()

    var cacheCount int
    err = cacheDB.QueryRow("SELECT COUNT(*) FROM xxx_cache").Scan(&cacheCount)
    require.NoError(t, err)
    require.Greater(t, cacheCount, 0, "Cache should have entries")

    initialCacheCount := cacheCount

    // SECOND RUN: Should use cache
    err = ParseXXXWithParams(...)
    require.NoError(t, err)

    // Verify cache count unchanged
    err = cacheDB.QueryRow("SELECT COUNT(*) FROM xxx_cache").Scan(&cacheCount)
    require.NoError(t, err)
    require.Equal(t, initialCacheCount, cacheCount,
        "Cache count should be unchanged on second run")
}
```

**Cache table names**:
- Goodreads: `openlibrary_cache`
- IMDb: `omdb_cache`
- Letterboxd: `letterboxd_cache`
- Steam: `steam_cache`

**Import needed**: `"github.com/lepinkainen/hermes/internal/cache"`

**Estimated time**: 4-5 hours (1 hour per importer + debugging)

---

## 🎯 Final Steps

### Run Full Test Suite
```bash
task test
```

### Run Linter
```bash
task lint
```

### Commit Changes
```bash
git add .
git commit -m "feat: add comprehensive E2E test coverage for all importers

- Add Steam E2E test with cache mocking pattern
- Add datasette disabled tests for all 4 importers
- Add markdown/JSON output verification for all importers
- Add cmd/diff E2E test with fixture databases
- Add cache behavior tests for all importers

Addresses Codex E2E test suggestions. All tests pass offline
using fixtures and cache pre-population.

🤖 Generated with Claude Code"
```

---

## 📊 Progress Tracker

**Phase 1: Foundation** (3-5 hours)
- [x] Steam E2E test (2-3 hours) ✓
- [x] Goodreads datasette disabled (0.5 hours) ✓
- [ ] IMDb datasette disabled (0.5 hours)
- [ ] Letterboxd datasette disabled (0.5 hours)
- [ ] Steam datasette disabled (0.5 hours)

**Phase 2: Output Verification** (6-8 hours)
- [ ] Markdown verification (3-4 hours)
- [ ] JSON verification (3-4 hours)

**Phase 3: Complex Commands** (6-8 hours)
- [ ] cmd/diff E2E test (6-8 hours)

**Phase 4: Advanced Scenarios** (4-5 hours)
- [ ] Cache behavior tests (4-5 hours)

**Total**: 19-26 hours estimated | **2-3 hours completed** | **17-23 hours remaining**

---

## 🔑 Key Learnings & Gotchas

### 1. Cache Population
**Wrong**: Manually insert into DB before `GetGlobalCache()` is called
```go
// ❌ DON'T DO THIS - tables get recreated
db.Exec("INSERT INTO steam_cache ...")
cache.GetGlobalCache() // Creates new tables!
```

**Right**: Use cache API after resetting global cache
```go
// ✅ DO THIS
cache.ResetGlobalCache()
globalCache, _ := cache.GetGlobalCache()
globalCache.Set("steam_cache", key, data)
```

### 2. Viper Config Order
Always set viper config BEFORE resetting cache:
```go
viper.Set("cache.dbfile", testPath)  // First
cache.ResetGlobalCache()             // Then reset
```

### 3. Schema Drift Prevention
Export schemas as public constants:
```go
// cmd/imdb/parser.go
const IMDbMoviesSchema = `CREATE TABLE...` // Uppercase!
```

Then use in tests:
```go
db.Exec(imdb.IMDbMoviesSchema) // No drift!
```

### 4. File Existence Check
Use `os.Stat()`, not `filepath.Glob()`:
```go
func fileExists(path string) bool {
    _, err := os.Stat(path)
    return err == nil
}
```

### 5. Steam-Specific Cache Format
Cache stores `GameDetails` as JSON, not raw API response:
```go
// Parse fixture to extract GameDetails
var result map[string]struct {
    Success bool
    Data    GameDetails
}
json.Unmarshal(fixture, &result)

// Cache the GameDetails, not the whole response
detailsJSON, _ := json.Marshal(gameDetails)
globalCache.Set("steam_cache", appID, string(detailsJSON))
```

---

## 📁 Files Modified So Far

### Created
- `cmd/steam/import_e2e_test.go` (208 lines)

### Modified
- `cmd/steam/steam.go` (added lines 98-102: mockable function variable)
- `cmd/steam/parser.go` (line 67: use mockable function)
- `cmd/goodreads/import_e2e_test.go` (added lines 101-156: datasette test)

### To be modified
- `cmd/imdb/import_e2e_test.go` (extend)
- `cmd/imdb/parser.go` (export schema)
- `cmd/letterboxd/import_e2e_test.go` (extend)
- `cmd/letterboxd/parser.go` (export schema)
- `cmd/diff/imdb_letterboxd_e2e_test.go` (create)

---

## 🚀 Quick Start for Next Session

1. **Resume from Phase 1.3**: Add datasette disabled tests to IMDb, Letterboxd, Steam
   - Copy pattern from `cmd/goodreads/import_e2e_test.go:101-156`
   - Update fixture paths and counts
   - Run tests individually: `go test ./cmd/imdb -v -run DatasetteDisabled`

2. **Then move to Phase 2.1**: Add markdown verification
   - Add to existing E2E tests (don't create new functions)
   - Use targeted assertions, not golden files
   - Import `"sort"` for deterministic file selection

3. **Follow the plan**: Each phase builds on the previous
   - Test incrementally after each addition
   - Commit after each phase completes

Good luck! 🎉
