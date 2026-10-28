package obsidian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAndWriteNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Note.md")

	fm := NewFrontmatterWithTitle("Note")
	fm.Set("type", "movie")

	require.NoError(t, BuildAndWriteNote(path, fm, "\n\nBody text\n\n", false))

	got, err := os.ReadFile(path)
	require.NoError(t, err)

	want, err := BuildNoteMarkdown(fm, "Body text")
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got), "file must contain exactly the built note")
	assert.Equal(t, "---\ntitle: Note\ntype: movie\n---\nBody text", string(got))

	t.Run("does not overwrite existing file by default", func(t *testing.T) {
		fm2 := NewFrontmatterWithTitle("Changed")
		require.NoError(t, BuildAndWriteNote(path, fm2, "New body", false))

		after, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, string(got), string(after))
	})

	t.Run("overwrites when asked", func(t *testing.T) {
		fm2 := NewFrontmatterWithTitle("Changed")
		require.NoError(t, BuildAndWriteNote(path, fm2, "New body", true))

		after, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "---\ntitle: Changed\n---\nNew body", string(after))
	})
}
