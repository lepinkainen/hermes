package cmd

import (
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/lepinkainen/hermes/cmd/goodreads"
	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/letterboxd"
	"github.com/lepinkainen/hermes/cmd/steam"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/humanlog"
	"github.com/spf13/viper"
)

// CLI represents the complete command structure for the hermes application
type CLI struct {
	// Global flags
	Overwrite bool `help:"Overwrite existing markdown files when processing"`

	// Datasette flags
	Datasette     bool   `help:"Enable Datasette output"`
	DatasetteMode string `help:"Datasette mode (local or remote)" default:"local" enum:"local,remote"`
	DatasetteDB   string `help:"Path to local SQLite database file" default:"./hermes.db"`
	DatasetteURL  string `help:"URL of remote Datasette instance"`
	DatasetteToken string `help:"API token for remote Datasette instance"`

	Import ImportCmd `cmd:"" help:"Import data from various sources"`

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
	Input      string `short:"f" help:"Path to Goodreads library export CSV file" required:""`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for Goodreads files" default:"goodreads"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/goodreads.json)"`
}

// IMDBCmd represents the imdb import command
type IMDBCmd struct {
	Input      string `short:"f" help:"Path to IMDB CSV file" required:""`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for IMDB files" default:"imdb"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/imdb.json)"`
}

// LetterboxdCmd represents the letterboxd import command
type LetterboxdCmd struct {
	Input      string `short:"f" help:"Path to Letterboxd CSV file" required:""`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for Letterboxd files" default:"letterboxd"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/letterboxd.json)"`
}

// SteamCmd represents the steam import command
type SteamCmd struct {
	SteamID    string `help:"Steam ID to fetch data for" required:""`
	APIKey     string `help:"Steam API key"`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for Steam files" default:"steam"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/steam.json)"`
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
	viper.SetDefault("datasette.enabled", false)
	viper.SetDefault("datasette.mode", "local")
	viper.SetDefault("datasette.dbfile", "./hermes.db")
	viper.SetDefault("datasette.remote_url", "")
	viper.SetDefault("datasette.api_token", "")

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
	
	// Update datasette config
	viper.Set("datasette.enabled", cli.Datasette)
	viper.Set("datasette.mode", cli.DatasetteMode)
	viper.Set("datasette.dbfile", cli.DatasetteDB)
	viper.Set("datasette.remote_url", cli.DatasetteURL)
	viper.Set("datasette.api_token", cli.DatasetteToken)
}

// Run methods for each command

func (g *GoodreadsCmd) Run() error {
	return goodreads.ParseGoodreadsWithParams(g.Input, g.Output, g.JSON, g.JSONOutput, false)
}

func (i *IMDBCmd) Run() error {
	return imdb.ParseImdbWithParams(i.Input, i.Output, i.JSON, i.JSONOutput, false)
}

func (l *LetterboxdCmd) Run() error {
	return letterboxd.ParseLetterboxdWithParams(l.Input, l.Output, l.JSON, l.JSONOutput, false)
}

func (s *SteamCmd) Run() error {
	return steam.ParseSteamWithParams(s.SteamID, s.APIKey, s.Output, s.JSON, s.JSONOutput, false)
}

func initLogging() {
	// Create a human-readable handler for logging
	handler := humanlog.NewHandler(os.Stdout, &humanlog.Options{
		Level:        slog.LevelInfo,
		TimeFormat:   "15:04:05",
		DisableColor: false,
	})

	// Set the default logger
	slog.SetDefault(slog.New(handler))
}