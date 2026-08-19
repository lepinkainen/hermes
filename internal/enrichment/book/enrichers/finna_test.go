package enrichers

import (
	"encoding/json"
	"net/http"
	"sort"
	"testing"

	"github.com/lepinkainen/hermes/internal/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinnaEnricher_Name(t *testing.T) {
	e := NewFinnaEnricher()
	assert.Equal(t, "Finna", e.Name())
}

func TestFinnaEnricher_Priority(t *testing.T) {
	e := NewFinnaEnricher()
	assert.Equal(t, 4, e.Priority())
}

func TestFlattenFinnaSubjects(t *testing.T) {
	tests := []struct {
		name string
		in   [][]string
		want []string
	}{
		{
			name: "nil",
			in:   nil,
			want: nil,
		},
		{
			name: "trims trailing periods and whitespace",
			in:   [][]string{{"surrealismi."}, {"  taidehistoria.  "}},
			want: []string{"surrealismi", "taidehistoria"},
		},
		{
			name: "flattens nested groups and dedupes",
			in: [][]string{
				{"fanit"},
				{"Fans (Persons)"},
				{"Fans (Persons)", "United States."},
				{"fanit"},
			},
			want: []string{"fanit", "Fans (Persons)", "United States"},
		},
		{
			name: "drops empty strings",
			in:   [][]string{{"", "  ", "kirjat"}},
			want: []string{"kirjat"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, flattenFinnaSubjects(tc.in))
		})
	}
}

func TestFinnaAuthors_UnmarshalJSON_EmptySubmaps(t *testing.T) {
	// Finna serializes empty author sub-maps as JSON arrays, not objects.
	raw := []byte(`{"primary": [], "secondary": [], "corporate": []}`)
	var a finnaAuthors
	require.NoError(t, json.Unmarshal(raw, &a))
	assert.Nil(t, a.Primary)
}

func TestFinnaAuthors_UnmarshalJSON_PopulatedPrimary(t *testing.T) {
	raw := []byte(`{
		"primary": {
			"Kyrö, Tuomas": {"role": ["kirjoittaja"]},
			"  Padded, Author ": {"role": ["-"]}
		},
		"secondary": {"Litja, Antti, lukija": {"role": ["lukija"]}},
		"corporate": {"WSOY, kustantaja": {"role": ["kustantaja"]}}
	}`)
	var a finnaAuthors
	require.NoError(t, json.Unmarshal(raw, &a))
	got := append([]string(nil), a.Primary...)
	sort.Strings(got)
	assert.Equal(t, []string{"Kyrö, Tuomas", "Padded, Author"}, got)
}

func TestFinnaAuthors_UnmarshalJSON_InvalidPrimaryReturnsError(t *testing.T) {
	raw := []byte(`{"primary": "not a map"}`)
	var a finnaAuthors
	assert.Error(t, json.Unmarshal(raw, &a))
}

func TestExtractFinnaEnrichmentData_FullRecord(t *testing.T) {
	rec := &finnaRecord{
		Title:      "Mielensäpahoittaja",
		Authors:    finnaAuthors{Primary: []string{"Kyrö, Tuomas"}},
		Publishers: []string{"WSOY"},
		Year:       "2010",
		Languages:  []string{"fin"},
		Subjects:   [][]string{{"romaanit."}, {"vanhukset"}},
		ID:         "fikka.5605354",
		CleanISBN:  "9510366439",
	}

	data := extractFinnaEnrichmentData(rec)
	require.NotNil(t, data)
	require.NotNil(t, data.Title)
	assert.Equal(t, "Mielensäpahoittaja", *data.Title)
	require.NotNil(t, data.Publisher)
	assert.Equal(t, "WSOY", *data.Publisher)
	require.NotNil(t, data.PublishDate)
	assert.Equal(t, "2010", *data.PublishDate)
	require.NotNil(t, data.Language)
	assert.Equal(t, "fin", *data.Language)
	assert.Equal(t, []string{"romaanit", "vanhukset"}, data.Subjects)
	assert.Equal(t, []string{"Kyrö, Tuomas"}, data.Authors)
	assert.Nil(t, data.CoverURL)
	assert.Nil(t, data.Description)
}

func TestExtractFinnaEnrichmentData_EmptyRecord(t *testing.T) {
	data := extractFinnaEnrichmentData(&finnaRecord{})
	require.NotNil(t, data)
	assert.Nil(t, data.Title)
	assert.Nil(t, data.Publisher)
	assert.Nil(t, data.PublishDate)
	assert.Nil(t, data.Language)
	assert.Nil(t, data.Subjects)
	assert.Nil(t, data.Authors)
}

func TestFinnaEnricher_FetchFromAPI_HappyPath(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "api.finna.fi", req.URL.Host)
		assert.Equal(t, "/api/v1/search", req.URL.Path)
		q := req.URL.Query()
		assert.Equal(t, "9510366439", q.Get("lookfor"))
		assert.Equal(t, "ISN", q.Get("type"))
		assert.Equal(t, "1", q.Get("limit"))
		assert.ElementsMatch(t, finnaSearchFields, q["field[]"])

		return jsonResponse(`{
			"resultCount": 1,
			"records": [{
				"title": "Mielensäpahoittaja",
				"authors": {
					"primary": {"Kyrö, Tuomas": {"role": ["kirjoittaja"]}},
					"secondary": [],
					"corporate": []
				},
				"publishers": ["WSOY"],
				"year": "2010",
				"languages": ["fin"],
				"subjects": [["romaanit"], ["vanhukset."]],
				"id": "fikka.5605354",
				"cleanIsbn": "9510366439"
			}],
			"status": "OK"
		}`, http.StatusOK), nil
	})

	e := &FinnaEnricher{
		getHTTPClient: func() *http.Client {
			return &http.Client{Transport: transport}
		},
		getRateLimiter: func() *ratelimit.Limiter {
			return ratelimit.NewWithBurst("Finna test", 1000, 1000)
		},
	}

	result, err := e.fetchFromAPI(t.Context(), "9510366439")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.NotFound)
	require.NotNil(t, result.Data)
	require.NotNil(t, result.Data.Title)
	assert.Equal(t, "Mielensäpahoittaja", *result.Data.Title)
	assert.Equal(t, []string{"Kyrö, Tuomas"}, result.Data.Authors)
	assert.Equal(t, []string{"romaanit", "vanhukset"}, result.Data.Subjects)
}

func TestFinnaEnricher_FetchFromAPI_NotFound(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(`{"resultCount": 0, "status": "OK"}`, http.StatusOK), nil
	})

	e := &FinnaEnricher{
		getHTTPClient: func() *http.Client {
			return &http.Client{Transport: transport}
		},
		getRateLimiter: func() *ratelimit.Limiter {
			return ratelimit.NewWithBurst("Finna test", 1000, 1000)
		},
	}

	result, err := e.fetchFromAPI(t.Context(), "9780000000000")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.NotFound)
	assert.Nil(t, result.Data)
}
