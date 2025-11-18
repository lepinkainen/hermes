package enrichment

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/tmdb"
)

func TestFindExactMatch(t *testing.T) {
	results := []tmdb.SearchResult{
		{
			ID:          1,
			Title:       "Guardians of the Galaxy Vol. 2",
			ReleaseDate: "2017-04-19",
			VoteCount:   22400,
		},
		{
			ID:          2,
			Title:       "Guardians",
			ReleaseDate: "2017-02-23",
			VoteCount:   794,
		},
		{
			ID:          3,
			Title:       "Naruto the Movie: Guardians of the Crescent Moon Kingdom",
			ReleaseDate: "2006-08-05",
			VoteCount:   468,
		},
	}

	tests := []struct {
		name    string
		title   string
		year    int
		wantID  int
		wantNil bool
	}{
		{
			name:   "exact match",
			title:  "Guardians",
			year:   2017,
			wantID: 2,
		},
		{
			name:   "case insensitive match",
			title:  "GUARDIANS",
			year:   2017,
			wantID: 2,
		},
		{
			name:   "with whitespace",
			title:  "  Guardians  ",
			year:   2017,
			wantID: 2,
		},
		{
			name:    "wrong year",
			title:   "Guardians",
			year:    2018,
			wantNil: true,
		},
		{
			name:    "partial title match",
			title:   "Guardians of the Galaxy",
			year:    2017,
			wantNil: true,
		},
		{
			name:    "no match",
			title:   "Avengers",
			year:    2017,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findExactMatch(results, tt.title, tt.year)
			if tt.wantNil {
				if result != nil {
					t.Errorf("findExactMatch() = %v, want nil", result.ID)
				}
			} else {
				if result == nil {
					t.Errorf("findExactMatch() = nil, want ID %d", tt.wantID)
				} else if result.ID != tt.wantID {
					t.Errorf("findExactMatch() ID = %d, want %d", result.ID, tt.wantID)
				}
			}
		})
	}
}

func TestFindExactMatch_TVShows(t *testing.T) {
	results := []tmdb.SearchResult{
		{
			ID:           1,
			Name:         "The Office",
			FirstAirDate: "2005-03-24",
			VoteCount:    3000,
		},
		{
			ID:          2,
			Name:        "Office Space",
			ReleaseDate: "1999-02-19",
			VoteCount:   2000,
		},
	}

	result := findExactMatch(results, "The Office", 2005)
	if result == nil {
		t.Error("findExactMatch() = nil, want TV show match")
	} else if result.ID != 1 {
		t.Errorf("findExactMatch() ID = %d, want 1", result.ID)
	}
}

func TestFindExactMatch_EmptyResults(t *testing.T) {
	result := findExactMatch([]tmdb.SearchResult{}, "Test", 2020)
	if result != nil {
		t.Errorf("findExactMatch() = %v, want nil for empty results", result)
	}
}

func TestFindExactMatch_MultipleExactMatches(t *testing.T) {
	// Three different "Pocahontas" movies released in 1995
	results := []tmdb.SearchResult{
		{
			ID:          1,
			Title:       "Pocahontas",
			ReleaseDate: "1995-06-23",
			VoteCount:   5000,
		},
		{
			ID:          2,
			Title:       "Pocahontas",
			ReleaseDate: "1995-01-01",
			VoteCount:   100,
		},
		{
			ID:          3,
			Title:       "Pocahontas: The Legend",
			ReleaseDate: "1995-09-01",
			VoteCount:   200,
		},
	}

	// Should return nil because there are multiple exact matches for "Pocahontas" (1995)
	result := findExactMatch(results, "Pocahontas", 1995)
	if result != nil {
		t.Errorf("findExactMatch() = %d, want nil for multiple exact matches", result.ID)
	}

	// But "Pocahontas: The Legend" should still match uniquely
	result = findExactMatch(results, "Pocahontas: The Legend", 1995)
	if result == nil {
		t.Error("findExactMatch() = nil, want unique match for 'Pocahontas: The Legend'")
	} else if result.ID != 3 {
		t.Errorf("findExactMatch() ID = %d, want 3", result.ID)
	}
}
