package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/lepinkainen/hermes/cmd/enhance"
	"github.com/lepinkainen/hermes/cmd/goodreads"
	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/letterboxd"
	"github.com/lepinkainen/hermes/cmd/steam"
	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/humanlog"
	"github.com/spf13/viper"
)

var (
	parseGoodreads  = goodreads.ParseGoodreadsWithParams
	parseIMDB       = imdb.ParseImdbWithParams
	parseLetterboxd = letterboxd.ParseLetterboxdWithParams
	parseSteam      = steam.ParseSteamWithParams
	runEnhancement  = enhance.EnhanceNotes
)

// CLI represents the complete command structure for the hermes application
type CLI struct {
	// Global flags
	Overwrite    bool `help:"Overwrite existing markdown files when processing"`
	UpdateCovers bool `help:"Re-download cover images even if they already exist"`

	// Datasette flags
	Datasette   bool   `help:"Enable Datasette output" default:"true"`
	DatasetteDB string `help:"Path to SQLite database file" default:"./hermes.db"`

	// Cache flags
	CacheDBFile string `help:"Path to cache SQLite database file" default:"./cache.db"`
	CacheTTL    string `help:"Cache time-to-live duration (e.g., 720h for 30 days)" default:"720h"`

	// TMDB cover cache flags (used by importers and enhance command)
	UseTMDBCoverCache  bool   `help:"Use development cache for TMDB cover images to avoid repeated downloads" default:"false"`
	TMDBCoverCachePath string `help:"Path to TMDB cover cache directory" default:"tmdb-cover-cache"`

	Import  ImportCmd  `cmd:"" help:"Import data from various sources"`
	Enhance EnhanceCmd `cmd:"" help:"Enhance existing markdown notes with TMDB data"`
	Cache   CacheCmd   `cmd:"" help:"Manage cache database"`

	// Version command could be added here in the future
}

// ImportCmd represents the import command and its subcommands
type ImportCmd struct {
	Goodreads  GoodreadsCmd  `cmd:"" help:"Import books from Goodreads library export"`
	IMDB       IMDBCmd       `cmd:"" help:"Import movies/shows from IMDB lists"`
	Letterboxd LetterboxdCmd `cmd:"" help:"Import movies from Letterboxd"`
	Steam      SteamCmd      `cmd:"" help:"Import games from Steam"`
}

// GoodreadsCmd represents the goodreads import command
type GoodreadsCmd struct {
	Input             string        `short:"f" help:"Path to Goodreads library export CSV file"`
	Output            string        `short:"o" help:"Subdirectory under markdown output directory for Goodreads files" default:"goodreads"`
	JSON              bool          `help:"Write data to JSON format"`
	JSONOutput        string        `help:"Path to JSON output file (defaults to json/goodreads.json)"`
	Automated         bool          `help:"Automatically download Goodreads export via browser automation"`
	GoodreadsEmail    string        `help:"Goodreads account email for automation"`
	GoodreadsPassword string        `help:"Goodreads account password for automation"`
	Headful           bool          `help:"Run automation with a visible browser window (default is headless)"`
	DownloadDir       string        `help:"Directory for automated Goodreads export download (defaults to exports/)"`
	AutomationTimeout time.Duration `help:"Timeout for Goodreads automation flow" default:"3m"`
}

// IMDBCmd represents the imdb import command
type IMDBCmd struct {
	Input      string `short:"f" help:"Path to IMDB CSV file"`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for IMDB files" default:"imdb"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/imdb.json)"`
	// TMDB enrichment options
	TMDBEnabled         bool     `help:"Enable TMDB enrichment" default:"true"`
	TMDBDownloadCover   bool     `help:"Download cover images from TMDB" default:"true"`
	TMDBGenerateContent bool     `help:"Generate TMDB content sections" default:"false"`
	TMDBNoInteractive   bool     `help:"Disable interactive TUI for TMDB selection (auto-select first result)" default:"false"`
	TMDBContentSections []string `help:"Specific TMDB content sections to generate (empty = all)"`
}

// LetterboxdCmd represents the letterboxd import command
type LetterboxdCmd struct {
	Input      string `short:"f" help:"Path to Letterboxd CSV file"`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for Letterboxd files" default:"letterboxd"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/letterboxd.json)"`
	// TMDB enrichment options
	TMDBEnabled         bool     `help:"Enable TMDB enrichment" default:"true"`
	TMDBDownloadCover   bool     `help:"Download cover images from TMDB" default:"true"`
	TMDBGenerateContent bool     `help:"Generate TMDB content sections" default:"false"`
	TMDBNoInteractive   bool     `help:"Disable interactive TUI for TMDB selection (auto-select first result)" default:"false"`
	TMDBContentSections []string `help:"Specific TMDB content sections to generate (empty = all)"`
}

// SteamCmd represents the steam import command
type SteamCmd struct {
	SteamID    string `help:"Steam ID to fetch data for"`
	APIKey     string `help:"Steam API key"`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for Steam files" default:"steam"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/steam.json)"`
}

// EnhanceCmd represents the enhance command
type EnhanceCmd struct {
	InputDirs           []string `short:"d" help:"Directories containing markdown files to enhance (can specify multiple)" required:""`
	Recursive           bool     `short:"r" help:"Scan subdirectories recursively" default:"false"`
	DryRun              bool     `help:"Show what would be done without making changes" default:"false"`
	OverwriteTMDB       bool     `help:"Overwrite existing TMDB content in notes" default:"false"`
	Force               bool     `short:"f" help:"Force re-enrichment even when TMDB ID exists in frontmatter" default:"false"`
	TMDBNoInteractive   bool     `help:"Disable interactive TUI for TMDB selection (auto-select first result)" default:"false"`
	TMDBContentSections []string `help:"Specific TMDB content sections to generate (empty = all)"`
}

// CacheCmd represents the cache management command
type CacheCmd struct {
	Invalidate InvalidateCacheCmd `cmd:"" help:"Invalidate (clear) cache for a specific source"`
}

// InvalidateCacheCmd represents the cache invalidate subcommand
type InvalidateCacheCmd struct {
	Source string `arg:"" help:"Cache source to invalidate: tmdb, omdb, steam, letterboxd, openlibrary" required:""`
}

// Execute runs the Kong-based CLI
func Execute() {
	initLogging()
	initConfig()

	// Create CLI instance
	var cli CLI

	// Parse command line with Kong
	ctx := kong.Parse(&cli,
		kong.Name("hermes"),
		kong.Description("A tool to import data from various sources into a unified format."),
		kong.UsageOnError(),
	)

	// Update global config based on parsed flags
	updateGlobalConfig(&cli)

	// Execute the selected command
	err := ctx.Run()
	if err != nil {
		slog.Error("Command failed", "error", err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetDefault("MarkdownOutputDir", "./markdown/")
	viper.SetDefault("JSONOutputDir", "./json/")
	viper.SetDefault("OverwriteFiles", false)

	// Datasette defaults
	viper.SetDefault("datasette.enabled", true)
	viper.SetDefault("datasette.dbfile", "./hermes.db")

	// Cache defaults
	viper.SetDefault("cache.dbfile", "./cache.db")
	viper.SetDefault("cache.ttl", "720h") // 30 days

	// Goodreads automation defaults
	viper.SetDefault("goodreads.automation.timeout", "3m")
	viper.SetDefault("goodreads.automation.download_dir", "exports")

	// Enable environment variable support
	viper.AutomaticEnv()
	// Bind specific environment variables to config keys
	if err := viper.BindEnv("TMDBAPIKey", "TMDB_API_KEY"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}
	if err := viper.BindEnv("goodreads.automation.headful", "GOODREADS_HEADFUL"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}
	if err := viper.BindEnv("goodreads.automation.download_dir", "GOODREADS_DOWNLOAD_DIR"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}
	if err := viper.BindEnv("goodreads.automation.timeout", "GOODREADS_AUTOMATION_TIMEOUT"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found, writing default config file...")
			if err := viper.SafeWriteConfig(); err != nil {
				slog.Error("Error writing config file", "error", err)
			}
			os.Exit(0)
		} else {
			slog.Error("Fatal error config file", "error", err)
			os.Exit(1)
		}
	}

	// Initialize global config
	config.InitConfig()
}

func updateGlobalConfig(cli *CLI) {
	// Update config based on CLI flags
	config.SetOverwriteFiles(cli.Overwrite)
	config.SetUpdateCovers(cli.UpdateCovers)

	// Update datasette config
	viper.Set("datasette.enabled", cli.Datasette)
	viper.Set("datasette.dbfile", cli.DatasetteDB)

	// Update cache config
	viper.Set("cache.dbfile", cli.CacheDBFile)
	viper.Set("cache.ttl", cli.CacheTTL)

	// Update TMDB cover cache config
	viper.Set("tmdb.cover_cache.enabled", cli.UseTMDBCoverCache)
	viper.Set("tmdb.cover_cache.path", cli.TMDBCoverCachePath)
}

// Run methods for each command

func (g *GoodreadsCmd) Run() error {
	// Read from config if value not provided via flag
	input := g.Input
	if input == "" && !g.Automated {
		input = viper.GetString("goodreads.csvfile")
	}

	automationTimeout := g.AutomationTimeout
	if automationTimeout == 0 {
		automationTimeout = viper.GetDuration("goodreads.automation.timeout")
		if automationTimeout == 0 {
			automationTimeout = 3 * time.Minute
		}
	}

	headful := g.Headful
	if !headful && viper.IsSet("goodreads.automation.headful") {
		headful = viper.GetBool("goodreads.automation.headful")
	}

	downloadDir := g.DownloadDir
	if downloadDir == "" {
		downloadDir = viper.GetString("goodreads.automation.download_dir")
	}

	email := g.GoodreadsEmail
	if email == "" {
		email = viper.GetString("goodreads.automation.email")
	}
	password := g.GoodreadsPassword
	if password == "" {
		password = viper.GetString("goodreads.automation.password")
	}

	// Check if required value is still missing when not automated
	if input == "" && !g.Automated {
		return fmt.Errorf("input CSV file is required (provide via --input flag or goodreads.csvfile in config)")
	}

	params := goodreads.ParseParams{
		CSVPath:    input,
		OutputDir:  g.Output,
		JSONOutput: g.JSONOutput,
		WriteJSON:  g.JSON,
		Automated:  g.Automated,
		AutomationOptions: goodreads.AutomationOptions{
			Email:       email,
			Password:    password,
			DownloadDir: downloadDir,
			Headless:    !headful,
			Timeout:     automationTimeout,
		},
	}

	if params.Automated && (params.AutomationOptions.Email == "" || params.AutomationOptions.Password == "") {
		return fmt.Errorf("goodreads automation requires email and password (use flags, environment variables, or config)")
	}

	return parseGoodreads(params)
}

func (i *IMDBCmd) Run() error {
	// Read from config if value not provided via flag
	input := i.Input
	if input == "" {
		input = viper.GetString("imdb.csvfile")
	}

	// Check if required value is still missing
	if input == "" {
		return fmt.Errorf("input CSV file is required (provide via --input flag or imdb.csvfile in config)")
	}

	return parseIMDB(
		input,
		i.Output,
		i.JSON,
		i.JSONOutput,
		false, // overwrite
		i.TMDBEnabled,
		i.TMDBDownloadCover,
		i.TMDBGenerateContent,
		!i.TMDBNoInteractive, // Invert: default is interactive
		i.TMDBContentSections,
		viper.GetBool("tmdb.cover_cache.enabled"),
		viper.GetString("tmdb.cover_cache.path"),
	)
}

func (l *LetterboxdCmd) Run() error {
	// Read from config if value not provided via flag
	input := l.Input
	if input == "" {
		input = viper.GetString("letterboxd.csvfile")
	}

	// Check if required value is still missing
	if input == "" {
		return fmt.Errorf("input CSV file is required (provide via --input flag or letterboxd.csvfile in config)")
	}

	return parseLetterboxd(
		input,
		l.Output,
		l.JSON,
		l.JSONOutput,
		false, // overwrite
		l.TMDBEnabled,
		l.TMDBDownloadCover,
		l.TMDBGenerateContent,
		!l.TMDBNoInteractive, // Invert: default is interactive
		l.TMDBContentSections,
		viper.GetBool("tmdb.cover_cache.enabled"),
		viper.GetString("tmdb.cover_cache.path"),
	)
}

func (s *SteamCmd) Run() error {
	// Read from config if values not provided via flags
	steamID := s.SteamID
	if steamID == "" {
		steamID = viper.GetString("steam.steamid")
	}

	apiKey := s.APIKey
	if apiKey == "" {
		apiKey = viper.GetString("steam.apikey")
	}

	// Check if required values are still missing
	if steamID == "" {
		return fmt.Errorf("steam ID is required (provide via --steamid flag or steam.steamid in config)")
	}
	if apiKey == "" {
		return fmt.Errorf("steam API key is required (provide via --apikey flag or steam.apikey in config)")
	}

	return parseSteam(steamID, apiKey, s.Output, s.JSON, s.JSONOutput, false)
}

func (e *EnhanceCmd) Run() error {
	for _, inputDir := range e.InputDirs {
		opts := enhance.Options{
			InputDir:            inputDir,
			Recursive:           e.Recursive,
			DryRun:              e.DryRun,
			Overwrite:           e.OverwriteTMDB,
			Force:               e.Force,
			TMDBDownloadCover:   true,                 // Always download covers
			TMDBGenerateContent: true,                 // Always generate content
			TMDBInteractive:     !e.TMDBNoInteractive, // Invert: default is interactive
			TMDBContentSections: e.TMDBContentSections,
			UseTMDBCoverCache:   viper.GetBool("tmdb.cover_cache.enabled"),
			TMDBCoverCachePath:  viper.GetString("tmdb.cover_cache.path"),
		}

		if err := runEnhancement(opts); err != nil {
			return err
		}
	}

	return nil
}

func (i *InvalidateCacheCmd) Run() error {
	cacheDB := viper.GetString("cache.dbfile")

	slog.Info("Invalidating cache", "source", i.Source, "database", cacheDB)

	// Map source name to cache table name
	tableName := i.Source + "_cache"

	// Validate source
	validSources := map[string]bool{
		"tmdb":        true,
		"omdb":        true,
		"steam":       true,
		"letterboxd":  true,
		"openlibrary": true,
	}

	if !validSources[i.Source] {
		return fmt.Errorf("invalid cache source '%s'; valid sources are: tmdb, omdb, steam, letterboxd, openlibrary", i.Source)
	}

	// Get or create cache database
	cacheInstance, err := cache.GetGlobalCache()
	if err != nil {
		return fmt.Errorf("failed to open cache database: %w", err)
	}

	rowsDeleted, err := cacheInstance.InvalidateSource(tableName)
	if err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	slog.Info("Cache invalidated", "source", i.Source, "rows_deleted", rowsDeleted)
	return nil
}

func initLogging() {
	// Create a human-readable handler for logging
	handler := humanlog.NewHandler(os.Stdout, &humanlog.Options{
		Level: slog.LevelInfo,
	})

	// Set the default logger
	slog.SetDefault(slog.New(handler))
}
