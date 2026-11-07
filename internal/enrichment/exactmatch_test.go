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

func TestSortExactTitleFirst(t *testing.T) {
	// The real 7-result "The Collector" survivor set: popularity-ranked TMDB
	// results bury the exact title match ("The Collector", ID 21407) behind
	// substring matches like "The Debt Collector".
	results := []tmdb.SearchResult{
		{ID: 1630409, Title: "The Debt Collector", ReleaseDate: "2026-07-20", VoteCount: 133, MediaType: "movie"},
		{ID: 513691, Title: "The Debt Collector", ReleaseDate: "2018-06-05", VoteCount: 261, MediaType: "movie"},
		{ID: 531499, Title: "The Tax Collector", ReleaseDate: "2020-08-07", VoteCount: 568, MediaType: "movie"},
		{ID: 9481, Title: "The Bone Collector", ReleaseDate: "1999-11-04", VoteCount: 3987, MediaType: "movie"},
		{ID: 21407, Title: "The Collector", ReleaseDate: "2009-07-09", VoteCount: 1726, MediaType: "movie"},
		{ID: 42740, Title: "The Collector", ReleaseDate: "1965-06-17", VoteCount: 235, MediaType: "movie"},
		{ID: 95272, Name: "Lincoln Rhyme: Hunt for the Bone Collector", FirstAirDate: "2020-01-10", VoteCount: 147, MediaType: "tv"},
	}

	sorted := sortExactTitleFirst(results, "The Collector")
	if len(sorted) != len(results) {
		t.Fatalf("sortExactTitleFirst() returned %d results, want %d", len(sorted), len(results))
	}
	if sorted[0].ID != 21407 {
		t.Errorf("sortExactTitleFirst() first ID = %d, want 21407", sorted[0].ID)
	}
	if sorted[1].ID != 42740 {
		t.Errorf("sortExactTitleFirst() second ID = %d, want 42740", sorted[1].ID)
	}
}

func TestSortExactTitleFirst_NoExactMatch(t *testing.T) {
	results := []tmdb.SearchResult{
		{ID: 1, Title: "The Debt Collector", VoteCount: 261},
		{ID: 2, Title: "The Tax Collector", VoteCount: 568},
	}

	sorted := sortExactTitleFirst(results, "The Collector")
	if len(sorted) != len(results) || sorted[0].ID != 1 || sorted[1].ID != 2 {
		t.Errorf("sortExactTitleFirst() = %v, want unchanged input order", sorted)
	}
}
