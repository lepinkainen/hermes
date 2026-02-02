package steam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGameDetails_Success(t *testing.T) {
	// Load test fixture
	fixtureData, err := os.ReadFile("testdata/app_details_success.json")
	require.NoError(t, err)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "appids=12345")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixtureData)
	}))
	defer server.Close()

	// Test using the base URL injection helper
	details, err := getGameDetailsWithBaseURL(12345, server.URL)
	require.NoError(t, err)
	require.NotNil(t, details)

	// Verify the parsed details
	assert.Equal(t, "Test Game", details.Name)
	assert.Equal(t, 12345, details.AppID)
	assert.Equal(t, "This is a detailed description of the test game with lots of information about gameplay, features, and story.", details.Description)
	assert.Equal(t, []string{"Developer One", "Developer Two"}, details.Developers)
	assert.Equal(t, []string{"Publisher One"}, details.Publishers)
}

func TestGetGameDetails_ParseResponse(t *testing.T) {
	// Test the JSON parsing logic directly
	fixtureData, err := os.ReadFile("testdata/app_details_success.json")
	require.NoError(t, err)

	var result map[string]struct {
		Success bool        `json:"success"`
		Data    GameDetails `json:"data"`
	}

	err = json.Unmarshal(fixtureData, &result)
	require.NoError(t, err)

	appData, exists := result["12345"]
	require.True(t, exists, "Expected app ID 12345 in response")
	require.True(t, appData.Success, "Expected success to be true")

	details := appData.Data

	// Verify parsed data matches expectations
	assert.Equal(t, "Test Game", details.Name)
	assert.Equal(t, "This is a detailed description of the test game with lots of information about gameplay, features, and story.", details.Description)
	assert.Equal(t, "A test game for unit testing", details.ShortDesc)
	assert.Equal(t, "https://cdn.akamai.steamstatic.com/steam/apps/12345/header.jpg", details.HeaderImage)
	assert.Equal(t, []string{"Developer One", "Developer Two"}, details.Developers)
	assert.Equal(t, []string{"Publisher One"}, details.Publishers)
	assert.Equal(t, "15 Jan, 2023", details.ReleaseDate.Date)
	assert.Equal(t, false, details.ReleaseDate.ComingSoon)
	assert.Equal(t, 85, details.Metacritic.Score)
	assert.Equal(t, "https://www.metacritic.com/game/pc/test-game", details.Metacritic.URL)

	// Verify categories
	require.Len(t, details.Categories, 3)
	assert.Equal(t, "Single-player", details.Categories[0].Description)
	assert.Equal(t, "Multi-player", details.Categories[1].Description)
	assert.Equal(t, "Steam Achievements", details.Categories[2].Description)

	// Verify genres
	require.Len(t, details.Genres, 2)
	assert.Equal(t, "Action", details.Genres[0].Description)
	assert.Equal(t, "Adventure", details.Genres[1].Description)

	// Verify screenshots
	require.Len(t, details.Screenshots, 2)
	assert.Equal(t, "https://cdn.akamai.steamstatic.com/steam/apps/12345/ss_1.jpg", details.Screenshots[0].PathURL)
	assert.Equal(t, "https://cdn.akamai.steamstatic.com/steam/apps/12345/ss_2.jpg", details.Screenshots[1].PathURL)
}

func TestGetGameDetails_ParseMinimalResponse(t *testing.T) {
	// Test parsing minimal game data
	fixtureData, err := os.ReadFile("testdata/app_details_minimal.json")
	require.NoError(t, err)

	var result map[string]struct {
		Success bool        `json:"success"`
		Data    GameDetails `json:"data"`
	}

	err = json.Unmarshal(fixtureData, &result)
	require.NoError(t, err)

	appData, exists := result["55555"]
	require.True(t, exists)
	require.True(t, appData.Success)

	details := appData.Data

	assert.Equal(t, "Minimal Game", details.Name)
	assert.Equal(t, "Basic game", details.Description)
	assert.Equal(t, []string{"Solo Developer"}, details.Developers)
	assert.Len(t, details.Screenshots, 0)
}

func TestGetGameDetails_ParseNotFoundResponse(t *testing.T) {
	// Test parsing a not found response
	fixtureData, err := os.ReadFile("testdata/app_details_not_found.json")
	require.NoError(t, err)

	var result map[string]struct {
		Success bool        `json:"success"`
		Data    GameDetails `json:"data"`
	}

	err = json.Unmarshal(fixtureData, &result)
	require.NoError(t, err)

	appData, exists := result["99999"]
	require.True(t, exists)
	assert.False(t, appData.Success, "Expected success to be false for not found game")
}

func TestImportSteamGames_ParseResponse(t *testing.T) {
	// Test the JSON parsing logic for owned games
	fixtureData, err := os.ReadFile("testdata/owned_games_response.json")
	require.NoError(t, err)

	var steamResp SteamResponse
	err = json.Unmarshal(fixtureData, &steamResp)
	require.NoError(t, err)

	games := steamResp.Response.Games

	// Verify we got all games
	require.Len(t, games, 3)

	// Verify first game
	assert.Equal(t, 12345, games[0].AppID)
	assert.Equal(t, "Test Game One", games[0].Name)
	assert.Equal(t, 120, games[0].PlaytimeForever)
	assert.Equal(t, 30, games[0].PlaytimeRecent)

	// Verify second game
	assert.Equal(t, 67890, games[1].AppID)
	assert.Equal(t, "Test Game Two", games[1].Name)
	assert.Equal(t, 0, games[1].PlaytimeForever)
	assert.Equal(t, 0, games[1].PlaytimeRecent)

	// Verify third game (no recent playtime field)
	assert.Equal(t, 11111, games[2].AppID)
	assert.Equal(t, "Test Game Three", games[2].Name)
	assert.Equal(t, 3600, games[2].PlaytimeForever)
	assert.Equal(t, 0, games[2].PlaytimeRecent) // Should default to 0
}

func TestImportSteamGames_WithTestServer(t *testing.T) {
	// Load test fixture
	fixtureData, err := os.ReadFile("testdata/owned_games_response.json")
	require.NoError(t, err)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		assert.Equal(t, "test-api-key", r.URL.Query().Get("key"))
		assert.Equal(t, "12345678901234567", r.URL.Query().Get("steamid"))
		assert.Equal(t, "json", r.URL.Query().Get("format"))
		assert.Equal(t, "true", r.URL.Query().Get("include_appinfo"))
		assert.Equal(t, "true", r.URL.Query().Get("include_played_free_games"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixtureData)
	}))
	defer server.Close()

	// Test using the base URL injection helper
	games, err := importSteamGamesWithBaseURL("12345678901234567", "test-api-key", server.URL)
	require.NoError(t, err)
	require.Len(t, games, 3)

	// Verify first game
	assert.Equal(t, 12345, games[0].AppID)
	assert.Equal(t, "Test Game One", games[0].Name)
	assert.Equal(t, 120, games[0].PlaytimeForever)
}

func TestImportSteamGames_ErrorHandling(t *testing.T) {
	// Test error cases with a test server
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "API returns 500",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "Internal server error"}`,
			expectedErrMsg: "steam API returned status code 500",
		},
		{
			name:           "API returns 401",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Invalid API key"}`,
			expectedErrMsg: "steam API returned status code 401",
		},
		{
			name:           "Invalid JSON response",
			statusCode:     http.StatusOK,
			responseBody:   `{invalid json}`,
			expectedErrMsg: "failed to parse JSON response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			_, err := importSteamGamesWithBaseURL("test-steam-id", "test-api-key", server.URL)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string // duration string for comparison
	}{
		{
			name:     "valid seconds",
			header:   "60",
			expected: "1m0s",
		},
		{
			name:     "zero seconds",
			header:   "0",
			expected: "0s",
		},
		{
			name:     "large value",
			header:   "3600",
			expected: "1h0m0s",
		},
		{
			name:     "small value",
			header:   "5",
			expected: "5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryAfter(tt.header)
			require.Equal(t, tt.expected, result.String())
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Test parsing HTTP-date format (future time)
	// Use a fixed time for testing
	tests := []struct {
		name           string
		header         string
		expectPositive bool
	}{
		{
			name:           "RFC1123 format - future",
			header:         "Mon, 02 Jan 2030 15:04:05 GMT",
			expectPositive: true,
		},
		{
			name:           "RFC850 format - future",
			header:         "Monday, 02-Jan-30 15:04:05 GMT",
			expectPositive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryAfter(tt.header)
			if tt.expectPositive {
				require.Greater(t, result.Seconds(), float64(0), "should parse future date as positive duration")
			}
		})
	}
}

func TestParseRetryAfter_InvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "empty string",
			header: "",
		},
		{
			name:   "invalid number",
			header: "abc",
		},
		{
			name:   "invalid date",
			header: "not-a-date",
		},
		{
			name:   "past HTTP date",
			header: "Mon, 02 Jan 2000 15:04:05 GMT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryAfter(tt.header)
			require.Equal(t, int64(0), result.Nanoseconds(), "should return 0 duration for invalid input")
		})
	}
}

func TestRateLimitHandling_429Response(t *testing.T) {
	tests := []struct {
		name                string
		retryAfterHeader    string
		expectRetryDuration bool
		expectedContains    string
	}{
		{
			name:                "429 with retry-after seconds",
			retryAfterHeader:    "120",
			expectRetryDuration: true,
			expectedContains:    "2m0s",
		},
		{
			name:                "429 without retry-after",
			retryAfterHeader:    "",
			expectRetryDuration: false,
			expectedContains:    "Please try again later",
		},
		{
			name:                "429 with invalid retry-after",
			retryAfterHeader:    "invalid",
			expectRetryDuration: false,
			expectedContains:    "Please try again later",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server that returns 429
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.retryAfterHeader != "" {
					w.Header().Set("Retry-After", tt.retryAfterHeader)
				}
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("Too Many Requests"))
			}))
			defer server.Close()

			// Make request directly (not through getGameDetails since URL is hardcoded)
			resp, err := http.Get(server.URL)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Verify response
			require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

			// Parse retry-after header
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))

			if tt.expectRetryDuration {
				require.Greater(t, retryAfter.Seconds(), float64(0), "should have positive retry duration")
				require.Contains(t, retryAfter.String(), tt.expectedContains)
			} else {
				require.Equal(t, int64(0), retryAfter.Nanoseconds(), "should have zero retry duration")
			}
		})
	}
}

func TestRateLimitError_Integration(t *testing.T) {
	// Test that rate limit errors are properly detected and formatted
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Rate limit exceeded"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Verify we can detect 429 status
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	// Verify we can parse the retry-after header
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	require.Equal(t, "30s", retryAfter.String())
}
