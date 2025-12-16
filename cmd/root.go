package cmd

import (
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/lepinkainen/hermes/cmd/enhance"
	"github.com/lepinkainen/hermes/cmd/goodreads"
	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/letterboxd"
	"github.com/lepinkainen/hermes/cmd/steam"
	"github.com/lepinkainen/hermes/internal/automation"
	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/humanlog"
	"github.com/spf13/viper"
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

	Import  ImportCmd          `cmd:"" help:"Import data from various sources"`
	Enhance enhance.EnhanceCmd `cmd:"" help:"Enhance existing markdown notes with TMDB data"`
	Cache   CacheCmd           `cmd:"" help:"Manage cache database"`

	// Version command could be added here in the future
}

// ImportCmd represents the import command and its subcommands
type ImportCmd struct {
	Goodreads  goodreads.GoodreadsCmd   `cmd:"" help:"Import books from Goodreads library export"`
	IMDB       imdb.IMDBCmd             `cmd:"" help:"Import movies/shows from IMDB lists"`
	Letterboxd letterboxd.LetterboxdCmd `cmd:"" help:"Import movies from Letterboxd"`
	Steam      steam.SteamCmd           `cmd:"" help:"Import games from Steam"`
}

// CacheCmd represents the cache management command
type CacheCmd struct {
	Invalidate cache.InvalidateCacheCmd `cmd:"" help:"Invalidate (clear) cache for a specific source"`
}

// Execute runs the Kong-based CLI
func Execute() {
	initLogging()
	initConfig()

	// Create CLI instance
	var cli CLI
	cli.Import.Goodreads.Init(goodreads.DefaultParseGoodreadsFunc, goodreads.DefaultDownloadGoodreadsCSVFunc, &automation.DefaultCDPRunner{})

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

	// Letterboxd automation defaults
	viper.SetDefault("letterboxd.automation.timeout", "3m")
	viper.SetDefault("letterboxd.automation.download_dir", "exports")

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
	if err := viper.BindEnv("letterboxd.automation.username", "LETTERBOXD_USERNAME"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}
	if err := viper.BindEnv("letterboxd.automation.password", "LETTERBOXD_PASSWORD"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}
	if err := viper.BindEnv("letterboxd.automation.headful", "LETTERBOXD_HEADFUL"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}
	if err := viper.BindEnv("letterboxd.automation.download_dir", "LETTERBOXD_DOWNLOAD_DIR"); err != nil {
		slog.Error("Failed to bind environment variable", "error", err)
	}
	if err := viper.BindEnv("letterboxd.automation.timeout", "LETTERBOXD_AUTOMATION_TIMEOUT"); err != nil {
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

func initLogging() {
	// Determine log level from environment variable
	logLevel := slog.LevelInfo
	if levelStr := os.Getenv("HERMES_LOG_LEVEL"); levelStr != "" {
		switch levelStr {
		case "debug", "DEBUG":
			logLevel = slog.LevelDebug
		case "info", "INFO":
			logLevel = slog.LevelInfo
		case "warn", "WARN":
			logLevel = slog.LevelWarn
		case "error", "ERROR":
			logLevel = slog.LevelError
		}
	}

	// Create a human-readable handler for logging
	handler := humanlog.NewHandler(os.Stdout, &humanlog.Options{
		Level:      logLevel,
		TimeFormat: "2006-01-02 15:04:05", // Include date and time
	})

	// Set the default logger
	slog.SetDefault(slog.New(handler))
}
