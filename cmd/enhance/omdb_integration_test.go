package enhance

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment/omdb"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/stretchr/testify/assert"
)

func TestAddOMDBData(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Title:       "The Shawshank Redemption",
		Year:        1994,
		IMDBID:      "tt0111161",
	}

	omdbData := &omdb.RatingsEnrichment{
		IMDbRating:     9.3,
		RottenTomatoes: "91%",
		RTTomatometer:  91,
		Metacritic:     81,
	}

	note.AddOMDBData(omdbData)

	// Verify frontmatter was updated
	imdbRating, ok := note.Frontmatter.Get("imdb_rating")
	assert.True(t, ok)
	assert.Equal(t, 9.3, imdbRating)

	assert.Equal(t, "91%", note.Frontmatter.GetString("rt_score"))
	assert.Equal(t, 91, note.Frontmatter.GetInt("rt_tomatometer"))
	assert.Equal(t, 81, note.Frontmatter.GetInt("metacritic_score"))
}

func TestAddOMDBData_NilData(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Title:       "Test Movie",
	}

	// Should not panic with nil data
	note.AddOMDBData(nil)

	// Frontmatter should remain empty
	_, ok := note.Frontmatter.Get("imdb_rating")
	assert.False(t, ok)
	assert.Equal(t, "", note.Frontmatter.GetString("rt_score"))
}

func TestAddOMDBData_PartialData(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Title:       "Test Movie",
	}

	// Only IMDb rating available
	omdbData := &omdb.RatingsEnrichment{
		IMDbRating:     7.5,
		RottenTomatoes: "",
		RTTomatometer:  0,
		Metacritic:     0,
	}

	note.AddOMDBData(omdbData)

	// Only IMDb rating should be set
	imdbRating, ok := note.Frontmatter.Get("imdb_rating")
	assert.True(t, ok)
	assert.Equal(t, 7.5, imdbRating)

	assert.Equal(t, "", note.Frontmatter.GetString("rt_score"))
	assert.Equal(t, 0, note.Frontmatter.GetInt("rt_tomatometer"))
	assert.Equal(t, 0, note.Frontmatter.GetInt("metacritic_score"))
}

func TestBuildMarkdown_WithOMDBRatings(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Body:        "Original content here.",
	}
	note.Frontmatter.Set("title", "Test Movie")
	note.Frontmatter.Set("year", 2021)

	originalContent := `---
title: Test Movie
year: 2021
---

Original content here.`

	omdbRatings := &omdb.RatingsEnrichment{
		IMDbRating:     8.5,
		RottenTomatoes: "89%",
		RTTomatometer:  89,
		Metacritic:     75,
	}

	result := note.BuildMarkdown(originalContent, nil, omdbRatings, false)

	// Check that ratings table is present
	assert.Contains(t, result, "## Ratings")
	assert.Contains(t, result, "| IMDb | ⭐ 8.5/10 |")
	assert.Contains(t, result, "| Rotten Tomatoes | 🍅 89% |")
	assert.Contains(t, result, "| Metacritic | 📊 75/100 |")

	// Check that original content is preserved
	assert.Contains(t, result, "Original content here.")
}

func TestBuildMarkdown_WithOMDBAndTMDB(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Body:        "Original content here.",
	}
	note.Frontmatter.Set("title", "Test Movie")
	note.Frontmatter.Set("year", 2021)

	originalContent := `---
title: Test Movie
year: 2021
---

Original content here.`

	omdbRatings := &omdb.RatingsEnrichment{
		IMDbRating:     8.5,
		RottenTomatoes: "89%",
		RTTomatometer:  89,
		Metacritic:     75,
	}

	result := note.BuildMarkdown(originalContent, nil, omdbRatings, false)

	// Ratings table should come before TMDB content (in this case, no TMDB content)
	assert.Contains(t, result, "## Ratings")

	// Both ratings and original content should be present
	assert.Contains(t, result, "| IMDb | ⭐ 8.5/10 |")
	assert.Contains(t, result, "Original content here.")

	// Check that markers are present
	assert.Contains(t, result, "<!-- TMDB_DATA_START -->")
	assert.Contains(t, result, "<!-- TMDB_DATA_END -->")
}

func TestBuildMarkdown_NoOMDBData(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Body:        "Original content here.",
	}
	note.Frontmatter.Set("title", "Test Movie")

	originalContent := `---
title: Test Movie
---

Original content here.`

	// No OMDB data provided
	result := note.BuildMarkdown(originalContent, nil, nil, false)

	// Should not contain ratings table
	assert.NotContains(t, result, "## Ratings")
	assert.NotContains(t, result, "IMDb")
	assert.NotContains(t, result, "Rotten Tomatoes")

	// Original content should still be present
	assert.Contains(t, result, "Original content here.")
}

func TestMetacriticScoreConflict(t *testing.T) {
	// Test that OMDB metacritic_score overwrites Steam metacritic_score
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Title:       "Test Game",
	}

	// Set Steam metacritic score first
	note.Frontmatter.Set("metacritic_score", 70)
	assert.Equal(t, 70, note.Frontmatter.GetInt("metacritic_score"))

	// Add OMDB data with different metacritic score
	omdbData := &omdb.RatingsEnrichment{
		Metacritic: 85,
	}
	note.AddOMDBData(omdbData)

	// OMDB score should overwrite Steam score
	assert.Equal(t, 85, note.Frontmatter.GetInt("metacritic_score"))
}

func TestOMDBEnrichment_WithoutIMDbID(t *testing.T) {
	note := &Note{
		Frontmatter: obsidian.NewFrontmatter(),
		Title:       "Test Movie",
		Year:        2021,
		IMDBID:      "", // No IMDb ID
	}

	originalContent := `---
title: Test Movie
year: 2021
---

Original content.`

	// When IMDb ID is missing, OMDB enrichment should be skipped
	// This is handled at the caller level (cmd.go), not in parser
	result := note.BuildMarkdown(originalContent, nil, nil, false)

	// Should not have ratings table
	assert.NotContains(t, result, "## Ratings")
}
