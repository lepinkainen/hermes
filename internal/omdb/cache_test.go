package omdb

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type omdbTestRecord struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
}

func setupOMDBCache(t *testing.T) *cache.CacheDB {
	t.Helper()

	ResetRateLimit()

	require.NoError(t, cache.ResetGlobalCache())
	viper.Reset()

	t.Cleanup(func() {
		ResetRateLimit()
		_ = cache.ResetGlobalCache()
		viper.Reset()
	})

	env := testutil.NewTestEnv(t)
	viper.Set("cache.dbfile", env.Path("omdb-cache.db"))
	viper.Set("cache.ttl", "24h")

	cacheDB, err := cache.GetGlobalCache()
	require.NoError(t, err)
	require.NoError(t, cacheDB.ClearAll("omdb_cache"))

	return cacheDB
}

func TestGetCached_CachesData(t *testing.T) {
	setupOMDBCache(t)

	fetchCalls := 0
	fetcher := func() (*omdbTestRecord, error) {
		fetchCalls++
		return &omdbTestRecord{Title: "Spirited Away", Year: 2001}, nil
	}

	result, fromCache, err := GetCached("tt0245429", fetcher)
	require.NoError(t, err)
	assert.False(t, fromCache)
	assert.Equal(t, "Spirited Away", result.Title)
	assert.Equal(t, 1, fetchCalls)

	result, fromCache, err = GetCached("tt0245429", fetcher)
	require.NoError(t, err)
	assert.True(t, fromCache)
	assert.Equal(t, "Spirited Away", result.Title)
	assert.Equal(t, 1, fetchCalls)
}

func TestGetCached_RateLimitErrorMarksLimit(t *testing.T) {
	setupOMDBCache(t)

	fetcher := func() (*omdbTestRecord, error) {
		return nil, errors.NewRateLimitError("rate limit hit")
	}

	_, _, err := GetCached("tt1234567", fetcher)
	require.Error(t, err)
	assert.True(t, errors.IsRateLimitError(err))
	assert.False(t, RequestsAllowed())
}

func TestGetCached_FetcherErrorDoesNotMarkRateLimit(t *testing.T) {
	setupOMDBCache(t)

	fetcher := func() (*omdbTestRecord, error) {
		return nil, assert.AnError
	}

	_, _, err := GetCached("tt7654321", fetcher)
	require.Error(t, err)
	assert.True(t, RequestsAllowed())
}

func TestSeedCacheByID_StoresJSON(t *testing.T) {
	cacheDB := setupOMDBCache(t)

	record := omdbTestRecord{Title: "The Matrix", Year: 1999}
	require.NoError(t, SeedCacheByID("tt0133093", record))

	cached, fromCache, err := cacheDB.Get("omdb_cache", "tt0133093", 24*time.Hour)
	require.NoError(t, err)
	assert.True(t, fromCache)

	var decoded omdbTestRecord
	require.NoError(t, json.Unmarshal([]byte(cached), &decoded))
	assert.Equal(t, record, decoded)
}
