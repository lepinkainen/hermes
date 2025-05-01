# Google Books API Enricher Implementation Plan

> **FUTURE DEVELOPMENT IDEA**: This enricher is not yet implemented and represents a planned future addition to the Hermes project's Goodreads importer.

## Overview

This component will enhance the existing Goodreads importer by fetching additional book metadata from the Google Books API, complementing or replacing data currently sourced from OpenLibrary. It will be implemented alongside a common `BookEnricher` interface to allow for multiple data sources.

## Data Source

- Google Books API (specifically the [Volumes API](https://developers.google.com/books/docs/v1/using#PerformingSearch))
- Requires an API key obtained from the Google Cloud Console.
- Provides rich metadata including descriptions, subjects, page counts, cover images, publisher info, etc.

## Implementation Plan

### Interface Definition (`cmd/goodreads/enricher.go`)

- [ ] Define a `BookEnricher` interface:
  ```go
  type BookEnricher interface {
      Enrich(book *Book) error
      Name() string
  }
  ```

### Refactor OpenLibrary (`cmd/goodreads/openlibrary.go`, `cmd/goodreads/enricher.go`)

- [ ] Create an `OpenLibraryClient` struct in `openlibrary.go` to encapsulate API call logic.
- [ ] Create an `OpenLibraryEnricher` struct in `enricher.go` that holds the client.
- [ ] Implement the `BookEnricher` interface for `OpenLibraryEnricher`.
- [ ] Update `cmd/goodreads/parser.go` to use the `OpenLibraryEnricher` via the interface.

### Google Books Implementation (`cmd/goodreads/googlebooks.go`, `cmd/goodreads/enricher.go`)

- [ ] Create `cmd/goodreads/googlebooks.go`.
- [ ] Define `GoogleBooksVolumeInfo`, `GoogleBooksItem`, etc., structs to match the API response.
- [ ] Implement `GoogleBooksClient` struct:
  - [ ] Method to search volumes by ISBN (`volumes?q=isbn:<isbn>`).
  - [ ] Method to get volume by Google Books ID (if needed).
  - [ ] Integrate HTTP client (reuse `getHTTPClient` or create a new one).
  - [ ] Handle API key authentication (passed via config).
- [ ] Implement caching:
  - [ ] Create `cache/goodreads/googlebooks/` directory.
  - [ ] Add functions like `getCachedGoogleBooksData`, `cacheGoogleBooksData` similar to existing cache implementations.
- [ ] Create `GoogleBooksEnricher` struct in `enricher.go`:
  - [ ] Hold an instance of `GoogleBooksClient`.
  - [ ] Implement the `BookEnricher` interface:
    - The `Enrich` method should:
      - Check if the book already has sufficient data (e.g., description).
      - Prioritize ISBN/ISBN13 for lookup.
      - Call the `GoogleBooksClient` to fetch data (checking cache first).
      - Map relevant fields from the Google Books API response to the `Book` struct (e.g., `Description`, `Subjects`, `Publisher`, `NumberOfPages`, `CoverURL`). Be careful not to overwrite existing valid data from Goodreads CSV unless the API data is clearly better.
      - Handle potential errors (API errors, not found, rate limits).
    - The `Name` method should return `"GoogleBooks"`.

### Configuration (`config.yaml`, `cmd/root.go`, `cmd/goodreads/cmd.go`)

- [ ] Add `googlebooks.apikey` to `config.yaml.example`.
- [ ] Add Viper key reading for `googlebooks.apikey` (likely in `cmd/goodreads/googlebooks.go` or during enricher initialization).
- [ ] Consider adding a flag/config option to specify enricher preference (e.g., `--enricher=googlebooks`, `--enricher=openlibrary`, `--enricher=all`). Default could be OpenLibrary for backward compatibility.

### Update Parser (`cmd/goodreads/parser.go`)

- [ ] Modify `ParseGoodreads` function:
  - [ ] Initialize configured `BookEnricher` instances based on config/flags.
  - [ ] In the book processing loop, iterate through the chosen enricher(s).
  - [ ] Call `enricher.Enrich(&book)` for each selected enricher.
  - [ ] Add logging to indicate which enricher is being used and whether it succeeded or failed for a given book.

### Testing (`cmd/goodreads/googlebooks_test.go`)

- [ ] Add unit tests for the `GoogleBooksClient` (using mock HTTP responses).
- [ ] Add unit tests for the `GoogleBooksEnricher.Enrich` method.
- [ ] Add test data (sample API responses) in `cmd/goodreads/testdata/`.

### Documentation

- [ ] Update `README.md` or relevant `docs/` files to mention the new enricher and configuration.
- [ ] Ensure Go doc comments are added for new public types and functions.

## Technical Considerations

- **API Key Management**: Ensure the API key is handled securely and not committed to the repository.
- **Rate Limiting**: Be mindful of Google Books API quotas and implement delays or backoff if necessary (though standard usage is usually fine).
- **Data Merging**: Decide on the strategy for merging data from Goodreads CSV, OpenLibrary, and Google Books (e.g., prefer Google Books description if available?).
- **ISBN Lookup**: Google Books API works best with ISBNs. Handle cases where books might lack an ISBN.
- **Error Handling**: Gracefully handle API errors, books not found, or missing data fields.

## External Dependencies

- Google Go Client libraries (optional, standard `net/http` is likely sufficient)
- Viper for configuration
- Standard Go libraries (`encoding/json`, `net/http`, `fmt`, `log/slog`)
