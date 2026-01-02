package steam

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// MigrateAchievementColumns adds achievement-related columns to steam_games table
// if they don't exist. This is a one-time migration for users with old databases.
// This function is idempotent and can be called multiple times safely.
func MigrateAchievementColumns(db *sql.DB) error {
	// Check if migration needed by querying table schema
	query := `SELECT COUNT(*) FROM pragma_table_info('steam_games')
              WHERE name IN ('achievements_total', 'achievements_unlocked', 'achievements_data')`

	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check schema: %w", err)
	}

	// All three columns exist, no migration needed
	if count == 3 {
		slog.Debug("Achievement columns already exist, skipping migration")
		return nil
	}

	// Partially migrated state shouldn't happen, but handle it
	if count > 0 {
		return fmt.Errorf("partial migration detected (%d/3 columns), manual intervention required", count)
	}

	slog.Info("Migrating steam_games table to add achievement columns")

	// Add columns with ALTER TABLE (SQLite supports adding columns with defaults)
	alterStatements := []string{
		"ALTER TABLE steam_games ADD COLUMN achievements_total INTEGER",
		"ALTER TABLE steam_games ADD COLUMN achievements_unlocked INTEGER",
		"ALTER TABLE steam_games ADD COLUMN achievements_data TEXT",
	}

	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	slog.Info("Successfully migrated steam_games table")
	return nil
}
