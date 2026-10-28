package obsidian

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/hermes/internal/fileutil"
)

// NewFrontmatterWithTitle returns a Frontmatter initialized with the title field.
func NewFrontmatterWithTitle(title string) *Frontmatter {
	fm := NewFrontmatter()
	fm.Set("title", title)
	return fm
}

// ApplyTagSet assigns a TagSet to the frontmatter in sorted form.
func ApplyTagSet(fm *Frontmatter, tags *TagSet) {
	fm.Set("tags", tags.GetSorted())
}

// BuildNoteMarkdown builds markdown for a note using trimmed body content.
func BuildNoteMarkdown(fm *Frontmatter, body string) ([]byte, error) {
	note := &Note{
		Frontmatter: fm,
		Body:        strings.TrimSpace(body),
	}

	return note.Build()
}

// BuildAndWriteNote builds a markdown note from frontmatter and body and
// writes it to filePath, respecting the overwrite flag. The file contains
// exactly BuildNoteMarkdown's output, with no extra trailing newlines.
func BuildAndWriteNote(filePath string, fm *Frontmatter, body string, overwrite bool) error {
	markdown, err := BuildNoteMarkdown(fm, body)
	if err != nil {
		return fmt.Errorf("failed to build markdown for %s: %w", filepath.Base(filePath), err)
	}

	return fileutil.WriteMarkdownFile(filePath, string(markdown), overwrite)
}
