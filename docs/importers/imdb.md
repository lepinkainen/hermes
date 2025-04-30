# IMDb Importer

This document describes the IMDb importer in Hermes, which processes IMDb export data and converts it to JSON and Markdown formats.

## Overview

The IMDb importer parses CSV files exported from IMDb (Internet Movie Database) and enriches them with additional metadata from the OMDb API (Open Movie Database). It then generates structured JSON and Markdown files for each movie or TV show in your ratings or watchlist.

## Data Source

### IMDb Export

IMDb allows users to export their ratings and watchlist data as CSV files:

1. Log in to your IMDb account
2. Go to your ratings page or watchlist
3. Click on the "..." (three dots) menu
4. Select "Export"
5. Download the CSV file

The CSV file contains basic information about your rated movies and TV shows, including:

- IMDb ID
- Title
- Your rating (1-10)
- Date rated
- Release year
- Runtime
- Genres
- Directors

### Data Enrichment

The importer enriches the basic IMDb data with additional information from the [OMDb API](http://www.omdbapi.com/), including:

- Poster images
- Plot summaries
- Content ratings (PG, R, etc.)
- Awards information
- More accurate genre and director information

## Usage

### Command-Line Usage

```bash
./hermes imdb --input path/to/imdb_ratings.csv --apikey your_omdb_api_key
```

### Configuration

In your `config.yaml` file:

```yaml
imdb:
  csvfile: "./path/to/imdb_ratings.csv"
  omdb_api_key: "your_omdb_api_key" # Get from http://www.omdbapi.com/apikey.aspx
  output:
    markdown: "./markdown/movies"
    json: "./json/movies"
```

### Command-Line Options

- `--input`, `-f`: Path to the IMDb CSV export file
- `--output-dir`: Directory for Markdown output (default: `./markdown/imdb`)
- `--write-json`: Enable JSON output
- `--json-output`: Path for JSON output file (default: `./json/imdb.json`)
- `--skip-invalid`: Skip invalid entries instead of failing
- `--overwrite`: Overwrite existing files (default: false)

## Output Format

### Markdown Files

Each movie or TV show is saved as a separate Markdown file with YAML frontmatter containing metadata. The filename is derived from the title.

Example Markdown output:

```markdown
---
title: "The Shawshank Redemption"
type: movie
imdb_id: "tt0111161"
year: 1994
imdb_rating: 9.3
my_rating: 10
date_rated: "2021-07-22"
runtime_mins: 142
duration: 2h 22m
genres:
  - "Drama"
directors:
  - "Frank Darabont"
tags:
  - imdb/movie
  - rating/10
  - year/1990s
content_rating: "R"
awards: "Nominated for 7 Oscars. 21 wins & 43 nominations total"
---

# The Shawshank Redemption

![](https://m.media-amazon.com/images/M/MV5BMDFkYTc0MGEtZmNhMC00ZDIzLWFmNTEtODM1ZmRlYWMwMWFmXkEyXkFqcGdeQXVyMTMxODk2OTU@._V1_SX300.jpg)

> [!summary] Plot
> Over the course of several years, two convicts form a friendship, seeking consolation and, eventually, redemption through basic compassion.

> [!award] Awards
> Nominated for 7 Oscars. 21 wins & 43 nominations total

> [!info] IMDb
> [View on IMDb](https://www.imdb.com/title/tt0111161/)
```

### JSON Output

All movies and TV shows are saved in a single JSON file as an array of objects.

Example JSON output:

```json
[
  {
    "ImdbId": "tt0111161",
    "My Rating": 10,
    "Date Rated": "2021-07-22",
    "Title": "The Shawshank Redemption",
    "Original Title": "The Shawshank Redemption",
    "URL": "https://www.imdb.com/title/tt0111161/",
    "Title Type": "Movie",
    "IMDb Rating": 9.3,
    "Runtime (mins)": 142,
    "Year": 1994,
    "Genres": ["Drama"],
    "Num Votes": 2578136,
    "Release Date": "1994-10-14",
    "Directors": ["Frank Darabont"],
    "Plot": "Over the course of several years, two convicts form a friendship, seeking consolation and, eventually, redemption through basic compassion.",
    "Content Rated": "R",
    "Awards": "Nominated for 7 Oscars. 21 wins & 43 nominations total",
    "Poster URL": "https://m.media-amazon.com/images/M/MV5BMDFkYTc0MGEtZmNhMC00ZDIzLWFmNTEtODM1ZmRlYWMwMWFmXkEyXkFqcGdeQXVyMTMxODk2OTU@._V1_SX300.jpg"
  }
]
```

## Caching

The IMDb importer implements caching for OMDb API responses to:

1. Respect API rate limits (OMDb has a limit of 1,000 requests per day for the free tier)
2. Improve performance for subsequent imports
3. Allow for offline processing of previously fetched data

Cache files are stored in the `cache/omdb/` directory, with filenames based on the IMDb ID (e.g., `tt0111161.json`).

## Implementation Details

### CSV Parsing

The importer reads the IMDb CSV export file and extracts the following information:

- IMDb ID (e.g., tt0111161)
- Your rating (1-10)
- Date rated
- Title and original title
- Title type (Movie, TV Series, etc.)
- IMDb rating
- Runtime in minutes
- Release year
- Genres
- Number of votes
- Release date
- Directors

### OMDb API Integration

For each movie or TV show, the importer:

1. Checks if data is already cached
2. If not cached, fetches data from OMDb API using the IMDb ID
3. Extracts additional metadata (poster, plot, content rating, awards)
4. Caches the response for future use

The importer handles rate limiting by detecting when the OMDb API limit is reached and stopping further requests.

### Output Generation

The importer generates:

1. One Markdown file per movie or TV show, with a filename derived from the title
2. A single JSON file containing all movies and TV shows (if JSON output is enabled)

## Troubleshooting

### API Rate Limits

OMDb has a limit of 1,000 requests per day for the free tier. If you have a large collection, you may hit this limit during import. The importer will detect this and stop with an error message. You can resume the import the next day, and it will continue from where it left off thanks to caching.

### Missing or Incorrect Data

If you notice missing or incorrect data in the output:

1. Check if the IMDb ID in your export file is correct
2. Verify that the OMDb API has data for that IMDb ID
3. Consider updating your IMDb rating or review to refresh the export data

### CSV Format Changes

IMDb occasionally changes the format of their export files. If you encounter parsing errors, check if the CSV format has changed and report the issue.

## OMDb API Key

To use the IMDb importer, you need an OMDb API key:

1. Go to [OMDb API](http://www.omdbapi.com/apikey.aspx)
2. Request a free API key (1,000 requests per day)
3. Activate the key via the email you receive
4. Add the key to your `config.yaml` file or use the `--apikey` flag
