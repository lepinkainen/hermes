package tmdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchResultDisplayTitle(t *testing.T) {
	assert.Equal(t, "Movie Title", SearchResult{Title: "Movie Title", Name: "Alt"}.DisplayTitle())
	assert.Equal(t, "Show Name", SearchResult{Name: "Show Name"}.DisplayTitle())
}

func TestSearchResultYearInt(t *testing.T) {
	tests := []struct {
		name   string
		result SearchResult
		want   int
	}{
		{
			name:   "movie year",
			result: SearchResult{ReleaseDate: "1999-03-31", MediaType: "movie"},
			want:   1999,
		},
		{
			name:   "tv year",
			result: SearchResult{FirstAirDate: "2008-01-20", MediaType: "tv"},
			want:   2008,
		},
		{
			name:   "invalid date",
			result: SearchResult{ReleaseDate: "bad", MediaType: "movie"},
			want:   0,
		},
		{
			name:   "missing date",
			result: SearchResult{MediaType: "movie"},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.YearInt())
		})
	}
}
