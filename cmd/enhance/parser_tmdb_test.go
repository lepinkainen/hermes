package enhance

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/stretchr/testify/require"
)

func TestHasTMDBData(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "has tmdb_id and content markers",
			note: &Note{
				TMDBID: 12345,
				Body:   "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: true,
		},
		{
			name: "has tmdb_id but no content markers",
			note: &Note{
				TMDBID: 12345,
				Body:   "Some content without markers",
			},
			want: false,
		},
		{
			name: "no tmdb_id",
			note: &Note{
				TMDBID: 0,
				Body:   "Some content",
			},
			want: false,
		},
		{
			name: "has content markers but no tmdb_id",
			note: &Note{
				TMDBID: 0,
				Body:   "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
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
		Frontmatter: obsidian.NewFrontmatter(),
	}
	note.Frontmatter.Set("title", "Test Movie")
	note.Frontmatter.Set("type", "movie")
	note.Frontmatter.Set("year", 2021)

	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:      12345,
		TMDBType:    "movie",
		RuntimeMins: 120,
		GenreTags:   []string{"Action", "Adventure"},
		CoverPath:   "_attachments/cover.jpg",
	}

	note.AddTMDBData(tmdbData)

	if note.Frontmatter.GetInt("tmdb_id") != 12345 {
		t.Errorf("tmdb_id not set correctly")
	}
	if note.Frontmatter.GetString("tmdb_type") != "movie" {
		t.Errorf("tmdb_type not set correctly")
	}
	if note.Frontmatter.GetInt("runtime") != 120 {
		t.Errorf("runtime not set correctly")
	}
	if note.Frontmatter.GetString("cover") != "_attachments/cover.jpg" {
		t.Errorf("cover not set correctly")
	}

	tags := note.Frontmatter.GetStringArray("tags")
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestAddTMDBDataFinishedField(t *testing.T) {
	t.Run("setsFinishedTrueForEndedTVShow", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Ended Show")
		fm.Set("tmdb_type", "tv")
		note := &Note{
			Title:       "Ended Show",
			Type:        "tv",
			Frontmatter: fm,
		}

		finished := true
		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:        12345,
			TMDBType:      "tv",
			RuntimeMins:   45,
			TotalEpisodes: 100,
			Finished:      &finished,
		}

		note.AddTMDBData(tmdbData)

		require.Equal(t, true, note.Frontmatter.GetBool("finished"))
	})

	t.Run("setsFinishedFalseForOngoingTVShow", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Ongoing Show")
		fm.Set("tmdb_type", "tv")
		note := &Note{
			Title:       "Ongoing Show",
			Type:        "tv",
			Frontmatter: fm,
		}

		finished := false
		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:        12346,
			TMDBType:      "tv",
			RuntimeMins:   45,
			TotalEpisodes: 50,
			Finished:      &finished,
		}

		note.AddTMDBData(tmdbData)

		require.Equal(t, false, note.Frontmatter.GetBool("finished"))
	})

	t.Run("doesNotSetFinishedForMovie", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Test Movie")
		fm.Set("tmdb_type", "movie")
		note := &Note{
			Title:       "Test Movie",
			Type:        "movie",
			Frontmatter: fm,
		}

		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:      12347,
			TMDBType:    "movie",
			RuntimeMins: 120,
			Finished:    nil, // Movies don't have this field
		}

		note.AddTMDBData(tmdbData)

		_, exists := note.Frontmatter.Get("finished")
		require.False(t, exists, "finished field should not be set for movies")
	})

	t.Run("doesNotSetFinishedWhenStatusNotAvailable", func(t *testing.T) {
		fm := obsidian.NewFrontmatter()
		fm.Set("title", "Show Without Status")
		fm.Set("tmdb_type", "tv")
		note := &Note{
			Title:       "Show Without Status",
			Type:        "tv",
			Frontmatter: fm,
		}

		tmdbData := &enrichment.TMDBEnrichment{
			TMDBID:      12348,
			TMDBType:    "tv",
			RuntimeMins: 30,
			Finished:    nil, // Status not available from TMDB
		}

		note.AddTMDBData(tmdbData)

		// Should not set finished field when TMDB status is not available
		_, exists := note.Frontmatter.Get("finished")
		require.False(t, exists, "finished field should not be set when TMDB status is unavailable")
	})
}
