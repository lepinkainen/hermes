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

// SteamAchievementsCacheSchema defines the schema for Steam player achievements cache
const SteamAchievementsCacheSchema = `
CREATE TABLE IF NOT EXISTS steam_achievements_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_steam_achievements_cached_at ON steam_achievements_cache(cached_at);
`

// SteamOwnedGamesCacheSchema defines the schema for Steam owned games list cache
const SteamOwnedGamesCacheSchema = `
CREATE TABLE IF NOT EXISTS steam_owned_games_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_steam_owned_games_cached_at ON steam_owned_games_cache(cached_at);
`

// SteamSearchCacheSchema defines the schema for Steam Store search results cache
const SteamSearchCacheSchema = `
CREATE TABLE IF NOT EXISTS steam_search_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_steam_search_cached_at ON steam_search_cache(cached_at);
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

// GoogleBooksCacheSchema defines the schema for Google Books API cache
const GoogleBooksCacheSchema = `
CREATE TABLE IF NOT EXISTS googlebooks_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_googlebooks_cached_at ON googlebooks_cache(cached_at);
`

// LetterboxdMappingCacheSchema defines the schema for Letterboxd URI → TMDB/IMDB ID mappings
// This cache persists user-confirmed TMDB selections to avoid re-prompting on subsequent imports
const LetterboxdMappingCacheSchema = `
CREATE TABLE IF NOT EXISTS letterboxd_mapping_cache (
	letterboxd_uri TEXT PRIMARY KEY NOT NULL,
	tmdb_id INTEGER,
	tmdb_type TEXT,
	imdb_id TEXT,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_letterboxd_mapping_cached_at ON letterboxd_mapping_cache(cached_at);
`

// ISBNdbCacheSchema defines the schema for ISBNdb book cache
const ISBNdbCacheSchema = `
CREATE TABLE IF NOT EXISTS isbndb_cache (
	cache_key TEXT PRIMARY KEY NOT NULL,
	data TEXT NOT NULL,
	cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_isbndb_cached_at ON isbndb_cache(cached_at);
`

// AllCacheSchemas contains all cache table schemas for easy initialization
var AllCacheSchemas = []string{
	OMDBCacheSchema,
	OpenLibraryCacheSchema,
	SteamCacheSchema,
	SteamAchievementsCacheSchema,
	SteamOwnedGamesCacheSchema,
	SteamSearchCacheSchema,
	LetterboxdCacheSchema,
	TMDBCacheSchema,
	GoogleBooksCacheSchema,
	LetterboxdMappingCacheSchema,
	ISBNdbCacheSchema,
}

// ValidCacheTableNames is the whitelist of allowed cache table names
// Used to prevent SQL injection when interpolating table names
var ValidCacheTableNames = map[string]bool{
	"omdb_cache":               true,
	"openlibrary_cache":        true,
	"steam_cache":              true,
	"steam_achievements_cache": true,
	"steam_owned_games_cache":  true,
	"steam_search_cache":       true,
	"letterboxd_cache":         true,
	"tmdb_cache":               true,
	"googlebooks_cache":        true,
	"letterboxd_mapping_cache": true,
	"isbndb_cache":             true,
}
