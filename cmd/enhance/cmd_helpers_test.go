package enhance

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func completeNote(title string, tmdbID int) string {
	return fmt.Sprintf(`---
title: %s
tmdb_type: movie
tmdb_id: %d
cover: attachments/%s - cover.jpg
---

Existing body

<!-- TMDB_DATA_START -->
Existing TMDB data
<!-- TMDB_DATA_END -->
`, title, tmdbID, title)
}

func relPaths(t *testing.T, root string, files []string) []string {
	t.Helper()

	var rels []string
	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		require.NoError(t, err)
		rels = append(rels, filepath.ToSlash(rel))
	}

	sort.Strings(rels)
	return rels
}
