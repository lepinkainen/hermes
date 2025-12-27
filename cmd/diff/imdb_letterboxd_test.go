package diff

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiffMatchesByImdbID(t *testing.T) {
	imdbMovies := []imdbMovie{
		{ImdbID: "tt0133093", Title: "The Matrix", Year: 1999, MyRating: 9},
		{ImdbID: "tt0468569", Title: "The Dark Knight", Year: 2008, MyRating: 10},
		{ImdbID: "tt0111161", Title: "The Shawshank Redemption", Year: 1994, MyRating: 10},
	}
	letterboxdMovies := []letterboxdMovie{
		{Name: "Matrix", Year: 1999, ImdbID: "tt0133093", LetterboxdURI: "https://boxd.it/abc1"},
		{Name: "Dark Knight", Year: 2008, ImdbID: "tt0468569", LetterboxdURI: "https://boxd.it/abc2"},
		{Name: "Shawshank", Year: 1994, ImdbID: "tt0111161", LetterboxdURI: "https://boxd.it/abc3"},
	}

	imdbOnly, letterboxdOnly, resolved := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)

	require.Equal(t, 0, len(imdbOnly), "all IMDb movies should be matched by IMDb ID")
	require.Equal(t, 0, len(letterboxdOnly), "all Letterboxd movies should be matched by IMDb ID")
	require.Equal(t, 0, resolved, "no title+year resolution needed when IMDb IDs match")
}

func TestDiffReportsUnmatchedMovies(t *testing.T) {
	imdbMovies := []imdbMovie{
		{ImdbID: "tt0133093", Title: "The Matrix", Year: 1999, MyRating: 9},
		{ImdbID: "tt9999999", Title: "IMDb Only Movie", Year: 2020, MyRating: 7},
	}
	letterboxdMovies := []letterboxdMovie{
		{Name: "The Matrix", Year: 1999, ImdbID: "tt0133093", LetterboxdURI: "https://boxd.it/abc1"},
		{Name: "Letterboxd Only Movie", Year: 2021, LetterboxdURI: "https://boxd.it/abc2"},
	}

	imdbOnly, letterboxdOnly, _ := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)

	require.Equal(t, 1, len(imdbOnly), "should have one IMDb-only movie")
	require.Equal(t, "IMDb Only Movie", imdbOnly[0].Title)
	require.Equal(t, 2020, imdbOnly[0].Year)
	require.Equal(t, "tt9999999", imdbOnly[0].ImdbID)

	require.Equal(t, 1, len(letterboxdOnly), "should have one Letterboxd-only movie")
	require.Equal(t, "Letterboxd Only Movie", letterboxdOnly[0].Title)
	require.Equal(t, 2021, letterboxdOnly[0].Year)
	require.Equal(t, "https://boxd.it/abc2", letterboxdOnly[0].LetterboxdURI)
}

func TestDiffAutoResolvesTitleYearExactMatches(t *testing.T) {
	imdbMovies := []imdbMovie{
		{Title: "10 Cloverfield Lane", Year: 2016, MyRating: 8},
		{Title: "12 Strong", Year: 2018, MyRating: 7},
		{Title: "15 Minutes", Year: 2001, MyRating: 7},
		{Title: "17 Again", Year: 2009, MyRating: 7},
		{Title: "2 Guns", Year: 2013, MyRating: 6},
	}
	letterboxdMovies := []letterboxdMovie{
		{Name: "10 Cloverfield Lane", Year: 2016, LetterboxdURI: "https://boxd.it/aZiu"},
		{Name: "10 Cloverfield Lane", Year: 2016, LetterboxdURI: "https://boxd.it/aZiu/diary/1"},
		{Name: "12 Strong", Year: 2018, LetterboxdURI: "https://boxd.it/fdnc"},
		{Name: "15 Minutes", Year: 2001, LetterboxdURI: "https://boxd.it/26nS"},
		{Name: "17 Again", Year: 2009, LetterboxdURI: "https://boxd.it/1Jnq"},
		{Name: "2 Guns", Year: 2013, LetterboxdURI: "https://boxd.it/4oAk"},
	}

	imdbOnly, letterboxdOnly, resolved := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)

	if resolved != 5 {
		t.Fatalf("expected 5 auto-resolved matches, got %d", resolved)
	}
	if len(imdbOnly) != 0 {
		t.Fatalf("expected no imdb-only items, got %d", len(imdbOnly))
	}
	if len(letterboxdOnly) != 0 {
		t.Fatalf("expected no letterboxd-only items, got %d", len(letterboxdOnly))
	}
}

func TestDiffAutoResolvesTitleYearWithMultipleEntries(t *testing.T) {
	imdbMovies := []imdbMovie{
		{ImdbID: "tt1179933", Title: "10 Cloverfield Lane", Year: 2016, MyRating: 8},
		{ImdbID: "tt1179933-alt", Title: "10 Cloverfield Lane", Year: 2016, MyRating: 8},
	}
	letterboxdMovies := []letterboxdMovie{
		{Name: "10 Cloverfield Lane", Year: 2016, LetterboxdURI: "https://boxd.it/aZiu"},
		{Name: "10 Cloverfield Lane", Year: 2016, LetterboxdURI: "https://boxd.it/aZiu/diary/1"},
		{Name: "10 Cloverfield Lane", Year: 2016, LetterboxdURI: "https://boxd.it/aZiu/diary/2"},
	}

	imdbOnly, letterboxdOnly, resolved := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)

	if resolved != 1 {
		t.Fatalf("expected 1 auto-resolved title+year group, got %d", resolved)
	}
	if len(imdbOnly) != 0 {
		t.Fatalf("expected no imdb-only items, got %d", len(imdbOnly))
	}
	if len(letterboxdOnly) != 0 {
		t.Fatalf("expected no letterboxd-only items, got %d", len(letterboxdOnly))
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"The Matrix", "the matrix"},
		{"  The Matrix  ", "the matrix"},
		{"The Matrix: Reloaded", "the matrix reloaded"},
		{"Ocean's Eleven", "ocean s eleven"},
		{"Amélie", "amélie"},
		{"10 Cloverfield Lane", "10 cloverfield lane"},
		{"", ""},
		{"   ", ""},
		{"A-B-C", "a b c"},
		{"Hello...World", "hello world"},
		{"Movie (2020)", "movie 2020"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTitle(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyLetterboxdMappings(t *testing.T) {
	movies := []letterboxdMovie{
		{Name: "Movie A", Year: 2020, LetterboxdURI: "https://boxd.it/aaa", ImdbID: ""},
		{Name: "Movie B", Year: 2021, LetterboxdURI: "https://boxd.it/bbb", ImdbID: "tt1234567"},
		{Name: "Movie C", Year: 2022, LetterboxdURI: "https://boxd.it/ccc", ImdbID: ""},
		{Name: "Movie D", Year: 2023, LetterboxdURI: "", ImdbID: ""},
	}

	mapping := map[string]string{
		"https://boxd.it/aaa": "tt0000001",
		"https://boxd.it/ccc": "tt0000003",
		"https://boxd.it/ddd": "tt0000004",
	}

	applyLetterboxdMappings(movies, mapping)

	require.Equal(t, "tt0000001", movies[0].ImdbID, "should apply mapping for Movie A")
	require.Equal(t, "tt1234567", movies[1].ImdbID, "should not overwrite existing IMDb ID")
	require.Equal(t, "tt0000003", movies[2].ImdbID, "should apply mapping for Movie C")
	require.Equal(t, "", movies[3].ImdbID, "should not apply mapping when URI is empty")
}

func TestApplyLetterboxdMappingsEmpty(t *testing.T) {
	movies := []letterboxdMovie{
		{Name: "Movie A", Year: 2020, LetterboxdURI: "https://boxd.it/aaa", ImdbID: ""},
	}

	applyLetterboxdMappings(movies, nil)
	require.Equal(t, "", movies[0].ImdbID, "should handle nil mapping")

	applyLetterboxdMappings(movies, map[string]string{})
	require.Equal(t, "", movies[0].ImdbID, "should handle empty mapping")
}

func TestDiffWithFuzzyMatches(t *testing.T) {
	// Test case: IMDb has "The Matrix" (1999) but Letterboxd has "The Matrix" (2000)
	// They won't match by IMDb ID (Letterboxd has none) or title+year (different years)
	// But the normalized title matches, so fuzzy matches should be suggested
	imdbMovies := []imdbMovie{
		{ImdbID: "tt0000001", Title: "The Matrix", Year: 1999, MyRating: 9},
	}
	letterboxdMovies := []letterboxdMovie{
		// Same title but different year - will be unmatched but should show as fuzzy match
		{Name: "The Matrix", Year: 2000, LetterboxdURI: "https://boxd.it/matrix1"},
	}

	imdbOnly, letterboxdOnly, _ := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)

	// Both should be in their respective "only" lists since years don't match
	require.Equal(t, 1, len(imdbOnly), "IMDb movie should be unmatched")
	require.Equal(t, 1, len(letterboxdOnly), "Letterboxd movie should be unmatched")

	// No fuzzy matches since title+year keys are different (different years)
	// The fuzzy matching uses the same title+year key, so different years won't match
	require.Equal(t, 0, len(imdbOnly[0].FuzzyMatches), "no fuzzy matches when years differ")
	require.Equal(t, 0, len(letterboxdOnly[0].FuzzyMatches), "no fuzzy matches when years differ")
}

func TestDiffWithFuzzyMatchesSameYear(t *testing.T) {
	// Test case: Original title matches but main title doesn't
	// IMDb has "Matrix" as title and "The Matrix" as original title
	// Letterboxd has "The Matrix" - should fuzzy match via original title
	imdbMovies := []imdbMovie{
		{ImdbID: "tt0000001", Title: "Matrix Reloaded", OriginalTitle: "The Matrix Reloaded", Year: 2003, MyRating: 8},
	}
	letterboxdMovies := []letterboxdMovie{
		{Name: "The Matrix Reloaded", Year: 2003, LetterboxdURI: "https://boxd.it/matrix1"},
	}

	imdbOnly, letterboxdOnly, resolved := diffIMDbLetterboxd(imdbMovies, letterboxdMovies)

	// Should resolve via title+year match on OriginalTitle
	require.Equal(t, 1, resolved, "should auto-resolve via original title")
	require.Equal(t, 0, len(imdbOnly), "should be matched")
	require.Equal(t, 0, len(letterboxdOnly), "should be matched")
}

func TestTitleYearKey(t *testing.T) {
	tests := []struct {
		title    string
		fallback string
		year     int
		expected string
	}{
		{"The Matrix", "", 1999, "the matrix|1999"},
		{"", "Original Title", 2000, "original title|2000"},
		{"", "", 2000, ""},
		{"Title", "", 0, ""},
		{"Title", "", -1, ""},
		{"  Spaced  ", "", 2020, "spaced|2020"},
	}

	for _, tt := range tests {
		t.Run(tt.title+tt.fallback, func(t *testing.T) {
			result := titleYearKey(tt.title, tt.fallback, tt.year)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDisplayTitle(t *testing.T) {
	tests := []struct {
		primary    string
		fallback   string
		identifier string
		expected   string
	}{
		{"The Matrix", "Matrix", "tt0133093", "The Matrix"},
		{"", "Matrix", "tt0133093", "Matrix"},
		{"", "", "tt0133093", "tt0133093"},
		{"", "", "", "Untitled"},
		{"  ", "  ", "", "Untitled"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := displayTitle(tt.primary, tt.fallback, tt.identifier)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestImdbURL(t *testing.T) {
	tests := []struct {
		imdbID   string
		existing string
		expected string
	}{
		{"tt0133093", "", "https://www.imdb.com/title/tt0133093/"},
		{"tt0133093", "https://custom.url/", "https://custom.url/"},
		{"", "", ""},
		{"", "https://custom.url/", "https://custom.url/"},
	}

	for _, tt := range tests {
		t.Run(tt.imdbID, func(t *testing.T) {
			result := imdbURL(tt.imdbID, tt.existing)
			require.Equal(t, tt.expected, result)
		})
	}
}
