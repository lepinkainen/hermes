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
			name: "has tmdb_id",
			note: &Note{TMDBID: 12345},
			want: true,
		},
		{
			name: "no tmdb_id",
			note: &Note{TMDBID: 0},
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
	if note.RawFrontmatter["runtime_mins"] != 120 {
		t.Errorf("runtime_mins not set correctly")
	}
	if note.RawFrontmatter["tmdb_cover"] != "_attachments/cover.jpg" {
		t.Errorf("tmdb_cover not set correctly")
	}

	genres, ok := note.RawFrontmatter["genres"].([]string)
	if !ok {
		t.Errorf("genres not set as []string")
	}
	if len(genres) != 2 {
		t.Errorf("expected 2 genres, got %d", len(genres))
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
