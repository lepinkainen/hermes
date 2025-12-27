package enhance

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/importer/mediaids"
)

func TestGetMediaIDs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    mediaids.MediaIDs
	}{
		{
			name: "TMDB ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
---

Content here.`,
			want: mediaids.MediaIDs{TMDBID: 949},
		},
		{
			name: "IMDB ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
imdb_id: "tt0113277"
---

Content here.`,
			want: mediaids.MediaIDs{IMDBID: "tt0113277"},
		},
		{
			name: "Letterboxd ID only",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
letterboxd_id: "2bg8"
---

Content here.`,
			want: mediaids.MediaIDs{LetterboxdID: "2bg8"},
		},
		{
			name: "All IDs present",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
tmdb_id: 949
imdb_id: "tt0113277"
letterboxd_id: "2bg8"
---

Content here.`,
			want: mediaids.MediaIDs{
				TMDBID:       949,
				IMDBID:       "tt0113277",
				LetterboxdID: "2bg8",
			},
		},
		{
			name: "No IDs",
			content: `---
title: "Test Movie"
tmdb_type: movie
year: 2021
---

Content here.`,
			want: mediaids.MediaIDs{},
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
tmdb_type: movie
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
tmdb_type: movie
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
tmdb_type: movie
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
tmdb_type: movie
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
tmdb_type: movie
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
tmdb_type: movie
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
tmdb_type: movie
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
tmdb_type: movie
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
tmdb_type: movie
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
