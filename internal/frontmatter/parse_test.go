package frontmatter

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(*testing.T, *ParsedNote)
	}{
		{
			name: "valid frontmatter",
			content: `---
title: Test Movie
year: 2024
tmdb_id: 12345
---
Body content here`,
			wantErr: false,
			check: func(t *testing.T, note *ParsedNote) {
				if note.GetString("title") != "Test Movie" {
					t.Errorf("expected title 'Test Movie', got %q", note.GetString("title"))
				}
				if note.GetInt("year") != 2024 {
					t.Errorf("expected year 2024, got %d", note.GetInt("year"))
				}
				if note.GetInt("tmdb_id") != 12345 {
					t.Errorf("expected tmdb_id 12345, got %d", note.GetInt("tmdb_id"))
				}
				if note.Body != "Body content here" {
					t.Errorf("expected body 'Body content here', got %q", note.Body)
				}
			},
		},
		{
			name:    "missing opening delimiter",
			content: `no frontmatter here`,
			wantErr: true,
		},
		{
			name: "missing closing delimiter",
			content: `---
title: Test
incomplete`,
			wantErr: true,
		},
		{
			name: "empty frontmatter",
			content: `---
---
Body only`,
			wantErr: false,
			check: func(t *testing.T, note *ParsedNote) {
				if note.Body != "Body only" {
					t.Errorf("expected body 'Body only', got %q", note.Body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := ParseMarkdown([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMarkdown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, note)
			}
		})
	}
}

func TestIntFromAny(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int
	}{
		{"int", 42, 42},
		{"int64", int64(123), 123},
		{"float64", float64(99.7), 99},
		{"string", "456", 456},
		{"string with spaces", "  789  ", 789},
		{"invalid string", "not a number", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntFromAny(tt.val)
			if got != tt.want {
				t.Errorf("IntFromAny(%v) = %d, want %d", tt.val, got, tt.want)
			}
		})
	}
}

func TestStringFromAny(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"string", "hello", "hello"},
		{"string with spaces", "  world  ", "world"},
		{"int", 42, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringFromAny(tt.val)
			if got != tt.want {
				t.Errorf("StringFromAny(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestParsedNote_GetInt(t *testing.T) {
	note := &ParsedNote{
		Frontmatter: map[string]any{
			"present": 42,
			"string":  "123",
		},
	}

	if got := note.GetInt("present"); got != 42 {
		t.Errorf("GetInt('present') = %d, want 42", got)
	}
	if got := note.GetInt("missing"); got != 0 {
		t.Errorf("GetInt('missing') = %d, want 0", got)
	}
	if got := note.GetInt("string"); got != 123 {
		t.Errorf("GetInt('string') = %d, want 123", got)
	}
}

func TestParsedNote_GetString(t *testing.T) {
	note := &ParsedNote{
		Frontmatter: map[string]any{
			"present": "hello",
			"int":     42,
		},
	}

	if got := note.GetString("present"); got != "hello" {
		t.Errorf("GetString('present') = %q, want 'hello'", got)
	}
	if got := note.GetString("missing"); got != "" {
		t.Errorf("GetString('missing') = %q, want ''", got)
	}
	if got := note.GetString("int"); got != "" {
		t.Errorf("GetString('int') = %q, want ''", got)
	}
}

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		want string
	}{
		{
			name: "tmdb_type movie",
			fm:   map[string]any{"tmdb_type": "movie"},
			want: "movie",
		},
		{
			name: "tmdb_type movie from YAML",
			fm: func() map[string]any {
				var m map[string]any
				_ = yaml.Unmarshal([]byte("tmdb_type: movie"), &m)
				return m
			}(),
			want: "movie",
		},
		{
			name: "tmdb_type tv",
			fm:   map[string]any{"tmdb_type": "tv"},
			want: "tv",
		},
		{
			name: "detect from movie tag",
			fm:   map[string]any{"tags": []any{"movie", "action"}},
			want: "movie",
		},
		{
			name: "detect from tv tag",
			fm:   map[string]any{"tags": []any{"tv", "drama"}},
			want: "tv",
		},
		{
			name: "tmdb_type takes precedence over tags",
			fm:   map[string]any{"tmdb_type": "movie", "tags": []any{"tv"}},
			want: "movie",
		},
		{
			name: "tv/ prefix beats movie/",
			fm:   map[string]any{"tags": []any{"movie/Crime", "tv/Action-and-Adventure"}},
			want: "tv",
		},
		{
			name: "nil frontmatter",
			fm:   nil,
			want: "",
		},
		{
			name: "no type info",
			fm:   map[string]any{"title": "Test"},
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
		fm   map[string]any
		want string
	}{
		{
			name: "picks tv from tags even if tmdb_type movie",
			fm:   map[string]any{"tmdb_type": "movie", "tags": []any{"tv", "to-watch"}},
			want: "tv",
		},
		{
			name: "returns movie from string slice tags",
			fm:   map[string]any{"tags": []string{"movie", "classic"}},
			want: "movie",
		},
		{
			name: "tv prefix wins over movie prefix",
			fm:   map[string]any{"tags": []string{"movie/Comedy", "tv/Animation"}},
			want: "tv",
		},
		{
			name: "empty when no tags",
			fm:   map[string]any{"title": "Example"},
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
