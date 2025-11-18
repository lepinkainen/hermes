package cache

// SQL schemas for cache tables
// All cache tables use "cache_key" as the primary key column for consistency

// OMDBCacheSchema defines the schema for OMDB (IMDb) movie/show cache
const OMDBCacheSchema = `
CREATE TABLE IF NOT EXISTS omdb_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_omdb_cached_at ON omdb_cache(cached_at);
`

// OpenLibraryCacheSchema defines the schema for OpenLibrary book cache
const OpenLibraryCacheSchema = `
CREATE TABLE IF NOT EXISTS openlibrary_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_openlibrary_cached_at ON openlibrary_cache(cached_at);
`

// SteamCacheSchema defines the schema for Steam game cache
const SteamCacheSchema = `
CREATE TABLE IF NOT EXISTS steam_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_steam_cached_at ON steam_cache(cached_at);
`

// LetterboxdCacheSchema defines the schema for Letterboxd movie cache
const LetterboxdCacheSchema = `
CREATE TABLE IF NOT EXISTS letterboxd_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_letterboxd_cached_at ON letterboxd_cache(cached_at);
`

// TMDBCacheSchema defines the schema for TMDB movie/show cache
const TMDBCacheSchema = `
CREATE TABLE IF NOT EXISTS tmdb_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tmdb_cached_at ON tmdb_cache(cached_at);
`

// AllCacheSchemas contains all cache table schemas for easy initialization
var AllCacheSchemas = []string{
	OMDBCacheSchema,
	OpenLibraryCacheSchema,
	SteamCacheSchema,
	LetterboxdCacheSchema,
	TMDBCacheSchema,
}

// ValidCacheTableNames is the whitelist of allowed cache table names
// Used to prevent SQL injection when interpolating table names
var ValidCacheTableNames = map[string]bool{
	"omdb_cache":        true,
	"openlibrary_cache": true,
	"steam_cache":       true,
	"letterboxd_cache":  true,
	"tmdb_cache":        true,
}
