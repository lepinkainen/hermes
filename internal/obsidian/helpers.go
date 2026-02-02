package obsidian

import "strings"

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
