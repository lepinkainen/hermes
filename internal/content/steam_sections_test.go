package content

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSteamContent_NilDetails(t *testing.T) {
	result := BuildSteamContent(nil, []string{"info", "description"}, "")
	assert.Equal(t, "", result)
}

func TestBuildSteamContent_EmptySections(t *testing.T) {
	details := &SteamGameDetails{
		AppID:     123,
		Name:      "Test Game",
		ShortDesc: "A short description",
	}

	result := BuildSteamContent(details, []string{}, "")

	// Empty sections should default to ["info", "description"]
	assert.Contains(t, result, "## Game Info")
	assert.Contains(t, result, "## Description")
}

func TestBuildSteamContent_WithCover(t *testing.T) {
	details := &SteamGameDetails{
		AppID:     123,
		Name:      "Test Game",
		ShortDesc: "A test game",
	}

	result := BuildSteamContent(details, []string{"info"}, "test-cover.jpg")

	// Should start with cover image
	assert.True(t, strings.HasPrefix(result, "![[test-cover.jpg|500]]"))
}

func TestBuildSteamContent_InfoSection(t *testing.T) {
	details := &SteamGameDetails{
		AppID:       440,
		Name:        "Team Fortress 2",
		Developers:  []string{"Valve"},
		Publishers:  []string{"Valve"},
		ReleaseDate: "Oct 10, 2007",
		ComingSoon:  false,
		Genres:      []string{"Action", "Free to Play"},
		Categories:  []string{"Multi-player", "Steam Achievements", "Steam Trading Cards"},
		Metacritic: struct {
			Score int
			URL   string
		}{
			Score: 92,
			URL:   "https://www.metacritic.com/game/pc/team-fortress-2",
		},
	}

	result := BuildSteamContent(details, []string{"info"}, "")

	assert.Contains(t, result, "## Game Info")
	assert.Contains(t, result, "| **Developer** | Valve |")
	assert.Contains(t, result, "| **Publisher** | Valve |")
	assert.Contains(t, result, "| **Release Date** | Oct 10, 2007 |")
	assert.Contains(t, result, "| **Genres** | Action, Free to Play |")
	assert.Contains(t, result, "| **Features** | Multi-player, Steam Achievements, Steam Trading Cards |")
	assert.Contains(t, result, "| **Metacritic** | [92/100](https://www.metacritic.com/game/pc/team-fortress-2) |")
	assert.Contains(t, result, "| **Steam** | [store.steampowered.com/app/440](https://store.steampowered.com/app/440) |")
}

func TestBuildSteamContent_InfoSection_ComingSoon(t *testing.T) {
	details := &SteamGameDetails{
		AppID:       999,
		ReleaseDate: "Q4 2025",
		ComingSoon:  true,
	}

	result := BuildSteamContent(details, []string{"info"}, "")

	assert.Contains(t, result, "| **Release Date** | Q4 2025 (Coming Soon) |")
}

func TestBuildSteamContent_InfoSection_MetacriticNoURL(t *testing.T) {
	details := &SteamGameDetails{
		AppID: 123,
		Metacritic: struct {
			Score int
			URL   string
		}{
			Score: 85,
			URL:   "",
		},
	}

	result := BuildSteamContent(details, []string{"info"}, "")

	assert.Contains(t, result, "| **Metacritic** | 85/100 |")
}

func TestBuildSteamContent_InfoSection_TooManyCategories(t *testing.T) {
	details := &SteamGameDetails{
		AppID:      123,
		Categories: []string{"Cat1", "Cat2", "Cat3", "Cat4", "Cat5", "Cat6", "Cat7"},
	}

	result := BuildSteamContent(details, []string{"info"}, "")

	// Should limit to 5 categories
	assert.Contains(t, result, "| **Features** | Cat1, Cat2, Cat3, Cat4, Cat5 |")
	assert.NotContains(t, result, "Cat6")
	assert.NotContains(t, result, "Cat7")
}

func TestBuildSteamContent_DescriptionSection_ShortDesc(t *testing.T) {
	details := &SteamGameDetails{
		AppID:     123,
		ShortDesc: "This is a short description",
	}

	result := BuildSteamContent(details, []string{"description"}, "")

	assert.Contains(t, result, "## Description")
	assert.Contains(t, result, "This is a short description")
}

func TestBuildSteamContent_DescriptionSection_LongDescFallback(t *testing.T) {
	details := &SteamGameDetails{
		AppID:       123,
		ShortDesc:   "",
		Description: "This is a long description",
	}

	result := BuildSteamContent(details, []string{"description"}, "")

	assert.Contains(t, result, "## Description")
	assert.Contains(t, result, "This is a long description")
}

func TestBuildSteamContent_DescriptionSection_EmptyReturnsEmpty(t *testing.T) {
	details := &SteamGameDetails{
		AppID:       123,
		ShortDesc:   "",
		Description: "",
	}

	result := BuildSteamContent(details, []string{"description"}, "")

	// Should not contain description section
	assert.NotContains(t, result, "## Description")
}

func TestBuildSteamContent_DescriptionSection_HTMLCleaning(t *testing.T) {
	details := &SteamGameDetails{
		AppID:     123,
		ShortDesc: "Text with <b>bold</b> and <i>italic</i> tags<br>and line breaks",
	}

	result := BuildSteamContent(details, []string{"description"}, "")

	assert.Contains(t, result, "Text with **bold** and _italic_ tags\nand line breaks")
}

func TestBuildSteamContent_BothSections(t *testing.T) {
	details := &SteamGameDetails{
		AppID:       123,
		Name:        "Test Game",
		Developers:  []string{"Dev Studio"},
		ShortDesc:   "A game description",
		ReleaseDate: "2024",
	}

	result := BuildSteamContent(details, []string{"info", "description"}, "")

	assert.Contains(t, result, "## Game Info")
	assert.Contains(t, result, "## Description")

	// Sections should be separated by double newline
	parts := strings.Split(result, "\n\n")
	assert.True(t, len(parts) >= 2)
}

func TestBuildSteamContent_OnlyCoverNoSections(t *testing.T) {
	details := &SteamGameDetails{
		AppID: 123,
	}

	// Empty string sections with cover only - should use defaults
	result := BuildSteamContent(details, []string{}, "cover.jpg")

	assert.True(t, strings.HasPrefix(result, "![[cover.jpg|500]]"))
}

func TestCleanHTMLTags_BasicTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold tags",
			input:    "Text with <b>bold</b> and <strong>strong</strong>",
			expected: "Text with **bold** and **strong**",
		},
		{
			name:     "italic tags",
			input:    "Text with <i>italic</i> and <em>emphasis</em>",
			expected: "Text with _italic_ and _emphasis_",
		},
		{
			name:     "line breaks",
			input:    "Line 1<br>Line 2<br/>Line 3<br />Line 4",
			expected: "Line 1\nLine 2\nLine 3\nLine 4",
		},
		{
			name:     "paragraph tags",
			input:    "<p>Paragraph 1</p><p>Paragraph 2</p>",
			expected: "Paragraph 1\nParagraph 2",
		},
		{
			name:     "list tags",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: "- Item 1\n- Item 2",
		},
		{
			name:     "mixed tags",
			input:    "<p>Text with <b>bold</b> and <i>italic</i><br>on multiple lines</p>",
			expected: "Text with **bold** and _italic_\non multiple lines",
		},
		{
			name:     "multiple newlines collapsed",
			input:    "Text\n\n\n\nwith\n\n\nmany\n\n\n\nnewlines",
			expected: "Text\n\nwith\n\nmany\n\nnewlines",
		},
		{
			name:     "no tags returns trimmed",
			input:    "  Plain text  ",
			expected: "Plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanHTMLTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSteamContent_MultipleDevelopers(t *testing.T) {
	details := &SteamGameDetails{
		AppID:      123,
		Developers: []string{"Studio A", "Studio B", "Studio C"},
		Publishers: []string{"Publisher X", "Publisher Y"},
	}

	result := BuildSteamContent(details, []string{"info"}, "")

	assert.Contains(t, result, "| **Developer** | Studio A, Studio B, Studio C |")
	assert.Contains(t, result, "| **Publisher** | Publisher X, Publisher Y |")
}

func TestBuildSteamContent_MinimalDetails(t *testing.T) {
	details := &SteamGameDetails{
		AppID: 123,
	}

	result := BuildSteamContent(details, []string{"info", "description"}, "")

	// Should still generate info section with at least Steam link
	assert.Contains(t, result, "## Game Info")
	assert.Contains(t, result, "| **Steam** | [store.steampowered.com/app/123]")

	// Description should be omitted if empty
	assert.NotContains(t, result, "## Description")
}
