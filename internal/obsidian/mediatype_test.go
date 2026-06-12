package obsidian

import "testing"

func fmFrom(fields map[string]any) *Frontmatter {
	fm := NewFrontmatter()
	for k, v := range fields {
		fm.Set(k, v)
	}
	return fm
}

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		name string
		fm   *Frontmatter
		want string
	}{
		{
			name: "tmdb_type movie",
			fm:   fmFrom(map[string]any{"tmdb_type": "movie"}),
			want: "movie",
		},
		{
			name: "tmdb_type tv",
			fm:   fmFrom(map[string]any{"tmdb_type": "tv"}),
			want: "tv",
		},
		{
			name: "detect from movie tag",
			fm:   fmFrom(map[string]any{"tags": []any{"movie", "action"}}),
			want: "movie",
		},
		{
			name: "detect from tv tag",
			fm:   fmFrom(map[string]any{"tags": []any{"tv", "drama"}}),
			want: "tv",
		},
		{
			name: "detect from game tag",
			fm:   fmFrom(map[string]any{"tags": []any{"steam/game"}}),
			want: "game",
		},
		{
			name: "tmdb_type takes precedence over tags",
			fm:   fmFrom(map[string]any{"tmdb_type": "movie", "tags": []any{"tv"}}),
			want: "movie",
		},
		{
			name: "tv/ prefix beats movie/",
			fm:   fmFrom(map[string]any{"tags": []any{"movie/Crime", "tv/Action-and-Adventure"}}),
			want: "tv",
		},
		{
			name: "nil frontmatter",
			fm:   nil,
			want: "",
		},
		{
			name: "no type info",
			fm:   fmFrom(map[string]any{"title": "Test"}),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectMediaType(tt.fm)
			if got != tt.want {
				t.Errorf("DetectMediaType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectMediaTypeFromTags(t *testing.T) {
	tests := []struct {
		name string
		fm   *Frontmatter
		want string
	}{
		{
			name: "picks tv from tags even if tmdb_type movie",
			fm:   fmFrom(map[string]any{"tmdb_type": "movie", "tags": []any{"tv", "to-watch"}}),
			want: "tv",
		},
		{
			name: "returns movie from string slice tags",
			fm:   fmFrom(map[string]any{"tags": []string{"movie", "classic"}}),
			want: "movie",
		},
		{
			name: "tv prefix wins over movie prefix",
			fm:   fmFrom(map[string]any{"tags": []string{"movie/Comedy", "tv/Animation"}}),
			want: "tv",
		},
		{
			name: "empty when no tags",
			fm:   fmFrom(map[string]any{"title": "Example"}),
			want: "",
		},
		{
			name: "nil frontmatter",
			fm:   nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectMediaTypeFromTags(tt.fm)
			if got != tt.want {
				t.Errorf("DetectMediaTypeFromTags() = %q, want %q", got, tt.want)
			}
		})
	}
}
