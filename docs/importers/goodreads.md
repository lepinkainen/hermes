# Goodreads Importer

This document describes the Goodreads importer in Hermes, which processes Goodreads library export data and converts it to JSON and Markdown formats.

## Overview

The Goodreads importer parses CSV files exported from Goodreads and enriches them with additional metadata from the OpenLibrary API. It then generates structured JSON and Markdown files for each book in your library.

## Data Source

### Goodreads Export

Goodreads allows users to export their library data as a CSV file:

1. Log in to your Goodreads account
2. Go to "My Books" (your shelves)
3. Click on "Import and export" at the bottom of the left sidebar
4. Click on "Export Library"
5. Wait for the export to be generated and download the CSV file

The CSV file contains basic information about your books, including:

- Book ID
- Title
- Author(s)
- ISBN/ISBN13
- Your rating and review
- Shelves
- Date added/read
- Publication information

### Data Enrichment

The importer enriches the basic Goodreads data with additional information from the [OpenLibrary API](https://openlibrary.org/developers/api), including:

- Cover images
- Detailed descriptions
- Subject categories
- Additional publication details
- More accurate page counts

## Usage

### Command-Line Usage

```bash
./hermes goodreads --input path/to/goodreads_export.csv
```

### Configuration

In your `config.yaml` file:

```yaml
goodreads:
  csvfile: "./path/to/goodreads_export.csv"
  output:
    markdown: "./markdown/books"
    json: "./json/books"
```

### Command-Line Options

- `--input`, `-f`: Path to the Goodreads CSV export file
- `--output-dir`: Directory for Markdown output (default: `./markdown/goodreads`)
- `--write-json`: Enable JSON output
- `--json-output`: Path for JSON output file (default: `./json/goodreads.json`)
- `--overwrite`: Overwrite existing files (default: false)

## Output Format

### Markdown Files

Each book is saved as a separate Markdown file with YAML frontmatter containing metadata. The filename is derived from the book title.

Example Markdown output:

```markdown
---
title: "1984"
type: book
goodreads_id: 5470
year: 1949
original_year: 1949
my_rating: 5
average_rating: 4.19
date_read: "2022-04-01"
date_added: "2022-03-15"
pages: 328
publisher: "Signet Classics"
binding: "Paperback"
isbn: "0451524934"
isbn13: "9780451524935"
authors:
  - "George Orwell"
bookshelves:
  - "classics"
  - "fiction"
  - "dystopian"
tags:
  - goodreads/book
  - rating/5
  - year/1940s
  - shelf/read
description: |
  Among the seminal texts of the 20th century, Nineteen Eighty-Four is a rare work that grows more haunting as its futuristic purgatory becomes more real...
subjects:
  - "Dystopian"
  - "Classics"
  - "Fiction"
cover_url: "https://covers.openlibrary.org/b/id/8575765-L.jpg"
---

# 1984

![Cover](https://covers.openlibrary.org/b/id/8575765-L.jpg)

## Review

A chilling dystopian masterpiece.
```

### JSON Output

All books are saved in a single JSON file as an array of book objects.

Example JSON output:

```json
[
  {
    "Book Id": 5470,
    "Title": "1984",
    "Authors": ["George Orwell"],
    "ISBN": "0451524934",
    "ISBN13": "9780451524935",
    "My Rating": 5,
    "Average Rating": 4.19,
    "Publisher": "Signet Classics",
    "Binding": "Paperback",
    "Number of Pages": 328,
    "Year Published": 1949,
    "Original Publication Year": 1949,
    "Date Read": "2022-04-01",
    "Date Added": "2022-03-15",
    "Bookshelves": ["classics", "fiction", "dystopian"],
    "Bookshelves with positions": ["classics", "fiction", "dystopian"],
    "Exclusive Shelf": "read",
    "My Review": "A chilling dystopian masterpiece.",
    "Spoiler": "",
    "Private Notes": "",
    "Read Count": 1,
    "Owned Copies": 1,
    "Description": "Among the seminal texts of the 20th century, Nineteen Eighty-Four is a rare work that grows more haunting as its futuristic purgatory becomes more real...",
    "Subjects": ["Dystopian", "Classics", "Fiction"],
    "Cover ID": 8575765,
    "Cover URL": "https://covers.openlibrary.org/b/id/8575765-L.jpg"
  }
]
```

## Caching

The Goodreads importer implements caching for OpenLibrary API responses to:

1. Respect API rate limits
2. Improve performance for subsequent imports
3. Allow for offline processing of previously fetched data

Cache files are stored in the `cache/goodreads/` directory, with filenames based on the book's ISBN.

## Implementation Details

### CSV Parsing

The importer reads the Goodreads CSV export file and extracts the following information:

- Basic book metadata (title, author, ISBN)
- Your ratings and reviews
- Reading status and dates
- Shelves and tags

### OpenLibrary Integration

For each book with an ISBN, the importer:

1. Checks if data is already cached
2. If not cached, fetches data from OpenLibrary API
3. Extracts additional metadata (cover images, descriptions, subjects)
4. Caches the response for future use

### Output Generation

The importer generates:

1. One Markdown file per book, with a filename derived from the book title
2. A single JSON file containing all books (if JSON output is enabled)

## Troubleshooting

### Missing Cover Images

If cover images are missing, it may be because:

- The book's ISBN is not in the OpenLibrary database
- The book doesn't have a cover image in OpenLibrary
- There was an error fetching the cover image

### Rate Limiting

OpenLibrary doesn't have strict rate limits, but the importer implements caching to be a good API citizen. If you're processing a large library, the initial import might take some time, but subsequent imports will be faster due to caching.

### ISBN Format Issues

Some Goodreads exports may have ISBN formatting issues. The importer attempts to clean up common formatting problems, but if you notice issues with specific books, check the ISBN format in the export file.
