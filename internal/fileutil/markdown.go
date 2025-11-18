package fileutil

import (
	"fmt"
	"strings"
	"time"
)

// MarkdownBuilder helps construct markdown documents with frontmatter
type MarkdownBuilder struct {
	frontmatter    strings.Builder
	content        strings.Builder
	hasFrontmatter bool
}

// NewMarkdownBuilder creates a new markdown builder
func NewMarkdownBuilder() *MarkdownBuilder {
	mb := &MarkdownBuilder{}
	mb.frontmatter.WriteString("---\n")
	mb.hasFrontmatter = true
	return mb
}

// AddTitle adds a title field to the frontmatter
func (mb *MarkdownBuilder) AddTitle(title string) *MarkdownBuilder {
	fmt.Fprintf(&mb.frontmatter, "title: \"%s\"\n", title)
	return mb
}

// AddType adds a type field to the frontmatter
func (mb *MarkdownBuilder) AddType(mediaType string) *MarkdownBuilder {
	fmt.Fprintf(&mb.frontmatter, "type: %s\n", mediaType)
	return mb
}

// AddYear adds a year field to the frontmatter
func (mb *MarkdownBuilder) AddYear(year int) *MarkdownBuilder {
	if year > 0 {
		fmt.Fprintf(&mb.frontmatter, "year: %d\n", year)
	}
	return mb
}

// AddField adds a simple key-value field to the frontmatter
func (mb *MarkdownBuilder) AddField(key string, value interface{}) *MarkdownBuilder {
	switch v := value.(type) {
	case string:
		if v != "" {
			fmt.Fprintf(&mb.frontmatter, "%s: \"%s\"\n", key, v)
		}
	case int:
		if v != 0 {
			fmt.Fprintf(&mb.frontmatter, "%s: %d\n", key, v)
		}
	case float64:
		if v > 0 {
			fmt.Fprintf(&mb.frontmatter, "%s: %.1f\n", key, v)
		}
	case bool:
		fmt.Fprintf(&mb.frontmatter, "%s: %t\n", key, v)
	}
	return mb
}

// AddStringArray adds an array of strings to the frontmatter
func (mb *MarkdownBuilder) AddStringArray(key string, values []string) *MarkdownBuilder {
	if len(values) == 0 {
		return mb
	}

	mb.frontmatter.WriteString(key + ":\n")
	for _, value := range values {
		if value != "" {
			fmt.Fprintf(&mb.frontmatter, "  - \"%s\"\n", strings.TrimSpace(value))
		}
	}
	return mb
}

// AddTags adds a list of tags to the frontmatter
func (mb *MarkdownBuilder) AddTags(tags ...string) *MarkdownBuilder {
	if len(tags) == 0 {
		return mb
	}

	mb.frontmatter.WriteString("tags:\n")
	for _, tag := range tags {
		if tag != "" {
			fmt.Fprintf(&mb.frontmatter, "  - %s\n", tag)
		}
	}
	return mb
}

// GetDecadeTag returns a decade tag based on the year
func (mb *MarkdownBuilder) GetDecadeTag(year int) string {
	switch {
	case year >= 2020:
		return "year/2020s"
	case year >= 2010:
		return "year/2010s"
	case year >= 2000:
		return "year/2000s"
	case year >= 1990:
		return "year/1990s"
	case year >= 1980:
		return "year/1980s"
	case year >= 1970:
		return "year/1970s"
	case year >= 1960:
		return "year/1960s"
	case year >= 1950:
		return "year/1950s"
	default:
		return "year/pre-1950s"
	}
}

// AddDuration adds a duration field to the frontmatter
func (mb *MarkdownBuilder) AddDuration(minutes int) *MarkdownBuilder {
	if minutes <= 0 {
		return mb
	}

	fmt.Fprintf(&mb.frontmatter, "duration: %s\n", FormatDuration(minutes))
	return mb
}

// AddParagraph adds a paragraph of text to the content
func (mb *MarkdownBuilder) AddParagraph(text string) *MarkdownBuilder {
	if text == "" {
		return mb
	}

	mb.content.WriteString(text)
	mb.content.WriteString("\n\n")
	return mb
}

// AddImage adds an image to the content using standard markdown syntax
func (mb *MarkdownBuilder) AddImage(imageURL string) *MarkdownBuilder {
	if imageURL == "" {
		return mb
	}

	fmt.Fprintf(&mb.content, "![](%s)\n\n", imageURL)
	return mb
}

// AddObsidianImage adds an image using Obsidian's wiki-link syntax with sizing.
// Format: ![[filename|width]]
func (mb *MarkdownBuilder) AddObsidianImage(filename string, width int) *MarkdownBuilder {
	if filename == "" {
		return mb
	}

	if width > 0 {
		fmt.Fprintf(&mb.content, "![[%s|%d]]\n\n", filename, width)
	} else {
		fmt.Fprintf(&mb.content, "![[%s]]\n\n", filename)
	}
	return mb
}

// AddCallout adds a callout section to the content
func (mb *MarkdownBuilder) AddCallout(calloutType, title, content string) *MarkdownBuilder {
	if content == "" {
		return mb
	}

	if title != "" {
		fmt.Fprintf(&mb.content, ">[!%s]- %s\n", calloutType, title)
	} else {
		fmt.Fprintf(&mb.content, ">[!%s]\n", calloutType)
	}

	// Add an empty line after the callout header for info callouts (Steam format)
	if calloutType == "info" && title == "Game Details" {
		mb.content.WriteString(">\n")
	}

	// Add indented content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fmt.Fprintf(&mb.content, "> %s\n", line)
	}

	mb.content.WriteString("\n")
	return mb
}

// AddExternalLink adds an external link to the content
func (mb *MarkdownBuilder) AddExternalLink(title, url string) *MarkdownBuilder {
	if url == "" {
		return mb
	}

	fmt.Fprintf(&mb.content, "[%s](%s)\n\n", title, url)
	return mb
}

// AddExternalLinksCallout adds a callout with external links
func (mb *MarkdownBuilder) AddExternalLinksCallout(title string, links map[string]string) *MarkdownBuilder {
	if len(links) == 0 {
		return mb
	}

	mb.content.WriteString(">[!info]- " + title + "\n")
	for linkTitle, linkURL := range links {
		fmt.Fprintf(&mb.content, "> [%s](%s)\n", linkTitle, linkURL)
	}
	mb.content.WriteString("\n")

	return mb
}

// AddDate adds a date field to the frontmatter in YYYY-MM-DD format
func (mb *MarkdownBuilder) AddDate(key string, dateStr string) *MarkdownBuilder {
	if dateStr == "" {
		return mb
	}

	// Try to parse and reformat the date to YYYY-MM-DD
	// Common date formats to try
	formats := []string{
		"2006-01-02",          // YYYY-MM-DD
		"01/02/2006",          // MM/DD/YYYY
		"02/01/2006",          // DD/MM/YYYY
		"Jan 2, 2006",         // Mon D, YYYY
		"2 Jan, 2006",         // D Mon, YYYY
		"January 2, 2006",     // Month D, YYYY
		"2 January, 2006",     // D Month, YYYY
		"02 Jan 2006",         // DD Mon YYYY
		"Jan 02, 2006",        // Mon DD, YYYY
		"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
		"2006/01/02",          // YYYY/MM/DD
		"02-Jan-2006",         // DD-Mon-YYYY
		time.RFC3339,          // ISO 8601
	}

	// Try each format
	var parsedDate time.Time
	var err error
	for _, format := range formats {
		parsedDate, err = time.Parse(format, dateStr)
		if err == nil {
			// Successfully parsed
			break
		}
	}

	if err != nil {
		// Couldn't parse, use as-is
		fmt.Fprintf(&mb.frontmatter, "%s: \"%s\"\n", key, dateStr)
	} else {
		// Successfully parsed, format as YYYY-MM-DD
		fmt.Fprintf(&mb.frontmatter, "%s: \"%s\"\n", key, parsedDate.Format("2006-01-02"))
	}

	return mb
}

// Build returns the complete markdown document as a string
func (mb *MarkdownBuilder) Build() string {
	if !mb.hasFrontmatter {
		return mb.content.String()
	}

	var doc strings.Builder
	doc.WriteString(mb.frontmatter.String())
	doc.WriteString("---\n\n")
	doc.WriteString(mb.content.String())

	return doc.String()
}

// FormatDuration formats minutes into human-readable duration (e.g. "2h 30m")
func FormatDuration(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
