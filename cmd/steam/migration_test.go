package steam

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigrateAchievementColumns_FreshDB(t *testing.T) {
	env := testutil.NewTestEnv(t)
	dbPath := filepath.Join(env.RootDir(), "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create steam_games table WITH achievement columns (fresh DB scenario)
	_, err = db.Exec(steamGamesSchema)
	require.NoError(t, err)

	// Run migration - should be a no-op
	err = MigrateAchievementColumns(db)
	require.NoError(t, err, "Migration should succeed on fresh DB")

	// Verify columns exist
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('steam_games')
                       WHERE name IN ('achievements_total', 'achievements_unlocked', 'achievements_data')`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "Should have all 3 achievement columns")
}

func TestMigrateAchievementColumns_OldSchema(t *testing.T) {
	env := testutil.NewTestEnv(t)
	dbPath := filepath.Join(env.RootDir(), "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create old schema WITHOUT achievement columns
	oldSchema := `CREATE TABLE IF NOT EXISTS steam_games (
		appid INTEGER PRIMARY KEY,
		name TEXT,
		playtime_forever INTEGER,
		playtime_2weeks INTEGER,
		last_played TEXT,
		details_fetched BOOLEAN,
		detailed_description TEXT,
		short_description TEXT,
		header_image TEXT,
		screenshots TEXT,
		developers TEXT,
		publishers TEXT,
		release_date TEXT,
		coming_soon BOOLEAN,
		categories TEXT,
		genres TEXT,
		metacritic_score INTEGER,
		metacritic_url TEXT
	)`

	_, err = db.Exec(oldSchema)
	require.NoError(t, err)

	// Verify columns don't exist yet
	var countBefore int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('steam_games')
                       WHERE name IN ('achievements_total', 'achievements_unlocked', 'achievements_data')`).Scan(&countBefore)
	require.NoError(t, err)
	assert.Equal(t, 0, countBefore, "Should have no achievement columns before migration")

	// Run migration
	err = MigrateAchievementColumns(db)
	require.NoError(t, err, "Migration should succeed")

	// Verify columns were added
	var countAfter int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('steam_games')
                       WHERE name IN ('achievements_total', 'achievements_unlocked', 'achievements_data')`).Scan(&countAfter)
	require.NoError(t, err)
	assert.Equal(t, 3, countAfter, "Should have all 3 achievement columns after migration")
}

func TestMigrateAchievementColumns_Idempotent(t *testing.T) {
	env := testutil.NewTestEnv(t)
	dbPath := filepath.Join(env.RootDir(), "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create table with achievement columns
	_, err = db.Exec(steamGamesSchema)
	require.NoError(t, err)

	// Run migration multiple times
	for i := 0; i < 3; i++ {
		err = MigrateAchievementColumns(db)
		require.NoError(t, err, "Migration should be idempotent (iteration %d)", i)
	}

	// Verify columns still exist and count is correct
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('steam_games')
                       WHERE name IN ('achievements_total', 'achievements_unlocked', 'achievements_data')`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "Should still have exactly 3 achievement columns")
}
