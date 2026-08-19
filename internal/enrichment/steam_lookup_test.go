package enrichment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/tui"
	"github.com/stretchr/testify/require"
)

// Note: fetchSteamStoreSearch makes real HTTP calls to Steam's API
// These tests verify the parsing logic but skip actual HTTP calls in CI
// To enable full HTTP testing, run with: go test -tags=integration

func TestFetchSteamStoreSearch_ParseResponse(t *testing.T) {
	// Test the response parsing logic with a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters are present
		query := r.URL.Query().Get("term")
		require.NotEmpty(t, query)

		// Return mock response
		resp := SteamStoreSearchResponse{
			Total: 2,
			Items: []SteamStoreSearchResult{
				{AppID: 70, Name: "Half-Life", Tiny: "https://example.com/hl.jpg"},
				{AppID: 220, Name: "Half-Life 2", Tiny: "https://example.com/hl2.jpg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	// Test the HTTP call and parsing directly with the mock server
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"?term=Half-Life&l=english&cc=US", nil)
	require.NoError(t, err)

	client := &http.Client{}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var searchResp SteamStoreSearchResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&searchResp))

	require.Equal(t, 2, searchResp.Total)
	require.Len(t, searchResp.Items, 2)
	require.Equal(t, 70, searchResp.Items[0].AppID)
	require.Equal(t, "Half-Life", searchResp.Items[0].Name)
}

func TestSteamStoreSearchResponse_EmptyResults(t *testing.T) {
	// Test parsing empty results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SteamStoreSearchResponse{
			Total: 0,
			Items: []SteamStoreSearchResult{},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	client := &http.Client{}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var searchResp SteamStoreSearchResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&searchResp))

	require.Equal(t, 0, searchResp.Total)
	require.Len(t, searchResp.Items, 0)
}

func TestSteamStoreSearch_HTTPErrorHandling(t *testing.T) {
	// Test handling of HTTP errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	client := &http.Client{}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestSteamStoreSearch_InvalidJSONHandling(t *testing.T) {
	// Test handling of invalid JSON responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	client := &http.Client{}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var searchResp SteamStoreSearchResponse
	err = json.NewDecoder(resp.Body).Decode(&searchResp)
	require.Error(t, err, "should fail to parse invalid JSON")
}

func TestSteamStoreSearch_ContextCancellation(t *testing.T) {
	// Test that context cancellation is respected
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler should never be called because context is already cancelled
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	client := &http.Client{}
	defer client.CloseIdleConnections()

	_, err = client.Do(req)
	require.Error(t, err, "should fail with cancelled context")
	require.Contains(t, err.Error(), "context canceled")
}

func TestSteamStoreSearch_QueryEscaping(t *testing.T) {
	// Test that URL query parameters are properly escaped
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify special characters are properly escaped
		require.Equal(t, "Grand Theft Auto V", r.URL.Query().Get("term"))

		resp := SteamStoreSearchResponse{
			Total: 1,
			Items: []SteamStoreSearchResult{
				{AppID: 271590, Name: "Grand Theft Auto V", Tiny: "https://example.com/gta5.jpg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"?term=Grand+Theft+Auto+V", nil)
	require.NoError(t, err)

	client := &http.Client{}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var searchResp SteamStoreSearchResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&searchResp))

	require.Len(t, searchResp.Items, 1)
	require.Equal(t, 271590, searchResp.Items[0].AppID)
}

func withSteamSearchServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	prevURL := steamSearchURL
	prevClient := steamSearchHTTPClient
	steamSearchURL = server.URL + "/?term=%s&l=english&cc=US"
	steamSearchHTTPClient = server.Client()
	t.Cleanup(func() {
		steamSearchURL = prevURL
		steamSearchHTTPClient = prevClient
	})
}

func TestFetchSteamStoreSearchSuccess(t *testing.T) {
	withSteamSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Half-Life", r.URL.Query().Get("term"))
		_ = json.NewEncoder(w).Encode(SteamStoreSearchResponse{
			Total: 1,
			Items: []SteamStoreSearchResult{
				{AppID: 70, Name: "Half-Life", Tiny: "hl.jpg"},
			},
		})
	})

	results, err := fetchSteamStoreSearch(t.Context(), "Half-Life")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, 70, results[0].AppID)
	require.Equal(t, "Half-Life", results[0].Name)
}

func TestSteamSearchTitle(t *testing.T) {
	require.Equal(t, "Star Wars Jedi Fallen Order", steamSearchTitle("Star Wars Jedi - Fallen Order"))
	require.Equal(t, "Half-Life", steamSearchTitle("Half-Life"))
	require.Equal(t, "Portal 2", steamSearchTitle("Portal 2"))
}

func TestFetchSteamStoreSearchHTTPError(t *testing.T) {
	withSteamSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	})

	_, err := fetchSteamStoreSearch(t.Context(), "foo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 500")
}

func TestFetchSteamStoreSearchInvalidJSON(t *testing.T) {
	withSteamSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})

	_, err := fetchSteamStoreSearch(t.Context(), "foo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse Steam search response")
}

func TestNormalizeSteamQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase and spaces to underscores",
			input:    "Half-Life 2",
			expected: "half-life_2",
		},
		{
			name:     "remove special characters",
			input:    "Grand Theft Auto: V",
			expected: "grand_theft_auto__v",
		},
		{
			name:     "trim whitespace",
			input:    "  Portal  ",
			expected: "portal",
		},
		{
			name:     "multiple spaces",
			input:    "The  Elder   Scrolls",
			expected: "the__elder___scrolls",
		},
		{
			name:     "keep numbers and dashes",
			input:    "Counter-Strike 1.6",
			expected: "counter-strike_1_6",
		},
		{
			name:     "unicode characters",
			input:    "Café Racer",
			expected: "caf__racer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSteamQuery(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSelectSteamResult_SingleResult(t *testing.T) {
	results := []tui.SteamSearchResult{
		{AppID: 123, Name: "Test Game", HeaderImage: "test.jpg"},
	}

	selected, err := selectSteamResult(results, "Test Game", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 123, selected.AppID)
}

func TestSelectSteamResult_EmptyResults(t *testing.T) {
	results := []tui.SteamSearchResult{}

	selected, err := selectSteamResult(results, "Test Game", false)
	require.NoError(t, err)
	require.Nil(t, selected)
}

func TestSelectSteamResult_ExactMatchNonInteractive(t *testing.T) {
	results := []tui.SteamSearchResult{
		{AppID: 70, Name: "Half-Life", HeaderImage: "hl.jpg"},
		{AppID: 220, Name: "Half-Life 2", HeaderImage: "hl2.jpg"},
		{AppID: 280, Name: "Half-Life: Source", HeaderImage: "hls.jpg"},
	}

	selected, err := selectSteamResult(results, "Half-Life 2", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 220, selected.AppID, "should select exact match")
}

func TestSelectSteamResult_NoExactMatchNonInteractive(t *testing.T) {
	results := []tui.SteamSearchResult{
		{AppID: 70, Name: "Half-Life", HeaderImage: "hl.jpg"},
		{AppID: 220, Name: "Half-Life 2", HeaderImage: "hl2.jpg"},
		{AppID: 280, Name: "Half-Life: Source", HeaderImage: "hls.jpg"},
	}

	// "Half-Life 3" has no exact match, but is title-compatible with the
	// plain "Half-Life" entry (a prefix match) and not with the "2" or
	// "Source" variants, so it should fall back to that sole candidate
	// rather than blindly taking the first search result.
	selected, err := selectSteamResult(results, "Half-Life 3", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 70, selected.AppID, "should select the sole title-compatible candidate when no exact match")
}

func TestSelectSteamResult_NoTitleCompatibleResultFallsThrough(t *testing.T) {
	// Regression test: Steam's storesearch can return a single, wholly
	// unrelated result for titles never released on Steam (e.g. "Uncharted
	// 2", a PS exclusive). Previously this was auto-selected as "the only
	// result", writing junk data and permanently blocking the RAWG
	// fallback (which only runs when Steam yields nil). It must now fall
	// through to nil so the caller's RAWG fallback gets a chance.
	results := []tui.SteamSearchResult{
		{AppID: 1155880, Name: "Some Indie Citybuilder", HeaderImage: "indie.jpg"},
	}

	selected, err := selectSteamResult(results, "Uncharted 2", false)
	require.NoError(t, err)
	require.Nil(t, selected, "wholly unrelated Steam result should fall through instead of being auto-selected")
}

func TestSelectSteamResult_TrademarkSymbolStillMatches(t *testing.T) {
	// Regression test: Steam titles commonly carry a trademark symbol
	// ("STAR WARS Jedi: Fallen Order™") that RAWG/note titles never have.
	// normalizeGameTitleTokens must strip it so the title-compatibility gate
	// doesn't itself reject legitimate Steam matches.
	results := []tui.SteamSearchResult{
		{AppID: 1172380, Name: "STAR WARS Jedi: Fallen Order™", HeaderImage: "swjfo.jpg"},
	}

	selected, err := selectSteamResult(results, "Star Wars Jedi Fallen Order", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 1172380, selected.AppID)
}

func TestSelectSteamResult_OnlyOneCompatibleAmongMultipleResults(t *testing.T) {
	// A multi-result search where only one entry is actually the game in
	// question — the incompatible entry sorts first, so a naive
	// "auto-select the first result" rule would pick the wrong one.
	results := []tui.SteamSearchResult{
		{AppID: 999, Name: "Totally Unrelated Game", HeaderImage: "unrelated.jpg"},
		{AppID: 620, Name: "Portal 2", HeaderImage: "portal2.jpg"},
	}

	selected, err := selectSteamResult(results, "Portal 2", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 620, selected.AppID, "should pick the title-compatible candidate over an incompatible earlier entry")
}

func TestFindExactSteamMatch_Found(t *testing.T) {
	results := []tui.SteamSearchResult{
		{AppID: 70, Name: "Half-Life", HeaderImage: "hl.jpg"},
		{AppID: 220, Name: "Half-Life 2", HeaderImage: "hl2.jpg"},
	}

	match := findExactSteamMatch(results, "half-life 2")
	require.NotNil(t, match)
	require.Equal(t, 220, match.AppID)
}

func TestFindExactSteamMatch_NotFound(t *testing.T) {
	results := []tui.SteamSearchResult{
		{AppID: 70, Name: "Half-Life", HeaderImage: "hl.jpg"},
		{AppID: 220, Name: "Half-Life 2", HeaderImage: "hl2.jpg"},
	}

	match := findExactSteamMatch(results, "Portal")
	require.Nil(t, match)
}

func TestFindExactSteamMatch_Ambiguous(t *testing.T) {
	results := []tui.SteamSearchResult{
		{AppID: 70, Name: "Game", HeaderImage: "g1.jpg"},
		{AppID: 220, Name: "Game", HeaderImage: "g2.jpg"},
	}

	match := findExactSteamMatch(results, "Game")
	require.Nil(t, match, "should return nil for ambiguous matches")
}

func TestFindExactSteamMatch_CaseInsensitive(t *testing.T) {
	results := []tui.SteamSearchResult{
		{AppID: 70, Name: "Half-Life", HeaderImage: "hl.jpg"},
	}

	match := findExactSteamMatch(results, "HALF-LIFE")
	require.NotNil(t, match)
	require.Equal(t, 70, match.AppID)
}
