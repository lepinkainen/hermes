package letterboxd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/spf13/viper"
)

const defaultAutomationTimeout = 3 * time.Minute

type ParseLetterboxdFuncType func(
	inputFile, outputDirParam string,
	writeJSONFlag bool,
	jsonOutputPath string,
	tmdbEnabledFlag bool,
	tmdbDownloadCoverFlag bool,
	tmdbGenerateContentFlag bool,
	tmdbInteractiveFlag bool,
	tmdbContentSectionsFlag []string,
	useTMDBCoverCacheFlag bool,
	tmdbCoverCachePathFlag string,
) error
type DownloadLetterboxdCSVFuncType func(ctx context.Context, opts AutomationOptions) (string, error)

// Define package-level variables for flags
var (
	csvFile     string
	skipInvalid bool
	cmdConfig   *cmdutil.BaseCommandConfig
	// Global variables referenced by the parser
	outputDir  string
	writeJSON  bool
	jsonOutput string
	overwrite  bool
	// TMDB enrichment options
	tmdbEnabled         bool
	tmdbDownloadCover   bool
	tmdbGenerateContent bool
	tmdbInteractive     bool
	tmdbContentSections []string
	// TMDB cover cache options
	useTMDBCoverCache  bool
	tmdbCoverCachePath string
)

var parseLetterboxdFunc = ParseLetterboxd
var downloadLetterboxdCSV = downloadLetterboxdZip

// DownloadLetterboxdCSV wraps the automation function for use by root.go
func DownloadLetterboxdCSV(ctx context.Context, opts AutomationOptions) (string, error) {
	return downloadLetterboxdCSV(ctx, opts)
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
	// Automation options
	Automated          bool          `help:"Automatically download Letterboxd export"`
	LetterboxdUsername string        `help:"Letterboxd account username"`
	LetterboxdPassword string        `help:"Letterboxd account password"`
	Headful            bool          `help:"Show browser window during automation"`
	DownloadDir        string        `help:"Directory for downloaded exports" default:"exports"`
	AutomationTimeout  time.Duration `help:"Timeout for automation process" default:"3m"`
	DryRun             bool          `help:"Run automation without importing (testing)"`
}

func (l *LetterboxdCmd) Run() error {
	// Read from config if value not provided via flag
	input := l.Input
	if input == "" && !l.Automated {
		input = viper.GetString("letterboxd.csvfile")
	}

	automationTimeout := l.AutomationTimeout
	if automationTimeout == 0 {
		automationTimeout = viper.GetDuration("letterboxd.automation.timeout")
		if automationTimeout == 0 {
			automationTimeout = 3 * time.Minute
		}
	}

	headful := l.Headful
	if !headful && viper.IsSet("letterboxd.automation.headful") {
		headful = viper.GetBool("letterboxd.automation.headful")
	}

	downloadDir := l.DownloadDir
	if downloadDir == "" {
		downloadDir = viper.GetString("letterboxd.automation.download_dir")
	}

	username := l.LetterboxdUsername
	if username == "" {
		username = viper.GetString("letterboxd.automation.username")
	}
	password := l.LetterboxdPassword
	if password == "" {
		password = viper.GetString("letterboxd.automation.password")
	}

	// Check if required value is still missing when not automated
	if input == "" && !l.Automated {
		return fmt.Errorf("input CSV file is required (provide via --input flag or letterboxd.csvfile in config)")
	}

	// Handle automation
	if l.Automated {
		if username == "" || password == "" {
			return fmt.Errorf("letterboxd automation requires both username and password")
		}

		ctx, cancel := context.WithTimeout(context.Background(), automationTimeout)
		defer cancel()

		opts := AutomationOptions{
			Username:    username,
			Password:    password,
			DownloadDir: downloadDir,
			Headless:    !headful,
			Timeout:     automationTimeout,
		}

		csvPath, err := DownloadLetterboxdCSV(ctx, opts)
		if err != nil {
			return fmt.Errorf("letterboxd automation failed: %w", err)
		}

		if l.DryRun {
			slog.Info("Dry-run mode: automation completed successfully", "csv_path", csvPath)
			return nil
		}

		input = csvPath
	}

	return ParseLetterboxdWithParams(
		input,
		l.Output,
		l.JSON,
		l.JSONOutput,
		l.TMDBEnabled,
		l.TMDBDownloadCover,
		l.TMDBGenerateContent,
		!l.TMDBNoInteractive, // Invert: default is interactive
		l.TMDBContentSections,
		viper.GetBool("tmdb.cover_cache.enabled"),
		viper.GetString("tmdb.cover_cache.path"),
	)
}

// ParseLetterboxdWithParams allows calling letterboxd parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseLetterboxdWithParams(
	inputFile, outputDirParam string,
	writeJSONFlag bool,
	jsonOutputPath string,
	tmdbEnabledFlag bool,
	tmdbDownloadCoverFlag bool,
	tmdbGenerateContentFlag bool,
	tmdbInteractiveFlag bool,
	tmdbContentSectionsFlag []string,
	useTMDBCoverCacheFlag bool,
	tmdbCoverCachePathFlag string,
) error {
	// Set the global variables that ParseLetterboxd expects
	csvFile = inputFile
	skipInvalid = false // Default value

	// Set TMDB flags
	tmdbEnabled = tmdbEnabledFlag
	tmdbDownloadCover = tmdbDownloadCoverFlag
	tmdbGenerateContent = tmdbGenerateContentFlag
	tmdbInteractive = tmdbInteractiveFlag
	tmdbContentSections = tmdbContentSectionsFlag
	useTMDBCoverCache = useTMDBCoverCacheFlag
	tmdbCoverCachePath = tmdbCoverCachePathFlag

	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDirParam,
		ConfigKey:  "letterboxd",
		WriteJSON:  writeJSONFlag,
		JSONOutput: jsonOutputPath,
	}

	if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
		return err
	}

	// Update package-level global variables with processed paths for parser usage
	outputDir = cmdConfig.OutputDir
	writeJSON = cmdConfig.WriteJSON
	jsonOutput = cmdConfig.JSONOutput

	// Call the existing parser
	return parseLetterboxdFunc()
}
