package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestEnv_Path(t *testing.T) {
	env := NewTestEnv(t)

	// Test basic path
	path := env.Path("subdir", "file.txt")
	assert.True(t, filepath.IsAbs(path))
	assert.Contains(t, path, "subdir")
	assert.Contains(t, path, "file.txt")
}

func TestTestEnv_Path_WithinSandbox(t *testing.T) {
	env := NewTestEnv(t)

	// These should work
	_ = env.Path("subdir")
	_ = env.Path("subdir", "nested")
	_ = env.Path("file.txt")
}

func TestTestEnv_WriteReadFile(t *testing.T) {
	env := NewTestEnv(t)

	content := []byte("test content")
	env.WriteFile("test.txt", content)

	read := env.ReadFile("test.txt")
	assert.Equal(t, content, read)
}

func TestTestEnv_WriteReadFileString(t *testing.T) {
	env := NewTestEnv(t)

	content := "test string content"
	env.WriteFileString("test.txt", content)

	read := env.ReadFileString("test.txt")
	assert.Equal(t, content, read)
}

func TestTestEnv_MkdirAll(t *testing.T) {
	env := NewTestEnv(t)

	env.MkdirAll("nested/dir/structure")

	path := env.Path("nested/dir/structure")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestTestEnv_FileExists(t *testing.T) {
	env := NewTestEnv(t)

	assert.False(t, env.FileExists("nonexistent.txt"))

	env.WriteFileString("exists.txt", "content")
	assert.True(t, env.FileExists("exists.txt"))
}

func TestTestEnv_RequireFileExists(t *testing.T) {
	env := NewTestEnv(t)
	env.WriteFileString("exists.txt", "content")

	// This should not panic
	env.RequireFileExists("exists.txt")
}

func TestTestEnv_RequireFileNotExists(t *testing.T) {
	env := NewTestEnv(t)

	// This should not panic
	env.RequireFileNotExists("nonexistent.txt")
}

func TestTestEnv_CopyFile(t *testing.T) {
	env := NewTestEnv(t)

	// Create a source file outside the env
	srcFile, err := os.CreateTemp("", "test-source-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(srcFile.Name()) }()

	content := []byte("source content")
	_, err = srcFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, srcFile.Close())

	// Copy to env
	env.CopyFile(srcFile.Name(), "copied.txt")

	read := env.ReadFile("copied.txt")
	assert.Equal(t, content, read)
}

func TestTestEnv_ListFiles(t *testing.T) {
	env := NewTestEnv(t)

	env.WriteFileString("file1.txt", "1")
	env.WriteFileString("file2.txt", "2")
	env.MkdirAll("subdir")

	files := env.ListFiles(".")
	assert.Len(t, files, 3)
	assert.Contains(t, files, "file1.txt")
	assert.Contains(t, files, "file2.txt")
	assert.Contains(t, files, "subdir")
}

func TestTestEnv_AssertFileContains(t *testing.T) {
	env := NewTestEnv(t)

	env.WriteFileString("test.txt", "hello world")
	env.AssertFileContains("test.txt", "world")
}

func TestTestEnv_AssertFileEquals(t *testing.T) {
	env := NewTestEnv(t)

	content := "exact content"
	env.WriteFileString("test.txt", content)
	env.AssertFileEquals("test.txt", content)
}

func TestTestEnv_SetEnv(t *testing.T) {
	env := NewTestEnv(t)

	// Set a test environment variable
	env.SetEnv("TEST_VAR", "test_value")
	assert.Equal(t, "test_value", os.Getenv("TEST_VAR"))
}

func TestTestEnv_SetEnv_Cleanup(t *testing.T) {
	// Set an initial value
	require.NoError(t, os.Setenv("CLEANUP_TEST_VAR", "original"))
	defer func() { _ = os.Unsetenv("CLEANUP_TEST_VAR") }()

	t.Run("inner", func(t *testing.T) {
		env := NewTestEnv(t)
		env.SetEnv("CLEANUP_TEST_VAR", "modified")
		assert.Equal(t, "modified", os.Getenv("CLEANUP_TEST_VAR"))
	})

	// After the inner test, the value should be restored
	assert.Equal(t, "original", os.Getenv("CLEANUP_TEST_VAR"))
}

func TestTestEnv_Remove(t *testing.T) {
	env := NewTestEnv(t)

	env.WriteFileString("to-remove.txt", "content")
	assert.True(t, env.FileExists("to-remove.txt"))

	env.Remove("to-remove.txt")
	assert.False(t, env.FileExists("to-remove.txt"))
}

func TestTestEnv_RemoveAll(t *testing.T) {
	env := NewTestEnv(t)

	env.MkdirAll("dir/nested")
	env.WriteFileString("dir/nested/file.txt", "content")
	assert.True(t, env.FileExists("dir/nested/file.txt"))

	env.RemoveAll("dir")
	assert.False(t, env.FileExists("dir"))
}

func TestTestEnv_String(t *testing.T) {
	env := NewTestEnv(t)

	str := env.String()
	assert.Contains(t, str, "TestEnv")
	assert.Contains(t, str, env.RootDir())
}

// GoldenHelper tests

func TestGoldenHelper_AssertGolden(t *testing.T) {
	env := NewTestEnv(t)

	// Create a golden directory
	goldenDir := env.Path("golden")
	env.MkdirAll("golden")

	// Create a golden file
	expectedContent := []byte("expected content")
	env.WriteFile("golden/test.golden", expectedContent)

	// Create helper
	golden := NewGoldenHelper(t, goldenDir)

	// Test assertion
	golden.AssertGolden("test.golden", expectedContent)
}

func TestGoldenHelper_AssertGoldenString(t *testing.T) {
	env := NewTestEnv(t)

	goldenDir := env.Path("golden")
	env.MkdirAll("golden")

	expectedContent := "expected string content"
	env.WriteFileString("golden/test.golden", expectedContent)

	golden := NewGoldenHelper(t, goldenDir)
	golden.AssertGoldenString("test.golden", expectedContent)
}

func TestGoldenHelper_GoldenPath(t *testing.T) {
	golden := NewGoldenHelper(t, "/some/golden/dir")

	path := golden.GoldenPath("test.golden")
	assert.Equal(t, "/some/golden/dir/test.golden", path)
}

func TestGoldenHelper_IsUpdateMode(t *testing.T) {
	// Without UPDATE_GOLDEN env var
	golden := NewGoldenHelper(t, "testdata")
	assert.False(t, golden.IsUpdateMode())
}

func TestGoldenHelper_MustReadGolden(t *testing.T) {
	env := NewTestEnv(t)

	goldenDir := env.Path("golden")
	env.MkdirAll("golden")

	content := []byte("golden content")
	env.WriteFile("golden/test.golden", content)

	golden := NewGoldenHelper(t, goldenDir)
	read := golden.MustReadGolden("test.golden")
	assert.Equal(t, content, read)
}

func TestGoldenHelper_Exists(t *testing.T) {
	env := NewTestEnv(t)

	goldenDir := env.Path("golden")
	env.MkdirAll("golden")

	golden := NewGoldenHelper(t, goldenDir)
	assert.False(t, golden.Exists("nonexistent.golden"))

	env.WriteFileString("golden/exists.golden", "content")
	assert.True(t, golden.Exists("exists.golden"))
}

// Config management tests

func TestResetConfig(t *testing.T) {
	// Save current state
	origOverwrite := config.OverwriteFiles
	origUpdateCovers := config.UpdateCovers

	t.Run("inner", func(t *testing.T) {
		ResetConfig(t)

		// Modify config
		config.OverwriteFiles = !origOverwrite
		config.UpdateCovers = !origUpdateCovers

		// Verify modified
		assert.NotEqual(t, origOverwrite, config.OverwriteFiles)
		assert.NotEqual(t, origUpdateCovers, config.UpdateCovers)
	})

	// After inner test, config should be restored
	assert.Equal(t, origOverwrite, config.OverwriteFiles)
	assert.Equal(t, origUpdateCovers, config.UpdateCovers)
}

func TestSetTestConfig(t *testing.T) {
	// Save current state
	origOverwrite := config.OverwriteFiles
	origUpdateCovers := config.UpdateCovers
	origTMDBKey := config.TMDBAPIKey
	origOMDBKey := config.OMDBAPIKey

	t.Run("inner", func(t *testing.T) {
		SetTestConfig(t)

		// Verify test defaults are set
		assert.True(t, config.OverwriteFiles)
		assert.False(t, config.UpdateCovers)
		assert.Equal(t, "test-tmdb-key", config.TMDBAPIKey)
		assert.Equal(t, "test-omdb-key", config.OMDBAPIKey)
	})

	// After inner test, config should be restored
	assert.Equal(t, origOverwrite, config.OverwriteFiles)
	assert.Equal(t, origUpdateCovers, config.UpdateCovers)
	assert.Equal(t, origTMDBKey, config.TMDBAPIKey)
	assert.Equal(t, origOMDBKey, config.OMDBAPIKey)
}

func TestSetTestConfigWithOptions(t *testing.T) {
	origOverwrite := config.OverwriteFiles

	t.Run("inner", func(t *testing.T) {
		SetTestConfigWithOptions(t,
			WithOverwriteFiles(false),
			WithTMDBAPIKey("custom-key"),
		)

		assert.False(t, config.OverwriteFiles)
		assert.Equal(t, "custom-key", config.TMDBAPIKey)
	})

	assert.Equal(t, origOverwrite, config.OverwriteFiles)
}

func TestSetViperValue(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	t.Run("inner", func(t *testing.T) {
		SetViperValue(t, "test.key", "test-value")
		assert.Equal(t, "test-value", viper.GetString("test.key"))
	})
}

func TestSetupTestCache(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	env := NewTestEnv(t)
	cacheDir := SetupTestCache(t, env)

	assert.DirExists(t, cacheDir)
	assert.Contains(t, viper.GetString("cache.dbfile"), "test-cache.db")
	assert.Equal(t, "24h", viper.GetString("cache.ttl"))
}

func TestSaveRestoreConfigState(t *testing.T) {
	// Set known values
	config.OverwriteFiles = true
	config.UpdateCovers = true
	config.TMDBAPIKey = "saved-tmdb"
	config.OMDBAPIKey = "saved-omdb"

	// Save state
	state := SaveConfigState()

	// Modify
	config.OverwriteFiles = false
	config.UpdateCovers = false
	config.TMDBAPIKey = "modified"
	config.OMDBAPIKey = "modified"

	// Restore
	RestoreConfigState(state)

	// Verify restored
	assert.True(t, config.OverwriteFiles)
	assert.True(t, config.UpdateCovers)
	assert.Equal(t, "saved-tmdb", config.TMDBAPIKey)
	assert.Equal(t, "saved-omdb", config.OMDBAPIKey)
}
