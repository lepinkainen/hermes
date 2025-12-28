package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSteamWithParams(t *testing.T) {
	env := testutil.NewTestEnv(t)
	dir := env.RootDir()

	// Mock the actual ParseSteam function (ParseSteamFunc points to this)
	mockParseSteam := func() error {
		if steamID != "123" || apiKey != "key" {
			t.Fatalf("steamID/apiKey not set")
		}
		if !strings.Contains(outputDir, dir) {
			t.Fatalf("outputDir = %s, want to contain %s", outputDir, dir)
		}
		if !writeJSON {
			t.Fatalf("flags not propagated")
		}
		if jsonOutput == "" {
			t.Fatalf("jsonOutput should be set")
		}
		return nil
	}
	origParseSteamFunc := ParseSteamFunc
	ParseSteamFunc = mockParseSteam
	defer func() { ParseSteamFunc = origParseSteamFunc }() // Restore original after test

	jsonPath := filepath.Join(dir, "steam.json")
	if err := os.WriteFile(jsonPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write json path: %v", err)
	}

	// Call ParseSteamWithParams, which will internally call the mocked ParseSteamFunc
	if err := ParseSteamWithParams("123", "key", dir, true, jsonPath); err != nil {
		t.Fatalf("ParseSteamWithParams error = %v", err)
	}
}

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

func TestSteamCmd_Run_Success(t *testing.T) {
	env := testutil.NewTestEnv(t)
	testutil.SetTestConfig(t)

	// Mock ParseSteamWithParamsFunc
	mockCalled := false
	mockFunc := func(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string) error {
		mockCalled = true
		assert.Equal(t, "12345", steamIDParam)
		assert.Equal(t, "test-key", apiKeyParam)
		return nil
	}

	origFunc := ParseSteamWithParamsFunc
	ParseSteamWithParamsFunc = mockFunc
	defer func() { ParseSteamWithParamsFunc = origFunc }()

	cmd := SteamCmd{
		SteamID: "12345",
		APIKey:  "test-key",
		Output:  env.RootDir(),
	}

	err := cmd.Run()
	require.NoError(t, err)
	assert.True(t, mockCalled, "ParseSteamWithParamsFunc should have been called")
}

func TestSteamCmd_Run_WithConfig(t *testing.T) {
	testutil.SetTestConfig(t)

	// Set values in config
	viper.Set("steam.steamid", "config-steam-id")
	viper.Set("steam.apikey", "config-api-key")

	mockCalled := false
	mockFunc := func(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string) error {
		mockCalled = true
		// Should use values from config since flags are empty
		assert.Equal(t, "config-steam-id", steamIDParam)
		assert.Equal(t, "config-api-key", apiKeyParam)
		return nil
	}

	origFunc := ParseSteamWithParamsFunc
	ParseSteamWithParamsFunc = mockFunc
	defer func() { ParseSteamWithParamsFunc = origFunc }()

	// Create command with empty flags (should read from config)
	cmd := SteamCmd{
		SteamID: "",
		APIKey:  "",
	}

	err := cmd.Run()
	require.NoError(t, err)
	assert.True(t, mockCalled)
}

func TestSteamCmd_Run_FlagOverridesConfig(t *testing.T) {
	testutil.SetTestConfig(t)

	// Set values in config
	viper.Set("steam.steamid", "config-steam-id")
	viper.Set("steam.apikey", "config-api-key")

	mockCalled := false
	mockFunc := func(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string) error {
		mockCalled = true
		// Should use flag values, not config
		assert.Equal(t, "flag-steam-id", steamIDParam)
		assert.Equal(t, "flag-api-key", apiKeyParam)
		return nil
	}

	origFunc := ParseSteamWithParamsFunc
	ParseSteamWithParamsFunc = mockFunc
	defer func() { ParseSteamWithParamsFunc = origFunc }()

	// Create command with flags set (should override config)
	cmd := SteamCmd{
		SteamID: "flag-steam-id",
		APIKey:  "flag-api-key",
	}

	err := cmd.Run()
	require.NoError(t, err)
	assert.True(t, mockCalled)
}

func TestSteamCmd_Run_PropagatesError(t *testing.T) {
	testutil.SetTestConfig(t)

	expectedErr := fmt.Errorf("mock error from parser")
	mockFunc := func(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string) error {
		return expectedErr
	}

	origFunc := ParseSteamWithParamsFunc
	ParseSteamWithParamsFunc = mockFunc
	defer func() { ParseSteamWithParamsFunc = origFunc }()

	cmd := SteamCmd{
		SteamID: "12345",
		APIKey:  "test-key",
	}

	err := cmd.Run()
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
