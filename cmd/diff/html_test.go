package diff

import (
	"strings"
	"testing"
	"time"
)

func TestRenderDiffHTML(t *testing.T) {
	report := &diffReport{
		ImdbOnly: []diffItem{
			{
				Title:      "The Matrix",
				Year:       1999,
				ImdbID:     "tt0133093",
				ImdbURL:    "https://www.imdb.com/title/tt0133093/",
				ImdbRating: 9,
			},
		},
		LetterboxdOnly: []diffItem{
			{
				Title:            "Amélie",
				Year:             2001,
				LetterboxdURI:    "https://letterboxd.com/film/amelie/",
				LetterboxdRating: 4.2,
			},
		},
		Stats: diffStats{
			imdbOnlyCount:       1,
			letterboxdOnlyCount: 1,
			resolvedTitleYear:   5,
		},
		GeneratedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		MainDBPath:  "/path/to/hermes.db",
		CacheDBPath: "/path/to/cache.db",
	}

	html, err := renderDiffHTML(report)
	if err != nil {
		t.Fatalf("renderDiffHTML failed: %v", err)
	}

	content := string(html)

	// Check basic structure
	checks := []string{
		"<!DOCTYPE html>",
		"IMDb vs Letterboxd Diff",
		"2024-01-15",
		"The Matrix",
		"1999",
		"tt0133093",
		"Amélie",
		"2001",
		"letterboxd.com/film/amelie",
		"Add to Letterboxd",
		"Add to IMDb",
		"Auto-resolved: 5",
		"/path/to/hermes.db",
		"/path/to/cache.db",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("HTML output missing expected content: %q", check)
		}
	}
}

func TestRenderDiffHTMLWithFuzzyMatches(t *testing.T) {
	report := &diffReport{
		ImdbOnly: []diffItem{
			{
				Title:      "Solaris",
				Year:       2002,
				ImdbID:     "tt0307479",
				ImdbURL:    "https://www.imdb.com/title/tt0307479/",
				ImdbRating: 6,
				FuzzyMatches: []diffMatch{
					{
						Title:            "Solaris",
						Year:             2002,
						LetterboxdURI:    "https://letterboxd.com/film/solaris-2002/",
						LetterboxdRating: 3.5,
					},
				},
			},
		},
		LetterboxdOnly: nil,
		Stats: diffStats{
			imdbOnlyCount:     1,
			imdbOnlyWithFuzzy: 1,
		},
		GeneratedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		MainDBPath:  "/path/to/hermes.db",
	}

	html, err := renderDiffHTML(report)
	if err != nil {
		t.Fatalf("renderDiffHTML failed: %v", err)
	}

	content := string(html)

	checks := []string{
		"Possible matches:",
		"letterboxd.com/film/solaris-2002",
		"3.5/5",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("HTML output missing expected fuzzy match content: %q", check)
		}
	}
}

func TestRenderDiffHTMLEmpty(t *testing.T) {
	report := &diffReport{
		ImdbOnly:       nil,
		LetterboxdOnly: nil,
		Stats:          diffStats{},
		GeneratedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		MainDBPath:     "/path/to/hermes.db",
	}

	html, err := renderDiffHTML(report)
	if err != nil {
		t.Fatalf("renderDiffHTML failed: %v", err)
	}

	content := string(html)

	// Check empty state message
	if !strings.Contains(content, "All synced!") {
		t.Error("HTML output missing 'All synced!' message for empty lists")
	}
}

func TestLetterboxdSearchURL(t *testing.T) {
	tests := []struct {
		title    string
		year     int
		expected string
	}{
		{
			title:    "The Matrix",
			year:     1999,
			expected: "https://letterboxd.com/search/The%20Matrix%201999/",
		},
		{
			title:    "Amélie",
			year:     2001,
			expected: "https://letterboxd.com/search/Am%C3%A9lie%202001/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := letterboxdSearchURL(tt.title, tt.year)
			if result != tt.expected {
				t.Errorf("letterboxdSearchURL(%q, %d) = %q, want %q", tt.title, tt.year, result, tt.expected)
			}
		})
	}
}

func TestImdbSearchURL(t *testing.T) {
	tests := []struct {
		title    string
		year     int
		expected string
	}{
		{
			title:    "The Matrix",
			year:     1999,
			expected: "https://www.imdb.com/find/?q=The+Matrix+1999",
		},
		{
			title:    "Amélie",
			year:     2001,
			expected: "https://www.imdb.com/find/?q=Am%C3%A9lie+2001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := imdbSearchURL(tt.title, tt.year)
			if result != tt.expected {
				t.Errorf("imdbSearchURL(%q, %d) = %q, want %q", tt.title, tt.year, result, tt.expected)
			}
		})
	}
}

func TestRenderDiffHTMLWithConvertedRatings(t *testing.T) {
	report := &diffReport{
		ImdbOnly: []diffItem{
			{
				Title:      "The Matrix",
				Year:       1999,
				ImdbID:     "tt0133093",
				ImdbURL:    "https://www.imdb.com/title/tt0133093/",
				ImdbRating: 8,
			},
		},
		LetterboxdOnly: []diffItem{
			{
				Title:            "Amélie",
				Year:             2001,
				LetterboxdURI:    "https://letterboxd.com/film/amelie/",
				LetterboxdRating: 4.5, // User's personal rating (5-star scale)
			},
			{
				Title:            "Clueless",
				Year:             1995,
				LetterboxdURI:    "https://letterboxd.com/film/clueless/",
				LetterboxdRating: 6.9, // TMDB enrichment (should NOT be displayed)
			},
		},
		Stats: diffStats{
			imdbOnlyCount:       1,
			letterboxdOnlyCount: 2,
		},
		GeneratedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		MainDBPath:  "/path/to/hermes.db",
	}

	html, err := renderDiffHTML(report)
	if err != nil {
		t.Fatalf("renderDiffHTML failed: %v", err)
	}

	content := string(html)

	// Should be present
	shouldContain := []string{
		"8/10",      // Original IMDb rating
		"4.0/5",     // Converted to Letterboxd scale
		"4.0 stars", // In the button text
		"4.5/5",     // User's Letterboxd rating (5-scale)
		"9/10",      // Converted to IMDb scale (4.5 * 2 = 9)
	}

	for _, check := range shouldContain {
		if !strings.Contains(content, check) {
			t.Errorf("HTML output missing expected content: %q", check)
		}
	}

	// TMDB enrichment ratings (> 5) should NOT be displayed
	shouldNotContain := []string{
		"6.9/5",  // TMDB rating incorrectly shown as /5
		"6.9/10", // TMDB rating shown as /10
		"13/10",  // Incorrect conversion
		"14/10",  // Incorrect conversion
	}

	for _, check := range shouldNotContain {
		if strings.Contains(content, check) {
			t.Errorf("HTML output should NOT contain TMDB enrichment rating: %q", check)
		}
	}
}

func TestImdbToLB(t *testing.T) {
	tests := []struct {
		imdb     int
		expected float64
	}{
		{1, 0.5},
		{2, 0.5},
		{3, 1.0},
		{4, 1.5},
		{5, 2.0},
		{6, 3.0},
		{7, 3.5},
		{8, 4.0},
		{9, 4.5},
		{10, 5.0},
		// Edge cases: clamping
		{0, 0.5},   // below min, clamps to 1 -> 0.5
		{-5, 0.5},  // below min, clamps to 1 -> 0.5
		{11, 5.0},  // above max, clamps to 10 -> 5.0
		{100, 5.0}, // above max, clamps to 10 -> 5.0
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.imdb)), func(t *testing.T) {
			result := imdbToLB(tt.imdb)
			if result != tt.expected {
				t.Errorf("imdbToLB(%d) = %.1f, want %.1f", tt.imdb, result, tt.expected)
			}
		})
	}
}

func TestLbToImdb(t *testing.T) {
	tests := []struct {
		stars    float64
		expected int
	}{
		{0.5, 2},
		{1.0, 3},
		{1.5, 4},
		{2.0, 5},
		{2.5, 5},
		{3.0, 6},
		{3.5, 7},
		{4.0, 8},
		{4.5, 9},
		{5.0, 10},
		// Edge cases: rounding to 0.5
		{0.3, 2}, // rounds to 0.5 -> 2
		{0.7, 2}, // rounds to 0.5 -> 2
		{1.2, 3}, // rounds to 1.0 -> 3
		{1.3, 4}, // rounds to 1.5 -> 4
		{2.7, 5}, // rounds to 2.5 -> 5
		{3.2, 6}, // rounds to 3.0 -> 6
		{4.7, 9}, // rounds to 4.5 -> 9
		// Edge cases: clamping
		{0.0, 2},  // clamps to 0.5 -> 2
		{-1.0, 2}, // clamps to 0.5 -> 2
		{5.5, 10}, // clamps to 5.0 -> 10
		{6.0, 10}, // clamps to 5.0 -> 10
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := lbToImdb(tt.stars)
			if result != tt.expected {
				t.Errorf("lbToImdb(%.1f) = %d, want %d", tt.stars, result, tt.expected)
			}
		})
	}
}
