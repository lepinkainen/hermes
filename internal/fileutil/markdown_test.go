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
