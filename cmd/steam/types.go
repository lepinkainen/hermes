package steam

import (
	"time"
)

// Game represents a Steam game with its details
type Game struct {
	AppID           int       `json:"appid"`
	Name            string    `json:"name"`
	PlaytimeForever int       `json:"playtime_forever"` // Total playtime in minutes
	PlaytimeRecent  int       `json:"playtime_2weeks"`  // Recent playtime in minutes (optional)
	LastPlayed      time.Time `json:"last_played"`
	DetailsFetched  bool      `json:"details_fetched"`
}

// SteamResponse represents the response structure from Steam API
type SteamResponse struct {
	Response struct {
		Games []Game `json:"games"`
	} `json:"response"`
}

type Screenshot struct {
	ID      int    `json:"id"`
	PathURL string `json:"path_full"`
}

type Category struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

type Genre struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type MetacriticData struct {
	Score int    `json:"score"`
	URL   string `json:"url"`
}

type Achievement struct {
	APIName     string  `json:"apiname"`
	Achieved    int     `json:"achieved"` // 0 = locked, 1 = unlocked
	UnlockTime  int64   `json:"unlocktime"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Percent     float64 `json:"percent,omitempty"` // Global unlock percentage
}

type GameDetails struct {
	Game
	Description string       `json:"detailed_description"`
	ShortDesc   string       `json:"short_description"`
	HeaderImage string       `json:"header_image"`
	Screenshots []Screenshot `json:"screenshots"`
	Developers  []string     `json:"developers"`
	Publishers  []string     `json:"publishers"`
	ReleaseDate struct {
		ComingSoon bool   `json:"coming_soon"`
		Date       string `json:"date"`
	} `json:"release_date"`
	Categories   []Category     `json:"categories"`
	Genres       []Genre        `json:"genres"`
	Metacritic   MetacriticData `json:"metacritic,omitempty"`
	Achievements []Achievement  `json:"-"` // Populated separately from User Stats API, not from Store API
}

// SteamAchievementsResponse represents the response structure for player achievements
type SteamAchievementsResponse struct {
	PlayerStats struct {
		SteamID      string        `json:"steamID"`
		GameName     string        `json:"gameName"`
		Achievements []Achievement `json:"achievements"`
		Success      bool          `json:"success"`
		Error        string        `json:"error"`
	} `json:"playerstats"`
}

// CachedAchievements wraps achievement data for negative caching
type CachedAchievements struct {
	Achievements   []Achievement `json:"achievements"`
	NoAchievements bool          `json:"no_achievements"`
}
