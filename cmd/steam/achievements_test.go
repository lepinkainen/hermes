package steam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	internalerrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlayerAchievements_ParseSuccess(t *testing.T) {
	fixtureData, err := os.ReadFile("testdata/achievements_success.json")
	require.NoError(t, err, "Failed to read test fixture")

	var resp SteamAchievementsResponse
	err = json.Unmarshal(fixtureData, &resp)
	require.NoError(t, err, "Failed to parse fixture JSON")

	require.True(t, resp.PlayerStats.Success, "Response should indicate success")
	require.Len(t, resp.PlayerStats.Achievements, 2, "Should have 2 achievements")

	// Check unlocked achievement
	assert.Equal(t, "FIRST_KILL", resp.PlayerStats.Achievements[0].APIName)
	assert.Equal(t, 1, resp.PlayerStats.Achievements[0].Achieved)
	assert.Equal(t, "First Blood", resp.PlayerStats.Achievements[0].Name)
	assert.Equal(t, "Get your first kill", resp.PlayerStats.Achievements[0].Description)
	assert.Equal(t, float64(75.5), resp.PlayerStats.Achievements[0].Percent)

	// Check locked achievement
	assert.Equal(t, "WIN_GAME", resp.PlayerStats.Achievements[1].APIName)
	assert.Equal(t, 0, resp.PlayerStats.Achievements[1].Achieved)
	assert.Equal(t, "Victory", resp.PlayerStats.Achievements[1].Name)
}

func TestGetPlayerAchievements_ParseNoAchievements(t *testing.T) {
	fixtureData, err := os.ReadFile("testdata/achievements_none.json")
	require.NoError(t, err, "Failed to read test fixture")

	var resp SteamAchievementsResponse
	err = json.Unmarshal(fixtureData, &resp)
	require.NoError(t, err, "Failed to parse fixture JSON")

	require.False(t, resp.PlayerStats.Success, "Response should indicate failure")
	require.Contains(t, resp.PlayerStats.Error, "no stats", "Error should mention 'no stats'")
	require.Empty(t, resp.PlayerStats.Achievements, "Should have no achievements")
}

func TestGetPlayerAchievements_ParsePrivateProfile(t *testing.T) {
	fixtureData, err := os.ReadFile("testdata/achievements_private.json")
	require.NoError(t, err, "Failed to read test fixture")

	var resp SteamAchievementsResponse
	err = json.Unmarshal(fixtureData, &resp)
	require.NoError(t, err, "Failed to parse fixture JSON")

	require.False(t, resp.PlayerStats.Success, "Response should indicate failure")
	require.Contains(t, resp.PlayerStats.Error, "private", "Error should mention 'private'")
	require.Empty(t, resp.PlayerStats.Achievements, "Should have no achievements")
}

func TestGetPlayerAchievements_HTTPSuccess(t *testing.T) {
	fixtureData, err := os.ReadFile("testdata/achievements_success.json")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "key123", r.URL.Query().Get("key"))
		assert.Equal(t, "76561198000000000", r.URL.Query().Get("steamid"))
		assert.Equal(t, "440", r.URL.Query().Get("appid"))
		assert.Equal(t, "en", r.URL.Query().Get("l"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixtureData)
	}))
	defer server.Close()

	achievements, err := getPlayerAchievementsWithBaseURL("76561198000000000", "key123", 440, server.URL)
	require.NoError(t, err)
	require.Len(t, achievements, 2)
	assert.Equal(t, "FIRST_KILL", achievements[0].APIName)
	assert.Equal(t, 1, achievements[0].Achieved)
}

func TestGetPlayerAchievements_HTTPNoAchievements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"playerstats":{"success":false,"error":"Requested app has no stats"}}`))
	}))
	defer server.Close()

	achievements, err := getPlayerAchievementsWithBaseURL("76561198000000000", "key123", 999, server.URL)
	require.NoError(t, err)
	assert.Nil(t, achievements)
}

func TestGetPlayerAchievements_HTTPPrivateProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"playerstats":{"success":false,"error":"Profile is not public"}}`))
	}))
	defer server.Close()

	_, err := getPlayerAchievementsWithBaseURL("76561198000000000", "key123", 440, server.URL)
	require.Error(t, err)
	var profErr *internalerrors.SteamProfileError
	require.ErrorAs(t, err, &profErr)
}

func TestGetPlayerAchievements_HTTPInvalidAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<html>401 Unauthorized</html>`))
	}))
	defer server.Close()

	_, err := getPlayerAchievementsWithBaseURL("76561198000000000", "bad", 440, server.URL)
	require.Error(t, err)
	var profErr *internalerrors.SteamProfileError
	require.ErrorAs(t, err, &profErr)
	assert.Equal(t, 401, profErr.StatusCode)
}

func TestGetPlayerAchievements_HTTPRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := getPlayerAchievementsWithBaseURL("76561198000000000", "key", 440, server.URL)
	require.Error(t, err)
	assert.True(t, internalerrors.IsRateLimitError(err))
}

func TestGetPlayerAchievements_HTTPBadRequestNoAchievements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`unparseable`))
	}))
	defer server.Close()

	achievements, err := getPlayerAchievementsWithBaseURL("76561198000000000", "key", 440, server.URL)
	require.NoError(t, err)
	assert.Nil(t, achievements)
}
