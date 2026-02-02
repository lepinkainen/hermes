package steam

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/errors"
	"github.com/spf13/viper"
)

const steamGamesSchema = `CREATE TABLE IF NOT EXISTS steam_games (
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
		metacritic_url TEXT,
		achievements_total INTEGER,
		achievements_unlocked INTEGER,
		achievements_data TEXT
	)`

// Convert GameDetails to map[string]any for database insertion
func gameDetailsToMap(details GameDetails) map[string]any {
	// Calculate achievement stats
	achievementsTotal := len(details.Achievements)
	achievementsUnlocked := 0
	for _, ach := range details.Achievements {
		if ach.Achieved == 1 {
			achievementsUnlocked++
		}
	}

	// Serialize achievements to JSON
	var achievementsJSON string
	if achievementsTotal > 0 {
		data, _ := json.Marshal(details.Achievements)
		achievementsJSON = string(data)
	}

	record := cmdutil.StructToMap(details, cmdutil.StructToMapOptions{
		JoinStringSlices: true,
		OmitFields: map[string]bool{
			"Achievements": true,
			"Categories":   true,
			"Genres":       true,
			"Metacritic":   true,
			"ReleaseDate":  true,
			"Screenshots":  true,
		},
		KeyOverrides: map[string]string{
			"AppID":          "appid",
			"PlaytimeRecent": "playtime_2weeks",
			"Description":    "detailed_description",
			"ShortDesc":      "short_description",
		},
	})

	record["release_date"] = details.ReleaseDate.Date
	record["coming_soon"] = details.ReleaseDate.ComingSoon
	record["metacritic_score"] = details.Metacritic.Score
	record["metacritic_url"] = details.Metacritic.URL
	record["achievements_total"] = achievementsTotal
	record["achievements_unlocked"] = achievementsUnlocked
	record["achievements_data"] = achievementsJSON
	record["screenshots"] = ""
	record["categories"] = ""
	record["genres"] = ""

	return record
}

// logGameProgress logs progress at percentage milestones (10%, 25%, 50%, 75%, 90%, 100%)
func logGameProgress(processed, total int) {
	if total == 0 || processed == 0 {
		return
	}

	percentage := float64(processed) / float64(total) * 100

	// Log at milestones: 10%, 25%, 50%, 75%, 90%, 100%
	milestones := []float64{10, 25, 50, 75, 90, 100}
	prevPercentage := float64(processed-1) / float64(total) * 100

	for _, milestone := range milestones {
		if percentage >= milestone && prevPercentage < milestone {
			slog.Info("Processing games",
				"processed", processed,
				"total", total,
				"percentage", fmt.Sprintf("%.1f%%", percentage),
			)
			break
		}
	}
}

func ParseSteam() error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	games, fromCache, err := GetCachedOwnedGames(steamID, apiKey)
	if err != nil {
		return fmt.Errorf("error importing games: %w", err)
	}
	if fromCache {
		slog.Debug("Using cached owned games list")
	} else {
		slog.Debug("Fetched owned games list from Steam API")
	}

	var processedGames []GameDetails
	totalGames := len(games)

	for _, game := range games {
		slog.Debug("Fetching game details", "game", game.Name)

		appID := strconv.Itoa(game.AppID)
		// Use the generic cache utility with SQLite backend
		details, _, err := cache.GetOrFetch("steam_cache", appID, func() (*GameDetails, error) {
			_, detailsData, fetchErr := fetchGameData(appID)
			if fetchErr != nil {
				return nil, fetchErr
			}
			// Ensure the AppID is set before caching
			detailsData.AppID = game.AppID
			return detailsData, nil
		})
		if err != nil {
			// Check for RateLimitError first (preserves RetryAfter timing)
			if errors.IsRateLimitError(err) {
				return err
			}

			// Fallback: string matching for backward compatibility
			if strings.Contains(err.Error(), "status code 429") {
				return errors.NewRateLimitError("Rate limit reached. Please try again later (usually after a few minutes)")
			}

			slog.Warn("Error fetching game details", "game", game.Name, "error", err)
			continue
		}

		// Ensure we preserve the original game's AppID and other important fields
		details.AppID = game.AppID
		details.PlaytimeForever = game.PlaytimeForever
		details.PlaytimeRecent = game.PlaytimeRecent
		details.LastPlayed = game.LastPlayed
		details.DetailsFetched = true

		// Fetch achievements if enabled
		if fetchAchievements {
			slog.Debug("Fetching achievements", "game", game.Name, "appid", game.AppID)
			achievements, fromCache, err := getCachedAchievements(steamID, apiKey, game.AppID)
			if err != nil {
				// Check for profile access errors (should stop processing)
				if errors.IsSteamProfileError(err) {
					return fmt.Errorf("steam profile access error (check API key and profile privacy): %w", err)
				}
				// Check if rate limit error - should stop processing
				if errors.IsRateLimitError(err) {
					return err
				}
				slog.Warn("Error fetching achievements", "game", game.Name, "error", err)
			} else {
				details.Achievements = achievements
				if fromCache {
					slog.Debug("Achievement cache hit", "count", len(achievements))
				} else {
					slog.Debug("Fetched achievements from API", "count", len(achievements))
				}
			}
		}

		if err := CreateMarkdownFile(game, details, outputDir); err != nil {
			slog.Error("Error creating markdown", "game", game.Name, "error", err)
			continue
		}

		processedGames = append(processedGames, *details)
		logGameProgress(len(processedGames), totalGames)
	}

	// Datasette integration
	// Run migration before writing to datastore
	if viper.GetBool("datasette.enabled") {
		dbPath := viper.GetString("datasette.dbfile")
		db, err := sql.Open("sqlite", dbPath)
		if err == nil {
			// Run migration to add achievement columns if they don't exist
			if migErr := MigrateAchievementColumns(db); migErr != nil {
				slog.Warn("Achievement column migration failed", "error", migErr)
			}
			_ = db.Close()
		}
	}

	if err := cmdutil.WriteToDatastore(processedGames, steamGamesSchema, "steam_games", "Steam games", gameDetailsToMap); err != nil {
		return err
	}

	// Write to JSON if enabled
	if writeJSON {
		if err := writeGameToJson(processedGames, jsonOutput); err != nil {
			slog.Error("Error writing games to JSON", "error", err)
		}
	}

	return nil
}

func fetchGameData(appID string) (*Game, *GameDetails, error) {
	apiKey := viper.GetString("steam.apikey")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("steam.apikey not set in config")
	}

	appIDInt, _ := strconv.Atoi(appID)
	details, err := getGameDetails(appIDInt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get game details: %w", err)
	}

	// Create a game object with the correct AppID and other details
	game := &Game{
		AppID:           appIDInt,
		Name:            details.Name,
		PlaytimeForever: details.PlaytimeForever,
		PlaytimeRecent:  details.PlaytimeRecent,
		LastPlayed:      details.LastPlayed,
		DetailsFetched:  true,
	}

	// Ensure the details also have the correct AppID
	details.AppID = appIDInt

	return game, details, nil
}
