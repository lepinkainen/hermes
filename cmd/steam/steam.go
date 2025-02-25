package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GetGameDetails fetches additional details for a game from Steam Store API
func GetGameDetails(appID int) (*GameDetails, error) {
	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%d", appID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game details: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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

// ImportSteamGames fetches games from a user's Steam library
func ImportSteamGames(steamID string, apiKey string) ([]Game, error) {
	// Steam Web API endpoint
	baseURL := "http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/"

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
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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
