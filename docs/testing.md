# Testing Guide

## Golden File Testing

### Overview

Golden file testing is a testing pattern where expected output is stored in reference files (called "golden files") that are committed to the repository. During tests, generated output is compared against these golden files to detect unexpected changes.

**Implementation Location:** `internal/testutil/golden.go`

**Why Use Golden Files:**
- Captures complex expected output (markdown files, JSON responses, formatted text)
- Makes it easy to review changes in test expectations via git diffs
- Serves as documentation of expected behavior
- Simplifies tests for formatters, parsers, and code generators

### How It Works

The `GoldenHelper` operates in two modes:

#### Normal Mode (Testing)
When running tests normally:
1. Reads the golden file from the `testdata/` directory
2. Compares it with the actual generated content
3. Test fails if they don't match (shows diff in test output)

#### Update Mode (Setting Baseline)
When `UPDATE_GOLDEN=true` environment variable is set:
1. Writes the actual output as the new golden file
2. Creates the golden directory if it doesn't exist
3. Logs: `"Updated golden file: {path}"`
4. Use this to establish or update test expectations

**Update Command:**
```bash
# Update all golden files in the entire project
UPDATE_GOLDEN=true go test ./...

# Update golden files in specific package
UPDATE_GOLDEN=true go test ./cmd/enhance/...
```

### API Reference

#### NewGoldenHelper
```go
func NewGoldenHelper(t *testing.T, goldenDir string) *GoldenHelper
```
Creates a new golden file helper pointing to a specific directory for golden files.

**Parameters:**
- `t` - The testing.T instance
- `goldenDir` - Path to directory containing golden files (e.g., `"testdata"` or `filepath.Join("testdata", "subdir")`)

**Returns:** `*GoldenHelper` instance

#### AssertGoldenString
```go
func (g *GoldenHelper) AssertGoldenString(name string, actual string)
```
Compares a string against a golden file, or updates the golden file if in update mode.

**Parameters:**
- `name` - Name of the golden file (e.g., `"output.txt"`)
- `actual` - The actual string content to compare

**Behavior:**
- Normal mode: Reads `goldenDir/name` and compares with `actual`
- Update mode: Writes `actual` to `goldenDir/name`

#### AssertGolden
```go
func (g *GoldenHelper) AssertGolden(name string, actual []byte)
```
Compares byte content against a golden file (used for binary or raw byte data).

**Parameters:**
- `name` - Name of the golden file
- `actual` - The actual byte content to compare

#### AssertGoldenJSON
```go
func (g *GoldenHelper) AssertGoldenJSON(name string, actual []byte)
```
Compares JSON content semantically, ignoring formatting differences (whitespace, key order).

**Parameters:**
- `name` - Name of the golden file (should be `.json`)
- `actual` - The actual JSON bytes to compare

**Use Case:** Ideal for testing JSON API responses or JSON serialization where formatting might vary.

#### MustReadGolden
```go
func (g *GoldenHelper) MustReadGolden(name string) []byte
```
Reads and returns the content of a golden file. Fails the test if the file doesn't exist.

**Parameters:**
- `name` - Name of the golden file

**Returns:** `[]byte` content of the file

#### IsUpdateMode
```go
func (g *GoldenHelper) IsUpdateMode() bool
```
Returns `true` if `UPDATE_GOLDEN` environment variable is set to `"true"`.

#### GoldenPath
```go
func (g *GoldenHelper) GoldenPath(name string) string
```
Returns the full file system path to a golden file.

**Parameters:**
- `name` - Name of the golden file

**Returns:** Full path string (e.g., `/path/to/project/cmd/enhance/testdata/output.txt`)

### Usage Examples

#### Basic Usage Pattern
```go
func TestMarkdownGeneration(t *testing.T) {
    // 1. Create golden helper pointing to testdata directory
    gh := testutil.NewGoldenHelper(t, "testdata")

    // 2. Generate your actual output
    actualMarkdown := generateMarkdown(someInput)

    // 3. Compare with golden file (or update it if UPDATE_GOLDEN=true)
    gh.AssertGoldenString("expected_output.md", actualMarkdown)
}
```

#### Table-Driven Test with Golden Files
```go
func TestMovieMarkdown(t *testing.T) {
    tests := []struct {
        name       string
        input      Movie
        goldenFile string
    }{
        {
            name:       "basic movie",
            input:      createBasicMovie(),
            goldenFile: "basic_movie.md",
        },
        {
            name:       "complex movie with all fields",
            input:      createComplexMovie(),
            goldenFile: "complex_movie.md",
        },
        {
            name:       "tv series",
            input:      createTVSeries(),
            goldenFile: "tv_series.md",
        },
    }

    gh := testutil.NewGoldenHelper(t, "testdata")

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            actual := generateMarkdown(tt.input)
            gh.AssertGoldenString(tt.goldenFile, actual)
        })
    }
}
```

#### Creating Golden Files for New Test
```bash
# 1. Write the test code with AssertGoldenString call
# 2. Run with UPDATE_GOLDEN to create initial golden files
UPDATE_GOLDEN=true go test ./cmd/mypackage/... -run TestMyNewFeature

# 3. Review the generated golden files
git diff cmd/mypackage/testdata/

# 4. If they look correct, commit them
git add cmd/mypackage/testdata/
git commit -m "test: add golden files for my new feature"

# 5. Future test runs will compare against these files
go test ./cmd/mypackage/... -run TestMyNewFeature
```

### Best Practices

#### When to Use Golden Files
**Good Use Cases:**
- Testing markdown/HTML generation
- Testing CSV/JSON formatters
- Testing parsers (output structured data)
- Testing code generators
- Complex multi-line text output
- Binary file generation

**When NOT to Use:**
- Simple boolean/numeric assertions
- Testing internal logic without formatting concerns
- Highly variable output (timestamps, random data)

#### Directory Organization
```
cmd/mypackage/
├── mypackage.go
├── mypackage_test.go
└── testdata/
    ├── input_data.csv          # Test input files
    ├── expected_output.md      # Golden files
    ├── complex_case.json       # More golden files
    └── subdirectory/           # Organize by test suite
        ├── case1.txt
        └── case2.txt
```

**Conventions:**
- Golden files live in `testdata/` subdirectory of the package being tested
- Name golden files descriptively (e.g., `basic_movie.md`, `error_response.json`)
- Group related golden files in subdirectories when you have many tests
- Keep test input data in `testdata/` alongside golden files

#### Naming Conventions
- Use descriptive names: `basic_movie.md` not `test1.md`
- Match file extension to content type: `.md`, `.json`, `.txt`, `.csv`
- For table-driven tests, name files after test case: `{test_case_name}.{ext}`

#### Updating Golden Files
- Review diffs carefully before committing updates
- Use `git diff` to see what changed in golden files
- Include golden file updates in the same commit as code changes
- Document in commit message why golden files changed

### Real Examples from Codebase

#### Example 1: File Discovery Testing
**Location:** `cmd/enhance/cmd_test.go`

```go
func TestFindMarkdownFiles_WithParentheses(t *testing.T) {
    env := testutil.NewTestEnv(t)

    env.WriteFileString("Plain.md", "ok")
    env.WriteFileString("Red Sonja (2025).md", "ok")
    env.WriteFileString("ignore.txt", "nope")
    env.MkdirAll("subdir")
    env.WriteFileString("subdir/Series (Pilot).md", "ok")

    gh := testutil.NewGoldenHelper(t, filepath.Join("testdata", "find_markdown_files"))

    files, err := findMarkdownFiles(env.RootDir(), false)
    require.NoError(t, err)
    gh.AssertGoldenString("non_recursive.txt", strings.Join(relPaths(t, env.RootDir(), files), "\n")+"\n")

    files, err = findMarkdownFiles(env.RootDir(), true)
    require.NoError(t, err)
    gh.AssertGoldenString("recursive.txt", strings.Join(relPaths(t, env.RootDir(), files), "\n")+"\n")
}
```

**Golden Files:**
- `cmd/enhance/testdata/find_markdown_files/non_recursive.txt`
- `cmd/enhance/testdata/find_markdown_files/recursive.txt`

#### Example 2: Markdown Generation Testing
**Location:** `cmd/imdb/markdown_test.go`

Tests for writing movies to markdown format, with golden files for different scenarios:

**Golden Files:**
- `cmd/imdb/testdata/basic_movie.md` - Simple movie with basic fields
- `cmd/imdb/testdata/complex_movie.md` - Movie with all optional fields populated
- `cmd/imdb/testdata/tv_series.md` - TV series with different structure

#### Example 3: CSV Parser Testing
**Location:** `cmd/goodreads/parser_test.go`, `cmd/steam/parser_test.go`, `cmd/letterboxd/parser_test.go`

These packages use golden files to test CSV parsing and transformation:

**Golden File Directories:**
- `cmd/goodreads/testdata/` - CSV parsing golden files
- `cmd/steam/testdata/` - Steam data transformation golden files
- `cmd/letterboxd/testdata/` - Letterboxd CSV processing golden files

### Testing the Tests

When working with golden files:

1. **Verify initial creation:**
   ```bash
   UPDATE_GOLDEN=true go test ./cmd/mypackage/...
   git status  # Should show new files in testdata/
   cat cmd/mypackage/testdata/my_golden.txt  # Review content
   ```

2. **Verify comparison works:**
   ```bash
   go test ./cmd/mypackage/...  # Should pass

   # Intentionally break output to verify test catches changes
   # Edit code to produce different output
   go test ./cmd/mypackage/...  # Should fail with diff
   ```

3. **Verify update mode:**
   ```bash
   # Make intentional change to output format
   UPDATE_GOLDEN=true go test ./cmd/mypackage/...
   git diff cmd/mypackage/testdata/  # Should show the change
   ```

### Common Issues

#### Golden File Not Found
```
Error: open testdata/my_file.txt: no such file or directory
```
**Solution:** Run with `UPDATE_GOLDEN=true` to create the golden file initially.

#### Test Fails After Refactoring
```
Expected: <golden file content>
Actual: <new output>
```
**Solution:**
1. Review the diff to understand what changed
2. If change is intentional, run `UPDATE_GOLDEN=true go test ...`
3. Review and commit the updated golden files

#### Line Ending Issues (Windows/Unix)
**Solution:** Ensure consistent line endings in golden files. Git should be configured with:
```bash
git config core.autocrlf input  # On Unix/Mac
git config core.autocrlf true   # On Windows
```

### Related Tools

- **TestEnv** (`internal/testutil/testutil.go`) - Creates isolated test directories
  - `env.WriteFileString(name, content)` - Create test files
  - `env.ReadFile(name)` - Read test files
  - `env.RootDir()` - Get test directory path

- **Test Utilities** - Helper functions for test setup
  - `SetTestConfig(t)` - Configure test environment
  - `NewTestEnv(t)` - Create isolated test directory
