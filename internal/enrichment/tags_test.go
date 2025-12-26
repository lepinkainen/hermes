// Package enrichment_test contains legacy tests for deprecated tag functions.
//
// These tests verify backward compatibility for enrichment.MergeTags and enrichment.TagsFromAny,
// which are deprecated in favor of obsidian.MergeTags and obsidian.TagsFromAny.
// These functions and tests will be removed in v2.0.0.
//
// New code should use the obsidian package equivalents, which provide built-in tag normalization
// according to Obsidian conventions.
package enrichment

import (
	"reflect"
	"testing"
)

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		new      []string
		want     []string
	}{
		{
			name:     "no overlap",
			existing: []string{"a", "b"},
			new:      []string{"c", "d"},
			want:     []string{"a", "b", "c", "d"},
		},
		{
			name:     "with overlap",
			existing: []string{"a", "b"},
			new:      []string{"b", "c"},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "empty existing",
			existing: []string{},
			new:      []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "empty new",
			existing: []string{"a", "b"},
			new:      []string{},
			want:     []string{"a", "b"},
		},
		{
			name:     "both empty",
			existing: []string{},
			new:      []string{},
			want:     []string{},
		},
		{
			name:     "filters empty strings",
			existing: []string{"a", ""},
			new:      []string{"", "b"},
			want:     []string{"a", "b"},
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
		name string
		val  any
		want []string
	}{
		{
			name: "interface slice",
			val:  []interface{}{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "string slice",
			val:  []string{"x", "y"},
			want: []string{"x", "y"},
		},
		{
			name: "mixed interface slice",
			val:  []interface{}{"a", 123, "b"},
			want: []string{"a", "b"},
		},
		{
			name: "filters empty strings",
			val:  []interface{}{"a", "", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "nil",
			val:  nil,
			want: nil,
		},
		{
			name: "wrong type",
			val:  "not a slice",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TagsFromAny(tt.val)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TagsFromAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
