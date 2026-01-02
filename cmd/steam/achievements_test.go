package steam

import (
	"encoding/json"
	"os"
	"testing"

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
