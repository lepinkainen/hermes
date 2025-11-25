package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/spf13/viper"
)

type TestData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func setupTestCache(t *testing.T) (*CacheDB, string) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	// Register test_cache as a valid table name for tests
	ValidCacheTableNames["test_cache"] = true
	t.Cleanup(func() {
		delete(ValidCacheTableNames, "test_cache")
	})

	// Use testutil for sandboxed test environment
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()
	dbPath := filepath.Join(tempDir, "test_cache.db")

	// Create cache database
	cache, err := NewCacheDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create cache database: %v", err)
	}

	// Create test table
	testSchema := `
		CREATE TABLE IF NOT EXISTS test_cache (
			cache_key TEXT PRIMARY KEY NOT NULL,
			data TEXT NOT NULL,
			cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	if err := cache.CreateTable(testSchema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Set viper config for TTL
	viper.Set("cache.ttl", "1h")

	return cache, dbPath
}

func withGlobalCache(t *testing.T, cache *CacheDB) {
	t.Helper()

	oldCache := globalCache
	globalCache = cache
	globalCacheOnce = sync.Once{}
	globalCacheOnce.Do(func() {})

	t.Cleanup(func() {
		globalCache = oldCache
		globalCacheOnce = sync.Once{}
	})
}

func setCachedAt(t *testing.T, cache *CacheDB, tableName, key string, at time.Time) {
	t.Helper()

	if _, err := cache.db.Exec("UPDATE "+tableName+" SET cached_at = ? WHERE cache_key = ?", at.UTC(), key); err != nil {
		t.Fatalf("Failed to update cached_at: %v", err)
	}
}

func TestGetOrFetch_CacheHit(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Pre-populate cache
	testKey := "test-key"
	testData := TestData{ID: 1, Name: "Test"}

	// Store in cache directly
	jsonData := `{"id":1,"name":"Test"}`
	if err := cache.Set("test_cache", testKey, jsonData); err != nil {
		t.Fatalf("Failed to pre-populate cache: %v", err)
	}

	// Override global cache for this test  - needs to happen BEFORE calling GetOrFetch
	withGlobalCache(t, cache)

	// Test GetOrFetch
	fetchCalled := false
	fetchFunc := func() (TestData, error) {
		fetchCalled = true
		return TestData{}, nil
	}

	result, fromCache, err := GetOrFetch("test_cache", testKey, fetchFunc)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !fromCache {
		t.Error("Expected fromCache to be true")
	}
	if fetchCalled {
		t.Error("Expected fetch function not to be called")
	}
	if result.ID != testData.ID || result.Name != testData.Name {
		t.Errorf("Expected %+v, got %+v", testData, result)
	}
}

func TestGetOrFetch_CacheMiss(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Override global cache for this test
	withGlobalCache(t, cache)

	testKey := "test-key"
	expectedData := TestData{ID: 2, Name: "Fetched"}

	// Test GetOrFetch with cache miss
	fetchCalled := 0
	fetchFunc := func() (TestData, error) {
		fetchCalled++
		return expectedData, nil
	}

	result, fromCache, err := GetOrFetch("test_cache", testKey, fetchFunc)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false")
	}
	if fetchCalled != 1 {
		t.Errorf("Expected fetch function to be called once, got %d", fetchCalled)
	}
	if result.ID != expectedData.ID || result.Name != expectedData.Name {
		t.Errorf("Expected %+v, got %+v", expectedData, result)
	}

	// Verify data was cached
	if !cache.CacheExists("test_cache", testKey) {
		t.Error("Expected cache entry to be created")
	}

	// Second call should hit cache and avoid fetch
	result, fromCache, err = GetOrFetch("test_cache", testKey, fetchFunc)
	if err != nil {
		t.Fatalf("Expected no error on second call, got %v", err)
	}
	if !fromCache {
		t.Error("Expected second call to return from cache")
	}
	if fetchCalled != 1 {
		t.Errorf("Expected fetch not to be called again, got %d calls", fetchCalled)
	}
	if result.ID != expectedData.ID || result.Name != expectedData.Name {
		t.Errorf("Expected %+v from cache, got %+v", expectedData, result)
	}
}

func TestGetOrFetch_RespectsTTLExpiration(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	withGlobalCache(t, cache)

	testKey := "test-key"
	staleData := `{"id":1,"name":"stale"}`
	freshData := TestData{ID: 2, Name: "Fresh"}

	if err := cache.Set("test_cache", testKey, staleData); err != nil {
		t.Fatalf("Failed to seed stale cache: %v", err)
	}
	setCachedAt(t, cache, "test_cache", testKey, time.Now().Add(-2*time.Hour))

	viper.Set("cache.ttl", "1h")

	fetchCalled := 0
	fetchFunc := func() (TestData, error) {
		fetchCalled++
		return freshData, nil
	}

	result, fromCache, err := GetOrFetch("test_cache", testKey, fetchFunc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Fatal("Expected cache miss due to TTL expiration")
	}
	if fetchCalled != 1 {
		t.Fatalf("Expected fetch to be called once, got %d", fetchCalled)
	}
	if result.ID != freshData.ID || result.Name != freshData.Name {
		t.Fatalf("Expected fresh data, got %+v", result)
	}

	cached, cachedHit, err := cache.Get("test_cache", testKey, time.Hour)
	if err != nil {
		t.Fatalf("Expected cached data to be stored, got error %v", err)
	}
	if !cachedHit {
		t.Fatal("Expected cached entry after refresh")
	}

	var cachedData TestData
	if err := json.Unmarshal([]byte(cached), &cachedData); err != nil {
		t.Fatalf("Failed to unmarshal cached data: %v", err)
	}
	if cachedData != freshData {
		t.Fatalf("Expected cached data %+v, got %+v", freshData, cachedData)
	}
}

func TestGetOrFetch_FetchError(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Override global cache for this test
	withGlobalCache(t, cache)

	testKey := "test-key"

	// Test GetOrFetch with fetch error
	fetchFunc := func() (TestData, error) {
		return TestData{}, &testError{"fetch failed"}
	}

	result, fromCache, err := GetOrFetch("test_cache", testKey, fetchFunc)

	// Assertions
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if fromCache {
		t.Error("Expected fromCache to be false")
	}
	if result.ID != 0 || result.Name != "" {
		t.Errorf("Expected zero value, got %+v", result)
	}
}

func TestCacheDB_GetSet(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	testKey := "test-key"
	testData := `{"id":1,"name":"Test"}`

	// Test Set
	err := cache.Set("test_cache", testKey, testData)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test Get
	data, fromCache, err := cache.Get("test_cache", testKey, time.Hour)
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}
	if !fromCache {
		t.Error("Expected fromCache to be true")
	}
	if data != testData {
		t.Errorf("Expected %s, got %s", testData, data)
	}
}

func TestCacheDB_GetExpired(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	testKey := "test-key"
	testData := `{"id":1,"name":"Test"}`

	// Set cache
	err := cache.Set("test_cache", testKey, testData)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	setCachedAt(t, cache, "test_cache", testKey, time.Now().Add(-2*time.Hour))

	data, fromCache, err := cache.Get("test_cache", testKey, time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false for expired cache")
	}
	if data != "" {
		t.Errorf("Expected empty string for expired cache, got %s", data)
	}
}

func TestCacheDB_ClearExpired(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Add some test entries
	_ = cache.Set("test_cache", "key1", `{"id":1}`)
	_ = cache.Set("test_cache", "key2", `{"id":2}`)
	_ = cache.Set("test_cache", "key3", `{"id":3}`)

	setCachedAt(t, cache, "test_cache", "key1", time.Now().Add(-2*time.Hour))
	setCachedAt(t, cache, "test_cache", "key2", time.Now().Add(-30*time.Minute))

	err := cache.ClearExpired("test_cache", 45*time.Minute)
	if err != nil {
		t.Fatalf("Failed to clear expired cache: %v", err)
	}

	if cache.CacheExists("test_cache", "key1") {
		t.Error("Expected key1 to be cleared")
	}
	if !cache.CacheExists("test_cache", "key2") {
		t.Error("Expected key2 to remain")
	}
	if !cache.CacheExists("test_cache", "key3") {
		t.Error("Expected key3 to remain")
	}
}

func TestCacheDB_ClearAll(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Add some test entries
	_ = cache.Set("test_cache", "key1", `{"id":1}`)
	_ = cache.Set("test_cache", "key2", `{"id":2}`)
	_ = cache.Set("test_cache", "key3", `{"id":3}`)

	// Clear all entries
	err := cache.ClearAll("test_cache")
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify all entries were removed
	if cache.CacheExists("test_cache", "key1") {
		t.Error("Expected key1 to be cleared")
	}
	if cache.CacheExists("test_cache", "key2") {
		t.Error("Expected key2 to be cleared")
	}
	if cache.CacheExists("test_cache", "key3") {
		t.Error("Expected key3 to be cleared")
	}
}

func TestCacheDB_CacheExists(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	existingKey := "existing"
	nonExistingKey := "non-existing"

	// Create one cache entry
	_ = cache.Set("test_cache", existingKey, `{"id":1}`)

	// Test existing entry
	if !cache.CacheExists("test_cache", existingKey) {
		t.Error("Expected cache to exist for existing key")
	}

	// Test non-existing entry
	if cache.CacheExists("test_cache", nonExistingKey) {
		t.Error("Expected cache not to exist for non-existing key")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestCacheDB_InvalidateSource(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Add some test entries
	_ = cache.Set("test_cache", "key1", `{"id":1}`)
	_ = cache.Set("test_cache", "key2", `{"id":2}`)
	_ = cache.Set("test_cache", "key3", `{"id":3}`)

	// Verify entries exist
	if !cache.CacheExists("test_cache", "key1") {
		t.Error("Expected key1 to exist before invalidation")
	}

	// Invalidate the entire table
	rowsDeleted, err := cache.InvalidateSource("test_cache")
	if err != nil {
		t.Fatalf("Failed to invalidate cache: %v", err)
	}

	// Should have deleted 3 rows
	if rowsDeleted != 3 {
		t.Errorf("Expected 3 rows deleted, got %d", rowsDeleted)
	}

	// Verify all entries were removed
	if cache.CacheExists("test_cache", "key1") {
		t.Error("Expected key1 to be invalidated")
	}
	if cache.CacheExists("test_cache", "key2") {
		t.Error("Expected key2 to be invalidated")
	}
	if cache.CacheExists("test_cache", "key3") {
		t.Error("Expected key3 to be invalidated")
	}
}

func TestCacheDB_InvalidateSource_InvalidTable(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Try to invalidate an invalid table name
	_, err := cache.InvalidateSource("invalid_table")
	if err == nil {
		t.Error("Expected error for invalid table name")
	}
}

func TestCacheDB_InvalidateSource_EmptyTable(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Invalidate empty table
	rowsDeleted, err := cache.InvalidateSource("test_cache")
	if err != nil {
		t.Fatalf("Failed to invalidate empty cache: %v", err)
	}

	// Should have deleted 0 rows
	if rowsDeleted != 0 {
		t.Errorf("Expected 0 rows deleted from empty table, got %d", rowsDeleted)
	}
}
