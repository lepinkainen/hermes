package enhance

import (
	"context"
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/stretchr/testify/assert"
)

func TestResolveSearchTitleYear(t *testing.T) {
	tests := []struct {
		name      string
		note      *Note
		wantTitle string
		wantYear  int
	}{
		{
			name:      "year already set",
			note:      &Note{Title: "Inception", Year: 2010},
			wantTitle: "Inception",
			wantYear:  2010,
		},
		{
			name:      "year missing, parsed from title",
			note:      &Note{Title: "The Matrix (1999)", Year: 0},
			wantTitle: "The Matrix",
			wantYear:  1999,
		},
		{
			name:      "year missing, title has no year",
			note:      &Note{Title: "Untitled", Year: 0},
			wantTitle: "Untitled",
			wantYear:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotYear := resolveSearchTitleYear(tt.note)
			assert.Equal(t, tt.wantTitle, gotTitle)
			assert.Equal(t, tt.wantYear, gotYear)
		})
	}
}

func TestBuildTMDBEnrichOpts(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test")
	fm.Set("tmdb_type", "movie")

	note := &Note{
		Type:        "movie",
		Frontmatter: fm,
	}
	opts := Options{
		TMDBDownloadCover:   true,
		TMDBInteractive:     true,
		TMDBContentSections: []string{"info", "cast"},
		UseTMDBCoverCache:   true,
		TMDBCoverCachePath:  "/tmp/covers",
		Force:               true,
		RefreshCache:        true,
		RegenerateData:      false,
	}

	got := buildTMDBEnrichOpts(note, opts, "/attach", "/note", true, true)

	assert.True(t, got.DownloadCover)
	assert.True(t, got.GenerateContent)
	assert.Equal(t, []string{"info", "cast"}, got.ContentSections)
	assert.Equal(t, "/attach", got.AttachmentsDir)
	assert.Equal(t, "/note", got.NoteDir)
	assert.True(t, got.Interactive)
	assert.True(t, got.Force)
	assert.True(t, got.RefreshCache)
	assert.Equal(t, "movie", got.StoredMediaType)
	assert.True(t, got.UseCoverCache)
	assert.Equal(t, "/tmp/covers", got.CoverCachePath)
}

func TestBuildTMDBEnrichOptsRegenerateForcesCoverAndContent(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test")

	note := &Note{Type: "movie", Frontmatter: fm}
	opts := Options{
		TMDBDownloadCover: false,
		RegenerateData:    true,
	}

	got := buildTMDBEnrichOpts(note, opts, "", "", false, false)

	assert.False(t, got.DownloadCover, "DownloadCover is false when TMDBDownloadCover is false even with RegenerateData")
	assert.True(t, got.GenerateContent, "RegenerateData forces GenerateContent")
}

func TestBuildTMDBEnrichOptsExpectedTypeFallback(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test")

	note := &Note{Type: "tv", Frontmatter: fm}
	got := buildTMDBEnrichOpts(note, Options{}, "", "", false, false)

	assert.Equal(t, "tv", got.ExpectedMediaType, "falls back to note.Type when tags do not specify media type")
}

func TestMovieTVNeedsOMDBShortCircuits(t *testing.T) {
	fmWithRatings := obsidian.NewFrontmatter()
	fmWithRatings.Set("imdb_rating", 9.0)

	tests := []struct {
		name string
		note *Note
		opts Options
	}{
		{
			name: "OMDB disabled",
			note: &Note{IMDBID: "tt0111161"},
			opts: Options{OMDBEnrich: false},
		},
		{
			name: "no IMDB ID",
			note: &Note{IMDBID: ""},
			opts: Options{OMDBEnrich: true},
		},
		{
			name: "already has OMDB data",
			note: &Note{IMDBID: "tt0111161", Frontmatter: fmWithRatings},
			opts: Options{OMDBEnrich: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.note.Frontmatter == nil {
				tt.note.Frontmatter = obsidian.NewFrontmatter()
			}
			assert.False(t, movieTVNeedsOMDB(tt.note, tt.opts))
		})
	}
}

func TestEnrichOMDBForMovieTVShortCircuits(t *testing.T) {
	tests := []struct {
		name     string
		note     *Note
		tmdbData *enrichment.TMDBEnrichment
		opts     Options
	}{
		{
			name:     "OMDB disabled",
			note:     &Note{IMDBID: "tt0111161"},
			tmdbData: &enrichment.TMDBEnrichment{IMDBID: "tt0111161"},
			opts:     Options{OMDBEnrich: false},
		},
		{
			name:     "no IMDB ID anywhere",
			note:     &Note{IMDBID: ""},
			tmdbData: &enrichment.TMDBEnrichment{IMDBID: ""},
			opts:     Options{OMDBEnrich: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enrichOMDBForMovieTV(context.Background(), tt.note, tt.tmdbData, tt.opts)
			assert.Nil(t, got)
		})
	}
}
