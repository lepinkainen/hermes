package cache

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

type TestData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func setupTestCache(t *testing.T) (*CacheDB, string) {
	t.Helper()

	// Register test_cache as a valid table name for tests
	ValidCacheTableNames["test_cache"] = true
	t.Cleanup(func() {
		delete(ValidCacheTableNames, "test_cache")
	})

	// Create temp directory and database file
	tempDir := t.TempDir()
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
	oldCache := globalCache
	defer func() {
		globalCache = oldCache
		globalCacheOnce = sync.Once{} // Reset for next test
	}()
	globalCache = cache
	globalCacheOnce = sync.Once{} // Reset so it uses the overridden cache
	globalCacheOnce.Do(func() {}) // Mark as done so GetGlobalCache doesn't try to reinit

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
	oldCache := globalCache
	defer func() {
		globalCache = oldCache
		globalCacheOnce = sync.Once{} // Reset for next test
	}()
	globalCache = cache
	globalCacheOnce = sync.Once{}
	globalCacheOnce.Do(func() {})

	testKey := "test-key"
	expectedData := TestData{ID: 2, Name: "Fetched"}

	// Test GetOrFetch with cache miss
	fetchCalled := false
	fetchFunc := func() (TestData, error) {
		fetchCalled = true
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
	if !fetchCalled {
		t.Error("Expected fetch function to be called")
	}
	if result.ID != expectedData.ID || result.Name != expectedData.Name {
		t.Errorf("Expected %+v, got %+v", expectedData, result)
	}

	// Verify data was cached
	if !cache.CacheExists("test_cache", testKey) {
		t.Error("Expected cache entry to be created")
	}
}

func TestGetOrFetch_FetchError(t *testing.T) {
	cache, dbPath := setupTestCache(t)
	defer func() { _ = cache.Close() }()
	defer func() { _ = os.Remove(dbPath) }()

	// Override global cache for this test
	oldCache := globalCache
	defer func() {
		globalCache = oldCache
		globalCacheOnce = sync.Once{} // Reset for next test
	}()
	globalCache = cache
	globalCacheOnce = sync.Once{}
	globalCacheOnce.Do(func() {})

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

	// Test Get with very short TTL (should be expired immediately)
	data, fromCache, err := cache.Get("test_cache", testKey, 1*time.Nanosecond)
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

	// Clear expired entries (with a short TTL so all should be cleared after a sleep)
	time.Sleep(10 * time.Millisecond)
	err := cache.ClearExpired("test_cache", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to clear expired cache: %v", err)
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
