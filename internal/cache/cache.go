package cache

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/spf13/viper"
	_ "modernc.org/sqlite"
)

const (
	// DefaultCacheTTL is the default time-to-live for cached entries (30 days)
	DefaultCacheTTL = 720 * time.Hour
	// NegativeCacheTTL is the TTL for "not found" responses (7 days)
	NegativeCacheTTL = 168 * time.Hour
)

// FetchFunc represents a function that fetches data from an external source
type FetchFunc[T any] func() (T, error)

// CacheDB manages the SQLite database connection for caching
type CacheDB struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string
}

var (
	globalCache     *CacheDB
	globalCacheOnce sync.Once
)

// ResetGlobalCache closes the current global cache and resets the singleton
// so the next call to GetGlobalCache will create a new instance.
// This is primarily for testing purposes.
func ResetGlobalCache() error {
	if globalCache != nil {
		if err := globalCache.Close(); err != nil {
			return err
		}
	}
	globalCache = nil
	globalCacheOnce = sync.Once{}
	return nil
}

// GetGlobalCache returns the singleton cache database instance
func GetGlobalCache() (*CacheDB, error) {
	var initErr error
	globalCacheOnce.Do(func() {
		dbPath := viper.GetString("cache.dbfile")
		if dbPath == "" {
			dbPath = "./cache.db"
		}
		globalCache, initErr = NewCacheDB(dbPath)
		if initErr != nil {
			return
		}
		// Initialize all cache tables
		for _, schema := range AllCacheSchemas {
			if err := globalCache.CreateTable(schema); err != nil {
				initErr = fmt.Errorf("failed to create cache table: %w", err)
				return
			}
		}
	})
	if initErr != nil {
		return nil, initErr
	}
	return globalCache, nil
}

// NewCacheDB creates a new CacheDB instance and opens the database connection
func NewCacheDB(dbPath string) (*CacheDB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		closeErr := db.Close()
		return nil, errors.Join(fmt.Errorf("failed to connect to cache database: %w", err), closeErr)
	}

	return &CacheDB{
		db:   db,
		path: dbPath,
	}, nil
}

// CreateTable creates a table using the provided schema
func (c *CacheDB) CreateTable(schema string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// Close closes the database connection
func (c *CacheDB) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// QueryRow executes a query that returns at most one row
func (c *CacheDB) QueryRow(query string, args ...any) *sql.Row {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db.QueryRow(query, args...)
}

// Exec executes a query without returning any rows
func (c *CacheDB) Exec(query string, args ...any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.db.Exec(query, args...)
	return err
}

// InvalidateSource deletes all entries from the specified cache table
// tableName must be one of the valid cache table names (e.g., "tmdb_cache", "omdb_cache")
// Returns the number of rows deleted
func (c *CacheDB) InvalidateSource(tableName string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate table name to prevent SQL injection
	if err := validateTableName(tableName); err != nil {
		return 0, err
	}

	// Delete all rows from the specified table
	query := fmt.Sprintf("DELETE FROM %s", tableName)
	result, err := c.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete cache entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Debug("Cache table cleared", "table", tableName, "rows_deleted", rowsAffected)
	return rowsAffected, nil
}

// validateTableName checks if the table name is in the whitelist
// to prevent SQL injection attacks
func validateTableName(tableName string) error {
	if !ValidCacheTableNames[tableName] {
		return fmt.Errorf("invalid cache table name: %s", tableName)
	}
	return nil
}

// GetOrFetch retrieves data from cache or fetches it using the provided function
// T is the type of data being cached
// tableName is the cache table to use (e.g., "omdb_cache", "openlibrary_cache", "steam_cache")
// cacheKey is the unique identifier for this cache entry (e.g., ISBN, IMDb ID, App ID)
// fetchFunc is called if the data is not found in cache or if the cache has expired
func GetOrFetch[T any](tableName, cacheKey string, fetchFunc FetchFunc[T]) (T, bool, error) {
	return getOrFetchWithPolicy(tableName, cacheKey, fetchFunc, nil)
}

// GetOrFetchWithPolicy retrieves data from cache or fetches it using the provided function, with optional control
// over whether a fetched value should be cached.
// If shouldCache is nil, all fetched values are cached (default behaviour).
func GetOrFetchWithPolicy[T any](tableName, cacheKey string, fetchFunc FetchFunc[T], shouldCache func(T) bool) (T, bool, error) {
	return getOrFetchWithPolicy(tableName, cacheKey, fetchFunc, shouldCache)
}

// GetOrFetchWithTTL retrieves data from cache or fetches it using the provided function, with a custom TTL.
// This is useful for negative caching where you want to cache "not found" responses with a shorter TTL.
// The ttlSelector function is called after fetching to determine which TTL to use for caching.
func GetOrFetchWithTTL[T any](tableName, cacheKey string, fetchFunc FetchFunc[T], ttlSelector func(T) time.Duration) (T, bool, error) {
	return getOrFetchWithTTLSelector(tableName, cacheKey, fetchFunc, ttlSelector)
}

// SelectNegativeCacheTTL returns a standard TTL selector for negative caching.
// Use this when you want to cache "not found" responses with a shorter TTL (7 days) than
// successful responses (30 days).
//
// The isNotFound function should return true if the result represents a "not found" response.
//
// Example:
//
//	cache.GetOrFetchWithTTL("openlibrary_cache", isbn,
//	    func() (*CachedBook, error) {
//	        book, err := fetchFromAPI(isbn)
//	        if err != nil && strings.Contains(err.Error(), "not found") {
//	            return &CachedBook{Book: nil, NotFound: true}, nil
//	        }
//	        return &CachedBook{Book: book, NotFound: false}, nil
//	    },
//	    cache.SelectNegativeCacheTTL(func(r *CachedBook) bool {
//	        return r.NotFound
//	    }))
func SelectNegativeCacheTTL[T any](isNotFound func(T) bool) func(T) time.Duration {
	return func(result T) time.Duration {
		if isNotFound(result) {
			return NegativeCacheTTL
		}
		return DefaultCacheTTL
	}
}

func getOrFetchWithPolicy[T any](tableName, cacheKey string, fetchFunc FetchFunc[T], shouldCache func(T) bool) (T, bool, error) {
	var zero T

	cache, err := GetGlobalCache()
	if err != nil {
		// If cache initialization fails, fall back to direct fetch
		slog.Warn("Failed to initialize cache, fetching directly", "error", err)
		data, fetchErr := fetchFunc()
		return data, false, fetchErr
	}

	// Get TTL duration from config
	ttlStr := viper.GetString("cache.ttl")
	if ttlStr == "" {
		ttlStr = "720h" // Default 30 days
	}
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		slog.Warn("Invalid cache TTL, using default", "ttl", ttlStr, "error", err)
		ttl = 720 * time.Hour
	}

	// Check cache first
	cached, fromCache, err := cache.Get(tableName, cacheKey, ttl)
	if err == nil && fromCache {
		var result T
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			slog.Debug("Cache hit", "table", tableName, "key", cacheKey)
			return result, true, nil
		}
		slog.Warn("Failed to unmarshal cached data, will refetch", "table", tableName, "key", cacheKey, "error", err)
	}

	// Fetch from external source if not in cache
	slog.Debug("Cache miss, fetching data", "table", tableName, "key", cacheKey)
	data, err := fetchFunc()
	if err != nil {
		return zero, false, fmt.Errorf("failed to fetch data: %w", err)
	}

	if shouldCache != nil && !shouldCache(data) {
		slog.Debug("Skipping cache store per policy", "table", tableName, "key", cacheKey)
		return data, false, nil
	}

	// Cache the result
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Warn("Failed to marshal data for caching", "table", tableName, "key", cacheKey, "error", err)
	} else {
		if err := cache.Set(tableName, cacheKey, string(jsonData)); err != nil {
			// Log error but don't fail - caching failure shouldn't stop the process
			slog.Warn("Failed to cache data", "table", tableName, "key", cacheKey, "error", err)
		} else {
			slog.Debug("Data cached successfully", "table", tableName, "key", cacheKey)
		}
	}

	return data, false, nil
}

func getOrFetchWithTTLSelector[T any](tableName, cacheKey string, fetchFunc FetchFunc[T], ttlSelector func(T) time.Duration) (T, bool, error) {
	var zero T

	cache, err := GetGlobalCache()
	if err != nil {
		// If cache initialization fails, fall back to direct fetch
		slog.Warn("Failed to initialize cache, fetching directly", "error", err)
		data, fetchErr := fetchFunc()
		return data, false, fetchErr
	}

	// Get default TTL from config for cache lookups
	ttlStr := viper.GetString("cache.ttl")
	if ttlStr == "" {
		ttlStr = "720h" // Default 30 days
	}
	defaultTTL, err := time.ParseDuration(ttlStr)
	if err != nil {
		slog.Warn("Invalid cache TTL, using default", "ttl", ttlStr, "error", err)
		defaultTTL = 720 * time.Hour
	}

	// Check cache first (use maximum TTL for lookup to find both short and long-lived entries)
	maxTTL := defaultTTL
	cached, fromCache, err := cache.Get(tableName, cacheKey, maxTTL)
	if err == nil && fromCache {
		var result T
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			slog.Debug("Cache hit", "table", tableName, "key", cacheKey)
			return result, true, nil
		}
		slog.Warn("Failed to unmarshal cached data, will refetch", "table", tableName, "key", cacheKey, "error", err)
	}

	// Fetch from external source if not in cache
	slog.Debug("Cache miss, fetching data", "table", tableName, "key", cacheKey)
	data, err := fetchFunc()
	if err != nil {
		return zero, false, fmt.Errorf("failed to fetch data: %w", err)
	}

	// Determine TTL based on the fetched data
	selectedTTL := defaultTTL
	if ttlSelector != nil {
		selectedTTL = ttlSelector(data)
	}

	// Cache the result with selected TTL
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Warn("Failed to marshal data for caching", "table", tableName, "key", cacheKey, "error", err)
	} else {
		if err := cache.Set(tableName, cacheKey, string(jsonData)); err != nil {
			// Log error but don't fail - caching failure shouldn't stop the process
			slog.Warn("Failed to cache data", "table", tableName, "key", cacheKey, "error", err)
		} else {
			slog.Debug("Data cached successfully", "table", tableName, "key", cacheKey, "ttl", selectedTTL)
		}
	}

	return data, false, nil
}

// Get retrieves a cached value from the specified table
// Returns the cached data, whether it was from cache, and any error
func (c *CacheDB) Get(tableName, key string, ttl time.Duration) (string, bool, error) {
	if err := validateTableName(tableName); err != nil {
		return "", false, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	query := fmt.Sprintf(`
		SELECT data, cached_at
		FROM %s
		WHERE cache_key = ?
	`, tableName)

	var data string
	var cachedAt time.Time
	err := c.db.QueryRow(query, key).Scan(&data, &cachedAt)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to query cache: %w", err)
	}

	// Check if cache has expired
	age := time.Now().UTC().Sub(cachedAt)
	if age > ttl {
		slog.Debug("Cache expired", "table", tableName, "key", key, "age", age)
		return "", false, nil
	}

	return data, true, nil
}

// Set stores a value in the cache
func (c *CacheDB) Set(tableName, key, data string) error {
	if err := validateTableName(tableName); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s (cache_key, data, cached_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, tableName)

	_, err := c.db.Exec(query, key, data)
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// ClearExpired removes expired cache entries from the specified table
func (c *CacheDB) ClearExpired(tableName string, ttl time.Duration) error {
	if err := validateTableName(tableName); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().UTC().Add(-ttl)
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE cached_at < ?
	`, tableName)

	result, err := c.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to clear expired cache: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		slog.Info("Cleared expired cache entries", "table", tableName, "count", rows)
	}

	return nil
}

// ClearAll removes all cache entries from the specified table
func (c *CacheDB) ClearAll(tableName string) error {
	if err := validateTableName(tableName); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	query := fmt.Sprintf("DELETE FROM %s", tableName)
	_, err := c.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	slog.Info("Cache cleared", "table", tableName)
	return nil
}

// CacheExists checks if a cache entry exists for the given key
func (c *CacheDB) CacheExists(tableName, key string) bool {
	if err := validateTableName(tableName); err != nil {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	query := fmt.Sprintf(`
		SELECT 1 FROM %s WHERE cache_key = ? LIMIT 1
	`, tableName)

	var exists int
	err := c.db.QueryRow(query, key).Scan(&exists)
	return err == nil
}
