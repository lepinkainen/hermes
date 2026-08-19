package enrichment

import (
	"github.com/lepinkainen/hermes/internal/content"
)

// BookEnrichmentOptions holds options for book enrichment.
type BookEnrichmentOptions struct {
	// DownloadCover determines whether to download the cover image
	DownloadCover bool
	// GenerateContent determines whether to generate book content sections
	GenerateContent bool
	// ContentSections specifies which sections to generate (empty = default:
	// "info", "description", "subjects")
	ContentSections []string
	// AttachmentsDir is the directory where images will be stored
	AttachmentsDir string
	// NoteDir is the directory where the note will be stored
	NoteDir string
	// Interactive enables TUI for multiple matches
	Interactive bool
	// Force forces re-enrichment/re-download even when data already exists
	Force bool
	// BaseDetails holds note-side fields that should be preserved rather
	// than overwritten by enrichment data. Only empty fields are filled in.
	BaseDetails *content.GoodreadsBookDetails
	// Authors is used to narrow title search when no ISBN is available
	Authors []string
}

// BookEnrichment holds book enrichment data.
type BookEnrichment struct {
	// ISBN is the ISBN-10, possibly discovered via search
	ISBN string
	// ISBN13 is the ISBN-13, possibly discovered via search
	ISBN13 string
	// CoverPath is the relative path to the downloaded cover image
	CoverPath string
	// CoverFilename is just the filename of the cover
	CoverFilename string
	// ContentMarkdown is the generated book content
	ContentMarkdown string
	// Description is the book's summary or blurb
	Description string
	// Subtitle is the secondary title or tagline
	Subtitle string
	// Publisher is the publishing company name
	Publisher string
	// PublishDate is the publication date (format varies by source)
	PublishDate string
	// Pages is the page count
	Pages int
	// Subjects are topic/category tags
	Subjects []string
	// SubjectPeople are people mentioned as subjects (biographies, etc.)
	SubjectPeople []string
	// Authors are the book's author names
	Authors []string
}
