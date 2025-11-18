# Output Formats

This document describes the output formats used by Hermes to store imported data. Hermes generates two types of output files: JSON and Markdown.

## Output Directories

By default, output files are stored in the following directories:

- **JSON**: `./json/{importer_name}/`
- **Markdown**: `./markdown/{importer_name}/`

For example, Goodreads data would be stored in `./json/goodreads/` and `./markdown/goodreads/`.

These directories can be customized in the configuration file or via command-line flags.

## JSON Format

JSON files provide a structured representation of the imported data, suitable for programmatic access and further processing.

### File Structure

Each imported item is stored in a separate JSON file, with a filename derived from the item's title or identifier. The JSON structure varies by importer but generally follows this pattern:

```json
{
  "id": "unique_identifier",
  "title": "Item Title",
  "source": {
    "name": "Source Name",
    "url": "https://source.url/item",
    "id": "source_specific_id"
  },
  "user_data": {
    "rating": 5,
    "date_added": "2023-01-15",
    "review": "User review text"
  },
  "metadata": {
    // Source-specific metadata
  }
}
```

### Example: Goodreads Book (JSON)

```json
{
  "id": "9780451524935",
  "title": "1984",
  "author": "George Orwell",
  "source": {
    "name": "Goodreads",
    "url": "https://www.goodreads.com/book/show/5470",
    "id": "5470"
  },
  "user_data": {
    "rating": 5,
    "date_added": "2022-03-15",
    "date_read": "2022-04-01",
    "review": "A chilling dystopian masterpiece."
  },
  "metadata": {
    "isbn": "9780451524935",
    "isbn13": "9780451524935",
    "publication_year": 1949,
    "page_count": 328,
    "genres": ["Dystopian", "Classics", "Fiction"],
    "description": "Among the seminal texts of the 20th century...",
    "cover_url": "https://covers.openlibrary.org/b/isbn/9780451524935-L.jpg"
  }
}
```

### Example: IMDb Movie (JSON)

```json
{
  "id": "tt0111161",
  "title": "The Shawshank Redemption",
  "source": {
    "name": "IMDb",
    "url": "https://www.imdb.com/title/tt0111161/",
    "id": "tt0111161"
  },
  "user_data": {
    "rating": 10,
    "date_rated": "2021-07-22",
    "watchlist": false
  },
  "metadata": {
    "year": 1994,
    "director": "Frank Darabont",
    "runtime": 142,
    "genres": ["Drama"],
    "plot": "Two imprisoned men bond over a number of years...",
    "cast": ["Tim Robbins", "Morgan Freeman", "Bob Gunton"],
    "poster_url": "https://m.media-amazon.com/images/M/MV5BMDFkYTc0MGEtZmNhMC00ZDIzLWFmNTEtODM1ZmRlYWMwMWFmXkEyXkFqcGdeQXVyMTMxODk2OTU@._V1_.jpg",
    "imdb_rating": 9.3
  }
}
```

## Markdown Format

Markdown files provide a human-readable representation of the imported data, suitable for knowledge management systems like Obsidian. The Markdown files include YAML frontmatter with structured metadata.

### File Structure

Each imported item is stored in a separate Markdown file, with a filename derived from the item's title or identifier. The Markdown structure includes:

1. **YAML Frontmatter**: Structured metadata at the top of the file
2. **Content**: The main content of the file, including user reviews and notes

```markdown
---
# YAML Frontmatter with metadata
---

# Main content
```

### Example: Goodreads Book (Markdown)

```markdown
---
title: "1984"
author: "George Orwell"
isbn: "9780451524935"
isbn13: "9780451524935"
year: 1949
pages: 328
genres:
  - "Dystopian"
  - "Classics"
  - "Fiction"
cover: "https://covers.openlibrary.org/b/isbn/9780451524935-L.jpg"
goodreads_id: "5470"
goodreads_url: "https://www.goodreads.com/book/show/5470"
date_added: "2022-03-15"
date_read: "2022-04-01"
rating: 5
---

# 1984

![Cover](https://covers.openlibrary.org/b/isbn/9780451524935-L.jpg)

## Metadata

- **Author**: George Orwell
- **Published**: 1949
- **Pages**: 328
- **ISBN**: 9780451524935

## Description

Among the seminal texts of the 20th century...

## My Rating: ⭐⭐⭐⭐⭐

**Date Read**: April 1, 2022

## My Review

A chilling dystopian masterpiece.
```

### Example: IMDb Movie (Markdown)

```markdown
---
title: "The Shawshank Redemption"
year: 1994
director: "Frank Darabont"
runtime: 142
genres:
  - "Drama"
cast:
  - "Tim Robbins"
  - "Morgan Freeman"
  - "Bob Gunton"
imdb_id: "tt0111161"
imdb_url: "https://www.imdb.com/title/tt0111161/"
imdb_rating: 9.3
poster: "https://m.media-amazon.com/images/M/MV5BMDFkYTc0MGEtZmNhMC00ZDIzLWFmNTEtODM1ZmRlYWMwMWFmXkEyXkFqcGdeQXVyMTMxODk2OTU@._V1_.jpg"
date_rated: "2021-07-22"
my_rating: 10
seen: true
---

# The Shawshank Redemption (1994)

![Poster](https://m.media-amazon.com/images/M/MV5BMDFkYTc0MGEtZmNhMC00ZDIzLWFmNTEtODM1ZmRlYWMwMWFmXkEyXkFqcGdeQXVyMTMxODk2OTU@._V1_.jpg)

## Metadata

- **Director**: Frank Darabont
- **Year**: 1994
- **Runtime**: 142 minutes
- **IMDb Rating**: 9.3/10

## Plot

Two imprisoned men bond over a number of years...

## Cast

- Tim Robbins
- Morgan Freeman
- Bob Gunton

## My Rating: ⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐

**Date Rated**: July 22, 2021
```

## Customizing Output Formats

The output formats can be customized by modifying the formatter implementations in each importer:

- **JSON**: `cmd/{importer}/json.go`
- **Markdown**: `cmd/{importer}/markdown.go`

The formatters use the `internal/fileutil` package for common formatting operations.

### Markdown Frontmatter

The Markdown frontmatter is particularly important for integration with knowledge management systems like Obsidian. The frontmatter follows YAML syntax and includes all relevant metadata for the item.

Common frontmatter fields across importers:

- `title`: The title of the item
- `source_name`: The name of the data source (e.g., "Goodreads", "IMDb")
- `source_url`: The URL to the item on the source website
- `source_id`: The ID of the item on the source website
- `date_added`: When the item was added to the user's collection
- `rating`: The user's rating of the item
- `seen`: Boolean flag indicating if the item has been watched/read. Automatically set to true when any rating is present. If omitted or false, the item is considered unwatched/unread.

Importer-specific frontmatter fields are documented in the respective importer documentation.

## File Naming

Files are named using a sanitized version of the item's title or identifier:

- Special characters are removed or replaced
- Spaces are replaced with underscores or hyphens
- File extensions are added (`.json` or `.md`)

For example, a book titled "Harry Potter & the Philosopher's Stone" might be saved as `harry_potter_the_philosophers_stone.md`.

## Next Steps

- See [Caching](06_caching.md) for details on the caching implementation
- See [Logging & Error Handling](07_logging_error_handling.md) for information about logging and error handling
- See the importer-specific documentation for details on the output formats for each importer
