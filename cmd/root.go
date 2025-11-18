package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/lepinkainen/hermes/cmd/enhance"
	"github.com/lepinkainen/hermes/cmd/goodreads"
	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/letterboxd"
	"github.com/lepinkainen/hermes/cmd/steam"
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

	Import  ImportCmd  `cmd:"" help:"Import data from various sources"`
	Enhance EnhanceCmd `cmd:"" help:"Enhance existing markdown notes with TMDB data"`

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
	Input      string `short:"f" help:"Path to Goodreads library export CSV file"`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for Goodreads files" default:"goodreads"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/goodreads.json)"`
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
	InputDir            string   `short:"d" help:"Directory containing markdown files to enhance" required:""`
	Recursive           bool     `short:"r" help:"Scan subdirectories recursively" default:"false"`
	DryRun              bool     `help:"Show what would be done without making changes" default:"false"`
	OverwriteTMDB       bool     `help:"Overwrite existing TMDB content in notes" default:"false"`
	Force               bool     `short:"f" help:"Force re-enrichment even when TMDB ID exists in frontmatter" default:"false"`
	TMDBNoInteractive   bool     `help:"Disable interactive TUI for TMDB selection (auto-select first result)" default:"false"`
	TMDBContentSections []string `help:"Specific TMDB content sections to generate (empty = all)"`
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

	// Enable environment variable support
	viper.AutomaticEnv()
	// Bind specific environment variables to config keys
	if err := viper.BindEnv("TMDBAPIKey", "TMDB_API_KEY"); err != nil {
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
}

// Run methods for each command

func (g *GoodreadsCmd) Run() error {
	// Read from config if value not provided via flag
	input := g.Input
	if input == "" {
		input = viper.GetString("goodreads.csvfile")
	}

	// Check if required value is still missing
	if input == "" {
		return fmt.Errorf("input CSV file is required (provide via --input flag or goodreads.csvfile in config)")
	}

	return parseGoodreads(input, g.Output, g.JSON, g.JSONOutput, false)
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
	opts := enhance.Options{
		InputDir:            e.InputDir,
		Recursive:           e.Recursive,
		DryRun:              e.DryRun,
		Overwrite:           e.OverwriteTMDB,
		Force:               e.Force,
		TMDBDownloadCover:   true,                 // Always download covers
		TMDBGenerateContent: true,                 // Always generate content
		TMDBInteractive:     !e.TMDBNoInteractive, // Invert: default is interactive
		TMDBContentSections: e.TMDBContentSections,
	}

	return runEnhancement(opts)
}

func initLogging() {
	// Create a human-readable handler for logging
	handler := humanlog.NewHandler(os.Stdout, &humanlog.Options{
		Level: slog.LevelInfo,
	})

	// Set the default logger
	slog.SetDefault(slog.New(handler))
}
