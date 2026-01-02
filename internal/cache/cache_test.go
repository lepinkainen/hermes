package cache

import (
	"database/sql"
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
	if err := cache.Set("test_cache", testKey, jsonData, 0); err != nil {
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

	if err := cache.Set("test_cache", testKey, staleData, 0); err != nil {
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
	err := cache.Set("test_cache", testKey, testData, 0)
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
	err := cache.Set("test_cache", testKey, testData, 0)
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
	_ = cache.Set("test_cache", "key1", `{"id":1}`, 0)
	_ = cache.Set("test_cache", "key2", `{"id":2}`, 0)
	_ = cache.Set("test_cache", "key3", `{"id":3}`, 0)

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
	_ = cache.Set("test_cache", "key1", `{"id":1}`, 0)
	_ = cache.Set("test_cache", "key2", `{"id":2}`, 0)
	_ = cache.Set("test_cache", "key3", `{"id":3}`, 0)

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
	_ = cache.Set("test_cache", existingKey, `{"id":1}`, 0)

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
	_ = cache.Set("test_cache", "key1", `{"id":1}`, 0)
	_ = cache.Set("test_cache", "key2", `{"id":2}`, 0)
	_ = cache.Set("test_cache", "key3", `{"id":3}`, 0)

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

func TestSelectNegativeCacheTTL(t *testing.T) {
	type CachedResult struct {
		Data     *string `json:"data"`
		NotFound bool    `json:"not_found"`
	}

	// Test with "not found" result
	notFoundResult := CachedResult{Data: nil, NotFound: true}
	selector := SelectNegativeCacheTTL(func(r CachedResult) bool {
		return r.NotFound
	})

	ttl := selector(notFoundResult)
	if ttl != NegativeCacheTTL {
		t.Errorf("Expected NegativeCacheTTL (%v) for not found result, got %v", NegativeCacheTTL, ttl)
	}
	if ttl != 168*time.Hour {
		t.Errorf("Expected 168h for not found result, got %v", ttl)
	}

	// Test with successful result
	data := "test data"
	foundResult := CachedResult{Data: &data, NotFound: false}
	ttl = selector(foundResult)
	if ttl != DefaultCacheTTL {
		t.Errorf("Expected DefaultCacheTTL (%v) for found result, got %v", DefaultCacheTTL, ttl)
	}
	if ttl != 720*time.Hour {
		t.Errorf("Expected 720h for found result, got %v", ttl)
	}
}

func TestCacheDB_QueryRow(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Insert test data
	testKey := "test-key"
	testData := `{"id":123,"name":"QueryRow Test"}`
	err := cache.Set("test_cache", testKey, testData, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Use QueryRow to retrieve the data
	var cachedKey string
	var cachedData string
	row := cache.QueryRow("SELECT cache_key, data FROM test_cache WHERE cache_key = ?", testKey)
	err = row.Scan(&cachedKey, &cachedData)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}

	// Verify the results
	if cachedKey != testKey {
		t.Errorf("Expected cache_key %s, got %s", testKey, cachedKey)
	}
	if cachedData != testData {
		t.Errorf("Expected data %s, got %s", testData, cachedData)
	}
}

func TestCacheDB_QueryRow_NoResults(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Query for non-existent key
	var cachedData string
	row := cache.QueryRow("SELECT data FROM test_cache WHERE cache_key = ?", "nonexistent")
	err := row.Scan(&cachedData)

	// Should return sql.ErrNoRows
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestCacheDB_Exec(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Insert test data using Set
	testKey := "test-key"
	testData := `{"id":456,"name":"Exec Test"}`
	err := cache.Set("test_cache", testKey, testData, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Use Exec to update the data
	newData := `{"id":789,"name":"Updated"}`
	err = cache.Exec("UPDATE test_cache SET data = ? WHERE cache_key = ?", newData, testKey)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	// Verify the update
	var cachedData string
	row := cache.QueryRow("SELECT data FROM test_cache WHERE cache_key = ?", testKey)
	err = row.Scan(&cachedData)
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if cachedData != newData {
		t.Errorf("Expected data %s, got %s", newData, cachedData)
	}
}

func TestCacheDB_Exec_Delete(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Insert test data
	testKey := "test-key"
	testData := `{"id":999,"name":"Delete Test"}`
	err := cache.Set("test_cache", testKey, testData, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Verify it exists
	if !cache.CacheExists("test_cache", testKey) {
		t.Fatal("Expected cache entry to exist before deletion")
	}

	// Use Exec to delete the data
	err = cache.Exec("DELETE FROM test_cache WHERE cache_key = ?", testKey)
	if err != nil {
		t.Fatalf("Exec delete failed: %v", err)
	}

	// Verify it's deleted
	if cache.CacheExists("test_cache", testKey) {
		t.Error("Expected cache entry to be deleted")
	}
}

func TestGetOrFetchWithPolicy_CacheAll(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	withGlobalCache(t, cache)

	testKey := "policy-test-key"
	expectedData := TestData{ID: 100, Name: "Policy Test"}

	fetchCalled := 0
	fetchFunc := func() (TestData, error) {
		fetchCalled++
		return expectedData, nil
	}

	// nil policy means cache everything (default behavior)
	result, fromCache, err := GetOrFetchWithPolicy("test_cache", testKey, fetchFunc, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false on first call")
	}
	if fetchCalled != 1 {
		t.Errorf("Expected fetch to be called once, got %d", fetchCalled)
	}

	// Verify it was cached
	if !cache.CacheExists("test_cache", testKey) {
		t.Error("Expected data to be cached")
	}

	// Second call should hit cache
	result, fromCache, err = GetOrFetchWithPolicy("test_cache", testKey, fetchFunc, nil)
	if err != nil {
		t.Fatalf("Expected no error on second call, got %v", err)
	}
	if !fromCache {
		t.Error("Expected fromCache to be true on second call")
	}
	if fetchCalled != 1 {
		t.Errorf("Expected fetch not to be called again, got %d calls", fetchCalled)
	}
	if result.ID != expectedData.ID {
		t.Errorf("Expected data %+v, got %+v", expectedData, result)
	}
}

func TestGetOrFetchWithPolicy_SkipCaching(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	withGlobalCache(t, cache)

	testKey := "skip-cache-key"

	fetchCalled := 0
	fetchFunc := func() (TestData, error) {
		fetchCalled++
		return TestData{ID: 0, Name: ""}, nil // Empty result
	}

	// Policy: don't cache if ID is 0
	shouldCache := func(data TestData) bool {
		return data.ID != 0
	}

	_, fromCache, err := GetOrFetchWithPolicy("test_cache", testKey, fetchFunc, shouldCache)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false")
	}
	if fetchCalled != 1 {
		t.Errorf("Expected fetch to be called once, got %d", fetchCalled)
	}

	// Verify it was NOT cached
	if cache.CacheExists("test_cache", testKey) {
		t.Error("Expected data NOT to be cached per policy")
	}

	// Second call should fetch again (not cached)
	_, fromCache, err = GetOrFetchWithPolicy("test_cache", testKey, fetchFunc, shouldCache)
	if err != nil {
		t.Fatalf("Expected no error on second call, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false on second call")
	}
	if fetchCalled != 2 {
		t.Errorf("Expected fetch to be called twice (not cached), got %d calls", fetchCalled)
	}
}

func TestGetOrFetchWithPolicy_SelectiveCaching(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	withGlobalCache(t, cache)

	// Policy: only cache if Name is not empty
	shouldCache := func(data TestData) bool {
		return data.Name != ""
	}

	// Test with data that should be cached
	testKey1 := "cache-key-1"
	expectedData1 := TestData{ID: 1, Name: "Cached"}
	fetchFunc1 := func() (TestData, error) {
		return expectedData1, nil
	}

	_, _, err := GetOrFetchWithPolicy("test_cache", testKey1, fetchFunc1, shouldCache)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !cache.CacheExists("test_cache", testKey1) {
		t.Error("Expected data to be cached (Name is not empty)")
	}

	// Test with data that should NOT be cached
	testKey2 := "cache-key-2"
	expectedData2 := TestData{ID: 2, Name: ""}
	fetchFunc2 := func() (TestData, error) {
		return expectedData2, nil
	}

	_, _, err = GetOrFetchWithPolicy("test_cache", testKey2, fetchFunc2, shouldCache)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cache.CacheExists("test_cache", testKey2) {
		t.Error("Expected data NOT to be cached (Name is empty)")
	}
}

func TestGetOrFetchWithTTL_CustomTTL(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	withGlobalCache(t, cache)

	testKey := "ttl-test-key"
	expectedData := TestData{ID: 200, Name: "TTL Test"}

	fetchCalled := 0
	fetchFunc := func() (TestData, error) {
		fetchCalled++
		return expectedData, nil
	}

	// TTL selector: use 2 hours for this test
	ttlSelector := func(data TestData) time.Duration {
		return 2 * time.Hour
	}

	_, fromCache, err := GetOrFetchWithTTL("test_cache", testKey, fetchFunc, ttlSelector)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false on first call")
	}
	if fetchCalled != 1 {
		t.Errorf("Expected fetch to be called once, got %d", fetchCalled)
	}

	// Verify it was cached
	if !cache.CacheExists("test_cache", testKey) {
		t.Error("Expected data to be cached")
	}

	// Second call should hit cache
	_, fromCache, err = GetOrFetchWithTTL("test_cache", testKey, fetchFunc, ttlSelector)
	if err != nil {
		t.Fatalf("Expected no error on second call, got %v", err)
	}
	if !fromCache {
		t.Error("Expected fromCache to be true on second call")
	}
	if fetchCalled != 1 {
		t.Errorf("Expected fetch not to be called again, got %d calls", fetchCalled)
	}
}

func TestGetOrFetchWithTTL_NegativeCaching(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	withGlobalCache(t, cache)

	type CachedBook struct {
		Title    string
		NotFound bool
	}

	// Test not-found result
	testKeyNotFound := "book-not-found"
	fetchFunc1 := func() (CachedBook, error) {
		return CachedBook{Title: "", NotFound: true}, nil
	}

	ttlSelector := SelectNegativeCacheTTL(func(r CachedBook) bool {
		return r.NotFound
	})

	result, fromCache, err := GetOrFetchWithTTL("test_cache", testKeyNotFound, fetchFunc1, ttlSelector)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false on first call")
	}
	if !result.NotFound {
		t.Error("Expected NotFound to be true")
	}

	// Verify it was cached
	if !cache.CacheExists("test_cache", testKeyNotFound) {
		t.Error("Expected not-found result to be cached")
	}

	// Test found result
	testKeyFound := "book-found"
	fetchFunc2 := func() (CachedBook, error) {
		return CachedBook{Title: "The Great Book", NotFound: false}, nil
	}

	result, fromCache, err = GetOrFetchWithTTL("test_cache", testKeyFound, fetchFunc2, ttlSelector)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fromCache {
		t.Error("Expected fromCache to be false on first call")
	}
	if result.NotFound {
		t.Error("Expected NotFound to be false")
	}
	if result.Title != "The Great Book" {
		t.Errorf("Expected title 'The Great Book', got '%s'", result.Title)
	}

	// Verify it was cached
	if !cache.CacheExists("test_cache", testKeyFound) {
		t.Error("Expected found result to be cached")
	}
}
