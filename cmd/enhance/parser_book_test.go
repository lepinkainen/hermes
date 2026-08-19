package enhance

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseNote_Book(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.CopyFile("testdata/basic_book.md", "basic_book.md")

	note, err := parseNoteFile(env.Path("basic_book.md"))
	require.NoError(t, err)

	require.True(t, note.IsBook())
	require.Equal(t, "book", note.Type)
	require.Equal(t, "Test Book", note.Title)
	require.Equal(t, "1234567890", note.ISBN)
	require.Equal(t, "1234567890123", note.ISBN13)
	require.Equal(t, "12345", note.GoodreadsID)
}

func TestParseNote_Book_Minimal(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.CopyFile("testdata/minimal_book.md", "minimal_book.md")

	note, err := parseNoteFile(env.Path("minimal_book.md"))
	require.NoError(t, err)

	require.True(t, note.IsBook())
	require.Empty(t, note.ISBN)
	require.Empty(t, note.ISBN13)
	require.Equal(t, "333", note.GoodreadsID)
}

func TestIsBook(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{name: "book type", note: &Note{Type: "book"}, want: true},
		{name: "movie type", note: &Note{Type: "movie"}, want: false},
		{name: "game type", note: &Note{Type: "game"}, want: false},
		{name: "empty type", note: &Note{Type: ""}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.note.IsBook())
		})
	}
}

func TestNeedsGoodreadsContent(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "no content markers",
			note: &Note{Body: "Some content without markers"},
			want: true,
		},
		{
			name: "has content markers",
			note: &Note{Body: "Some content\n\n<!-- GOODREADS_DATA_START -->\nBook info\n<!-- GOODREADS_DATA_END -->"},
			want: false,
		},
		{
			name: "empty body",
			note: &Note{Body: ""},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.note.NeedsGoodreadsContent())
		})
	}
}

func TestAddBookData_FillsOnlyEmptyFields(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test Book")
	fm.Set("isbn", "existing-isbn")
	fm.Set("publisher", "Existing Publisher")

	note := &Note{Frontmatter: fm}

	data := &enrichment.BookEnrichment{
		ISBN:          "new-isbn",
		ISBN13:        "new-isbn13",
		CoverPath:     "attachments/Test Book - cover.jpg",
		Pages:         321,
		Publisher:     "New Publisher",
		Description:   "A new description.",
		Subtitle:      "A Subtitle",
		Authors:       []string{"New Author"},
		Subjects:      []string{"Fiction"},
		SubjectPeople: []string{"Someone"},
	}

	note.AddBookData(data)

	// Already-present fields are preserved
	require.Equal(t, "existing-isbn", fm.GetString("isbn"))
	require.Equal(t, "Existing Publisher", fm.GetString("publisher"))

	// Empty fields are filled in
	require.Equal(t, "new-isbn13", fm.GetString("isbn13"))
	require.Equal(t, "attachments/Test Book - cover.jpg", fm.GetString("cover"))
	require.Equal(t, 321, fm.GetInt("pages"))
	require.Equal(t, "A new description.", fm.GetString("description"))
	require.Equal(t, "A Subtitle", fm.GetString("subtitle"))
	require.Equal(t, []string{"New Author"}, fm.GetStringArray("authors"))
	require.Equal(t, []string{"Fiction"}, fm.GetStringArray("subjects"))
	require.Equal(t, []string{"Someone"}, fm.GetStringArray("subject_people"))
}

func TestAddBookData_Nil(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test Book")
	note := &Note{Frontmatter: fm}

	// Should not panic and should be a no-op
	note.AddBookData(nil)

	require.Equal(t, "Test Book", fm.GetString("title"))
}

func TestBuildMarkdownForBook_AppendsWhenNoMarkers(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test Book")
	note := &Note{
		Frontmatter: fm,
		Body:        "Existing body",
	}

	data := &enrichment.BookEnrichment{
		ContentMarkdown: "## Book Info\n\nSome info",
	}

	result := note.BuildMarkdownForBook("original", data, false)

	require.Contains(t, result, "Existing body")
	require.Contains(t, result, "<!-- GOODREADS_DATA_START -->")
	require.Contains(t, result, "## Book Info")
	require.Contains(t, result, "<!-- GOODREADS_DATA_END -->")
}

func TestBuildMarkdownForBook_RegenerateReplaces(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test Book")
	note := &Note{
		Frontmatter: fm,
		Body:        "Existing body\n\n<!-- GOODREADS_DATA_START -->\nOld info\n<!-- GOODREADS_DATA_END -->",
	}

	data := &enrichment.BookEnrichment{
		ContentMarkdown: "New info",
	}

	result := note.BuildMarkdownForBook("original", data, true)

	require.NotContains(t, result, "Old info")
	require.Contains(t, result, "New info")
}

func TestBuildMarkdownForBook_NoRegenerateKeepsExisting(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test Book")
	note := &Note{
		Frontmatter: fm,
		Body:        "Existing body\n\n<!-- GOODREADS_DATA_START -->\nOld info\n<!-- GOODREADS_DATA_END -->",
	}

	data := &enrichment.BookEnrichment{
		ContentMarkdown: "New info",
	}

	result := note.BuildMarkdownForBook("original", data, false)

	require.Contains(t, result, "Old info")
	require.NotContains(t, result, "New info")
}

func TestBuildGoodreadsBaseDetails(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Test Book")
	fm.Set("subtitle", "A Subtitle")
	fm.Set("authors", []string{"Author One"})
	fm.Set("publisher", "Test Publisher")
	fm.Set("pages", 300)
	fm.Set("original_year", 2019)
	fm.Set("my_rating", 5) // int in frontmatter
	fm.Set("average_rating", 4.25)
	fm.Set("binding", "Paperback")
	fm.Set("description", "A description")
	fm.Set("subjects", []string{"Fiction"})
	fm.Set("subject_people", []string{"Someone"})

	note := &Note{
		Title:       "Test Book",
		Year:        2020,
		ISBN:        "111",
		ISBN13:      "1112223334445",
		GoodreadsID: "999",
		Frontmatter: fm,
	}

	details := note.buildGoodreadsBaseDetails()

	require.Equal(t, "Test Book", details.Title)
	require.Equal(t, "A Subtitle", details.Subtitle)
	require.Equal(t, []string{"Author One"}, details.Authors)
	require.Equal(t, "Test Publisher", details.Publisher)
	require.Equal(t, 300, details.Pages)
	require.Equal(t, 2020, details.YearPublished)
	require.Equal(t, 2019, details.OriginalPublicationYear)
	require.InEpsilon(t, 5.0, details.MyRating, 0.0001, "my_rating as int should coerce to float64")
	require.InEpsilon(t, 4.25, details.AverageRating, 0.0001)
	require.Equal(t, "111", details.ISBN)
	require.Equal(t, "1112223334445", details.ISBN13)
	require.Equal(t, "Paperback", details.Binding)
	require.Equal(t, "999", details.GoodreadsID)
	require.Equal(t, "A description", details.Description)
	require.Equal(t, []string{"Fiction"}, details.Subjects)
	require.Equal(t, []string{"Someone"}, details.SubjectPeople)
}

func TestFloatFromFrontmatter(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want float64
	}{
		{name: "float64", val: 4.5, want: 4.5},
		{name: "int", val: 5, want: 5.0},
		{name: "int64", val: int64(3), want: 3.0},
		{name: "string", val: "2.5", want: 2.5},
		{name: "invalid string", val: "not-a-number", want: 0},
		{name: "missing", val: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := obsidian.NewFrontmatter()
			if tt.val != nil {
				fm.Set("rating", tt.val)
			}
			got := floatFromFrontmatter(fm, "rating")
			if tt.want == 0 {
				require.Zero(t, got)
			} else {
				require.InEpsilon(t, tt.want, got, 0.0001)
			}
		})
	}
}
