package enhance

import (
	"testing"

	"github.com/stretchr/testify/require"
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
tmdb_type: movie
year: 2021
imdb_id: "tt1234567"
---

This is the content of the note.`,
			want: &Note{
				Title:  "Test Movie",
				Type:   "movie",
				Year:   2021,
				IMDBID: "tt1234567",
			},
			wantErr: false,
		},
		{
			name: "note with tmdb_id",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 12345
---

Content here.`,
			want: &Note{
				Title:  "Test Movie",
				Type:   "movie",
				Year:   2021,
				TMDBID: 12345,
			},
			wantErr: false,
		},
		{
			name:    "missing frontmatter",
			content: "Just some content without frontmatter",
			want: &Note{
				Title: "",
				Type:  "",
				Year:  0,
			},
			wantErr: false, // obsidian.ParseMarkdown accepts content without frontmatter
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
			// Body is checked via parsed content, not comparing directly
		})
	}
}

func TestParseNoteWithHeatFile(t *testing.T) {
	// Test with actual Heat (1995) file content
	content := `---
title: "Heat"
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
}

func TestExtractTitleFromPath_WithParentheses(t *testing.T) {
	title := extractTitleFromPath("/notes/Red Sonja (2025).md")
	require.Equal(t, "Red Sonja (2025)", title)
}

func TestParseTitleYearFromTitle(t *testing.T) {
	name, year, ok := parseTitleYearFromTitle("Legend (2015)")
	require.True(t, ok)
	require.Equal(t, "Legend", name)
	require.Equal(t, 2015, year)

	name, year, ok = parseTitleYearFromTitle("Legend")
	require.False(t, ok)
	require.Equal(t, "", name)
	require.Equal(t, 0, year)

	name, year, ok = parseTitleYearFromTitle("Dracula 3000")
	require.False(t, ok)
	require.Equal(t, "", name)
	require.Equal(t, 0, year)

	name, year, ok = parseTitleYearFromTitle("Legend (2015) Director's Cut")
	require.False(t, ok)
	require.Equal(t, "", name)
	require.Equal(t, 0, year)
}

func TestParseNote_EmptyYAMLValue(t *testing.T) {
	// Test parsing a file with an empty YAML value (like "episodes:")
	// This reproduces the Abbot Elementary.md case
	content := `---
tags: [to-watch, tv]
service: disneyplus
status: watching
episodes:
finished: false
cover: "#a8ceb7"
created: 2026-01-08 01:30
modified: 2026-01-08 01:31
---
`

	note, err := parseNote(content)
	if err != nil {
		t.Fatalf("Failed to parse note with empty YAML value: %v", err)
	}

	// Verify basic parsing worked
	require.NotNil(t, note)
	require.NotNil(t, note.Frontmatter)

	// Check that tags were parsed
	tags := note.Frontmatter.GetStringArray("tags")
	require.Contains(t, tags, "to-watch")
	require.Contains(t, tags, "tv")

	// Check that episodes field exists (should be nil or empty)
	episodesVal, exists := note.Frontmatter.Get("episodes")
	t.Logf("episodes field exists: %v, value: %v (type: %T)", exists, episodesVal, episodesVal)

	// Check finished field
	require.False(t, note.Frontmatter.GetBool("finished"))

	// Check cover field
	require.Equal(t, "#a8ceb7", note.Frontmatter.GetString("cover"))
}
