package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/errors"
)

// parseRetryAfter extracts retry timing from Retry-After header
// Returns 0 if header is missing or invalid
func parseRetryAfter(headerValue string) time.Duration {
	if headerValue == "" {
		return 0
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(headerValue); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date
	if t, err := http.ParseTime(headerValue); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	return 0
}

// getGameDetails fetches additional details for a game from Steam Store API.
func getGameDetails(appID int) (*GameDetails, error) {
	return getGameDetailsWithBaseURL(appID, "https://store.steampowered.com/api/appdetails")
}

// getGameDetailsWithBaseURL is a helper that accepts a custom base URL for testing
func getGameDetailsWithBaseURL(appID int, baseURL string) (*GameDetails, error) {
	url := fmt.Sprintf("%s?appids=%d", baseURL, appID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game details: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for rate limit BEFORE general status check
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		if retryAfter > 0 {
			return nil, errors.NewRateLimitErrorWithRetry(
				"Steam API rate limit reached",
				retryAfter,
			)
		}
		return nil, errors.NewRateLimitError(
			"Steam API rate limit reached. Please try again later (usually after a few minutes)",
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam store API returned status code %d. Response: %s", resp.StatusCode, string(body))
	}

	// Steam Store API returns a map with app ID as key
	var result map[string]struct {
		Success bool        `json:"success"`
		Data    GameDetails `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w. Body: %s", err, string(body))
	}

	// Get the data using the appID as string key
	appData, exists := result[fmt.Sprintf("%d", appID)]
	if !exists {
		return nil, fmt.Errorf("steam API response missing data for app ID %d (game might be removed or region-locked). Response: %s", appID, string(body))
	}

	if !appData.Success {
		return nil, fmt.Errorf("steam API indicated unsuccessful data fetch for app ID %d (game might be unavailable in store). Response: %s", appID, string(body))
	}

	// Set the AppID in the returned data
	appData.Data.AppID = appID
	return &appData.Data, nil
}

// importSteamGames fetches games from a user's Steam library.
func importSteamGames(steamID string, apiKey string) ([]Game, error) {
	return importSteamGamesWithBaseURL(steamID, apiKey, "https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/")
}

// importSteamGamesWithBaseURL is a helper that accepts a custom base URL for testing
func importSteamGamesWithBaseURL(steamID string, apiKey string, baseURL string) ([]Game, error) {
	// Create URL with query parameters
	params := url.Values{}
	params.Add("key", apiKey)
	params.Add("steamid", steamID)
	params.Add("format", "json")
	params.Add("include_appinfo", "true")
	params.Add("include_played_free_games", "true")

	// Create the full URL
	fullURL := baseURL + "?" + params.Encode()

	// Make the HTTP request
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Steam games: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for rate limit BEFORE general status check
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		if retryAfter > 0 {
			return nil, errors.NewRateLimitErrorWithRetry(
				"Steam API rate limit reached",
				retryAfter,
			)
		}
		return nil, errors.NewRateLimitError(
			"Steam API rate limit reached. Please try again later (usually after a few minutes)",
		)
	}

	// Check response status and include response body in error message
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam API returned status code %d. Response: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var steamResp SteamResponse
	if err := json.Unmarshal(body, &steamResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return steamResp.Response.Games, nil
}

// ImportSteamGamesFuncType is the signature of the importSteamGames function.
type ImportSteamGamesFuncType func(steamID string, apiKey string) ([]Game, error)

// ImportSteamGamesFunc is a variable that can be overridden for testing purposes.
var ImportSteamGamesFunc ImportSteamGamesFuncType = importSteamGames

// GetPlayerAchievements fetches achievements for a specific game and user
func GetPlayerAchievements(steamID string, apiKey string, appID int) ([]Achievement, error) {
	params := url.Values{}
	params.Add("key", apiKey)
	params.Add("steamid", steamID)
	params.Add("appid", fmt.Sprintf("%d", appID))
	params.Add("l", "en") // Request English names/descriptions

	fullURL := "https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v0001/?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch achievements: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for rate limit BEFORE general status check
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		if retryAfter > 0 {
			return nil, errors.NewRateLimitErrorWithRetry(
				"Steam API rate limit reached",
				retryAfter,
			)
		}
		return nil, errors.NewRateLimitError(
			"Steam API rate limit reached. Please try again later (usually after a few minutes)",
		)
	}

	if resp.StatusCode != http.StatusOK {
		// Parse response to distinguish between different error types
		var achResp SteamAchievementsResponse
		if err := json.Unmarshal(body, &achResp); err == nil && achResp.PlayerStats.Error != "" {
			errMsg := achResp.PlayerStats.Error
			errLower := strings.ToLower(errMsg)

			// Check for "no stats/achievements" (legitimate, not an error)
			if strings.Contains(errLower, "no stats") || strings.Contains(errLower, "no achievements") {
				slog.Debug("Game has no achievements", "appid", appID, "error", errMsg)
				return nil, nil
			}

			// Check for profile/permission errors (configuration issue)
			if strings.Contains(errLower, "private") || strings.Contains(errLower, "permission") {
				return nil, errors.NewSteamProfileError(resp.StatusCode, errMsg)
			}
		}

		// Fallback for specific status codes
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, errors.NewSteamProfileError(401, "Invalid API key")
		}

		if resp.StatusCode == http.StatusForbidden {
			// 403 without parseable error - could be profile privacy
			return nil, errors.NewSteamProfileError(403, string(body))
		}

		if resp.StatusCode == http.StatusBadRequest {
			// 400 typically means "no achievements" for this game
			slog.Debug("Game has no achievements (400)", "appid", appID)
			return nil, nil
		}

		// Other errors
		return nil, fmt.Errorf("steam API returned status code %d. Response: %s", resp.StatusCode, string(body))
	}

	var achResp SteamAchievementsResponse
	if err := json.Unmarshal(body, &achResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if !achResp.PlayerStats.Success {
		// success=false - check error message
		if achResp.PlayerStats.Error != "" {
			errLower := strings.ToLower(achResp.PlayerStats.Error)
			if strings.Contains(errLower, "no stats") || strings.Contains(errLower, "no achievements") {
				slog.Debug("Game has no achievements (success=false)", "appid", appID)
				return nil, nil
			}
		}
		// success=false without clear reason - treat as no achievements
		return nil, nil
	}

	return achResp.PlayerStats.Achievements, nil
}
