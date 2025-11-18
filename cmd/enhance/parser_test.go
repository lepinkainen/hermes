package enhance

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
)

func TestParseNote(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Note
		wantErr bool
	}{
		{
			name: "basic movie note",
			content: `---
title: "Test Movie"
type: movie
year: 2021
imdb_id: "tt1234567"
---

This is the content of the note.`,
			want: &Note{
				Title:  "Test Movie",
				Type:   "movie",
				Year:   2021,
				IMDBID: "tt1234567",
				RawFrontmatter: map[string]interface{}{
					"title":   "Test Movie",
					"type":    "movie",
					"year":    2021,
					"imdb_id": "tt1234567",
				},
				OriginalBody: "This is the content of the note.",
			},
			wantErr: false,
		},
		{
			name: "note with tmdb_id",
			content: `---
title: "Test Movie"
type: movie
year: 2021
tmdb_id: 12345
---

Content here.`,
			want: &Note{
				Title:  "Test Movie",
				Type:   "movie",
				Year:   2021,
				TMDBID: 12345,
				RawFrontmatter: map[string]interface{}{
					"title":   "Test Movie",
					"type":    "movie",
					"year":    2021,
					"tmdb_id": 12345,
				},
				OriginalBody: "Content here.",
			},
			wantErr: false,
		},
		{
			name:    "missing frontmatter",
			content: "Just some content without frontmatter",
			wantErr: true,
		},
		{
			name: "invalid yaml",
			content: `---
this is not: valid: yaml
---

Content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNote(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.Title != tt.want.Title {
				t.Errorf("Title = %v, want %v", got.Title, tt.want.Title)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Year != tt.want.Year {
				t.Errorf("Year = %v, want %v", got.Year, tt.want.Year)
			}
			if got.IMDBID != tt.want.IMDBID {
				t.Errorf("IMDBID = %v, want %v", got.IMDBID, tt.want.IMDBID)
			}
			if got.TMDBID != tt.want.TMDBID {
				t.Errorf("TMDBID = %v, want %v", got.TMDBID, tt.want.TMDBID)
			}
			if got.OriginalBody != tt.want.OriginalBody {
				t.Errorf("OriginalBody = %v, want %v", got.OriginalBody, tt.want.OriginalBody)
			}
		})
	}
}

func TestHasTMDBData(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "has tmdb_id and content markers",
			note: &Note{
				TMDBID:       12345,
				OriginalBody: "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: true,
		},
		{
			name: "has tmdb_id but no content markers",
			note: &Note{
				TMDBID:       12345,
				OriginalBody: "Some content without markers",
			},
			want: false,
		},
		{
			name: "no tmdb_id",
			note: &Note{
				TMDBID:       0,
				OriginalBody: "Some content",
			},
			want: false,
		},
		{
			name: "has content markers but no tmdb_id",
			note: &Note{
				TMDBID:       0,
				OriginalBody: "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.HasTMDBData(); got != tt.want {
				t.Errorf("HasTMDBData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddTMDBData(t *testing.T) {
	note := &Note{
		RawFrontmatter: map[string]interface{}{
			"title": "Test Movie",
			"type":  "movie",
			"year":  2021,
		},
	}

	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:      12345,
		TMDBType:    "movie",
		RuntimeMins: 120,
		GenreTags:   []string{"Action", "Adventure"},
		CoverPath:   "_attachments/cover.jpg",
	}

	note.AddTMDBData(tmdbData)

	if note.RawFrontmatter["tmdb_id"] != 12345 {
		t.Errorf("tmdb_id not set correctly")
	}
	if note.RawFrontmatter["tmdb_type"] != "movie" {
		t.Errorf("tmdb_type not set correctly")
	}
	if note.RawFrontmatter["runtime"] != 120 {
		t.Errorf("runtime not set correctly")
	}
	if note.RawFrontmatter["cover"] != "_attachments/cover.jpg" {
		t.Errorf("cover not set correctly")
	}

	tags, ok := note.RawFrontmatter["tags"].([]string)
	if !ok {
		t.Errorf("tags not set as []string")
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestBuildMarkdown(t *testing.T) {
	note := &Note{
		RawFrontmatter: map[string]interface{}{
			"title": "Test Movie",
			"type":  "movie",
			"year":  2021,
		},
		OriginalBody: "Original content here.",
	}

	originalContent := `---
title: Test Movie
type: movie
year: 2021
---

Original content here.`

	tmdbData := &enrichment.TMDBEnrichment{
		ContentMarkdown: "## TMDB Content\nSome TMDB data here.",
	}

	result := note.BuildMarkdown(originalContent, tmdbData, false)

	// Check that frontmatter is present
	if result[:3] != "---" {
		t.Errorf("Markdown should start with frontmatter delimiter")
	}

	// Check that original body is preserved
	if !containsSubstring(result, "Original content here.") {
		t.Errorf("Original content should be preserved")
	}

	// Check that TMDB content is appended
	if !containsSubstring(result, "## TMDB Content") {
		t.Errorf("TMDB content should be appended")
	}
}

// Helper function for substring checking
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNeedsCover(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "no cover field",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
				},
			},
			want: true,
		},
		{
			name: "empty cover field",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
					"cover": "",
				},
			},
			want: true,
		},
		{
			name: "cover field with value",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
					"cover": "_attachments/cover.jpg",
				},
			},
			want: false,
		},
		{
			name: "cover field is not a string",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
					"cover": 12345,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.NeedsCover(); got != tt.want {
				t.Errorf("NeedsCover() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsMetadata(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "no tmdb_id",
			note: &Note{
				TMDBID: 0,
				Type:   "movie",
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
				},
			},
			want: true,
		},
		{
			name: "has tmdb_id but no runtime",
			note: &Note{
				TMDBID: 12345,
				Type:   "movie",
				RawFrontmatter: map[string]interface{}{
					"title":   "Test Movie",
					"tmdb_id": 12345,
				},
			},
			want: true,
		},
		{
			name: "has tmdb_id and runtime but no tags",
			note: &Note{
				TMDBID: 12345,
				Type:   "movie",
				RawFrontmatter: map[string]interface{}{
					"title":   "Test Movie",
					"tmdb_id": 12345,
					"runtime": 120,
				},
			},
			want: true,
		},
		{
			name: "has tmdb_id, runtime, and empty tags",
			note: &Note{
				TMDBID: 12345,
				Type:   "movie",
				RawFrontmatter: map[string]interface{}{
					"title":   "Test Movie",
					"tmdb_id": 12345,
					"runtime": 120,
					"tags":    []string{},
				},
			},
			want: true,
		},
		{
			name: "has all metadata",
			note: &Note{
				TMDBID: 12345,
				Type:   "movie",
				RawFrontmatter: map[string]interface{}{
					"title":   "Test Movie",
					"tmdb_id": 12345,
					"runtime": 120,
					"tags":    []string{"Action", "Adventure"},
				},
			},
			want: false,
		},
		{
			name: "tv show with all metadata",
			note: &Note{
				TMDBID: 12345,
				Type:   "tv",
				RawFrontmatter: map[string]interface{}{
					"title":   "Test Show",
					"tmdb_id": 12345,
					"runtime": 45,
					"tags":    []string{"Drama"},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.NeedsMetadata(); got != tt.want {
				t.Errorf("NeedsMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsContent(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "no content markers",
			note: &Note{
				OriginalBody: "Some content without markers",
			},
			want: true,
		},
		{
			name: "has content markers",
			note: &Note{
				OriginalBody: "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: false,
		},
		{
			name: "empty body",
			note: &Note{
				OriginalBody: "",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.NeedsContent(); got != tt.want {
				t.Errorf("NeedsContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMediaIDs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    MediaIDs
	}{
		{
			name: "TMDB ID only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
tmdb_id: 949
---

Content here.`,
			want: MediaIDs{TMDBID: 949},
		},
		{
			name: "IMDB ID only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
imdb_id: "tt0113277"
---

Content here.`,
			want: MediaIDs{IMDBID: "tt0113277"},
		},
		{
			name: "Letterboxd ID only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
letterboxd_id: "2bg8"
---

Content here.`,
			want: MediaIDs{LetterboxdID: "2bg8"},
		},
		{
			name: "All IDs present",
			content: `---
title: "Test Movie"
type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
letterboxd_id: "2bg8"
---

Content here.`,
			want: MediaIDs{
				TMDBID:       949,
				IMDBID:       "tt0113277",
				LetterboxdID: "2bg8",
			},
		},
		{
			name: "No IDs",
			content: `---
title: "Test Movie"
type: movie
year: 2021
---

Content here.`,
			want: MediaIDs{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parseNote(tt.content)
			if err != nil {
				t.Fatalf("Failed to parse note: %v", err)
			}

			ids := note.GetMediaIDs()
			if ids.TMDBID != tt.want.TMDBID {
				t.Errorf("TMDBID = %v, want %v", ids.TMDBID, tt.want.TMDBID)
			}
			if ids.IMDBID != tt.want.IMDBID {
				t.Errorf("IMDBID = %v, want %v", ids.IMDBID, tt.want.IMDBID)
			}
			if ids.LetterboxdID != tt.want.LetterboxdID {
				t.Errorf("LetterboxdID = %v, want %v", ids.LetterboxdID, tt.want.LetterboxdID)
			}
		})
	}
}

func TestHasAnyID(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "TMDB ID only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
tmdb_id: 949
---

Content here.`,
			want: true,
		},
		{
			name: "IMDB ID only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
imdb_id: "tt0113277"
---

Content here.`,
			want: true,
		},
		{
			name: "Letterboxd ID only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
letterboxd_id: "2bg8"
---

Content here.`,
			want: true,
		},
		{
			name: "All IDs present",
			content: `---
title: "Test Movie"
type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
letterboxd_id: "2bg8"
---

Content here.`,
			want: true,
		},
		{
			name: "No IDs",
			content: `---
title: "Test Movie"
type: movie
year: 2021
---

Content here.`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parseNote(tt.content)
			if err != nil {
				t.Fatalf("Failed to parse note: %v", err)
			}

			hasAny := note.HasAnyID()
			if hasAny != tt.want {
				t.Errorf("HasAnyID() = %v, want %v", hasAny, tt.want)
			}
		})
	}
}

func TestGetIDSummary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "All IDs present",
			content: `---
title: "Test Movie"
type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
letterboxd_id: "2bg8"
---

Content here.`,
			want: "tmdb:949, imdb:tt0113277, letterboxd:2bg8",
		},
		{
			name: "TMDB and IMDB only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
---

Content here.`,
			want: "tmdb:949, imdb:tt0113277",
		},
		{
			name: "IMDB only",
			content: `---
title: "Test Movie"
type: movie
year: 2021
imdb_id: "tt0113277"
---

Content here.`,
			want: "imdb:tt0113277",
		},
		{
			name: "No IDs",
			content: `---
title: "Test Movie"
type: movie
year: 2021
---

Content here.`,
			want: "no IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parseNote(tt.content)
			if err != nil {
				t.Fatalf("Failed to parse note: %v", err)
			}

			summary := note.GetIDSummary()
			if summary != tt.want {
				t.Errorf("GetIDSummary() = %v, want %v", summary, tt.want)
			}
		})
	}
}

func TestParseNoteWithHeatFile(t *testing.T) {
	// Test with actual Heat (1995) file content
	content := `---
title: "Heat"
type: movie
year: 1995
date_watched: "2019-04-12"
letterboxd_rating: 8.3
runtime: 170
duration: 2h 50m
directors:
  - "Michael Mann"
tags:
  - letterboxd/movie
  - rating/8
  - year/1990s
  - genre/Action
  - genre/Crime
  - genre/Drama
letterboxd_uri: "https://boxd.it/2bg8"
letterboxd_id: "2bg8"
cover: "attachments/Heat - cover.jpg"
tmdb_id: 949
tmdb_type: "movie"
---

![[Heat - cover.jpg|250]]

>[!summary]- Plot
> A group of high-end professional thieves start to feel the heat from the LAPD when they unknowingly leave a verbal clue at their latest heist.

>[!cast]- Cast
> - Al Pacino
> - Robert De Niro
> - Val Kilmer

>[!info]- Letterboxd
> [View on Letterboxd](https://boxd.it/2bg8)
`

	note, err := parseNote(content)
	if err != nil {
		t.Fatalf("Failed to parse Heat note: %v", err)
	}

	// Verify parsing
	if note.Title != "Heat" {
		t.Errorf("Title = %v, want Heat", note.Title)
	}
	if note.Type != "movie" {
		t.Errorf("Type = %v, want movie", note.Type)
	}
	if note.Year != 1995 {
		t.Errorf("Year = %v, want 1995", note.Year)
	}
	if note.TMDBID != 949 {
		t.Errorf("TMDBID = %v, want 949", note.TMDBID)
	}
	if note.LetterboxdID != "2bg8" {
		t.Errorf("LetterboxdID = %v, want 2bg8", note.LetterboxdID)
	}

	// Test ID extraction
	ids := note.GetMediaIDs()
	if ids.TMDBID != 949 {
		t.Errorf("GetMediaIDs().TMDBID = %v, want 949", ids.TMDBID)
	}
	if ids.LetterboxdID != "2bg8" {
		t.Errorf("GetMediaIDs().LetterboxdID = %v, want 2bg8", ids.LetterboxdID)
	}

	// Test utility functions
	if !note.HasAnyID() {
		t.Errorf("HasAnyID() = false, want true")
	}

	summary := note.GetIDSummary()
	expectedSummary := "tmdb:949, letterboxd:2bg8"
	if summary != expectedSummary {
		t.Errorf("GetIDSummary() = %v, want %v", summary, expectedSummary)
	}

	// Verify the original issue: HasTMDBData should return false because no TMDB content markers
	if note.HasTMDBData() {
		t.Errorf("HasTMDBData() = true, want false (Heat file has TMDB ID but no TMDB content markers)")
	}

	// Test why this file would trigger TMDB search: needs content but has TMDB ID
	if note.NeedsContent() {
		t.Logf("✓ Heat file needs TMDB content (no markers found)")
	} else {
		t.Errorf("Heat file should need TMDB content (no markers found)")
	}

	if !note.NeedsMetadata() {
		t.Logf("✓ Heat file has all metadata (TMDB ID, runtime, tags)")
	} else {
		t.Logf("ℹ Heat file might need some metadata")
	}
}

func TestHasSeenField(t *testing.T) {
	tests := []struct {
		name     string
		note     *Note
		expected bool
	}{
		{
			name: "has seen field",
			note: &Note{
				Seen: true,
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
					"seen":  true,
				},
			},
			expected: true,
		},
		{
			name: "does not have seen field",
			note: &Note{
				Seen: false,
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
				},
			},
			expected: false,
		},
		{
			name: "has seen field set to false",
			note: &Note{
				Seen: false,
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
					"seen":  false,
				},
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.note.hasSeenField()
			if result != tc.expected {
				t.Errorf("hasSeenField() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestHasAnyRating(t *testing.T) {
	tests := []struct {
		name     string
		note     *Note
		expected bool
	}{
		{
			name: "has imdb_rating",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title":       "Test Movie",
					"imdb_rating": 8.5,
				},
			},
			expected: true,
		},
		{
			name: "has my_rating",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title":     "Test Movie",
					"my_rating": 9,
				},
			},
			expected: true,
		},
		{
			name: "has letterboxd_rating",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title":           "Test Movie",
					"letterboxd_rating": 4.5,
				},
			},
			expected: true,
		},
		{
			name: "has zero ratings",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title":           "Test Movie",
					"imdb_rating":     0.0,
					"my_rating":       0,
					"letterboxd_rating": 0.0,
				},
			},
			expected: false,
		},
		{
			name: "has no ratings",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title": "Test Movie",
					"year":  2020,
				},
			},
			expected: false,
		},
		{
			name: "has mixed rating types (int and float)",
			note: &Note{
				RawFrontmatter: map[string]interface{}{
					"title":     "Test Movie",
					"my_rating": 8, // int
					"imdb_rating": 7.5, // float64
				},
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.note.hasAnyRating()
			if result != tc.expected {
				t.Errorf("hasAnyRating() = %v, want %v", result, tc.expected)
			}
		})
	}
}
