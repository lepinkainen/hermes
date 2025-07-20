# Netflix Importer Implementation Plan

> **FUTURE DEVELOPMENT IDEA**: This importer is not yet implemented and represents a planned future addition to the Hermes project.

## Overview

This importer will parse Netflix viewing history data and convert it to JSON and Markdown formats for local storage.

## Data Source

- Netflix allows users to download their viewing history as CSV files from [Account Settings](https://www.netflix.com/account/getmyinfo)
- CSV contains basic information like title, date watched, and device
- No official API available, we'll need to enrich data from external sources

## Implementation Plan

### Directory Structure

```
cmd/
  └── netflix/
      ├── cmd.go          # Command registration with Kong
      ├── parser.go       # CSV parsing logic
      ├── types.go        # Data models
      ├── tmdb.go         # TMDB integration for metadata enrichment
      ├── cache.go        # Caching for API calls
      ├── json.go         # JSON output formatter
      ├── markdown.go     # Markdown output formatter
      └── testdata/       # Test files
```

### Implementation Steps

1. **Command Setup**

   - Add a new `netflix` command to the CLI using Kong
   - Configure file input/output flags similar to existing importers

2. **Data Parsing**

   - Implement CSV parser for Netflix viewing history
   - Map data to internal structures

3. **Data Enrichment**

   - Integrate with TMDB (The Movie Database) API to fetch additional metadata
   - Implement caching similar to OMDB implementation
   - Fetch posters, descriptions, genres, cast, etc.

4. **Output Generation**

   - Implement JSON formatter
   - Implement Markdown formatter with similar structure to IMDB output

5. **Testing**
   - Add unit tests for parser logic
   - Create sample test data

## Technical Considerations

- **Ambiguous Titles**: Netflix's export doesn't include unique IDs, making exact matches on TMDB challenging
- **Rate Limiting**: TMDB has API rate limits; implement proper caching and throttling
- **Title vs. Episode**: Distinguish between shows and individual episodes
- **Watch Progress**: Netflix exports don't include watch percentage; only completed items

## External Dependencies

- TMDb Go Client for API integration (`github.com/ryanbradynd05/go-tmdb` or similar)
