package tmdb

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/stretchr/testify/require"
)

func TestCacheTMDBValueStoresPayload(t *testing.T) {
	setupTMDBCache(t)

	client := NewClient("key")
	payload := &CachedMetadata{Metadata: &Metadata{TMDBID: 42, TMDBType: "movie"}}
	client.cacheTMDBValue("metadata_movie_42", payload)

	cacheDB, err := cache.GetGlobalCache()
	require.NoError(t, err)

	cached, fromCache, err := cacheDB.Get("tmdb_cache", "metadata_movie_42", 24*time.Hour)
	require.NoError(t, err)
	require.True(t, fromCache)

	var decoded CachedMetadata
	require.NoError(t, json.Unmarshal([]byte(cached), &decoded))
	require.NotNil(t, decoded.Metadata)
	require.Equal(t, 42, decoded.Metadata.TMDBID)
	require.Equal(t, "movie", decoded.Metadata.TMDBType)
}
