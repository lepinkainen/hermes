package enrichers

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string, status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestBookBrainzEnricher_Name(t *testing.T) {
	e := NewBookBrainzEnricher()
	assert.Equal(t, "BookBrainz", e.Name())
}

func TestBookBrainzEnricher_Priority(t *testing.T) {
	e := NewBookBrainzEnricher()
	assert.Equal(t, 3, e.Priority())
}

func TestFindBestEditionMatch(t *testing.T) {
	results := []bbEditionSearchResult{
		{
			BBID: "wrong-type",
			Type: "Work",
			IdentifierSet: &bbIdentifierSet{Identifiers: []bbIdentifier{
				{Value: "9780441172719"},
			}},
		},
		{
			BBID: "right",
			Type: "Edition",
			IdentifierSet: &bbIdentifierSet{Identifiers: []bbIdentifier{
				{Value: "978-0441172719"},
			}},
		},
	}

	match := findBestEditionMatch(results, "9780441172719")
	require.NotNil(t, match)
	assert.Equal(t, "right", match.BBID)
}

func TestFindBestEditionMatch_NoIdentifierMatch(t *testing.T) {
	results := []bbEditionSearchResult{
		{
			BBID: "wrong",
			Type: "Edition",
			IdentifierSet: &bbIdentifierSet{Identifiers: []bbIdentifier{
				{Value: "9780261103573"},
			}},
		},
	}

	assert.Nil(t, findBestEditionMatch(results, "9780441172719"))
}

func TestExtractEditionEnrichmentData(t *testing.T) {
	searchResult := &bbEditionSearchResult{
		BBID:  "fb7d0a29-e03a-4d53-81ce-25c2712d4845",
		Name:  "Fallback Dune",
		Type:  "Edition",
		Pages: 700,
	}
	details := &bbEditionResponse{
		DefaultAlias:     &bbDefaultAlias{Name: "Dune"},
		Pages:            896,
		Languages:        []string{"eng"},
		ReleaseEventDate: "+001990-09-01",
		Publishers:       []bbPublisher{{Name: "Ace"}},
		AuthorCredits: &bbAuthorCredits{Names: []bbAuthorCreditName{
			{Name: "Frank Herbert"},
		}},
	}

	data := extractEditionEnrichmentData(searchResult, details)
	require.NotNil(t, data)
	require.NotNil(t, data.Title)
	assert.Equal(t, "Dune", *data.Title)
	require.NotNil(t, data.NumberOfPages)
	assert.Equal(t, 896, *data.NumberOfPages)
	require.NotNil(t, data.Publisher)
	assert.Equal(t, "Ace", *data.Publisher)
	require.NotNil(t, data.PublishDate)
	assert.Equal(t, "1990-09-01", *data.PublishDate)
	require.NotNil(t, data.Language)
	assert.Equal(t, "eng", *data.Language)
	assert.Equal(t, []string{"Frank Herbert"}, data.Authors)
}

func TestExtractEditionEnrichmentData_FallsBackToSearchResult(t *testing.T) {
	searchResult := &bbEditionSearchResult{
		Name:  "Dune",
		Pages: 896,
	}

	data := extractEditionEnrichmentData(searchResult, nil)
	require.NotNil(t, data)
	require.NotNil(t, data.Title)
	assert.Equal(t, "Dune", *data.Title)
	require.NotNil(t, data.NumberOfPages)
	assert.Equal(t, 896, *data.NumberOfPages)
}

func TestBookBrainzEnricher_FetchFromAPI(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "bookbrainz.org" && req.URL.Path == "/search/search":
			assert.Equal(t, "edition", req.URL.Query().Get("type"))
			assert.Equal(t, "9780441172719", req.URL.Query().Get("q"))
			return jsonResponse(`{
				"results": [{
					"bbid": "fb7d0a29-e03a-4d53-81ce-25c2712d4845",
					"name": "Dune",
					"type": "Edition",
					"pages": 896,
					"identifierSet": {
						"identifiers": [
							{"typeId": 9, "value": "978-0441172719"}
						]
					}
				}],
				"total": 1
			}`, http.StatusOK), nil
		case req.URL.Host == "api.bookbrainz.org" && req.URL.Path == "/1/edition/fb7d0a29-e03a-4d53-81ce-25c2712d4845":
			return jsonResponse(`{
				"bbid": "fb7d0a29-e03a-4d53-81ce-25c2712d4845",
				"defaultAlias": {"name": "Dune", "language": "eng"},
				"pages": 896,
				"languages": ["eng"],
				"publishers": [{"name": "Ace"}],
				"releaseEventDate": "+001990-09-01",
				"authorCredits": {
					"names": [{"name": "Frank Herbert"}]
				}
			}`, http.StatusOK), nil
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}
	})

	e := &BookBrainzEnricher{
		getHTTPClient: func() *http.Client {
			return &http.Client{Transport: transport}
		},
		getRateLimiter: func() *ratelimit.Limiter {
			return ratelimit.NewWithBurst("BookBrainz test", 1000, 1000)
		},
	}

	result, err := e.fetchFromAPI(t.Context(), "9780441172719")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.NotFound)
	require.NotNil(t, result.Data)
	require.NotNil(t, result.Data.Title)
	assert.Equal(t, "Dune", *result.Data.Title)
	assert.Equal(t, []string{"Frank Herbert"}, result.Data.Authors)
}

func TestBookBrainzEnricher_FetchFromAPI_NotFound(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(`{"results": [], "total": 0}`, http.StatusOK), nil
	})

	e := &BookBrainzEnricher{
		getHTTPClient: func() *http.Client {
			return &http.Client{Transport: transport}
		},
		getRateLimiter: func() *ratelimit.Limiter {
			return ratelimit.NewWithBurst("BookBrainz test", 1000, 1000)
		},
	}

	result, err := e.fetchFromAPI(t.Context(), "missing")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.NotFound)
	assert.Nil(t, result.Data)
}
