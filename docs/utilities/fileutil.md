# File Utilities

This document describes the file utility functions in Hermes, which provide common operations for file handling and Markdown generation.

## Overview

The `fileutil` package provides a set of utilities for:

1. File operations (creating, writing, and checking files)
2. Filename sanitization and path generation
3. Markdown document generation with frontmatter
4. Formatting utilities for dates, durations, and other content

These utilities are used throughout the Hermes importers to ensure consistent file handling and output formatting.

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

Example usage:

```go
content := []byte("# My Document\n\nThis is some content.")
written, err := fileutil.WriteFileWithOverwrite("markdown/document.md", content, 0644, false)
if err != nil {
    log.Errorf("Failed to write file: %v", err)
} else if written {
    log.Infof("Wrote file: markdown/document.md")
} else {
    log.Debugf("Skipped existing file: markdown/document.md")
}
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

## Markdown Generation

The `MarkdownBuilder` provides a fluent interface for constructing Markdown documents with YAML frontmatter.

### Creating a Markdown Builder

```go
mb := fileutil.NewMarkdownBuilder()
```

### Adding Frontmatter

The builder provides methods for adding various types of frontmatter fields:

```go
// Basic metadata
mb.AddTitle("My Book")
mb.AddType("book")
mb.AddYear(2023)

// Simple key-value fields
mb.AddField("author", "Jane Doe")
mb.AddField("rating", 4.5)
mb.AddField("pages", 320)
mb.AddField("published", true)

// Arrays
mb.AddStringArray("genres", []string{"Fiction", "Mystery", "Thriller"})

// Tags
mb.AddTags("book", "fiction", "mystery")

// Dates
mb.AddDate("published_date", "2023-04-15")

// Durations
mb.AddDuration(320) // 5h 20m
```

### Adding Content

The builder also provides methods for adding content to the document:

```go
// Add a paragraph
mb.AddParagraph("# My Book\n\nThis is a great book about...")

// Add an image
mb.AddImage("https://example.com/cover.jpg")

// Add a callout
mb.AddCallout("summary", "Plot", "This book follows the adventures of...")

// Add external links
mb.AddExternalLink("Publisher Website", "https://example.com")

// Add a callout with multiple links
links := map[string]string{
    "Goodreads": "https://goodreads.com/book/123",
    "Amazon": "https://amazon.com/dp/123456",
}
mb.AddExternalLinksCallout("External Links", links)
```

### Building the Document

Once all content has been added, build the complete document:

```go
document := mb.Build()
```

This returns a string containing the complete Markdown document with frontmatter.

### Complete Example

```go
mb := fileutil.NewMarkdownBuilder()
mb.AddTitle("The Great Gatsby")
mb.AddType("book")
mb.AddYear(1925)
mb.AddField("author", "F. Scott Fitzgerald")
mb.AddStringArray("genres", []string{"Fiction", "Classic"})
mb.AddTags("book", "classic", "fiction", mb.GetDecadeTag(1925))
mb.AddParagraph("# The Great Gatsby")
mb.AddImage("https://example.com/gatsby.jpg")
mb.AddCallout("summary", "Plot", "Set in the Jazz Age, this novel tells the story...")

document := mb.Build()
```

Output:

```markdown
---
title: "The Great Gatsby"
type: book
year: 1925
author: "F. Scott Fitzgerald"
genres:
  - "Fiction"
  - "Classic"
tags:
  - book
  - classic
  - fiction
  - year/1920s
---

# The Great Gatsby

![](https://example.com/gatsby.jpg)

> [!summary]- Plot
> Set in the Jazz Age, this novel tells the story...
```

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

### Decade Tag Generation

The `GetDecadeTag` method generates a decade tag based on a year:

```go
decadeTag := mb.GetDecadeTag(1985)
// Returns: "year/1980s"
```

This is useful for categorizing content by decade in the frontmatter tags.

## API Reference

### File Operations

| Function                                                                                               | Description                                              |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| `FileExists(filePath string) bool`                                                                     | Checks if a file exists at the given path                |
| `WriteFileWithOverwrite(filePath string, data []byte, perm os.FileMode, overwrite bool) (bool, error)` | Writes data to a file, respecting the overwrite flag     |
| `GetMarkdownFilePath(name string, directory string) string`                                            | Returns the expected markdown file path for a given name |
| `SanitizeFilename(name string) string`                                                                 | Cleans a filename by replacing problematic characters    |

### MarkdownBuilder Methods

| Method                                                                            | Description                                               |
| --------------------------------------------------------------------------------- | --------------------------------------------------------- |
| `NewMarkdownBuilder() *MarkdownBuilder`                                           | Creates a new markdown builder                            |
| `AddTitle(title string) *MarkdownBuilder`                                         | Adds a title field to the frontmatter                     |
| `AddType(mediaType string) *MarkdownBuilder`                                      | Adds a type field to the frontmatter                      |
| `AddYear(year int) *MarkdownBuilder`                                              | Adds a year field to the frontmatter                      |
| `AddField(key string, value interface{}) *MarkdownBuilder`                        | Adds a simple key-value field to the frontmatter          |
| `AddStringArray(key string, values []string) *MarkdownBuilder`                    | Adds an array of strings to the frontmatter               |
| `AddTags(tags ...string) *MarkdownBuilder`                                        | Adds a list of tags to the frontmatter                    |
| `GetDecadeTag(year int) string`                                                   | Returns a decade tag based on the year                    |
| `AddDuration(minutes int) *MarkdownBuilder`                                       | Adds a duration field to the frontmatter                  |
| `AddParagraph(text string) *MarkdownBuilder`                                      | Adds a paragraph of text to the content                   |
| `AddImage(imageURL string) *MarkdownBuilder`                                      | Adds an image to the content                              |
| `AddCallout(calloutType, title, content string) *MarkdownBuilder`                 | Adds a callout section to the content                     |
| `AddExternalLink(title, url string) *MarkdownBuilder`                             | Adds an external link to the content                      |
| `AddExternalLinksCallout(title string, links map[string]string) *MarkdownBuilder` | Adds a callout with external links                        |
| `AddDate(key string, dateStr string) *MarkdownBuilder`                            | Adds a date field to the frontmatter in YYYY-MM-DD format |
| `Build() string`                                                                  | Returns the complete markdown document as a string        |

### Formatting Utilities

| Function                             | Description                                                  |
| ------------------------------------ | ------------------------------------------------------------ |
| `FormatDuration(minutes int) string` | Formats minutes into human-readable duration (e.g. "2h 30m") |

## Usage Examples

### Creating a Markdown File for a Book

```go
func CreateBookMarkdown(book Book, outputDir string) error {
    mb := fileutil.NewMarkdownBuilder()

    // Add frontmatter
    mb.AddTitle(book.Title)
    mb.AddType("book")
    mb.AddYear(book.Year)
    mb.AddField("author", book.Author)
    mb.AddStringArray("genres", book.Genres)
    mb.AddTags("book", mb.GetDecadeTag(book.Year))

    // Add content
    mb.AddParagraph(fmt.Sprintf("# %s", book.Title))
    mb.AddImage(book.CoverURL)
    mb.AddCallout("summary", "Description", book.Description)

    // Build the document
    content := mb.Build()

    // Get the file path
    filePath := fileutil.GetMarkdownFilePath(book.Title, outputDir)

    // Write the file
    written, err := fileutil.WriteFileWithOverwrite(filePath, []byte(content), 0644, false)
    if err != nil {
        return err
    }

    if written {
        log.Infof("Created markdown for: %s", book.Title)
    } else {
        log.Debugf("Skipped existing file: %s", filePath)
    }

    return nil
}
```

### Creating a Markdown File for a Movie

```go
func CreateMovieMarkdown(movie Movie, outputDir string) error {
    mb := fileutil.NewMarkdownBuilder()

    // Add frontmatter
    mb.AddTitle(movie.Title)
    mb.AddType("movie")
    mb.AddYear(movie.Year)
    mb.AddField("director", movie.Director)
    mb.AddStringArray("cast", movie.Cast)
    mb.AddStringArray("genres", movie.Genres)
    mb.AddDuration(movie.RuntimeMinutes)
    mb.AddTags("movie", mb.GetDecadeTag(movie.Year))

    // Add content
    mb.AddParagraph(fmt.Sprintf("# %s", movie.Title))
    mb.AddImage(movie.PosterURL)
    mb.AddCallout("summary", "Plot", movie.Plot)

    // Add external links
    links := map[string]string{
        "View on IMDb": fmt.Sprintf("https://www.imdb.com/title/%s", movie.ImdbID),
    }
    mb.AddExternalLinksCallout("External Links", links)

    // Build the document
    content := mb.Build()

    // Get the file path
    filePath := fileutil.GetMarkdownFilePath(movie.Title, outputDir)

    // Write the file
    written, err := fileutil.WriteFileWithOverwrite(filePath, []byte(content), 0644, false)
    if err != nil {
        return err
    }

    if written {
        log.Infof("Created markdown for: %s", movie.Title)
    } else {
        log.Debugf("Skipped existing file: %s", filePath)
    }

    return nil
}
```
