package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func resetConfigState(t *testing.T) {
	originalOverwrite := OverwriteFiles
	originalUpdate := UpdateCovers
	originalTMDBKey := TMDBAPIKey
	originalOMDBKey := OMDBAPIKey

	t.Cleanup(func() {
		OverwriteFiles = originalOverwrite
		UpdateCovers = originalUpdate
		TMDBAPIKey = originalTMDBKey
		OMDBAPIKey = originalOMDBKey
		viper.Reset()
	})

	viper.Reset()
}

func TestSetOverwriteFiles(t *testing.T) {
	resetConfigState(t)

	// Save the original value to restore after the test
	originalValue := OverwriteFiles

	testCases := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "set to true",
			input:    true,
			expected: true,
		},
		{
			name:     "set to false",
			input:    false,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the value
			SetOverwriteFiles(tc.input)

			// Check that the global variable was updated
			assert.Equal(t, tc.expected, OverwriteFiles)
		})
	}

	// Restore the original value
	OverwriteFiles = originalValue
}

func TestSetUpdateCovers(t *testing.T) {
	resetConfigState(t)

	originalValue := UpdateCovers
	defer func() {
		UpdateCovers = originalValue
	}()

	testCases := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "set to true",
			input:    true,
			expected: true,
		},
		{
			name:     "set to false",
			input:    false,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			SetUpdateCovers(tc.input)
			assert.Equal(t, tc.expected, UpdateCovers)
		})
	}
}

func TestInitConfigSetsDefaults(t *testing.T) {
	resetConfigState(t)

	InitConfig()

	assert.False(t, OverwriteFiles)
	assert.False(t, UpdateCovers)
	assert.Empty(t, TMDBAPIKey)
	assert.Empty(t, OMDBAPIKey)

	assert.Equal(t, "./markdown/", viper.GetString("MarkdownOutputDir"))
	assert.Equal(t, "./json/", viper.GetString("JSONOutputDir"))
	assert.False(t, viper.GetBool("OverwriteFiles"))
	assert.False(t, viper.GetBool("UpdateCovers"))
}

func TestInitConfigUsesExistingValues(t *testing.T) {
	resetConfigState(t)

	t.Setenv("OVERWRITEFILES", "true")
	t.Setenv("UPDATECOVERS", "true")
	t.Setenv("TMDB_API_KEY", "tmdb-key")
	t.Setenv("OMDBAPIKEY", "omdb-key")

	viper.AutomaticEnv()
	_ = viper.BindEnv("TMDBAPIKey", "TMDB_API_KEY")

	InitConfig()

	assert.True(t, OverwriteFiles)
	assert.True(t, UpdateCovers)
	assert.Equal(t, "tmdb-key", TMDBAPIKey)
	assert.Equal(t, "omdb-key", OMDBAPIKey)
}
