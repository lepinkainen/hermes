# Letterboxd Importer

This document describes the Letterboxd importer in Hermes, which processes Letterboxd export data and converts it to JSON and Markdown formats.

## Overview

The Letterboxd importer parses CSV files exported from Letterboxd (a social network for sharing film reviews and ratings) and enriches them with additional metadata from the OMDb API (Open Movie Database). It then generates structured JSON and Markdown files for each movie in your diary or watchlist.

## Data Source

### Letterboxd Export

Letterboxd allows users to export their diary and watchlist data as CSV files:

1. Log in to your Letterboxd account
2. Go to your profile settings
3. Click on "Import & Export"
4. Click on "Export Your Data"
5. Download the CSV file(s)

The CSV file contains basic information about your watched movies, including:

- Date watched
- Movie title
- Release year
- Letterboxd URI
- Rating (if provided)

### Data Enrichment

The importer enriches the basic Letterboxd data with additional information from the [OMDb API](http://www.omdbapi.com/), including:

- Poster images
- Plot summaries
- Director information
- Cast lists
- Genres
- Runtime
- IMDb ID

## Usage

### Command-Line Usage

```bash
./hermes letterboxd --input path/to/letterboxd_export.csv --apikey your_omdb_api_key
```

### Configuration

In your `config.yml` (or `config.yaml`) file:

```yaml
letterboxd:
  csvfile: "./path/to/letterboxd_export.csv"
  omdb_api_key: "your_omdb_api_key" # Get from http://www.omdbapi.com/apikey.aspx
  output:
    markdown: "./markdown/films"
    json: "./json/films"
```

### Command-Line Options

- `--input`, `-f`: Path to the Letterboxd CSV export file
- `--output-dir`: Directory for Markdown output (default: `./markdown/letterboxd`)
- `--write-json`: Enable JSON output
- `--json-output`: Path for JSON output file (default: `./json/letterboxd.json`)
- `--skip-invalid`: Skip invalid entries instead of failing
- `--skip-enrich`: Skip enriching data with OMDB API
- `--overwrite`: Overwrite existing files (default: false)

## Output Format

### Markdown Files

Each movie is saved as a separate Markdown file with YAML frontmatter containing metadata. The filename is derived from the movie title and year.

Example Markdown output:

```markdown
---
title: "The Godfather"
type: movie
year: 1972
date_watched: "2023-01-15"
letterboxd_rating: 5.0
runtime_mins: 175
duration: 2h 55m
directors:
  - "Francis Ford Coppola"
genres:
  - "Crime"
  - "Drama"
tags:
  - letterboxd/movie
  - rating/5.0
  - year/1970s
letterboxd_uri: "https://letterboxd.com/film/the-godfather/"
letterboxd_id: "the-godfather"
imdb_id: "tt0068646"
cover: "https://m.media-amazon.com/images/M/MV5BM2MyNjYxNmUtYTAwNi00MTYxLWJmNWYtYzZlODY3ZTk3OTFlXkEyXkFqcGdeQXVyNzkwMjQ5NzM@._V1_SX300.jpg"
---

# The Godfather

![](https://m.media-amazon.com/images/M/MV5BM2MyNjYxNmUtYTAwNi00MTYxLWJmNWYtYzZlODY3ZTk3OTFlXkEyXkFqcGdeQXVyNzkwMjQ5NzM@._V1_SX300.jpg)

> [!summary] Plot
> The aging patriarch of an organized crime dynasty transfers control of his clandestine empire to his reluctant son.

> [!cast] Cast
>
> - Marlon Brando
> - Al Pacino
> - James Caan
> - Richard S. Castellano
> - Robert Duvall

> [!info] Letterboxd
> [View on Letterboxd](https://letterboxd.com/film/the-godfather/) > [View on IMDb](https://www.imdb.com/title/tt0068646)
```

### JSON Output

All movies are saved in a single JSON file as an array of objects.

Example JSON output:

```json
[
  {
    "Date": "2023-01-15",
    "Name": "The Godfather",
    "Year": 1972,
    "LetterboxdID": "the-godfather",
    "LetterboxdURI": "https://letterboxd.com/film/the-godfather/",
    "ImdbID": "tt0068646",
    "Director": "Francis Ford Coppola",
    "Cast": [
      "Marlon Brando",
      "Al Pacino",
      "James Caan",
      "Richard S. Castellano",
      "Robert Duvall"
    ],
    "Genres": ["Crime", "Drama"],
    "Runtime": 175,
    "Rating": 5.0,
    "PosterURL": "https://m.media-amazon.com/images/M/MV5BM2MyNjYxNmUtYTAwNi00MTYxLWJmNWYtYzZlODY3ZTk3OTFlXkEyXkFqcGdeQXVyNzkwMjQ5NzM@._V1_SX300.jpg",
    "Description": "The aging patriarch of an organized crime dynasty transfers control of his clandestine empire to his reluctant son."
  }
]
```

## Caching

The Letterboxd importer implements caching for OMDb API responses to:

1. Respect API rate limits (OMDb has a limit of 1,000 requests per day for the free tier)
2. Improve performance for subsequent imports
3. Allow for offline processing of previously fetched data

Cache files are stored in two locations:

- `cache/letterboxd/`: Movie data cached by title and year
- `cache/omdb/`: Movie data cached by IMDb ID (shared with the IMDb importer)

The importer first checks the Letterboxd cache, then the OMDB cache, and finally falls back to the OMDb API if needed.

## Implementation Details

### CSV Parsing

The importer reads the Letterboxd CSV export file and extracts the following information:

- Date watched
- Movie title
- Release year
- Letterboxd URI
- Letterboxd ID (extracted from the URI)

### OMDb API Integration

For each movie, the importer:

1. Checks if data is already cached in the Letterboxd cache
2. If not found, checks if data is already cached in the OMDB cache
3. If not found in either cache, fetches data from OMDb API using the movie title and year
4. Extracts additional metadata (poster, plot, director, cast, genres, runtime)
5. Caches the response in both caches for future use

The importer handles rate limiting by detecting when the OMDb API limit is reached and stopping further requests.

### Output Generation

The importer generates:

1. One Markdown file per movie, with a filename derived from the title and year
2. A single JSON file containing all movies (if JSON output is enabled)

## Troubleshooting

### API Rate Limits

OMDb has a limit of 1,000 requests per day for the free tier. If you have a large collection, you may hit this limit during import. The importer will detect this and stop with an error message. You can resume the import the next day, and it will continue from where it left off thanks to caching.

### Missing or Incorrect Data

If you notice missing or incorrect data in the output:

1. Check if the movie title and year in your export file are correct
2. Verify that the OMDb API has data for that movie
3. Try using the `--skip-enrich` flag if you only need the basic Letterboxd data

### CSV Format Changes

Letterboxd occasionally changes the format of their export files. If you encounter parsing errors, check if the CSV format has changed and report the issue.

## OMDb API Key

To use the Letterboxd importer with data enrichment, you need an OMDb API key:

1. Go to [OMDb API](http://www.omdbapi.com/apikey.aspx)
2. Request a free API key (1,000 requests per day)
3. Activate the key via the email you receive
4. Add the key to your `config.yml` (or `config.yaml`) file or use the `--apikey` flag

Note that the Letterboxd importer can use the same OMDb API key as the IMDb importer, so you only need to obtain one key for both importers.
