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
