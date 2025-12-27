package tmdb

import (
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
)

func setupTMDBCache(t *testing.T) {
	t.Helper()

	// Reset any existing global cache to ensure isolation between tests
	if err := cache.ResetGlobalCache(); err != nil {
		t.Fatalf("Failed to reset global cache: %v", err)
	}

	viper.Reset()
	t.Cleanup(func() {
		// Clean up the global cache on test completion
		_ = cache.ResetGlobalCache()
		viper.Reset()
	})

	// Use testutil for sandboxed test environment
	env := testutil.NewTestEnv(t)
	tmpDir := env.RootDir()

	viper.Set("cache.dbfile", filepath.Join(tmpDir, "tmdb-cache.db"))
	viper.Set("cache.ttl", "24h")

	cacheDB, err := cache.GetGlobalCache()
	if err != nil {
		t.Fatalf("Failed to init cache: %v", err)
	}
	if err := cacheDB.ClearAll("tmdb_cache"); err != nil {
		t.Fatalf("Failed to reset tmdb_cache table: %v", err)
	}
}
