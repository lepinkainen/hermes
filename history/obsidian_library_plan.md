# Obsidian Library Plan

## Implementation Status

**Last Updated:** 2024-12-27

### ✅ Completed (PR #1)
- **Phase 1: Core Infrastructure** - 100% complete
- **Phase 2: Update cmd/enhance** - 100% complete

### 🔄 In Progress
- None currently

### ⏳ Remaining Work (PR #2)
- **Phase 3: Migrate Enrichment** - Not started
- **Phase 4: Update Importers** - Not started
- **Phase 5: Documentation & Cleanup** - Not started

### Key Decisions Made
1. **Always write flow-style tags** (not optional via flag)
2. **Support reading block-style** (backwards compatible)
3. **Tag normalization always enabled** (no opt-out flag)
4. **Existing notes left unchanged** (no migration required)

---

## Goals

- Centralize Obsidian Markdown handling into a single internal library (`internal/obsidian`).
- Enforce consistent tag normalization and frontmatter parsing/serialization.
- Provide safe, typed access to frontmatter fields with deterministic output.
- Support flow-style YAML for tags while maintaining backwards compatibility.

## Scope and Current Touchpoints

- Frontmatter parsing and tag detection: `internal/frontmatter`
- Markdown builder and tag collection: `internal/fileutil`
- Enrichment merge logic: `internal/enrichment` and `cmd/enhance`
- Tag generation in importers: `cmd/*/markdown.go`
- TMDB tag sanitization: `internal/tmdb`

## Tag Rules

**Normalization Steps:**
1. Preserve case (no lowercasing)
2. Strip leading `#` if present
3. Trim leading/trailing whitespace
4. Convert all whitespace to hyphens (`\s+` → `-`)
5. Strip leading/trailing hyphens
6. **Collapse repeated hyphens** (`-{2,}` → `-`)
7. Remove special characters (`&`, `#`, etc.) but preserve `/` for hierarchy
8. Sort tags alphabetically and dedupe

**Output Format:**
- Flow-style YAML: `tags: [game, genre/Indie, genre/RPG, genre/Simulation, to-watch]` ✅ **IMPLEMENTED**
  - Always writes flow-style (no flag needed)
  - Reads both flow-style and block-style (backwards compatible)

**Examples:**
```
"Action  Comedy"      → "Action-Comedy"
"#Sci-Fi"           → "Sci-Fi"
"  genre/Horror  "  → "genre/Horror"
"game//action"      → "game-action" (if / is removed)
"& Other"           → "and-Other"
```

## Frontmatter Rules

- Keys sorted alphabetically on write (using custom sorted-key wrapper)
- Missing frontmatter is valid and yields an empty map
- Re-serialization is acceptable even if formatting changes
- String arrays are deduped on serialization
- Preserve custom formatting in body content

## Proposed Package and API

**New package:** `internal/obsidian`

**Types:**
```go
// Note represents a complete markdown document
type Note struct {
    Frontmatter *Frontmatter
    Body       string
}

// Frontmatter provides typed access to YAML frontmatter with sorted keys
type Frontmatter struct {
    fields map[string]any
    keys   []string // Sorted key order for deterministic serialization
}
```

**Core Functions:**
```go
// Parsing
func ParseMarkdown([]byte) (*Note, error)

// Serialization
func (n *Note) Build(flowTags bool) ([]byte, error)
func (f *Frontmatter) MarshalYAML() (interface{}, error) // Custom sorted marshaling

// Typed access (existing patterns from internal/frontmatter)
func (f *Frontmatter) GetString(key string) string
func (f *Frontmatter) GetInt(key string) int
func (f *Frontmatter) GetBool(key string) bool
func (f *Frontmatter) GetStringArray(key string) []string

// Mutation
func (f *Frontmatter) Set(key string, value any)
func (f *Frontmatter) Delete(key string)
func (f *Frontmatter) Get(key string) (any, bool)
```

**Tag Functions:**
```go
// Normalization
func NormalizeTag(tag string) string
func NormalizeTags(tags []string) []string

// Tag merging with normalization
func MergeTags(existing, new []string) []string

// Type extraction (from internal/enrichment)
func TagsFromAny(val any) []string

// TagSet for deduping and stable ordering (replaces internal/fileutil.TagCollector)
type TagSet struct {
    tags map[string]bool
}

func NewTagSet() *TagSet
func (ts *TagSet) Add(tag string)
func (ts *TagSet) AddIf(condition bool, tag string)
func (ts *TagSet) AddFormat(format string, args ...interface{})
func (ts *TagSet) GetSorted() []string
```

## Implementation Details

### Tag Normalization Implementation

```go
import "regexp"

func NormalizeTag(tag string) string {
    tag = strings.TrimSpace(tag)
    tag = strings.TrimPrefix(tag, "#")

    // Convert whitespace to hyphens
    re := regexp.MustCompile(`\s+`)
    tag = re.ReplaceAllString(tag, "-")

    // Collapse repeated hyphens
    re = regexp.MustCompile(`-+`)
    tag = re.ReplaceAllString(tag, "-")

    // Trim leading/trailing hyphens
    tag = strings.Trim(tag, "-")

    // Remove special characters (preserve / for tags like genre/Action)
    tag = strings.ReplaceAll(tag, "&", "and")
    tag = strings.ReplaceAll(tag, "#", "")

    return tag
}
```

### Flow-Style YAML for Tags

```go
import yaml "gopkg.in/yaml.v3"

func (f *Frontmatter) MarshalYAML() (interface{}, error) {
    // Create sorted sequence for key-value pairs
    nodes := make([]*yaml.Node, 0, len(f.keys))

    for _, key := range f.keys {
        val := f.fields[key]

        // Create key node
        keyNode := &yaml.Node{
            Kind:  yaml.ScalarNode,
            Value: key,
        }

        // Create value node with flow style for tags
        var valueNode *yaml.Node
        if key == "tags" && tags, ok := val.([]string); ok {
            // Flow-style sequence: [a, b, c]
            valueNode = &yaml.Node{
                Kind:  yaml.SequenceNode,
                Style: yaml.FlowStyle,
            }
            for _, tag := range tags {
                valueNode.Content = append(valueNode.Content, &yaml.Node{
                    Kind:  yaml.ScalarNode,
                    Value: tag,
                })
            }
        } else {
            // Normal value
            valueNode = encodeValue(val)
        }

        // Append key-value pair
        nodes = append(nodes, keyNode, valueNode)
    }

    // Create mapping node
    return &yaml.Node{
        Kind:    yaml.MappingNode,
        Content: nodes,
    }, nil
}
```

### Frontmatter Key Sorting

```go
func (f *Frontmatter) Set(key string, value any) {
    _, exists := f.fields[key]
    f.fields[key] = value

    if !exists {
        // Insert in sorted position
        f.keys = append(f.keys, key)
        sort.Strings(f.keys)
    }
}

func (f *Frontmatter) Delete(key string) {
    delete(f.fields, key)
    for i, k := range f.keys {
        if k == key {
            f.keys = append(f.keys[:i], f.keys[i+1:]...)
            break
        }
    }
}
```

## Migration Strategy

### ✅ Phase 1: Core Infrastructure (COMPLETED)
**Status:** 100% complete, all tests passing (92.6% coverage)

**Completed Tasks:**
- ✅ Created `internal/obsidian` package structure
  - `internal/obsidian/note.go` - Note and Frontmatter types
  - `internal/obsidian/tags.go` - Tag functions and TagSet
  - `internal/obsidian/note_test.go` - Frontmatter tests
  - `internal/obsidian/tags_test.go` - Tag normalization tests
- ✅ Implemented `Note` and `Frontmatter` types with sorted-key marshaling
  - Keys automatically sorted on Set()
  - Custom MarshalYAML() for flow-style tags
  - Typed getters: GetString, GetInt, GetBool, GetStringArray
  - Keys() method for iteration
- ✅ Implemented `NormalizeTag/NormalizeTags` with hyphen collapse
  - All 8 normalization steps implemented
  - Comprehensive edge case testing
- ✅ Implemented flow-style YAML serialization (always on, not optional)
- ✅ Added comprehensive unit tests for tag normalization edge cases
  - 34 test cases for NormalizeTag
  - 7 test cases for NormalizeTags
  - 6 test cases for TagSet
  - 7 test cases for MergeTags
  - 9 test cases for TagsFromAny
- ✅ Added `TagSet`, `MergeTags`, `TagsFromAny` functions
- ⏳ Golden file tests for YAML output - deferred to Phase 5

### ✅ Phase 2: Update cmd/enhance (COMPLETED)
**Status:** 100% complete, all tests passing (53.1% coverage)

**Completed Tasks:**
- ✅ Replaced `enhance/parser.go:parseNote()` with `obsidian.ParseMarkdown()`
- ✅ Updated `enhance.Note` to use `*obsidian.Frontmatter` (not replaced, enhanced)
  - Kept domain-specific typed fields (TMDBID, IMDBID, etc.)
  - Replaced RawFrontmatter with *obsidian.Frontmatter
  - Replaced OriginalBody with Body
- ✅ Updated `AddTMDBData()` and `AddSteamData()` to use `obsidian.MergeTags()`
- ✅ Updated `cmd.go` to use Frontmatter getter/setter methods
- ✅ Updated all tests to use new obsidian types
  - Fixed 30+ test cases
  - Updated struct literals to use Frontmatter
  - Migrated from RawFrontmatter access to Frontmatter methods
- ✅ Verified backwards compatibility with existing notes
  - Reads both flow-style and block-style tags
  - Accepts notes without frontmatter
- ❌ No `--flow-tags` flag added (decision: always use flow-style)

### ⏳ Phase 3: Migrate Enrichment (TODO - PR #2)
**Status:** Not started

**Remaining Tasks:**
- ⏳ Move `enrichment.MergeTags()` to `internal/obsidian` with normalization
  - **Note:** Already done in Phase 1! `obsidian.MergeTags()` exists
  - Need to deprecate `enrichment.MergeTags()` and update callers
- ⏳ Move `enrichment.TagsFromAny()` to `internal/obsidian`
  - **Note:** Already done in Phase 1! `obsidian.TagsFromAny()` exists
  - Need to deprecate `enrichment.TagsFromAny()` and update callers
- ⏳ Update all enrichment calls to use `obsidian.MergeTags()`
  - Already done in `cmd/enhance`
  - Need to update other packages if they use enrichment tags
- ⏳ Deprecate `internal/enrichment/tags.go` (keep exports for compatibility)
  - Add deprecation comments
  - Re-export from obsidian package for compatibility

**Files to Update:**
- `internal/enrichment/tags.go` - Add deprecation warnings, re-export obsidian functions
- Search codebase for `enrichment.MergeTags` and `enrichment.TagsFromAny` usage

### ⏳ Phase 4: Update Importers (TODO - PR #2)
**Status:** Not started

**Remaining Tasks:**
- ⏳ Replace `fileutil.TagCollector` with `obsidian.TagSet`
  - Update `cmd/goodreads/markdown.go` (currently uses string concat, not TagCollector!)
  - Update `cmd/imdb/markdown.go`
  - Update `cmd/letterboxd/markdown.go`
  - Update `cmd/steam/markdown.go`
- ⏳ Update TMDB tag generation to use `obsidian.NormalizeTag()`
  - Update `internal/tmdb/helpers.go:sanitizeGenreName()` to use `obsidian.NormalizeTag()`
- ⏳ Update Steam enrichment to use normalized tags
  - Fix `internal/enrichment/steam_content.go:extractGenreTags()` to normalize
- ⏳ Update importer tests for flow-style output
- ⏳ Deprecate `fileutil.TagCollector`
  - Add deprecation comment
  - Re-export obsidian.TagSet for compatibility
- ❌ No `--flow-tags` flag (decision: always use flow-style)

**Files to Update:**
- `cmd/goodreads/markdown.go` - Switch from string concat to TagSet
- `cmd/imdb/markdown.go` - Replace TagCollector with TagSet
- `cmd/letterboxd/markdown.go` - Replace TagCollector with TagSet
- `cmd/steam/markdown.go` - Replace TagCollector with TagSet
- `internal/tmdb/helpers.go` - Use obsidian.NormalizeTag
- `internal/enrichment/steam_content.go` - Use obsidian.NormalizeTag
- `internal/fileutil/markdown.go` - Deprecate TagCollector

### ⏳ Phase 5: Documentation & Cleanup (TODO)
**Status:** Not started

**Remaining Tasks:**
- ⏳ Create `docs/obsidian.md` with tag rules and library usage
- ⏳ Update `docs/05_output_formats.md` with tag format examples
- ⏳ Add migration guide for users with existing notes (if needed)
- ⏳ Document breaking changes in CHANGELOG
  - Flow-style tags always used
  - Tag normalization always applied
  - Sorted frontmatter keys
- ⏳ Add golden file tests for YAML output
- ⏳ Remove deprecated exports after grace period (future version)

## Tests and Documentation

### Unit Tests
- Tag normalization edge cases (multiple hyphens, whitespace, special chars)
- Frontmatter round-trip parsing with/without tags
- Flow-style vs block-style YAML output
- Key sorting behavior
- `MergeTags()` with normalization

### Integration Tests
- Full enhance workflow with flow-style tags
- Importer output verification with normalized tags
- Backwards compatibility with existing block-style notes

### Golden File Tests
- Sample flow-style frontmatter output
- Sample block-style frontmatter output (flag-controlled)
- Before/after tag normalization examples

### Documentation
- `docs/obsidian.md`: Tag normalization rules, API reference
- Migration guide for existing notes
- Breaking change warnings in CHANGELOG
- Update importer-specific docs with new tag behavior

## Backwards Compatibility

### ✅ Implemented Approach
- ✅ **No flags needed** - Opinionated output (flow-style only)
- ✅ **Always writes flow-style YAML** for tags
- ✅ **Reads both formats** - Backwards compatible with block-style tags
- ✅ **Tag normalization always enabled** - Consistent output everywhere
- ✅ **Existing notes left unchanged** - No migration required
- ✅ **Sorted frontmatter keys** - Deterministic, diff-friendly output

### ⏳ Planned Graceful Deprecation
- ⏳ Keep `internal/enrichment/tags.go` exports for 2 versions
- ⏳ Keep `internal/fileutil.TagCollector` exports for 2 versions
- ⏳ Add deprecation warnings in comments (not logs - too noisy)
- ⏳ Document migration path in CHANGELOG

## Effort Estimation

| Phase | Tasks | Status | Actual Time |
|--------|--------|--------|-------------|
| 1. Core (`internal/obsidian`) | Package structure, types, normalization, flow-style YAML, tests | ✅ Complete | ~6 hours |
| 2. Update `cmd/enhance` | Replace parser/note types, update tests | ✅ Complete | ~4 hours |
| 3. Migrate enrichment | Move tag utilities, update all enrichment calls | ⏳ TODO | Est. 2-3 hours |
| 4. Update importers | Replace TagCollector, update all markdown.go files | ⏳ TODO | Est. 3-4 hours |
| 5. Documentation & cleanup | Docs, golden tests, deprecation warnings | ⏳ TODO | Est. 2-3 hours |
| **Total** | | **40% complete** | **10 hours done, ~8-10 hours remaining** |

### Actual Implementation Notes
- Phase 1 was faster than estimated due to clear plan and code examples
- Phase 2 took longer than estimated due to extensive test updates (30+ test cases)
- Phases 3-4 will be faster because the core design is validated and working
- Golden file tests deferred to Phase 5 to avoid blocking progress
