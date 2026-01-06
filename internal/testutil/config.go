package testutil

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/spf13/viper"
)

// ConfigState holds the state of the config package variables.
type ConfigState struct {
	OverwriteFiles bool
	UpdateCovers   bool
	TMDBAPIKey     string
	OMDBAPIKey     string
}

// SaveConfigState captures the current state of config package variables.
func SaveConfigState() ConfigState {
	return ConfigState{
		OverwriteFiles: config.OverwriteFiles,
		UpdateCovers:   config.UpdateCovers,
		TMDBAPIKey:     config.TMDBAPIKey,
		OMDBAPIKey:     config.OMDBAPIKey,
	}
}

// RestoreConfigState restores the config package variables to a saved state.
func RestoreConfigState(state ConfigState) {
	config.OverwriteFiles = state.OverwriteFiles
	config.UpdateCovers = state.UpdateCovers
	config.TMDBAPIKey = state.TMDBAPIKey
	config.OMDBAPIKey = state.OMDBAPIKey
}

// ResetConfig saves the current config state and schedules restoration
// when the test completes. It also resets viper.
func ResetConfig(t *testing.T) {
	t.Helper()

	// Save current config state
	state := SaveConfigState()

	// Reset viper
	viper.Reset()

	// Schedule restoration on test cleanup
	t.Cleanup(func() {
		RestoreConfigState(state)
		viper.Reset()
	})
}

// SetTestConfig sets up a test configuration with common defaults.
// It saves the current state and restores it when the test completes.
func SetTestConfig(t *testing.T) {
	t.Helper()

	// Save current config state
	state := SaveConfigState()

	// Reset viper and set test defaults
	viper.Reset()

	// Set common test defaults
	config.OverwriteFiles = true
	config.UpdateCovers = false
	config.TMDBAPIKey = "test-tmdb-key"
	config.OMDBAPIKey = "test-omdb-key"

	// Schedule restoration on test cleanup
	t.Cleanup(func() {
		RestoreConfigState(state)
		viper.Reset()
	})
}

// SetTestConfigOption is a functional option for configuring test config.
type SetTestConfigOption func(*testConfigOptions)

type testConfigOptions struct {
	overwriteFiles bool
	updateCovers   bool
	tmdbAPIKey     string
	omdbAPIKey     string
}

// WithOverwriteFiles sets the OverwriteFiles option.
func WithOverwriteFiles(v bool) SetTestConfigOption {
	return func(o *testConfigOptions) {
		o.overwriteFiles = v
	}
}

// WithUpdateCovers sets the UpdateCovers option.
func WithUpdateCovers(v bool) SetTestConfigOption {
	return func(o *testConfigOptions) {
		o.updateCovers = v
	}
}

// WithTMDBAPIKey sets the TMDB API key.
func WithTMDBAPIKey(key string) SetTestConfigOption {
	return func(o *testConfigOptions) {
		o.tmdbAPIKey = key
	}
}

// WithOMDBAPIKey sets the OMDB API key.
func WithOMDBAPIKey(key string) SetTestConfigOption {
	return func(o *testConfigOptions) {
		o.omdbAPIKey = key
	}
}

// SetTestConfigWithOptions sets up a test configuration with custom options.
// It saves the current state and restores it when the test completes.
func SetTestConfigWithOptions(t *testing.T, opts ...SetTestConfigOption) {
	t.Helper()

	// Save current config state
	state := SaveConfigState()

	// Reset viper
	viper.Reset()

	// Set defaults
	options := testConfigOptions{
		overwriteFiles: true,
		updateCovers:   false,
		tmdbAPIKey:     "test-tmdb-key",
		omdbAPIKey:     "test-omdb-key",
	}

	// Apply options
	for _, opt := range opts {
		opt(&options)
	}

	// Apply to config
	config.OverwriteFiles = options.overwriteFiles
	config.UpdateCovers = options.updateCovers
	config.TMDBAPIKey = options.tmdbAPIKey
	config.OMDBAPIKey = options.omdbAPIKey

	// Schedule restoration on test cleanup
	t.Cleanup(func() {
		RestoreConfigState(state)
		viper.Reset()
	})
}

// SetViperValue sets a viper configuration value and schedules cleanup.
func SetViperValue(t *testing.T, key string, value any) {
	t.Helper()

	// Get the old value (if any)
	oldValue := viper.Get(key)
	hadValue := viper.IsSet(key)

	// Set the new value
	viper.Set(key, value)

	// Schedule cleanup
	t.Cleanup(func() {
		if hadValue {
			viper.Set(key, oldValue)
		}
		// Note: viper doesn't have an Unset function, so we can't
		// restore the "unset" state. This is a known limitation.
	})
}

// SetupTestCache configures viper for test caching with a temporary directory.
// It creates the cache directory and sets up viper configuration.
func SetupTestCache(t *testing.T, env *TestEnv) string {
	t.Helper()

	// Create cache directory
	cacheDir := env.Path("cache")
	env.MkdirAll("cache")

	// Configure viper
	viper.Set("cache.dbfile", env.Path("cache", "test-cache.db"))
	viper.Set("cache.ttl", "24h")

	return cacheDir
}

// SetupDatasetteDB configures datasette database for E2E tests.
// It creates a temporary database file and configures viper with automatic cleanup.
// Returns the database path.
func SetupDatasetteDB(t *testing.T, env *TestEnv) string {
	t.Helper()

	dbPath := env.Path("test.db")

	// Configure datasette using SetViperValue for automatic cleanup
	SetViperValue(t, "datasette.enabled", true)
	SetViperValue(t, "datasette.dbfile", dbPath)

	return dbPath
}

// SetupE2EMarkdownOutput configures markdown output directory for E2E tests.
// It sets the markdown output directory to the test environment and configures cleanup.
func SetupE2EMarkdownOutput(t *testing.T, env *TestEnv) {
	t.Helper()

	// Set markdown output to test directory with automatic cleanup
	SetViperValue(t, "markdownoutputdir", env.RootDir())
}
