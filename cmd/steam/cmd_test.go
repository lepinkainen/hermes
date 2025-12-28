package steam

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSteamCmd_Run_MissingConfig(t *testing.T) {
	testutil.SetTestConfig(t)

	tests := []struct {
		name        string
		cmd         SteamCmd
		configSetup func()
		wantErr     string
	}{
		{
			name: "missing steam ID from both flag and config",
			cmd: SteamCmd{
				SteamID: "",
				APIKey:  "test-key",
			},
			configSetup: func() {
				viper.Set("steam.steamid", "")
				viper.Set("steam.apikey", "test-key")
			},
			wantErr: "steam ID is required",
		},
		{
			name: "missing API key from both flag and config",
			cmd: SteamCmd{
				SteamID: "12345",
				APIKey:  "",
			},
			configSetup: func() {
				viper.Set("steam.steamid", "12345")
				viper.Set("steam.apikey", "")
			},
			wantErr: "steam API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.configSetup()
			err := tt.cmd.Run()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSteamCmd_Run_ConfigFallback(t *testing.T) {
	testutil.SetTestConfig(t)

	// Mock ParseSteamWithParamsFunc to verify config values are used
	var capturedSteamID, capturedAPIKey string
	mockFunc := func(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string) error {
		capturedSteamID = steamIDParam
		capturedAPIKey = apiKeyParam
		return nil
	}

	origFunc := ParseSteamWithParamsFunc
	ParseSteamWithParamsFunc = mockFunc
	defer func() { ParseSteamWithParamsFunc = origFunc }()

	// Set values in config
	viper.Set("steam.steamid", "config-steam-id")
	viper.Set("steam.apikey", "config-api-key")

	// Create command with empty flags (should read from config)
	cmd := SteamCmd{
		SteamID: "",
		APIKey:  "",
	}

	err := cmd.Run()
	require.NoError(t, err)

	// Verify config values were used
	assert.Equal(t, "config-steam-id", capturedSteamID, "Should use steam ID from config")
	assert.Equal(t, "config-api-key", capturedAPIKey, "Should use API key from config")
}

func TestSteamCmd_Run_FlagOverridesConfig(t *testing.T) {
	testutil.SetTestConfig(t)

	// Mock ParseSteamWithParamsFunc to verify flag values override config
	var capturedSteamID, capturedAPIKey string
	mockFunc := func(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string) error {
		capturedSteamID = steamIDParam
		capturedAPIKey = apiKeyParam
		return nil
	}

	origFunc := ParseSteamWithParamsFunc
	ParseSteamWithParamsFunc = mockFunc
	defer func() { ParseSteamWithParamsFunc = origFunc }()

	// Set values in config
	viper.Set("steam.steamid", "config-steam-id")
	viper.Set("steam.apikey", "config-api-key")

	// Create command with flags set (should override config)
	cmd := SteamCmd{
		SteamID: "flag-steam-id",
		APIKey:  "flag-api-key",
	}

	err := cmd.Run()
	require.NoError(t, err)

	// Verify flag values were used, not config
	assert.Equal(t, "flag-steam-id", capturedSteamID, "Should use steam ID from flag")
	assert.Equal(t, "flag-api-key", capturedAPIKey, "Should use API key from flag")
}
