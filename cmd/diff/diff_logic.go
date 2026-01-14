package diff

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func diffIMDbLetterboxd(imdbMovies []imdbMovie, letterboxdMovies []letterboxdMovie) ([]diffItem, []diffItem, int) {
	imdbByID := make(map[string]imdbMovie)
	imdbByKey := make(map[string][]imdbMovie)
	for _, movie := range imdbMovies {
		if movie.ImdbID != "" {
			imdbByID[movie.ImdbID] = movie
		}
		addImdbIndex(imdbByKey, movie.Title, movie)
		if movie.OriginalTitle != "" && movie.OriginalTitle != movie.Title {
			addImdbIndex(imdbByKey, movie.OriginalTitle, movie)
		}
	}

	letterboxdByID := make(map[string]letterboxdMovie)
	letterboxdByKey := make(map[string][]letterboxdMovie)
	for _, movie := range letterboxdMovies {
		if movie.ImdbID != "" {
			letterboxdByID[movie.ImdbID] = movie
		}
		addLetterboxdIndex(letterboxdByKey, movie.Name, movie)
	}

	matchedImdb := map[string]bool{}
	matchedLetterboxd := map[string]bool{}

	for imdbID, imdbMovie := range imdbByID {
		letterboxdMovie, ok := letterboxdByID[imdbID]
		if !ok {
			continue
		}
		matchedImdb[imdbItemKey(imdbMovie)] = true
		matchedLetterboxd[letterboxdItemKey(letterboxdMovie)] = true
	}

	resolvedTitleYear := 0
	for key, imdbMatches := range imdbByKey {
		letterboxdMatches, ok := letterboxdByKey[key]
		if !ok {
			continue
		}
		imdbUnique := uniqueImdbMovies(imdbMatches)
		letterboxdUnique := uniqueLetterboxdMovies(letterboxdMatches)
		if len(imdbUnique) == 0 || len(letterboxdUnique) == 0 {
			continue
		}
		resolvedTitleYear++
		for _, movie := range imdbUnique {
			imdbKey := imdbItemKey(movie)
			if imdbKey != "" {
				matchedImdb[imdbKey] = true
			}
		}
		for _, movie := range letterboxdUnique {
			letterboxdKey := letterboxdItemKey(movie)
			if letterboxdKey != "" {
				matchedLetterboxd[letterboxdKey] = true
			}
		}
	}

	imdbOnly := []diffItem{}
	for _, movie := range imdbMovies {
		if matchedImdb[imdbItemKey(movie)] {
			continue
		}
		item := diffItem{
			Title:      displayTitle(movie.Title, movie.OriginalTitle, movie.ImdbID),
			Year:       movie.Year,
			ImdbID:     movie.ImdbID,
			ImdbURL:    imdbURL(movie.ImdbID, movie.URL),
			ImdbRating: movie.MyRating,
		}
		key := titleYearKey(movie.Title, movie.OriginalTitle, movie.Year)
		if key != "" {
			item.FuzzyMatches = buildLetterboxdMatches(letterboxdByKey[key])
		}
		imdbOnly = append(imdbOnly, item)
	}

	letterboxdOnly := []diffItem{}
	for _, movie := range letterboxdMovies {
		if matchedLetterboxd[letterboxdItemKey(movie)] {
			continue
		}
		item := diffItem{
			Title:            displayTitle(movie.Name, "", movie.LetterboxdID),
			Year:             movie.Year,
			ImdbID:           movie.ImdbID,
			ImdbURL:          imdbURL(movie.ImdbID, ""),
			LetterboxdURI:    movie.LetterboxdURI,
			LetterboxdRating: movie.Rating,
		}
		key := titleYearKey(movie.Name, "", movie.Year)
		if key != "" {
			item.FuzzyMatches = buildIMDbMatches(imdbByKey[key])
		}
		letterboxdOnly = append(letterboxdOnly, item)
	}

	sortDiffItems(imdbOnly)
	sortDiffItems(letterboxdOnly)
	return imdbOnly, letterboxdOnly, resolvedTitleYear
}

func imdbItemKey(movie imdbMovie) string {
	if movie.ImdbID != "" {
		return "imdb:" + movie.ImdbID
	}
	key := titleYearKey(movie.Title, movie.OriginalTitle, movie.Year)
	if key != "" {
		return "title:" + key
	}
	return ""
}

func letterboxdItemKey(movie letterboxdMovie) string {
	if movie.ImdbID != "" {
		return "imdb:" + movie.ImdbID
	}
	if movie.LetterboxdURI != "" {
		return "uri:" + movie.LetterboxdURI
	}
	if movie.LetterboxdID != "" {
		return "id:" + movie.LetterboxdID
	}
	key := titleYearKey(movie.Name, "", movie.Year)
	if key != "" {
		return "title:" + key
	}
	return ""
}

func uniqueImdbMovies(movies []imdbMovie) []imdbMovie {
	seen := map[string]bool{}
	result := make([]imdbMovie, 0, len(movies))
	for _, movie := range movies {
		key := imdbItemKey(movie)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, movie)
	}
	return result
}

func uniqueLetterboxdMovies(movies []letterboxdMovie) []letterboxdMovie {
	seen := map[string]bool{}
	result := make([]letterboxdMovie, 0, len(movies))
	for _, movie := range movies {
		key := letterboxdItemKey(movie)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, movie)
	}
	return result
}

func addImdbIndex(index map[string][]imdbMovie, title string, movie imdbMovie) {
	key := titleYearKey(title, "", movie.Year)
	if key == "" {
		return
	}
	index[key] = append(index[key], movie)
}

func addLetterboxdIndex(index map[string][]letterboxdMovie, title string, movie letterboxdMovie) {
	key := titleYearKey(title, "", movie.Year)
	if key == "" {
		return
	}
	index[key] = append(index[key], movie)
}

func titleYearKey(title, fallback string, year int) string {
	candidate := strings.TrimSpace(title)
	if candidate == "" {
		candidate = strings.TrimSpace(fallback)
	}
	if candidate == "" || year <= 0 {
		return ""
	}
	normalized := normalizeTitle(candidate)
	if normalized == "" {
		return ""
	}
	return fmt.Sprintf("%s|%d", normalized, year)
}

func normalizeTitle(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))
	if lower == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(lower))
	lastSpace := false
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func displayTitle(primary, fallback, identifier string) string {
	primary = strings.TrimSpace(primary)
	if primary != "" {
		return primary
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	if identifier != "" {
		return identifier
	}
	return "Untitled"
}

func imdbURL(imdbID, existing string) string {
	if existing != "" {
		return existing
	}
	if imdbID == "" {
		return ""
	}
	return fmt.Sprintf("https://www.imdb.com/title/%s/", imdbID)
}

func buildLetterboxdMatches(matches []letterboxdMovie) []diffMatch {
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	result := make([]diffMatch, 0, len(matches))
	for _, match := range matches {
		key := match.LetterboxdURI
		if key == "" {
			key = match.LetterboxdID
		}
		if key != "" && seen[key] {
			continue
		}
		if key != "" {
			seen[key] = true
		}
		result = append(result, diffMatch{
			Title:            displayTitle(match.Name, "", match.LetterboxdID),
			Year:             match.Year,
			ImdbID:           match.ImdbID,
			ImdbURL:          imdbURL(match.ImdbID, ""),
			LetterboxdURI:    match.LetterboxdURI,
			LetterboxdRating: match.Rating,
		})
	}
	sortMatches(result)
	return result
}

func buildIMDbMatches(matches []imdbMovie) []diffMatch {
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	result := make([]diffMatch, 0, len(matches))
	for _, match := range matches {
		key := match.ImdbID
		if key != "" && seen[key] {
			continue
		}
		if key != "" {
			seen[key] = true
		}
		result = append(result, diffMatch{
			Title:      displayTitle(match.Title, match.OriginalTitle, match.ImdbID),
			Year:       match.Year,
			ImdbID:     match.ImdbID,
			ImdbURL:    imdbURL(match.ImdbID, match.URL),
			ImdbRating: match.MyRating,
		})
	}
	sortMatches(result)
	return result
}

func sortDiffItems(items []diffItem) {
	sort.Slice(items, func(i, j int) bool {
		iTitle := strings.ToLower(items[i].Title)
		jTitle := strings.ToLower(items[j].Title)
		if iTitle == jTitle {
			return items[i].Year < items[j].Year
		}
		return iTitle < jTitle
	})
}

func sortMatches(matches []diffMatch) {
	sort.Slice(matches, func(i, j int) bool {
		iTitle := strings.ToLower(matches[i].Title)
		jTitle := strings.ToLower(matches[j].Title)
		if iTitle == jTitle {
			return matches[i].Year < matches[j].Year
		}
		return iTitle < jTitle
	})
}
