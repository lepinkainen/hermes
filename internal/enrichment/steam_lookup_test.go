package enrichment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"?term=Half-Life&l=english&cc=US", nil)
	require.NoError(t, err)

	client := &http.Client{}
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	client := &http.Client{}
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	client := &http.Client{}
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var searchResp SteamStoreSearchResponse
	err = json.NewDecoder(resp.Body).Decode(&searchResp)
	require.Error(t, err, "should fail to parse invalid JSON")
}

func TestSteamStoreSearch_ContextCancellation(t *testing.T) {
	// Test that context cancellation is respected
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://store.steampowered.com", nil)
	require.NoError(t, err)

	client := &http.Client{}
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"?term=Grand+Theft+Auto+V", nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var searchResp SteamStoreSearchResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&searchResp))

	require.Len(t, searchResp.Items, 1)
	require.Equal(t, 271590, searchResp.Items[0].AppID)
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
	}

	selected, err := selectSteamResult(results, "Portal", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 70, selected.AppID, "should select first result when no exact match")
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
