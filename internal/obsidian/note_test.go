package obsidian

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTitle   string
		wantTags    []string
		wantBody    string
		wantErr     bool
		description string
	}{
		{
			name: "basic frontmatter",
			input: `---
title: Test Movie
tags: [action, comedy]
year: 2020
---
This is the body content.`,
			wantTitle: "Test Movie",
			wantTags:  []string{"action", "comedy"},
			wantBody:  "This is the body content.",
			wantErr:   false,
		},
		{
			name: "block-style tags",
			input: `---
title: Test Movie
tags:
  - action
  - comedy
  - drama
---
Body content here.`,
			wantTitle: "Test Movie",
			wantTags:  []string{"action", "comedy", "drama"},
			wantBody:  "Body content here.",
			wantErr:   false,
		},
		{
			name:      "no frontmatter",
			input:     "Just body content, no frontmatter.",
			wantTitle: "",
			wantTags:  []string{},
			wantBody:  "Just body content, no frontmatter.",
			wantErr:   false,
		},
		{
			name: "empty frontmatter",
			input: `---
---
Body content.`,
			wantTitle: "",
			wantTags:  []string{},
			wantBody:  "Body content.",
			wantErr:   false,
		},
		{
			name: "no closing delimiter",
			input: `---
title: Test
This is broken`,
			wantTitle: "",
			wantTags:  []string{},
			wantBody: `---
title: Test
This is broken`,
			wantErr: false,
		},
		{
			name: "multiline body",
			input: `---
title: Test
---
Line 1
Line 2
Line 3`,
			wantTitle: "Test",
			wantTags:  []string{},
			wantBody:  "Line 1\nLine 2\nLine 3",
			wantErr:   false,
		},
		{
			name:      "empty input",
			input:     "",
			wantTitle: "",
			wantTags:  []string{},
			wantBody:  "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := ParseMarkdown([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMarkdown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if tt.wantTitle != "" {
				got := note.Frontmatter.GetString("title")
				if got != tt.wantTitle {
					t.Errorf("title = %q, want %q", got, tt.wantTitle)
				}
			}

			if len(tt.wantTags) > 0 {
				got := note.Frontmatter.GetStringArray("tags")
				if !reflect.DeepEqual(got, tt.wantTags) {
					t.Errorf("tags = %v, want %v", got, tt.wantTags)
				}
			}

			if note.Body != tt.wantBody {
				t.Errorf("body = %q, want %q", note.Body, tt.wantBody)
			}
		})
	}
}

func TestFrontmatterSetGet(t *testing.T) {
	t.Run("Set and Get", func(t *testing.T) {
		fm := NewFrontmatter()
		fm.Set("title", "Test")
		fm.Set("year", 2020)
		fm.Set("seen", true)

		if got := fm.GetString("title"); got != "Test" {
			t.Errorf("GetString(title) = %q, want %q", got, "Test")
		}
		if got := fm.GetInt("year"); got != 2020 {
			t.Errorf("GetInt(year) = %d, want %d", got, 2020)
		}
		if got := fm.GetBool("seen"); got != true {
			t.Errorf("GetBool(seen) = %v, want %v", got, true)
		}
	})

	t.Run("Get missing keys", func(t *testing.T) {
		fm := NewFrontmatter()

		if got := fm.GetString("missing"); got != "" {
			t.Errorf("GetString(missing) = %q, want empty string", got)
		}
		if got := fm.GetInt("missing"); got != 0 {
			t.Errorf("GetInt(missing) = %d, want 0", got)
		}
		if got := fm.GetBool("missing"); got != false {
			t.Errorf("GetBool(missing) = %v, want false", got)
		}
		if got := fm.GetStringArray("missing"); len(got) != 0 {
			t.Errorf("GetStringArray(missing) = %v, want empty slice", got)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		fm := NewFrontmatter()
		fm.Set("title", "Test")
		fm.Set("year", 2020)

		fm.Delete("title")

		if got := fm.GetString("title"); got != "" {
			t.Errorf("After Delete, GetString(title) = %q, want empty", got)
		}
		if _, ok := fm.Get("title"); ok {
			t.Errorf("After Delete, Get(title) should return ok=false")
		}

		// year should still exist
		if got := fm.GetInt("year"); got != 2020 {
			t.Errorf("GetInt(year) = %d, want %d", got, 2020)
		}
	})

	t.Run("Sorted keys", func(t *testing.T) {
		fm := NewFrontmatter()
		fm.Set("zebra", "z")
		fm.Set("apple", "a")
		fm.Set("banana", "b")

		want := []string{"apple", "banana", "zebra"}
		if !reflect.DeepEqual(fm.keys, want) {
			t.Errorf("keys = %v, want %v", fm.keys, want)
		}
	})

	t.Run("Update existing key preserves order", func(t *testing.T) {
		fm := NewFrontmatter()
		fm.Set("zebra", "z1")
		fm.Set("apple", "a1")
		fm.Set("banana", "b1")

		// Update existing key
		fm.Set("banana", "b2")

		// Keys should remain sorted
		want := []string{"apple", "banana", "zebra"}
		if !reflect.DeepEqual(fm.keys, want) {
			t.Errorf("keys after update = %v, want %v", fm.keys, want)
		}

		// Value should be updated
		if got := fm.GetString("banana"); got != "b2" {
			t.Errorf("GetString(banana) = %q, want %q", got, "b2")
		}
	})
}

func TestNoteBuild(t *testing.T) {
	t.Run("flow-style tags", func(t *testing.T) {
		note := &Note{
			Frontmatter: NewFrontmatter(),
			Body:        "Test body content.",
		}
		note.Frontmatter.Set("title", "Test Movie")
		note.Frontmatter.Set("tags", []string{"action", "comedy", "drama"})
		note.Frontmatter.Set("year", 2020)

		output, err := note.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		outputStr := string(output)

		// Should have frontmatter delimiters
		if !strings.Contains(outputStr, "---\n") {
			t.Errorf("Output missing frontmatter delimiters")
		}

		// Should have flow-style tags
		if !strings.Contains(outputStr, "tags: [action, comedy, drama]") {
			t.Errorf("Output missing flow-style tags, got:\n%s", outputStr)
		}

		// Should have body
		if !strings.Contains(outputStr, "Test body content.") {
			t.Errorf("Output missing body content")
		}

		// Keys should be sorted alphabetically
		lines := strings.Split(outputStr, "\n")
		var frontmatterLines []string
		inFrontmatter := false
		for _, line := range lines {
			if line == "---" {
				if !inFrontmatter {
					inFrontmatter = true
					continue
				} else {
					break
				}
			}
			if inFrontmatter && line != "" {
				frontmatterLines = append(frontmatterLines, line)
			}
		}

		// Check that keys appear in alphabetical order
		if len(frontmatterLines) >= 3 {
			// Should be: tags, title, year (alphabetically)
			if !strings.HasPrefix(frontmatterLines[0], "tags:") {
				t.Errorf("First key should be 'tags', got: %s", frontmatterLines[0])
			}
			if !strings.HasPrefix(frontmatterLines[1], "title:") {
				t.Errorf("Second key should be 'title', got: %s", frontmatterLines[1])
			}
			if !strings.HasPrefix(frontmatterLines[2], "year:") {
				t.Errorf("Third key should be 'year', got: %s", frontmatterLines[2])
			}
		}
	})

	t.Run("empty frontmatter", func(t *testing.T) {
		note := &Note{
			Frontmatter: NewFrontmatter(),
			Body:        "Just body content.",
		}

		output, err := note.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		outputStr := string(output)

		// Should not have frontmatter delimiters
		if strings.HasPrefix(outputStr, "---") {
			t.Errorf("Empty frontmatter should not produce delimiters")
		}

		// Should just be body
		if outputStr != "Just body content." {
			t.Errorf("Output = %q, want %q", outputStr, "Just body content.")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		note := &Note{
			Frontmatter: NewFrontmatter(),
			Body:        "",
		}
		note.Frontmatter.Set("title", "Test")

		output, err := note.Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		outputStr := string(output)

		// Should have frontmatter
		if !strings.Contains(outputStr, "title: Test") {
			t.Errorf("Output missing frontmatter")
		}

		// Should end after closing delimiter
		if !strings.HasSuffix(outputStr, "---\n") {
			t.Errorf("Output should end with closing delimiter and newline")
		}
	})
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "block-style tags converted to flow-style",
			input: `---
title: Test Movie
tags:
  - action
  - comedy
  - drama
year: 2020
---
Body content here.`,
		},
		{
			name: "flow-style tags preserved",
			input: `---
tags: [action, comedy, drama]
title: Test Movie
year: 2020
---
Body content here.`,
		},
		{
			name: "complex frontmatter",
			input: `---
imdb_id: tt1234567
seen: true
tags: [movie, action, sci-fi]
title: Test Movie
tmdb_id: 12345
year: 2020
---
# Test Movie

This is the body content.
More lines here.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			note, err := ParseMarkdown([]byte(tt.input))
			if err != nil {
				t.Fatalf("ParseMarkdown() error = %v", err)
			}

			// Build
			output, err := note.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			// Parse again
			note2, err := ParseMarkdown(output)
			if err != nil {
				t.Fatalf("Second ParseMarkdown() error = %v", err)
			}

			// Compare frontmatter values
			for _, key := range note.Frontmatter.keys {
				val1, _ := note.Frontmatter.Get(key)
				val2, _ := note2.Frontmatter.Get(key)

				// Special handling for tags - compare as sorted slices
				if key == "tags" {
					tags1 := TagsFromAny(val1)
					tags2 := TagsFromAny(val2)
					if !reflect.DeepEqual(tags1, tags2) {
						t.Errorf("Round trip mismatch for tags: %v != %v", tags1, tags2)
					}
				} else {
					if !reflect.DeepEqual(val1, val2) {
						t.Errorf("Round trip mismatch for %s: %v != %v", key, val1, val2)
					}
				}
			}

			// Compare bodies
			if note.Body != note2.Body {
				t.Errorf("Round trip body mismatch:\n%q\n!=\n%q", note.Body, note2.Body)
			}

			// Verify output has flow-style tags
			outputStr := string(output)
			if strings.Contains(tt.input, "tags:") {
				// Should have flow-style format
				if !strings.Contains(outputStr, "[") || !strings.Contains(outputStr, "]") {
					t.Errorf("Output should have flow-style tags, got:\n%s", outputStr)
				}
			}

			// Verify keys are sorted in output
			lines := strings.Split(outputStr, "\n")
			var keys []string
			inFrontmatter := false
			for _, line := range lines {
				if line == "---" {
					if !inFrontmatter {
						inFrontmatter = true
						continue
					} else {
						break
					}
				}
				if inFrontmatter && strings.Contains(line, ":") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) > 0 {
						keys = append(keys, strings.TrimSpace(parts[0]))
					}
				}
			}

			// Verify keys are sorted
			sortedKeys := make([]string, len(keys))
			copy(sortedKeys, keys)
			sort.Strings(sortedKeys)
			if !reflect.DeepEqual(keys, sortedKeys) {
				t.Errorf("Keys not sorted in output: %v, want %v", keys, sortedKeys)
			}
		})
	}
}
