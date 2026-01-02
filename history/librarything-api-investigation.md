# LibraryThing API Integration Investigation

**Date:** 2026-01-02
**Status:** Investigation Complete

## Executive Summary

LibraryThing offers multiple APIs and export formats that could be integrated into Hermes. However, there are significant limitations that affect the approach:

1. **Primary constraint:** LibraryThing does NOT provide an API for exporting user's own library data
2. **Export-based approach:** Users must use LibraryThing's CSV/Tab-delimited export functionality
3. **Catalog APIs available:** Can enrich book data via Talpa Search API and Common Knowledge API
4. **Current API status:** Legacy APIs were disabled (Jan 2021), but Talpa Search API launched Aug 2024

---

## Available LibraryThing APIs and Data Sources

### 1. **Talpa Search API** (Recommended - Active)

**Status:** Active (Released August 2024)

**Purpose:** Natural language search for books and media

**Capabilities:**
- Search for books using natural language queries
- Returns JSON with results including ISBNs, UPCs, and work details
- Free tier: 50 queries per day, 1 query per second
- Commercial/scale usage available (contact talpa@librarything.com)

**API Endpoint Format:**
```
https://api.librarything.com/search (inferred from documentation)
```

**Integration Use Case:**
- Enrich LibraryThing CSV imports with additional metadata
- Verify ISBN/title matches
- Supplement user-provided data with catalog information

**Limitations:**
- Rate limited for free tier
- Designed for search, not bulk operations
- Cannot retrieve user library data

**References:**
- [Talpa Search API Documentation](https://www.librarything.com/developer/documentation/talpa)
- [Talpa API Released Blog Post](https://blog.librarything.com/2024/08/talpa-api-released/)

---

### 2. **Common Knowledge API** (Legacy - Status Uncertain)

**Status:** Disabled as of January 2021, but may still be partially functional

**Purpose:** Access LibraryThing's "Common Knowledge" fielded wiki data about books

**Methods:**
- `librarything.ck.getwork` - Get work data by ISBN or work ID
- `librarything.ck.getauthor` - Get author information

**Capabilities:**
- Retrieve book metadata: author, title, publication year, series, characters, places
- Data available under Creative Commons Attribution Share Alike license
- XML or JSON response format
- Requires API key

**Request Format:**
```
http://www.librarything.com/services/rest/1.1/?method=librarything.ck.getwork&isbn={isbn}&apikey={key}
```

**Integration Use Case:**
- Enrich CSV imports with detailed metadata from LibraryThing catalog
- Fill in missing publication years, series information, etc.
- Similar to how Goodreads integration uses OpenLibrary and Google Books APIs

**Limitations:**
- Official status: disabled since Jan 2021
- Unclear if methods still work (GitHub examples exist but no recent confirmation)
- Requires API key registration
- Unclear if new key registrations are accepted

**References:**
- [LibraryThing Web Services Documentation](https://www.librarything.com/services/webservices.php)
- [Web Services REST Documentation](https://www.librarything.com/services/rest/documentation/1.1/)
- [GitHub Example: Common Knowledge API Usage](https://gist.github.com/atbradley/f671d5d63b6cdb275e76b22ebb6f6e5e)

---

### 3. **Data Export Options** (User-Initiated - Working)

**Status:** Active and recommended by LibraryThing

**Available Formats:**
- **CSV** - Basic comma-separated values (limited fields)
- **Tab-delimited** - More complete field set (preferred for backups)
- **JSON** - Structured format for programmatic access (licensed for browser-use only)
- **MARC** - Library standard format

**Integration Approach:**
Parse CSV/Tab-delimited export files similar to Goodreads import:
- Standard column structure (must match sample CSV exactly)
- Can't be modified/rearranged
- Provides user's full library with ratings, dates, notes, bookshelves
- Similar to existing Goodreads CSV import pattern

**Limitations:**
- Manual export required by user (no API access)
- CSV/Tab formats have fixed column structure
- Does not support automated scheduled exports
- Does not support updating existing records

**References:**
- [Export Options](https://www.librarything.com/export)
- [Import/Export Documentation](https://www.librarything.com/more/import)

---

## Integration Approaches

### **Option A: CSV Import (Recommended - Lowest Risk)**

**What:** Parse LibraryThing's CSV/Tab-delimited export format

**Pros:**
- ✅ Works with current LibraryThing (no API issues)
- ✅ Follows existing Hermes pattern (reuse Goodreads CSV parser structure)
- ✅ Full access to user's library with all metadata
- ✅ No rate limiting or API key issues
- ✅ User has full control of data

**Cons:**
- ❌ Requires manual export by user
- ❌ No automated/headless download like Goodreads
- ❌ One-time import only (manual re-export needed for updates)

**Implementation Effort:**
- Low-Medium: Can reuse patterns from `cmd/goodreads/csv_loader.go`
- Need to understand LibraryThing CSV column structure
- Similar to Letterboxd import which also uses manual exports

**Output Handling:**
- Support same output formats as Goodreads: Markdown, JSON, SQLite/Datasette
- Consider LibraryThing-specific fields (bookshelves, private notes, etc.)

---

### **Option B: Talpa API Enrichment (Optional - Enhancement)**

**What:** Use Talpa Search API to enhance CSV import with additional metadata

**Pros:**
- ✅ Enriches user data with LibraryThing's latest metadata
- ✅ Active API (no status uncertainty)
- ✅ Natural language search capability
- ✅ Could improve matching for ambiguous titles/authors

**Cons:**
- ❌ Rate limited (50/day free tier)
- ❌ Additional complexity
- ❌ Still requires CSV import as primary data source
- ❌ May not be necessary if CSV export is complete

**Implementation Effort:**
- Medium: New API client needed, follow patterns from `cmd/goodreads/omdb.go` or `cmd/steam/steam.go`
- Would be enrichment step after parsing CSV

**Use Case:**
- Could be "phase 2" enhancement similar to how Goodreads uses OpenLibrary + Google Books
- Better for handling ISBN mismatches or finding additional editions

---

### **Option C: Common Knowledge API Enrichment (Not Recommended - High Risk)**

**What:** Use librarything.ck.getwork API for catalog data

**Pros:**
- ✅ Rich metadata from LibraryThing's Common Knowledge wiki
- ✅ More fields than Talpa API
- ✅ Open license (CC-BY-SA)

**Cons:**
- ❌ **CRITICAL:** Officially disabled since Jan 2021
- ❌ Status unclear in 2024-2025 (no recent confirmation)
- ❌ Unknown if new API keys still issued
- ❌ High risk of breaking in future
- ❌ No documented escalation path or SLA

**Implementation Effort:**
- Low-Medium if working, but unreliable

**Recommendation:**
- **Not recommended** due to official disabled status and lack of recent updates
- Monitor for re-enablement but don't build on it
- Talpa API is better alternative if enrichment needed

---

## Recommended Implementation Plan

### **Phase 1: CSV Import (MVP)**

**Goal:** Basic LibraryThing support via CSV export

1. Create `cmd/librarything/` package following standard Hermes structure:
   ```
   cmd/librarything/
   ├── cmd.go              # Command struct and Kong integration
   ├── parser.go           # CSV parsing logic
   ├── types.go            # Data structures
   ├── json.go             # JSON output formatting
   ├── markdown.go         # Markdown output formatting
   ├── cache.go            # Optional caching for API calls
   └── testdata/           # Test fixtures
   ```

2. **Research LibraryThing CSV format:**
   - Download sample export
   - Document column structure
   - Understand field semantics (compare with Goodreads)

3. **Implement CSV parser:**
   - Similar to `cmd/goodreads/csv_loader.go`
   - Map LibraryThing fields to standard types
   - Handle LibraryThing-specific fields (bookshelves, private notes)

4. **Define output structure:**
   - Create Book type with LibraryThing-specific fields
   - Implement JSON output (similar to `cmd/goodreads/json.go`)
   - Implement Markdown output (similar to `cmd/goodreads/markdown.go`)
   - Support Datasette output

5. **Add to CLI:**
   - Add `LibraryThingCmd` to `cmd/root.go` ImportCmd
   - Add config key: `librarything.csvfile`
   - Support same flags as Goodreads: `-o`, `--json`, `--overwrite`

6. **Testing:**
   - Unit tests for CSV parsing
   - E2E test with sample export
   - Coverage for LibraryThing-specific edge cases

---

### **Phase 2: Talpa API Enrichment (Optional)**

Only if Phase 1 shows data gaps:

1. Create Talpa API client (similar to OpenLibrary client)
2. Implement optional enrichment step after CSV parsing
3. Use for ISBN verification and metadata enhancement
4. Add caching to respect rate limits
5. Make optional via `--talpa-enrich` flag

---

### **Phase 3: Monitor for API Re-enablement**

- Track LibraryThing API development forum/blog
- Prepare Optional Phase C implementation if Common Knowledge API re-enabled
- Update documentation when status changes

---

## File Structure Comparison

### Existing Goodreads Pattern (for reference)

```
cmd/goodreads/
├── cmd.go                      # Command struct
├── parser.go                   # CSV parsing
├── types.go                    # Book struct
├── csv_loader.go              # CSV utilities
├── csv_utils.go               # Helper functions
├── openlibrary.go            # API client
├── googlebooks.go            # API client
├── cache.go                  # API caching
├── json.go                   # JSON output
├── markdown.go               # Markdown output
├── automation.go             # Browser automation (unique to GR)
└── testdata/                 # Test fixtures
```

### Proposed LibraryThing Structure

```
cmd/librarything/
├── cmd.go                      # Command struct (NO automation needed)
├── parser.go                   # CSV parsing
├── types.go                    # Book struct
├── json.go                     # JSON output
├── markdown.go                 # Markdown output
├── talpa.go                    # Talpa API client (Phase 2)
├── cache.go                    # Caching (Phase 2)
└── testdata/                   # Test fixtures
```

**Differences from Goodreads:**
- No automation module (no browser automation for export)
- Simpler structure (CSV-only, not API-first)
- Optional Talpa enrichment rather than required OpenLibrary/Google Books

---

## Key Differences: LibraryThing vs Goodreads

| Aspect | Goodreads | LibraryThing |
|--------|-----------|--------------|
| **Data Access** | CSV export + automated download | CSV export only (manual) |
| **Primary Data Source** | Goodreads API + CSV | CSV only |
| **Automation** | Browser automation for download | None (manual export) |
| **Enrichment APIs** | OpenLibrary, Google Books | Talpa Search (if used) |
| **Export Formats** | CSV only | CSV, Tab-delimited, JSON, MARC |
| **Ratings/Reviews** | Included in CSV | Included in export |
| **Custom Fields** | Limited | Supports "own copy" tracking |
| **Timestamps** | Date added, date read | Yes |

---

## Data Mapping Example

### LibraryThing Export Fields (Estimated based on research)

Expected columns in Tab-delimited export:
- Book ID
- Title
- Author(s)
- ISBN / ISBN13
- Rating
- Publication Year
- Edition
- Binding
- Pages
- Language
- Date Added
- Date Finished
- Status (currently reading, read, want to read)
- Collections (equivalent to bookshelves)
- Tags
- Private Notes
- Review/Comments
- # Copies Owned
- Cover URL

### To Hermes Book Type Mapping

```go
type LibraryThingBook struct {
    ID              string      `json:"bookId"`
    Title           string      `json:"title"`
    Authors         []string    `json:"authors"`
    ISBN            string      `json:"isbn"`
    ISBN13          string      `json:"isbn13"`
    MyRating        float64     `json:"myRating"`
    YearPublished   int         `json:"yearPublished"`
    DateAdded       string      `json:"dateAdded"`
    DateFinished    string      `json:"dateFinished"`
    Status          string      `json:"status"`
    Collections     []string    `json:"collections"`   // LT-specific
    Tags            []string    `json:"tags"`          // LT-specific
    PrivateNotes    string      `json:"privateNotes"`  // LT-specific
    Review          string      `json:"review"`
    OwnedCopies     int         `json:"ownedCopies"`   // LT-specific
    CoverURL        string      `json:"coverUrl"`
}
```

---

## Recommendations Summary

### **Recommended: Implement Phase 1 (CSV Import)**

1. **Why:**
   - Works immediately with current LibraryThing
   - Follows proven Hermes patterns
   - No API status uncertainty
   - Provides full user library access

2. **When:**
   - Start with CSV parser research
   - Can be implemented in parallel with other features
   - Estimated effort: 2-3 days for MVP

3. **Scope:**
   - CSV/Tab-delimited parsing
   - JSON and Markdown output
   - Datasette integration
   - Basic testing

### **Optional: Phase 2 (Talpa API Enrichment)**

1. **Consider if:**
   - LibraryThing CSV export data proves incomplete
   - Users need ISBN/title verification
   - Rate limits are acceptable (50/day free)

2. **Timeline:** After Phase 1 working and tested

### **Not Recommended: Phase 3 (Common Knowledge API)**

1. **Reason:** Officially disabled, no clear re-enablement timeline
2. **Alternative:** Use Talpa API for enrichment needs
3. **Monitor:** Watch for official API re-enablement announcement

---

## Next Steps

1. **Confirm CSV format:** Download actual sample export from LibraryThing
2. **Document schema:** Map actual columns to Hermes types
3. **Create issue:** Use `bd create` to track implementation work
4. **Start Phase 1:** Begin CSV parser implementation
5. **Plan enrichment:** Decide if Talpa API enrichment is needed based on Phase 1 results

---

## Sources and References

- [LibraryThing Web Services API](https://www.librarything.com/services/webservices.php)
- [Talpa Search API Documentation](https://www.librarything.com/developer/documentation/talpa)
- [Talpa API Released Blog Post (Aug 2024)](https://blog.librarything.com/2024/08/talpa-api-released/)
- [Web Services REST Documentation](https://www.librarything.com/services/rest/documentation/1.1/)
- [LibraryThing Export Options](https://www.librarything.com/export)
- [GitHub: Common Knowledge API Example](https://gist.github.com/atbradley/f671d5d63b6cdb275e76b22ebb6f6e5e)
- [Using the LibraryThing API from C#](https://saguiitay.com/using-the-librarything-api-from-c/)
