# Documentation Update Task List

This document outlines the remaining steps to complete the Hermes project documentation update.

## Remaining Importers Documentation

### 1. Letterboxd Importer

- [x] Read `cmd/letterboxd/markdown.go` to understand output formatting
- [x] Read `cmd/letterboxd/json.go` to understand JSON output
- [x] Create `docs/importers/letterboxd.md` with the following sections:
  - Overview
  - Data Source (Letterboxd export format)
  - Data Enrichment (OMDb API)
  - Usage (command-line options, configuration)
  - Output Format (Markdown and JSON examples)
  - Caching
  - Implementation Details
  - Troubleshooting

### 2. Steam Importer

- [x] Read `cmd/steam/cmd.go` to understand command structure
- [x] Read `cmd/steam/parser.go` to understand data parsing
- [x] Read `cmd/steam/steam.go` to understand API integration
- [x] Read `cmd/steam/cache.go` to understand caching implementation
- [x] Read `cmd/steam/markdown.go` to understand output formatting
- [x] Read `cmd/steam/json.go` to understand JSON output
- [x] Read `cmd/steam/types.go` to understand data structures
- [x] Create `docs/importers/steam.md` with the following sections:
  - Overview
  - Data Source (Steam API)
  - Data Enrichment
  - Usage (command-line options, configuration, API key setup)
  - Output Format (Markdown and JSON examples)
  - Caching
  - Implementation Details
  - Troubleshooting
  - Rate Limiting Considerations

## Internal Utilities Documentation

### 3. File Utilities

- [x] Read `internal/fileutil/fileutil.go` to understand file operations
- [x] Read `internal/fileutil/markdown.go` to understand Markdown generation
- [x] Create `docs/utilities/fileutil.md` with the following sections:
  - Overview
  - File Operations
  - Markdown Generation
  - Usage Examples
  - API Reference

### 4. Command Utilities

- [x] Read `internal/cmdutil/base.go` to understand command setup utilities
- [x] Create `docs/utilities/cmdutil.md` with the following sections:
  - Overview
  - Command Setup
  - Flag Management
  - Usage Examples
  - API Reference

### 5. Configuration Utilities

- [x] Read `internal/config/config.go` to understand configuration management
- [x] Create `docs/utilities/config.md` with the following sections:
  - Overview
  - Configuration Loading
  - Default Values
  - Environment Variables
  - Usage Examples
  - API Reference

### 6. Error Utilities

- [x] Read `internal/errors/rate_limit.go` to understand custom error types
- [x] Create `docs/utilities/errors.md` with the following sections:
  - Overview
  - Custom Error Types
  - Error Handling Patterns
  - Usage Examples
  - API Reference

## Final Review and Cleanup

### 7. Documentation Review

- [x] Review all created documentation files for consistency
- [x] Ensure all links between documentation files work correctly
- [x] Check for any missing information or sections
- [x] Verify that examples are accurate and up-to-date
- [x] Ensure adherence to the documentation rules in `.clinerules/project-rules.md`

### 8. README Update

- [x] Update the main README.md if necessary to reference the new documentation

## Notes

- Follow the established documentation style and format from the existing files
- Ensure all documentation is accurate and reflects the current state of the code
- Include practical examples where appropriate
- Focus on making the documentation useful for both users and developers
