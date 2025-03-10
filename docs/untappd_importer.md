# Untappd Importer Implementation Plan

## Overview

This importer will process Untappd check-in data to create a local collection of beer ratings and tasting notes in JSON and Markdown formats.

## Data Source

- Untappd allows users to export their check-ins and history via the website
- Export format is CSV containing beer name, brewery, rating, location, and check-in notes
- Untappd offers a public API that requires registration for an API key

## Implementation Plan

### Directory Structure

```
cmd/
  └── untappd/
      ├── cmd.go          # Command registration with Cobra
      ├── parser.go       # CSV parsing logic
      ├── types.go        # Data models
      ├── api.go          # Untappd API integration
      ├── cache.go        # Caching for API calls
      ├── json.go         # JSON output formatter
      ├── markdown.go     # Markdown output formatter
      └── testdata/       # Test files
```

### Implementation Steps

1. **Command Setup**

   - Add a new `untappd` command to the CLI using Cobra
   - Configure file input/output flags similar to existing importers
   - Add API key configuration via Viper

2. **Data Parsing**

   - Implement CSV parser for Untappd check-in history
   - Map data to internal structures

3. **Data Enrichment**

   - Integrate with Untappd API to fetch additional metadata
   - Implement caching to respect rate limits
   - Fetch beer images, detailed descriptions, brewery info, style categorization

4. **Output Generation**

   - Implement JSON formatter
   - Implement Markdown formatter with structured beer/brewery information
   - Include statistics and visualizations (top breweries, styles, etc.)

5. **Testing**
   - Add unit tests for parser logic
   - Create sample test data

## Technical Considerations

- **API Rate Limits**: Untappd has strict API rate limits (100 calls/hour for free tier)
- **Data Structure**: Group check-ins by beer and brewery for better organization
- **Location Data**: Handle geographical information for check-ins
- **Image Handling**: Process and store beer/brewery label images
- **Authentication**: Manage API tokens securely

## External Dependencies

- HTTP client for Untappd API calls
- `github.com/go-echarts/go-echarts` for optional visualization capabilities

## Notes

- If using the Untappd API rather than CSV exports, users would need to register for their own API credentials at [Untappd Developer Portal](https://untappd.com/api/docs)
