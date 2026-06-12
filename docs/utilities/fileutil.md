# File Utilities

This document describes the file utility functions in Hermes, which provide common operations for file handling and output writing.

## Overview

The `fileutil` package provides a set of utilities for:

1. File operations (creating, writing, and checking files)
2. Filename sanitization and path generation
3. Cover image downloading
4. Formatting utilities for durations

These utilities are used throughout the Hermes importers to ensure consistent file handling and output formatting.

> **Note:** Markdown documents with YAML frontmatter are composed with the
> `internal/obsidian` package (`Frontmatter`, `TagSet`, `Note`), not by
> `fileutil`. See the importer `markdown.go` files for the standard pattern:
> build frontmatter with `obsidian.NewFrontmatterWithTitle`, collect tags with
> `obsidian.NewTagSet`, render with `obsidian.BuildNoteMarkdown`, then write
> with `fileutil.WriteMarkdownFile`.

## File Operations

### File Existence Check

The `FileExists` function checks if a file exists at a given path:

```go
func FileExists(filePath string) bool
```

Example usage:

```go
if fileutil.FileExists("path/to/file.md") {
    // File exists, do something
} else {
    // File doesn't exist
}
```

### Writing Files with Overwrite Control

The `WriteFileWithOverwrite` function writes data to a file, respecting an overwrite flag:

```go
func WriteFileWithOverwrite(filePath string, data []byte, perm os.FileMode, overwrite bool) (bool, error)
```

This function:

- Checks if the file already exists
- Skips writing if the file exists and overwrite is false
- Creates any necessary directories
- Returns true if the file was written, false if it was skipped

Convenience wrappers exist for the two common output formats:

```go
func WriteMarkdownFile(filePath string, content string, overwrite bool) error
func WriteJSONFile(data any, filePath string, overwrite bool) (bool, error)
```

### Path Generation

The `GetMarkdownFilePath` function generates a standardized path for a Markdown file:

```go
func GetMarkdownFilePath(name string, directory string) string
```

This function:

- Sanitizes the filename to remove problematic characters
- Joins the directory and sanitized filename with a `.md` extension

Example usage:

```go
path := fileutil.GetMarkdownFilePath("My Book: A Story", "markdown/books")
// Returns: "markdown/books/My Book - A Story.md"
```

### Filename Sanitization

The `SanitizeFilename` function cleans a filename by replacing problematic characters:

```go
func SanitizeFilename(name string) string
```

This function replaces characters like `:`, `/`, and `\` with safe alternatives.

Example usage:

```go
filename := fileutil.SanitizeFilename("Star Wars: Episode IV")
// Returns: "Star Wars - Episode IV"
```

## Cover Images

The cover pipeline downloads cover art into an `attachments/` directory next to
the notes and reuses existing files unless an update is forced:

```go
func BuildCoverFilename(title string) string
func DownloadCover(ctx context.Context, opts CoverDownloadOptions) (*CoverDownloadResult, error)
```

Example usage:

```go
result, err := fileutil.DownloadCover(ctx, fileutil.CoverDownloadOptions{
    URL:          movie.PosterURL,
    OutputDir:    outputDir,
    Filename:     fileutil.BuildCoverFilename(movie.Title),
    UpdateCovers: config.UpdateCovers,
})
if err == nil && result != nil {
    fm.Set("cover", result.RelativePath)
}
```

Respect `CoverDownloadOptions.UpdateCovers` rather than deleting files manually.

## Formatting Utilities

### Duration Formatting

The `FormatDuration` function formats minutes into a human-readable duration:

```go
func FormatDuration(minutes int) string
```

Example usage:

```go
duration := fileutil.FormatDuration(150)
// Returns: "2h 30m"
```

## API Reference

### File Operations

| Function                                                                                               | Description                                              |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| `FileExists(filePath string) bool`                                                                     | Checks if a file exists at the given path                |
| `WriteFileWithOverwrite(filePath string, data []byte, perm os.FileMode, overwrite bool) (bool, error)` | Writes data to a file, respecting the overwrite flag     |
| `WriteMarkdownFile(filePath string, content string, overwrite bool) error`                             | Writes a markdown file and logs the result               |
| `WriteJSONFile(data any, filePath string, overwrite bool) (bool, error)`                               | Marshals data to JSON and writes it                      |
| `GetMarkdownFilePath(name string, directory string) string`                                            | Returns the expected markdown file path for a given name |
| `SanitizeFilename(name string) string`                                                                 | Cleans a filename by replacing problematic characters    |
| `RelativeTo(base, target string) (string, error)`                                                      | Returns target relative to base                          |
| `CopyFile(src, dst string) error`                                                                      | Copies a file                                            |

### Cover Images

| Function                                                                                | Description                                          |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| `BuildCoverFilename(title string) string`                                              | Returns the standard `"Title - cover.jpg"` filename  |
| `DownloadCover(ctx context.Context, opts CoverDownloadOptions) (*CoverDownloadResult, error)` | Downloads a cover into `attachments/`, reusing files |

### Formatting Utilities

| Function                             | Description                                                  |
| ------------------------------------ | ------------------------------------------------------------ |
| `FormatDuration(minutes int) string` | Formats minutes into human-readable duration (e.g. "2h 30m") |

---

*Document created: 2025-04-30*
*Last reviewed: 2026-06-12*
