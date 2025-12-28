package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapWithSteamMarkers(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "wrap non-empty content",
			content:  "Game description here",
			expected: "<!-- STEAM_DATA_START -->\nGame description here\n<!-- STEAM_DATA_END -->",
		},
		{
			name:     "wrap content with leading/trailing whitespace",
			content:  "  \n  Content with spaces  \n  ",
			expected: "<!-- STEAM_DATA_START -->\nContent with spaces\n<!-- STEAM_DATA_END -->",
		},
		{
			name:     "empty content returns empty string",
			content:  "",
			expected: "",
		},
		{
			name:     "whitespace-only content",
			content:  "   \n   ",
			expected: "<!-- STEAM_DATA_START -->\n\n<!-- STEAM_DATA_END -->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithSteamMarkers(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasSteamContentMarkers(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "has both markers",
			body:     "<!-- STEAM_DATA_START -->\nContent\n<!-- STEAM_DATA_END -->",
			expected: true,
		},
		{
			name:     "has both markers with surrounding text",
			body:     "Before\n<!-- STEAM_DATA_START -->\nContent\n<!-- STEAM_DATA_END -->\nAfter",
			expected: true,
		},
		{
			name:     "missing end marker",
			body:     "<!-- STEAM_DATA_START -->\nContent",
			expected: false,
		},
		{
			name:     "missing start marker",
			body:     "Content\n<!-- STEAM_DATA_END -->",
			expected: false,
		},
		{
			name:     "no markers",
			body:     "Just some content",
			expected: false,
		},
		{
			name:     "empty body",
			body:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSteamContentMarkers(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSteamContent(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		expectedContent string
		expectedFound   bool
	}{
		{
			name:            "extract content between markers",
			body:            "<!-- STEAM_DATA_START -->\nGame info here\n<!-- STEAM_DATA_END -->",
			expectedContent: "Game info here",
			expectedFound:   true,
		},
		{
			name:            "extract with surrounding text",
			body:            "Before\n<!-- STEAM_DATA_START -->\nGame info\n<!-- STEAM_DATA_END -->\nAfter",
			expectedContent: "Game info",
			expectedFound:   true,
		},
		{
			name:            "extract with whitespace",
			body:            "<!-- STEAM_DATA_START -->\n  \n  Game info  \n  \n<!-- STEAM_DATA_END -->",
			expectedContent: "Game info",
			expectedFound:   true,
		},
		{
			name:            "no markers returns empty and false",
			body:            "Just some content",
			expectedContent: "",
			expectedFound:   false,
		},
		{
			name:            "missing end marker",
			body:            "<!-- STEAM_DATA_START -->\nContent",
			expectedContent: "",
			expectedFound:   false,
		},
		{
			name:            "missing start marker",
			body:            "Content\n<!-- STEAM_DATA_END -->",
			expectedContent: "",
			expectedFound:   false,
		},
		{
			name:            "empty content between markers",
			body:            "<!-- STEAM_DATA_START --><!-- STEAM_DATA_END -->",
			expectedContent: "",
			expectedFound:   true,
		},
		{
			name:            "markers in wrong order",
			body:            "<!-- STEAM_DATA_END -->Content<!-- STEAM_DATA_START -->",
			expectedContent: "",
			expectedFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, found := GetSteamContent(tt.body)
			assert.Equal(t, tt.expectedContent, content)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}

func TestReplaceSteamContent(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		newContent string
		expected   string
	}{
		{
			name:       "replace content between markers",
			body:       "<!-- STEAM_DATA_START -->\nOld content\n<!-- STEAM_DATA_END -->",
			newContent: "New content",
			expected:   "<!-- STEAM_DATA_START -->\nNew content\n<!-- STEAM_DATA_END -->",
		},
		{
			name:       "replace with surrounding text preserved",
			body:       "Before\n\n<!-- STEAM_DATA_START -->\nOld content\n<!-- STEAM_DATA_END -->\n\nAfter",
			newContent: "New content",
			expected:   "Before\n\n<!-- STEAM_DATA_START -->\nNew content\n<!-- STEAM_DATA_END -->\nAfter",
		},
		{
			name:       "replace empty content",
			body:       "<!-- STEAM_DATA_START --><!-- STEAM_DATA_END -->",
			newContent: "New content",
			expected:   "<!-- STEAM_DATA_START -->\nNew content\n<!-- STEAM_DATA_END -->",
		},
		{
			name:       "no markers returns original body",
			body:       "Just some content",
			newContent: "New content",
			expected:   "Just some content",
		},
		{
			name:       "missing end marker returns original",
			body:       "<!-- STEAM_DATA_START -->\nContent",
			newContent: "New content",
			expected:   "<!-- STEAM_DATA_START -->\nContent",
		},
		{
			name:       "markers in wrong order returns original",
			body:       "<!-- STEAM_DATA_END -->Content<!-- STEAM_DATA_START -->",
			newContent: "New content",
			expected:   "<!-- STEAM_DATA_END -->Content<!-- STEAM_DATA_START -->",
		},
		{
			name:       "replace with multiline content",
			body:       "<!-- STEAM_DATA_START -->\nOld\n<!-- STEAM_DATA_END -->",
			newContent: "Line 1\nLine 2\nLine 3",
			expected:   "<!-- STEAM_DATA_START -->\nLine 1\nLine 2\nLine 3\n<!-- STEAM_DATA_END -->",
		},
		{
			name:       "replace preserves before text without extra newlines",
			body:       "Before<!-- STEAM_DATA_START -->\nOld\n<!-- STEAM_DATA_END -->",
			newContent: "New",
			expected:   "Before\n\n<!-- STEAM_DATA_START -->\nNew\n<!-- STEAM_DATA_END -->",
		},
		{
			name:       "replace preserves after text",
			body:       "<!-- STEAM_DATA_START -->\nOld\n<!-- STEAM_DATA_END -->After",
			newContent: "New",
			expected:   "<!-- STEAM_DATA_START -->\nNew\n<!-- STEAM_DATA_END -->\nAfter",
		},
		{
			name:       "replace with content containing whitespace",
			body:       "<!-- STEAM_DATA_START -->\nOld\n<!-- STEAM_DATA_END -->",
			newContent: "  New content  ",
			expected:   "<!-- STEAM_DATA_START -->\nNew content\n<!-- STEAM_DATA_END -->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceSteamContent(tt.body, tt.newContent)
			assert.Equal(t, tt.expected, result)
		})
	}
}
