package enhance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/obsidian"
)

func TestNeedsCover(t *testing.T) {
	// Create a temporary directory for testing file existence
	tempDir := t.TempDir()
	attachmentsDir := filepath.Join(tempDir, "attachments")
	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		t.Fatalf("Failed to create test attachments dir: %v", err)
	}

	// Create a test cover file
	existingCover := filepath.Join(attachmentsDir, "existing-cover.jpg")
	if err := os.WriteFile(existingCover, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test cover file: %v", err)
	}

	tests := []struct {
		name    string
		note    *Note
		noteDir string
		want    bool
	}{
		{
			name: "no cover field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				return &Note{Frontmatter: fm}
			}(),
			noteDir: tempDir,
			want:    true,
		},
		{
			name: "empty cover field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("cover", "")
				return &Note{Frontmatter: fm}
			}(),
			noteDir: tempDir,
			want:    true,
		},
		{
			name: "cover field with existing file",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("cover", "attachments/existing-cover.jpg")
				return &Note{Frontmatter: fm}
			}(),
			noteDir: tempDir,
			want:    false,
		},
		{
			name: "cover field with non-existent file",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("cover", "attachments/missing-cover.jpg")
				return &Note{Frontmatter: fm}
			}(),
			noteDir: tempDir,
			want:    true,
		},
		{
			name: "cover field is not a string",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("cover", 12345)
				return &Note{Frontmatter: fm}
			}(),
			noteDir: tempDir,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.NeedsCover(tt.noteDir); got != tt.want {
				t.Errorf("NeedsCover() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsContent(t *testing.T) {
	tests := []struct {
		name string
		note *Note
		want bool
	}{
		{
			name: "no content markers",
			note: &Note{
				Body: "Some content without markers",
			},
			want: true,
		},
		{
			name: "has content markers",
			note: &Note{
				Body: "Some content\n\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->",
			},
			want: false,
		},
		{
			name: "empty body",
			note: &Note{
				Body: "",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.note.NeedsContent(); got != tt.want {
				t.Errorf("NeedsContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasSeenField(t *testing.T) {
	tests := []struct {
		name     string
		note     *Note
		expected bool
	}{
		{
			name: "has seen field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("seen", true)
				return &Note{Seen: true, Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "does not have seen field",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				return &Note{Seen: false, Frontmatter: fm}
			}(),
			expected: false,
		},
		{
			name: "has seen field set to false",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("seen", false)
				return &Note{Seen: false, Frontmatter: fm}
			}(),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.note.hasSeenField()
			if result != tc.expected {
				t.Errorf("hasSeenField() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestHasAnyRating(t *testing.T) {
	tests := []struct {
		name     string
		note     *Note
		expected bool
	}{
		{
			name: "has imdb_rating",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("imdb_rating", 8.5)
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "has my_rating",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("my_rating", 9)
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "has letterboxd_rating",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("letterboxd_rating", 4.5)
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
		{
			name: "has zero ratings",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("imdb_rating", 0.0)
				fm.Set("my_rating", 0)
				fm.Set("letterboxd_rating", 0.0)
				return &Note{Frontmatter: fm}
			}(),
			expected: false,
		},
		{
			name: "has no ratings",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("year", 2020)
				return &Note{Frontmatter: fm}
			}(),
			expected: false,
		},
		{
			name: "has mixed rating types (int and float)",
			note: func() *Note {
				fm := obsidian.NewFrontmatter()
				fm.Set("title", "Test Movie")
				fm.Set("my_rating", 8)     // int
				fm.Set("imdb_rating", 7.5) // float64
				return &Note{Frontmatter: fm}
			}(),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.note.hasAnyRating()
			if result != tc.expected {
				t.Errorf("hasAnyRating() = %v, want %v", result, tc.expected)
			}
		})
	}
}
