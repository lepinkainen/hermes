package enhance

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/stretchr/testify/assert"
)

func TestHasOMDBData(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Note)
		expected bool
	}{
		{
			name: "has imdb_rating",
			setup: func(n *Note) {
				n.Frontmatter.Set("imdb_rating", 8.5)
			},
			expected: true,
		},
		{
			name: "has rt_score",
			setup: func(n *Note) {
				n.Frontmatter.Set("rt_score", "89%")
			},
			expected: true,
		},
		{
			name: "has metacritic_score",
			setup: func(n *Note) {
				n.Frontmatter.Set("metacritic_score", 75)
			},
			expected: true,
		},
		{
			name: "has multiple omdb fields",
			setup: func(n *Note) {
				n.Frontmatter.Set("imdb_rating", 8.5)
				n.Frontmatter.Set("rt_score", "89%")
			},
			expected: true,
		},
		{
			name:     "no omdb data",
			setup:    func(n *Note) {},
			expected: false,
		},
		{
			name: "has zero metacritic score",
			setup: func(n *Note) {
				n.Frontmatter.Set("metacritic_score", 0)
			},
			expected: false,
		},
		{
			name: "has empty rt_score",
			setup: func(n *Note) {
				n.Frontmatter.Set("rt_score", "")
			},
			expected: false,
		},
		{
			name: "has imdb_rating as int",
			setup: func(n *Note) {
				n.Frontmatter.Set("imdb_rating", 8)
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &Note{
				Frontmatter: obsidian.NewFrontmatter(),
			}
			tt.setup(note)
			assert.Equal(t, tt.expected, note.HasOMDBData())
		})
	}
}
