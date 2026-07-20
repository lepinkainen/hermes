package cache

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/sqliteutil"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"
)

const (
	// DefaultCacheTTL is the default time-to-live for cached entries (30 days)
	DefaultCacheTTL = 720 * time.Hour
	// NegativeCacheTTL is the TTL for "not found" responses (7 days)
	NegativeCacheTTL = 168 * time.Hour

	// sqliteTimestampLayout matches the format produced by SQLite's
	// CURRENT_TIMESTAMP ("YYYY-MM-DD HH:MM:SS", always UTC).
	sqliteTimestampLayout = "2006-01-02 15:04:05"

	// parseFailureEscalationThreshold is the number of per-key cached_at parse
	// failures for a single table after which we stop logging a Warn per key
	// and instead log a single Error noting the table is effectively uncached.
	parseFailureEscalationThreshold = 10
)

// cacheEntryV1 wraps cached data with TTL metadata
// This allows different cache entries to have different TTL values
type cacheEntryV1 struct {
	Data         string `json:"data"`
	TTLSeconds   int64  `json:"ttl_seconds"`
	CachedAtUnix int64  `json:"cached_at_unix"`
}

// FetchFunc represents a function that fetches data from an external source
type FetchFunc[T any] func() (T, error)

// CacheDB manages the SQLite database connection for caching
type CacheDB struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string

	// parseFailMu guards parseFailures, tracking per-table cached_at parse
	// failures so we can escalate from a per-key Warn to a single Error once
	// a table's data looks widely corrupt. Separate from mu since Get (the
	// caller) only takes mu.RLock().
	parseFailMu   sync.Mutex
	parseFailures map[string]int
}

var (
	globalCache      *CacheDB
	globalCacheOnce  sync.Once
	globalCacheMutex sync.Mutex //nolint:unused // test seam (cache_test.go)

	configTTL     time.Duration
	configTTLOnce sync.Once
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
	configTTLOnce = sync.Once{}
	return nil
}

// getConfigTTL returns the cache TTL from config, parsed once and cached.
func getConfigTTL() time.Duration {
	configTTLOnce.Do(func() {
		ttlStr := viper.GetString("cache.ttl")
		if ttlStr == "" {
			configTTL = DefaultCacheTTL
			return
		}
		parsed, err := time.ParseDuration(ttlStr)
		if err != nil {
			slog.Warn("Invalid cache TTL, using default", "ttl", ttlStr, "error", err)
			configTTL = DefaultCacheTTL
			return
		}
		configTTL = parsed
	})
	return configTTL
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
	db, err := sql.Open("sqlite", sqliteutil.DSN(dbPath))
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
		db:            db,
		path:          dbPath,
		parseFailures: make(map[string]int),
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

	// Check cache first
	ttl := getConfigTTL()
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

	// Cache the result (no custom TTL, use default)
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Warn("Failed to marshal data for caching", "table", tableName, "key", cacheKey, "error", err)
	} else {
		if err := cache.Set(tableName, cacheKey, string(jsonData), 0); err != nil {
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

	// Check cache first (use maximum TTL for lookup to find both short and long-lived entries)
	defaultTTL := getConfigTTL()
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
		if err := cache.Set(tableName, cacheKey, string(jsonData), selectedTTL); err != nil {
			// Log error but don't fail - caching failure shouldn't stop the process
			slog.Warn("Failed to cache data", "table", tableName, "key", cacheKey, "error", err)
		} else {
			slog.Debug("Data cached successfully", "table", tableName, "key", cacheKey, "ttl", selectedTTL)
		}
	}

	return data, false, nil
}

// parseCachedAt normalizes the value scanned from a cached_at column into a
// time.Time. Under STRICT tables the column is declared TEXT (STRICT tables
// reject DATETIME), so modernc.org/sqlite returns a string/[]byte rather than
// auto-converting to time.Time. Legacy (pre-STRICT) databases still declare
// the column DATETIME, in which case the driver hands back a time.Time
// directly - support both so old, un-migrated cache.db files keep working.
func parseCachedAt(v any) (time.Time, error) {
	switch val := v.(type) {
	case time.Time:
		return val.UTC(), nil
	case string:
		return parseCachedAtString(val)
	case []byte:
		return parseCachedAtString(string(val))
	default:
		return time.Time{}, fmt.Errorf("unsupported cached_at type %T", v)
	}
}

func parseCachedAtString(s string) (time.Time, error) {
	if t, err := time.ParseInLocation(sqliteTimestampLayout, s, time.UTC); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized cached_at format: %q", s)
}

// recordParseFailure logs a cached_at parse failure for tableName/key. Below
// parseFailureEscalationThreshold failures it logs a per-key Warn like
// before; at the threshold it logs one Error explaining the table's
// cached_at data is widely corrupt and that deleting cache.db will fix it,
// then stays quiet for subsequent failures on that table.
func (c *CacheDB) recordParseFailure(tableName, key string, parseErr error) {
	c.parseFailMu.Lock()
	c.parseFailures[tableName]++
	count := c.parseFailures[tableName]
	c.parseFailMu.Unlock()

	switch {
	case count == parseFailureEscalationThreshold:
		slog.Error("Cache table has widely corrupt cached_at data; caching is effectively disabled for it, delete cache.db to fix",
			"table", tableName, "failures", count)
	case count < parseFailureEscalationThreshold:
		slog.Warn("Failed to parse cached_at, treating as cache miss", "table", tableName, "key", key, "error", parseErr)
	}
}

// Get retrieves a cached value from the specified table
// Returns the cached data, whether it was from cache, and any error
// The ttl parameter is used as fallback for old cache entries without TTL metadata
func (c *CacheDB) Get(tableName, key string, fallbackTTL time.Duration) (string, bool, error) {
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
	var cachedAtRaw any
	err := c.db.QueryRow(query, key).Scan(&data, &cachedAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to query cache: %w", err)
	}

	cachedAt, err := parseCachedAt(cachedAtRaw)
	if err != nil {
		// Under a STRICT (TEXT) cached_at column we always get a parseable string,
		// so a failure here means genuinely corrupt data - treat as a cache miss
		// rather than crashing the caller.
		c.recordParseFailure(tableName, key, err)
		return "", false, nil
	}

	// Try to unmarshal as wrapped entry (new format with custom TTL)
	var wrapper cacheEntryV1
	if err := json.Unmarshal([]byte(data), &wrapper); err == nil && wrapper.Data != "" {
		// New format - use stored TTL
		ttl := time.Duration(wrapper.TTLSeconds) * time.Second
		age := time.Now().UTC().Sub(cachedAt)
		if age > ttl {
			slog.Debug("Cache expired (custom TTL)", "table", tableName, "key", key, "age", age, "ttl", ttl)
			return "", false, nil
		}
		return wrapper.Data, true, nil
	}

	// Old format (plain data) or unmarshal failed - use fallback TTL for backward compatibility
	age := time.Now().UTC().Sub(cachedAt)
	if age > fallbackTTL {
		slog.Debug("Cache expired (fallback TTL)", "table", tableName, "key", key, "age", age, "ttl", fallbackTTL)
		return "", false, nil
	}

	return data, true, nil
}

// Set stores a value in the cache with optional custom TTL
// If ttl is 0, the data is stored without TTL metadata (backward compatible)
// If ttl > 0, the data is wrapped with TTL metadata for custom expiration
func (c *CacheDB) Set(tableName, key, data string, ttl time.Duration) error {
	if err := validateTableName(tableName); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// If TTL is provided, wrap the data with metadata
	var dataToStore string
	if ttl > 0 {
		wrapper := cacheEntryV1{
			Data:         data,
			TTLSeconds:   int64(ttl.Seconds()),
			CachedAtUnix: time.Now().UTC().Unix(),
		}
		wrapperJSON, err := json.Marshal(wrapper)
		if err != nil {
			return fmt.Errorf("failed to marshal cache wrapper: %w", err)
		}
		dataToStore = string(wrapperJSON)
	} else {
		// No TTL provided, store data as-is (backward compatible)
		dataToStore = data
	}

	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s (cache_key, data, cached_at)
		VALUES (?, ?, ?)
	`, tableName)

	_, err := c.db.Exec(query, key, dataToStore, time.Now().UTC().Format(sqliteTimestampLayout))
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

	// cached_at is stored as TEXT (STRICT tables can't declare DATETIME), using
	// SQLite's CURRENT_TIMESTAMP format ("YYYY-MM-DD HH:MM:SS", UTC). Binding a
	// time.Time here would compare against a driver-formatted string that doesn't
	// match, so format the cutoff the same way for a correct lexicographic comparison.
	result, err := c.db.Exec(query, cutoff.Format(sqliteTimestampLayout))
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
