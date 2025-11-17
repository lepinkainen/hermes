package content

import (
	"testing"
)

func TestWrapWithMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "basic content",
			content: "Some TMDB content",
			want:    "<!-- TMDB_DATA_START -->\nSome TMDB content\n<!-- TMDB_DATA_END -->",
		},
		{
			name:    "multiline content",
			content: "Line 1\nLine 2\nLine 3",
			want:    "<!-- TMDB_DATA_START -->\nLine 1\nLine 2\nLine 3\n<!-- TMDB_DATA_END -->",
		},
		{
			name:    "content with leading/trailing whitespace",
			content: "  \n  Content  \n  ",
			want:    "<!-- TMDB_DATA_START -->\nContent\n<!-- TMDB_DATA_END -->",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "content with markdown",
			content: "## Overview\n\nThis is a **great** movie.",
			want:    "<!-- TMDB_DATA_START -->\n## Overview\n\nThis is a **great** movie.\n<!-- TMDB_DATA_END -->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapWithMarkers(tt.content)
			if got != tt.want {
				t.Errorf("WrapWithMarkers() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasTMDBContentMarkers(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "has both markers",
			body: "Some content\n<!-- TMDB_DATA_START -->\nTMDB content\n<!-- TMDB_DATA_END -->\nMore content",
			want: true,
		},
		{
			name: "only start marker",
			body: "Some content\n<!-- TMDB_DATA_START -->\nTMDB content",
			want: false,
		},
		{
			name: "only end marker",
			body: "Some content\nTMDB content\n<!-- TMDB_DATA_END -->",
			want: false,
		},
		{
			name: "no markers",
			body: "Just regular content",
			want: false,
		},
		{
			name: "empty body",
			body: "",
			want: false,
		},
		{
			name: "markers in wrong order",
			body: "<!-- TMDB_DATA_END -->\nContent\n<!-- TMDB_DATA_START -->",
			want: true, // Function just checks for presence, not order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasTMDBContentMarkers(tt.body)
			if got != tt.want {
				t.Errorf("HasTMDBContentMarkers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTMDBContent(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantContent string
		wantFound   bool
	}{
		{
			name:        "basic content between markers",
			body:        "Before\n<!-- TMDB_DATA_START -->\nTMDB content here\n<!-- TMDB_DATA_END -->\nAfter",
			wantContent: "TMDB content here",
			wantFound:   true,
		},
		{
			name:        "multiline content between markers",
			body:        "<!-- TMDB_DATA_START -->\n## Overview\n\nThis is a movie.\n\n## Cast\n\n- Actor 1\n<!-- TMDB_DATA_END -->",
			wantContent: "## Overview\n\nThis is a movie.\n\n## Cast\n\n- Actor 1",
			wantFound:   true,
		},
		{
			name:        "no markers",
			body:        "Just regular content",
			wantContent: "",
			wantFound:   false,
		},
		{
			name:        "only start marker",
			body:        "<!-- TMDB_DATA_START -->\nSome content",
			wantContent: "",
			wantFound:   false,
		},
		{
			name:        "empty content between markers",
			body:        "<!-- TMDB_DATA_START --><!-- TMDB_DATA_END -->",
			wantContent: "",
			wantFound:   true,
		},
		{
			name:        "whitespace between markers",
			body:        "<!-- TMDB_DATA_START -->\n   \n   \n<!-- TMDB_DATA_END -->",
			wantContent: "",
			wantFound:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContent, gotFound := GetTMDBContent(tt.body)
			if gotContent != tt.wantContent {
				t.Errorf("GetTMDBContent() content = %q, want %q", gotContent, tt.wantContent)
			}
			if gotFound != tt.wantFound {
				t.Errorf("GetTMDBContent() found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestReplaceTMDBContent(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		newContent string
		want       string
	}{
		{
			name:       "replace existing content",
			body:       "Before\n<!-- TMDB_DATA_START -->\nOld content\n<!-- TMDB_DATA_END -->\nAfter",
			newContent: "New content",
			want:       "Before\n\n<!-- TMDB_DATA_START -->\nNew content\n<!-- TMDB_DATA_END -->\nAfter",
		},
		{
			name:       "replace with multiline content",
			body:       "<!-- TMDB_DATA_START -->\nOld\n<!-- TMDB_DATA_END -->",
			newContent: "Line 1\nLine 2\nLine 3",
			want:       "<!-- TMDB_DATA_START -->\nLine 1\nLine 2\nLine 3\n<!-- TMDB_DATA_END -->",
		},
		{
			name:       "no markers - return unchanged",
			body:       "Just regular content",
			newContent: "New content",
			want:       "Just regular content",
		},
		{
			name:       "replace with empty content",
			body:       "Before\n<!-- TMDB_DATA_START -->\nOld content\n<!-- TMDB_DATA_END -->\nAfter",
			newContent: "",
			want:       "Before\n\n<!-- TMDB_DATA_START -->\n\n<!-- TMDB_DATA_END -->\nAfter",
		},
		{
			name:       "content before markers",
			body:       "Original intro\n\n<!-- TMDB_DATA_START -->\nOld\n<!-- TMDB_DATA_END -->",
			newContent: "New TMDB data",
			want:       "Original intro\n\n<!-- TMDB_DATA_START -->\nNew TMDB data\n<!-- TMDB_DATA_END -->",
		},
		{
			name:       "content after markers",
			body:       "<!-- TMDB_DATA_START -->\nOld\n<!-- TMDB_DATA_END -->\n\nOriginal outro",
			newContent: "New TMDB data",
			want:       "<!-- TMDB_DATA_START -->\nNew TMDB data\n<!-- TMDB_DATA_END -->\nOriginal outro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceTMDBContent(tt.body, tt.newContent)
			if got != tt.want {
				t.Errorf("ReplaceTMDBContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildCoverImageEmbed(t *testing.T) {
	tests := []struct {
		name          string
		coverFilename string
		want          string
	}{
		{
			name:          "basic filename",
			coverFilename: "movie-poster.jpg",
			want:          "![[movie-poster.jpg|250]]",
		},
		{
			name:          "filename with path",
			coverFilename: "_attachments/cover.jpg",
			want:          "![[_attachments/cover.jpg|250]]",
		},
		{
			name:          "empty filename",
			coverFilename: "",
			want:          "",
		},
		{
			name:          "filename with spaces",
			coverFilename: "My Movie Poster.png",
			want:          "![[My Movie Poster.png|250]]",
		},
		{
			name:          "filename with special characters",
			coverFilename: "poster (2023) [4K].webp",
			want:          "![[poster (2023) [4K].webp|250]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCoverImageEmbed(tt.coverFilename)
			if got != tt.want {
				t.Errorf("BuildCoverImageEmbed() = %q, want %q", got, tt.want)
			}
		})
	}
}
