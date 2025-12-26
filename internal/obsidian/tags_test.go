package obsidian

import (
	"reflect"
	"testing"
)

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Basic cases
		{"simple tag", "action", "action"},
		{"with spaces", "Action Comedy", "Action-Comedy"},
		{"multiple spaces", "Action  Comedy", "Action-Comedy"},
		{"leading hash", "#Sci-Fi", "Sci-Fi"},
		{"leading and trailing whitespace", "  genre/Horror  ", "genre/Horror"},
		{"ampersand", "& Other", "and-Other"},
		{"ampersand in middle", "Rock & Roll", "Rock-and-Roll"},

		// Edge cases from plan
		{"double spaces", "Action  Comedy", "Action-Comedy"},
		{"hash symbol", "#Sci-Fi", "Sci-Fi"},
		{"genre with spaces", "  genre/Horror  ", "genre/Horror"},
		{"ampersand prefix", "& Other", "and-Other"},

		// Hyphen handling
		{"multiple hyphens", "foo---bar", "foo-bar"},
		{"leading hyphens", "---test", "test"},
		{"trailing hyphens", "test---", "test"},
		{"mixed hyphens and spaces", "foo -- bar", "foo-bar"},

		// Special characters
		{"hash in middle", "test#value", "testvalue"},
		{"multiple hashes", "##test##", "test"},
		{"mixed special chars", "Rock & Roll #1", "Rock-and-Roll-1"},

		// Hierarchy preservation
		{"genre hierarchy", "genre/Action", "genre/Action"},
		{"nested hierarchy", "game/genre/RPG", "game/genre/RPG"},
		{"hierarchy with spaces", "genre / Action", "genre-/-Action"},

		// Empty and whitespace
		{"empty string", "", ""},
		{"only whitespace", "   ", ""},
		{"only hash", "#", ""},
		{"only hyphens", "---", ""},
		{"only ampersand", "&", "and"},

		// Case preservation
		{"preserve case", "MyTag", "MyTag"},
		{"preserve mixed case", "camelCaseTag", "camelCaseTag"},

		// Tab and newline handling
		{"tabs", "foo\tbar", "foo-bar"},
		{"newlines", "foo\nbar", "foo-bar"},
		{"mixed whitespace", "foo \t\n bar", "foo-bar"},

		// Real-world examples
		{"movie genre", "Science Fiction", "Science-Fiction"},
		{"tv genre", "Sci-Fi & Fantasy", "Sci-Fi-and-Fantasy"},
		{"rating tag", "rating/4", "rating/4"},
		{"decade tag", "year/2020s", "year/2020s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTag(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeTag(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "simple list",
			input: []string{"action", "comedy", "drama"},
			want:  []string{"action", "comedy", "drama"},
		},
		{
			name:  "with duplicates",
			input: []string{"action", "Action", "ACTION"},
			want:  []string{"ACTION", "Action", "action"}, // Case preserved and sorted
		},
		{
			name:  "with spaces and normalization",
			input: []string{"Action Comedy", "#Sci-Fi", "genre/Horror"},
			want:  []string{"Action-Comedy", "Sci-Fi", "genre/Horror"},
		},
		{
			name:  "with empty strings",
			input: []string{"action", "", "comedy", "   ", "drama"},
			want:  []string{"action", "comedy", "drama"},
		},
		{
			name:  "duplicates after normalization",
			input: []string{"Action  Comedy", "Action Comedy", "#Action-Comedy"},
			want:  []string{"Action-Comedy"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "all empty",
			input: []string{"", "   ", "#", "---"},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTags(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagSet(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		ts := NewTagSet()
		ts.Add("action")
		ts.Add("comedy")
		ts.Add("drama")

		got := ts.GetSorted()
		want := []string{"action", "comedy", "drama"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetSorted() = %v, want %v", got, want)
		}
	})

	t.Run("automatic normalization", func(t *testing.T) {
		ts := NewTagSet()
		ts.Add("Action  Comedy")
		ts.Add("#Sci-Fi")
		ts.Add("  genre/Horror  ")

		got := ts.GetSorted()
		want := []string{"Action-Comedy", "Sci-Fi", "genre/Horror"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetSorted() = %v, want %v", got, want)
		}
	})

	t.Run("deduplication", func(t *testing.T) {
		ts := NewTagSet()
		ts.Add("action")
		ts.Add("action")
		ts.Add("Action")
		ts.Add("#action")

		got := ts.GetSorted()
		// Case preserved - "action" and "Action" are different
		want := []string{"Action", "action"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetSorted() = %v, want %v", got, want)
		}
	})

	t.Run("AddIf conditional", func(t *testing.T) {
		ts := NewTagSet()
		ts.AddIf(true, "action")
		ts.AddIf(false, "comedy")
		ts.AddIf(true, "drama")

		got := ts.GetSorted()
		want := []string{"action", "drama"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetSorted() = %v, want %v", got, want)
		}
	})

	t.Run("AddFormat", func(t *testing.T) {
		ts := NewTagSet()
		ts.AddFormat("rating/%d", 4)
		ts.AddFormat("year/%ds", 2020)
		ts.AddFormat("genre/%s", "Action")

		got := ts.GetSorted()
		want := []string{"genre/Action", "rating/4", "year/2020s"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetSorted() = %v, want %v", got, want)
		}
	})

	t.Run("empty tags filtered", func(t *testing.T) {
		ts := NewTagSet()
		ts.Add("")
		ts.Add("   ")
		ts.Add("#")
		ts.Add("valid")

		got := ts.GetSorted()
		want := []string{"valid"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GetSorted() = %v, want %v", got, want)
		}
	})
}

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		new      []string
		want     []string
	}{
		{
			name:     "no overlap",
			existing: []string{"action", "comedy"},
			new:      []string{"drama", "thriller"},
			want:     []string{"action", "comedy", "drama", "thriller"},
		},
		{
			name:     "with overlap",
			existing: []string{"action", "comedy"},
			new:      []string{"comedy", "drama"},
			want:     []string{"action", "comedy", "drama"},
		},
		{
			name:     "empty existing",
			existing: []string{},
			new:      []string{"action", "comedy"},
			want:     []string{"action", "comedy"},
		},
		{
			name:     "empty new",
			existing: []string{"action", "comedy"},
			new:      []string{},
			want:     []string{"action", "comedy"},
		},
		{
			name:     "both empty",
			existing: []string{},
			new:      []string{},
			want:     []string{},
		},
		{
			name:     "with normalization",
			existing: []string{"Action  Comedy", "#Sci-Fi"},
			new:      []string{"genre/Horror", "Action-Comedy"},
			want:     []string{"Action-Comedy", "Sci-Fi", "genre/Horror"},
		},
		{
			name:     "empty strings filtered",
			existing: []string{"action", "", "comedy"},
			new:      []string{"drama", "   ", "thriller"},
			want:     []string{"action", "comedy", "drama", "thriller"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeTags(tt.existing, tt.new)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagsFromAny(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []string
	}{
		{
			name:  "nil",
			input: nil,
			want:  []string{},
		},
		{
			name:  "string slice",
			input: []string{"action", "comedy", "drama"},
			want:  []string{"action", "comedy", "drama"},
		},
		{
			name:  "string slice with empty",
			input: []string{"action", "", "comedy"},
			want:  []string{"action", "comedy"},
		},
		{
			name:  "interface slice",
			input: []interface{}{"action", "comedy", "drama"},
			want:  []string{"action", "comedy", "drama"},
		},
		{
			name:  "interface slice with mixed types",
			input: []interface{}{"action", 123, "comedy", nil, "drama"},
			want:  []string{"action", "comedy", "drama"},
		},
		{
			name:  "interface slice with empty strings",
			input: []interface{}{"action", "", "comedy"},
			want:  []string{"action", "comedy"},
		},
		{
			name:  "wrong type",
			input: "not a slice",
			want:  []string{},
		},
		{
			name:  "empty string slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "empty interface slice",
			input: []interface{}{},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TagsFromAny(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TagsFromAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
