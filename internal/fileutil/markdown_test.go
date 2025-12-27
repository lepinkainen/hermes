package fileutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownBuilder(t *testing.T) {
	// Basic builder test
	builder := NewMarkdownBuilder()

	doc := builder.
		AddTitle("Test Title").
		AddType("movie").
		AddYear(2021).
		AddField("custom_field", "custom value").
		AddField("rating", 4.5).
		AddField("runtime_mins", 120).
		AddDuration(120).
		AddStringArray("genres", []string{"Action", "Drama"}).
		AddTags("tag1", "tag2", builder.GetDecadeTag(2021)).
		AddParagraph("This is a test paragraph.").
		AddImage("https://example.com/image.jpg").
		AddCallout("summary", "Plot", "This is the plot summary.").
		AddExternalLink("View Website", "https://example.com").
		Build()

	// Check that frontmatter exists
	assert.True(t, strings.HasPrefix(doc, "---\n"))
	assert.True(t, strings.Contains(doc, "---\n\n"))

	// Check frontmatter fields
	assert.Contains(t, doc, "title: \"Test Title\"")
	assert.Contains(t, doc, "type: movie")
	assert.Contains(t, doc, "year: 2021")
	assert.Contains(t, doc, "custom_field: \"custom value\"")
	assert.Contains(t, doc, "rating: 4.5")
	assert.Contains(t, doc, "runtime_mins: 120")
	assert.Contains(t, doc, "duration: 2h 0m")

	// Check arrays
	assert.Contains(t, doc, "genres:")
	assert.Contains(t, doc, "  - \"Action\"")
	assert.Contains(t, doc, "  - \"Drama\"")

	// Check tags
	assert.Contains(t, doc, "tags:")
	assert.Contains(t, doc, "  - tag1")
	assert.Contains(t, doc, "  - tag2")
	assert.Contains(t, doc, "  - year/2020s")

	// Check content
	assert.Contains(t, doc, "This is a test paragraph.")
	assert.Contains(t, doc, "![](https://example.com/image.jpg)")

	// Check callout
	assert.Contains(t, doc, ">[!summary]- Plot")
	assert.Contains(t, doc, "> This is the plot summary.")

	// Check link
	assert.Contains(t, doc, "[View Website](https://example.com)")
}

func TestExternalLinksCallout(t *testing.T) {
	builder := NewMarkdownBuilder()

	links := map[string]string{
		"View on Website": "https://example.com",
		"View on IMDb":    "https://imdb.com/title/123",
	}

	doc := builder.AddExternalLinksCallout("External Links", links).Build()

	assert.Contains(t, doc, ">[!info]- External Links")
	assert.Contains(t, doc, "> [View on Website](https://example.com)")
	assert.Contains(t, doc, "> [View on IMDb](https://imdb.com/title/123)")
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		minutes  int
		expected string
	}{
		{120, "2h 0m"},
		{90, "1h 30m"},
		{45, "0h 45m"},
		{0, "0h 0m"},
		{135, "2h 15m"},
		{180, "3h 0m"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := FormatDuration(tc.minutes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAddDate(t *testing.T) {
	testCases := []struct {
		name     string
		dateStr  string
		expected string
	}{
		{
			name:     "already_iso_format",
			dateStr:  "2023-01-15",
			expected: "date: \"2023-01-15\"",
		},
		{
			name:     "mm_dd_yyyy",
			dateStr:  "01/15/2023",
			expected: "date: \"2023-01-15\"",
		},
		{
			name:     "dd_mm_yyyy",
			dateStr:  "15/01/2023",
			expected: "date: \"2023-01-15\"",
		},
		{
			name:     "month_name",
			dateStr:  "Jan 15, 2023",
			expected: "date: \"2023-01-15\"",
		},
		{
			name:     "steam_format",
			dateStr:  "15 Jan, 2023",
			expected: "date: \"2023-01-15\"",
		},
		{
			name:     "full_month_name",
			dateStr:  "January 15, 2023",
			expected: "date: \"2023-01-15\"",
		},
		{
			name:     "unparseable_format",
			dateStr:  "Some random text",
			expected: "date: \"Some random text\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mb := NewMarkdownBuilder()
			mb.AddDate("date", tc.dateStr)
			result := mb.Build()
			assert.Contains(t, result, tc.expected)
		})
	}
}

func TestAddTMDBEnrichmentFields_ZeroTMDBID(t *testing.T) {
	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")
	mb.AddTMDBEnrichmentFields(0, "movie", 0, "")

	result := mb.Build()

	assert.Contains(t, result, "title: \"Test Movie\"")
	assert.NotContains(t, result, "tmdb_id:")
	assert.NotContains(t, result, "tmdb_type:")
}

func TestAddTMDBEnrichmentFields_ValidTMDBID(t *testing.T) {
	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")
	mb.AddTMDBEnrichmentFields(12345, "movie", 0, "")

	result := mb.Build()

	assert.Contains(t, result, "tmdb_id: 12345")
	assert.Contains(t, result, "tmdb_type: \"movie\"")
}

func TestAddTMDBEnrichmentFields_EmptyContentMarkdown(t *testing.T) {
	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")
	mb.AddTMDBEnrichmentFields(12345, "movie", 0, "")

	result := mb.Build()

	// Should have frontmatter with TMDB fields but no additional content
	assert.Contains(t, result, "tmdb_id: 12345")
	assert.Contains(t, result, "tmdb_type: \"movie\"")

	// Check that after the closing --- there's no content besides whitespace
	parts := strings.Split(result, "---")
	if len(parts) >= 3 {
		// parts[2] is everything after the second ---
		contentAfterFrontmatter := strings.TrimSpace(parts[2])
		assert.Empty(t, contentAfterFrontmatter, "Expected no content after frontmatter")
	}
}

func TestAddTMDBEnrichmentFields_WithContentMarkdown(t *testing.T) {
	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")
	mb.AddTMDBEnrichmentFields(12345, "movie", 0, "## Cast\n- Actor 1\n- Actor 2")

	result := mb.Build()

	assert.Contains(t, result, "## Cast")
	assert.Contains(t, result, "- Actor 1")
	assert.Contains(t, result, "- Actor 2")
}

func TestAddTMDBEnrichmentFields_TotalEpisodes(t *testing.T) {
	testCases := []struct {
		name          string
		totalEpisodes int
		shouldContain bool
	}{
		{
			name:          "zero episodes not added",
			totalEpisodes: 0,
			shouldContain: false,
		},
		{
			name:          "positive episodes added",
			totalEpisodes: 24,
			shouldContain: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mb := NewMarkdownBuilder()
			mb.AddTitle("Test TV Show")
			mb.AddTMDBEnrichmentFields(12345, "tv", tc.totalEpisodes, "")

			result := mb.Build()

			if tc.shouldContain {
				assert.Contains(t, result, "total_episodes: 24")
			} else {
				assert.NotContains(t, result, "total_episodes:")
			}
		})
	}
}

func TestAddTMDBEnrichmentFields_FullEnrichment(t *testing.T) {
	mb := NewMarkdownBuilder()
	mb.AddTitle("Breaking Bad")
	mb.AddType("tv")
	mb.AddYear(2008)
	mb.AddTMDBEnrichmentFields(1396, "tv", 62, "## Overview\nA high school chemistry teacher...")

	result := mb.Build()

	// Check frontmatter
	assert.Contains(t, result, "title: \"Breaking Bad\"")
	assert.Contains(t, result, "type: tv")
	assert.Contains(t, result, "year: 2008")
	assert.Contains(t, result, "tmdb_id: 1396")
	assert.Contains(t, result, "tmdb_type: \"tv\"")
	assert.Contains(t, result, "total_episodes: 62")

	// Check content
	assert.Contains(t, result, "## Overview")
	assert.Contains(t, result, "A high school chemistry teacher...")
}

func TestAddTMDBEnrichmentFields_Chaining(t *testing.T) {
	mb := NewMarkdownBuilder()
	result := mb.AddTitle("Test").AddTMDBEnrichmentFields(123, "movie", 0, "content")
	assert.Same(t, mb, result)
}
