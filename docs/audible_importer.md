# Audible Importer Implementation Plan

## Overview

This importer will process Audible library data to create a local collection of audiobooks in JSON and Markdown formats.

## Data Source

- Audible provides a "Library Export" feature that generates a CSV file with your purchased audiobooks
- Export contains basic information: title, author, purchase date, and listening status
- No direct official API, but we can scrape additional data from public Audible pages

## Implementation Plan

### Directory Structure

```
cmd/
  └── audible/
      ├── cmd.go           # Command registration with Cobra
      ├── parser.go        # CSV parsing logic
      ├── types.go         # Data models
      ├── enricher.go      # Data enrichment via web scraping or OpenLibrary API
      ├── cache.go         # Caching for API/web requests
      ├── json.go          # JSON output formatter
      ├── markdown.go      # Markdown output formatter
      └── testdata/        # Test files
```

### Implementation Steps

1. **Command Setup**

   - Add a new `audible` command to the CLI using Cobra
   - Configure file input/output flags similar to existing importers

2. **Data Parsing**

   - Implement CSV parser for Audible library export
   - Map data to internal structures

3. **Data Enrichment**

   - Use OpenLibrary API to fetch additional book metadata
   - Optionally implement a web scraper to fetch Audible-specific data like ratings, duration
   - Fetch cover images, descriptions, narrator info, series info

4. **Output Generation**

   - Implement JSON formatter
   - Implement Markdown formatter with structured audiobook information

5. **Testing**
   - Add unit tests for parser logic
   - Create sample test data

## Technical Considerations

- **Web Scraping Ethics**: Respect Audible's robots.txt and implement proper rate limiting
- **Book Identification**: Use ISBN when available to get accurate metadata
- **Series Handling**: Group books by series and show reading order
- **Listening Progress**: Capture completion status if available
- **Audible Regions**: Handle differences in Audible's regional sites (US, UK, etc.)

## External Dependencies

- OpenLibrary API client or standard HTTP client for API calls
- `github.com/gocolly/colly` for optional web scraping capabilities
