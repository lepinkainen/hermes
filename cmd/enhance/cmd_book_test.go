package enhance

import (
	"context"
	"fmt"
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

// completeBookNote returns a book note that already has a valid local cover
// file reference and Goodreads content markers, i.e. one that processBookNote
// should skip by default.
func completeBookNote(title string) string {
	return fmt.Sprintf(`---
title: %s
type: book
authors:
    - Test Author
isbn: "1234567890"
cover: attachments/%s - cover.jpg
---

Existing body

<!-- GOODREADS_DATA_START -->
Existing book data
<!-- GOODREADS_DATA_END -->
`, title, title)
}

// withStubbedBookEnrichment redirects the enrichBookFunc seam for the
// duration of the test.
func withStubbedBookEnrichment(t *testing.T, fn func(ctx context.Context, title, isbn, isbn13 string, opts enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error)) {
	t.Helper()
	orig := enrichBookFunc
	enrichBookFunc = fn
	t.Cleanup(func() { enrichBookFunc = orig })
}

func TestProcessBookNote_Skip(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("Test Book.md", completeBookNote("Test Book"))
	env.WriteFileString("attachments/Test Book - cover.jpg", "fake cover data")

	called := false
	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		called = true
		return nil, nil
	})

	file := env.Path("Test Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, skip := processBookNote(context.Background(), file, note, Options{}, env.Path("attachments"))

	require.False(t, success)
	require.True(t, skip)
	require.False(t, called, "enrichment should not be called when note is already complete")
}

func TestProcessBookNote_DryRun(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("New Book.md", "---\ntitle: New Book\ntype: book\n---\n\nBody")

	called := false
	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		called = true
		return nil, nil
	})

	file := env.Path("New Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, skip := processBookNote(context.Background(), file, note, Options{DryRun: true}, env.Path("attachments"))

	require.True(t, success)
	require.False(t, skip)
	require.False(t, called, "dry-run must not call the enrichment seam")
}

func TestProcessBookNote_Success(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("New Book.md", "---\ntitle: New Book\ntype: book\nisbn: \"1234567890\"\n---\n\nBody")

	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		return &enrichment.BookEnrichment{
			ISBN13:          "1234567890123",
			Publisher:       "Test Publisher",
			ContentMarkdown: "## Book Info\n\nSome info",
			Authors:         []string{"Test Author"},
		}, nil
	})

	file := env.Path("New Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, skip := processBookNote(context.Background(), file, note, Options{}, env.Path("attachments"))

	require.True(t, success)
	require.False(t, skip)

	// Note is renamed to the "<author> - <title>.md" convention since no
	// authors were present in the frontmatter but the enrichment result had one.
	require.True(t, env.FileExists("Test Author - New Book.md"))
	require.False(t, env.FileExists("New Book.md"))

	body := env.ReadFileString("Test Author - New Book.md")
	require.Contains(t, body, `isbn13: "1234567890123"`)
	require.Contains(t, body, "publisher: Test Publisher")
	require.Contains(t, body, "<!-- GOODREADS_DATA_START -->")
	require.Contains(t, body, "## Book Info")
}

func TestProcessBookNote_NoDataFound(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("New Book.md", "---\ntitle: New Book\ntype: book\n---\n\nBody")

	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		return nil, nil
	})

	file := env.Path("New Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, skip := processBookNote(context.Background(), file, note, Options{}, env.Path("attachments"))

	require.False(t, success)
	require.True(t, skip)
}

func TestProcessBookNote_EnrichmentError(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("New Book.md", "---\ntitle: New Book\ntype: book\n---\n\nBody")

	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		return nil, fmt.Errorf("boom")
	})

	file := env.Path("New Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, skip := processBookNote(context.Background(), file, note, Options{}, env.Path("attachments"))

	require.False(t, success)
	require.False(t, skip)
}

func TestProcessBookNote_Rename_SkipsWhenTargetExistsNonInteractive(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("New Book.md", "---\ntitle: New Book\ntype: book\n---\n\nBody")
	env.WriteFileString("Test Author - New Book.md", "---\ntitle: Other\n---\n\nexisting target")

	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		return &enrichment.BookEnrichment{
			ContentMarkdown: "info",
			Authors:         []string{"Test Author"},
		}, nil
	})

	file := env.Path("New Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, skip := processBookNote(context.Background(), file, note, Options{BookInteractive: false}, env.Path("attachments"))

	require.True(t, success)
	require.False(t, skip)

	// Original file remains in place (rename was skipped) with the enriched content.
	require.True(t, env.FileExists("New Book.md"))
	require.Contains(t, env.ReadFileString("New Book.md"), "info")

	// The pre-existing target file is untouched.
	require.Equal(t, "---\ntitle: Other\n---\n\nexisting target", env.ReadFileString("Test Author - New Book.md"))
}

func TestProcessBookNote_NoRename_WhenNoAuthorKnown(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("New Book.md", "---\ntitle: New Book\ntype: book\n---\n\nBody")

	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		return &enrichment.BookEnrichment{ContentMarkdown: "info"}, nil
	})

	file := env.Path("New Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, _ := processBookNote(context.Background(), file, note, Options{}, env.Path("attachments"))
	require.True(t, success)

	require.True(t, env.FileExists("New Book.md"), "file should remain at original path when no author is known")
}

func TestProcessBookNote_Rename_NoOpWhenAlreadyNamedCorrectly(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("Test Author - New Book.md", "---\ntitle: New Book\ntype: book\nauthors:\n    - Test Author\n---\n\nBody")

	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		return &enrichment.BookEnrichment{ContentMarkdown: "info"}, nil
	})

	file := env.Path("Test Author - New Book.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, _ := processBookNote(context.Background(), file, note, Options{}, env.Path("attachments"))
	require.True(t, success)
	require.True(t, env.FileExists("Test Author - New Book.md"))
}

func TestProcessBookNote_Rename_NoOpWhenTitleAlreadyAuthorPrefixed(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.WriteFileString("Glynn Stewart - Exile.md", "---\ntitle: Glynn Stewart - Exile\ntype: book\nauthors:\n    - Glynn Stewart\n---\n\nBody")

	withStubbedBookEnrichment(t, func(_ context.Context, _, _, _ string, _ enrichment.BookEnrichmentOptions) (*enrichment.BookEnrichment, error) {
		return &enrichment.BookEnrichment{ContentMarkdown: "info"}, nil
	})

	file := env.Path("Glynn Stewart - Exile.md")
	note, err := parseNoteFile(file)
	require.NoError(t, err)

	success, _ := processBookNote(context.Background(), file, note, Options{}, env.Path("attachments"))
	require.True(t, success)
	// File should NOT be renamed to "Glynn Stewart - Glynn Stewart - Exile.md"
	require.True(t, env.FileExists("Glynn Stewart - Exile.md"))
}
