package enhance

import (
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestGolden_TMDBBasicEnrichment tests basic TMDB enrichment:
// Input: minimal note with just title and year
// Output: note with TMDB data (tmdb_id, runtime, genres, cover, content)
func TestGolden_TMDBBasicEnrichment(t *testing.T) {
	gh := testutil.NewGoldenHelper(t, filepath.Join("testdata", "golden"))

	// Read input from golden file
	input := gh.MustReadGoldenString("tmdb_basic_input.md")

	// Parse the input note
	note, err := parseNote(input)
	require.NoError(t, err)
	require.Equal(t, "Heat", note.Title)
	require.Equal(t, 1995, note.Year)

	// Create mock TMDB enrichment data
	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:      949,
		TMDBType:    "movie",
		RuntimeMins: 170,
		GenreTags:   []string{"genre/action", "genre/crime", "genre/drama"},
		CoverPath:   "attachments/Heat - cover.jpg",
		ContentMarkdown: `## Cast

| Actor | Character |
|-------|-----------|
| Al Pacino | Lt. Vincent Hanna |
| Robert De Niro | Neil McCauley |
| Val Kilmer | Chris Shiherlis |

## Crew

| Role | Name |
|------|------|
| Director | Michael Mann |
| Writer | Michael Mann |

## Similar Movies

- [[Collateral]]
- [[The Town]]
- [[Ronin]]`,
	}

	// Apply TMDB data to the note
	note.AddTMDBData(tmdbData)

	// Build the markdown output
	output := note.BuildMarkdown(input, tmdbData, nil, true)

	// Compare with golden file
	gh.AssertGoldenString("tmdb_basic_output.md", output)
}

// TestGolden_OMDBEnrichment tests OMDB ratings enrichment:
// Input: note with TMDB data but no OMDB ratings
// Output: note with OMDB ratings added to frontmatter and ratings table prepended to content
func TestGolden_OMDBEnrichment(t *testing.T) {
	gh := testutil.NewGoldenHelper(t, filepath.Join("testdata", "golden"))

	// Read input from golden file
	input := gh.MustReadGoldenString("omdb_input.md")

	// Parse the input note
	note, err := parseNote(input)
	require.NoError(t, err)
	require.Equal(t, "Heat", note.Title)
	require.Equal(t, 949, note.TMDBID)
	require.Equal(t, "tt0113277", note.IMDBID)
	require.False(t, note.HasOMDBData(), "input note should not have OMDB data")

	// Create mock OMDB ratings data
	omdbRatings := &omdb.RatingsEnrichment{
		IMDbRating:     8.3,
		RottenTomatoes: "87%",
		RTTomatometer:  87,
		Metacritic:     76,
	}

	// In the real flow, TMDB enrichment provides the existing content to preserve
	// When regenerating, we pass the existing TMDB content to keep it
	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:   949,
		TMDBType: "movie",
		IMDBID:   "tt0113277",
		ContentMarkdown: `## Cast

| Actor | Character |
|-------|-----------|
| Al Pacino | Lt. Vincent Hanna |
| Robert De Niro | Neil McCauley |

## Crew

| Role | Name |
|------|------|
| Director | Michael Mann |`,
	}

	// Apply OMDB data to the note
	note.AddOMDBData(omdbRatings)

	// Verify OMDB data was added
	require.True(t, note.HasOMDBData(), "note should have OMDB data after enrichment")

	// Build the markdown output - tmdbData provides content to preserve
	// The ratings table is prepended to the TMDB content
	output := note.BuildMarkdown(input, tmdbData, omdbRatings, true)

	// Compare with golden file
	gh.AssertGoldenString("omdb_output.md", output)
}

// TestGolden_CombinedEnrichment tests both TMDB and OMDB enrichment together
func TestGolden_CombinedEnrichment(t *testing.T) {
	// Start with minimal input
	input := `---
title: Heat
year: 1995
---

A classic heist movie.
`

	// Parse the input note
	note, err := parseNote(input)
	require.NoError(t, err)

	// Create mock TMDB data
	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:      949,
		TMDBType:    "movie",
		IMDBID:      "tt0113277",
		RuntimeMins: 170,
		GenreTags:   []string{"genre/action", "genre/crime"},
		CoverPath:   "attachments/Heat - cover.jpg",
		ContentMarkdown: `## Cast

| Actor | Character |
|-------|-----------|
| Al Pacino | Lt. Vincent Hanna |
| Robert De Niro | Neil McCauley |`,
	}

	// Create mock OMDB data
	omdbRatings := &omdb.RatingsEnrichment{
		IMDbRating:     8.3,
		RottenTomatoes: "87%",
		RTTomatometer:  87,
		Metacritic:     76,
	}

	// Apply both enrichments
	note.AddTMDBData(tmdbData)
	note.AddOMDBData(omdbRatings)

	// Verify data was added
	require.Equal(t, 949, note.TMDBID)
	require.Equal(t, "tt0113277", note.IMDBID)
	require.True(t, note.HasOMDBData())

	// Build the markdown output with both
	output := note.BuildMarkdown(input, tmdbData, omdbRatings, true)

	// Verify output contains expected elements
	require.Contains(t, output, "tmdb_id: 949")
	require.Contains(t, output, "imdb_id: tt0113277")
	require.Contains(t, output, "imdb_rating: 8.3")
	require.Contains(t, output, "rt_score: 87%")
	require.Contains(t, output, "metacritic_score: 76")
	require.Contains(t, output, "<!-- TMDB_DATA_START -->")
	require.Contains(t, output, "## Ratings")
	require.Contains(t, output, "## Cast")
	require.Contains(t, output, "<!-- TMDB_DATA_END -->")
}

// TestGolden_UpdateNoteWithTMDBData tests the full update flow with file I/O
func TestGolden_UpdateNoteWithTMDBData(t *testing.T) {
	env := testutil.NewTestEnv(t)
	gh := testutil.NewGoldenHelper(t, filepath.Join("testdata", "golden"))

	// Copy input to temp directory
	input := gh.MustReadGoldenString("tmdb_basic_input.md")
	env.WriteFileString("Heat.md", input)
	path := env.Path("Heat.md")

	// Parse the note
	note, err := parseNoteFile(path)
	require.NoError(t, err)

	// Create mock TMDB enrichment data
	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:      949,
		TMDBType:    "movie",
		RuntimeMins: 170,
		GenreTags:   []string{"genre/action", "genre/crime", "genre/drama"},
		CoverPath:   "attachments/Heat - cover.jpg",
		ContentMarkdown: `## Cast

| Actor | Character |
|-------|-----------|
| Al Pacino | Lt. Vincent Hanna |
| Robert De Niro | Neil McCauley |
| Val Kilmer | Chris Shiherlis |

## Crew

| Role | Name |
|------|------|
| Director | Michael Mann |
| Writer | Michael Mann |

## Similar Movies

- [[Collateral]]
- [[The Town]]
- [[Ronin]]`,
	}

	// Update the note file
	err = updateNoteWithTMDBData(path, note, tmdbData, nil, true)
	require.NoError(t, err)

	// Read the output and compare with golden file
	output := env.ReadFileString("Heat.md")
	gh.AssertGoldenString("tmdb_basic_output.md", output)
}

// TestGolden_UpdateNoteWithOMDBData tests adding OMDB ratings to existing TMDB note
func TestGolden_UpdateNoteWithOMDBData(t *testing.T) {
	env := testutil.NewTestEnv(t)
	gh := testutil.NewGoldenHelper(t, filepath.Join("testdata", "golden"))

	// Copy input to temp directory
	input := gh.MustReadGoldenString("omdb_input.md")
	env.WriteFileString("Heat.md", input)
	path := env.Path("Heat.md")

	// Parse the note
	note, err := parseNoteFile(path)
	require.NoError(t, err)

	// Create mock OMDB ratings data
	omdbRatings := &omdb.RatingsEnrichment{
		IMDbRating:     8.3,
		RottenTomatoes: "87%",
		RTTomatometer:  87,
		Metacritic:     76,
	}

	// In the real flow, TMDB enrichment provides the existing content to preserve
	tmdbData := &enrichment.TMDBEnrichment{
		TMDBID:   949,
		TMDBType: "movie",
		IMDBID:   "tt0113277",
		ContentMarkdown: `## Cast

| Actor | Character |
|-------|-----------|
| Al Pacino | Lt. Vincent Hanna |
| Robert De Niro | Neil McCauley |

## Crew

| Role | Name |
|------|------|
| Director | Michael Mann |`,
	}

	// Update the note file
	err = updateNoteWithTMDBData(path, note, tmdbData, omdbRatings, true)
	require.NoError(t, err)

	// Read the output and compare with golden file
	output := env.ReadFileString("Heat.md")
	gh.AssertGoldenString("omdb_output.md", output)
}
