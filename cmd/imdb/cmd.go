package imdb

import (
	"fmt"

	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/spf13/viper"
)

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

	return ParseImdbWithParams(
		input,
		i.Output,
		i.JSON,
		i.JSONOutput,
		i.TMDBEnabled,
		i.TMDBDownloadCover,
		i.TMDBGenerateContent,
		!i.TMDBNoInteractive, // Invert: default is interactive
		i.TMDBContentSections,
		viper.GetBool("tmdb.cover_cache.enabled"),
		viper.GetString("tmdb.cover_cache.path"),
	)
}

// Define package-level variables for flags
var (
	csvFile     string
	outputDir   string
	writeJSON   bool
	jsonOutput  string
	skipInvalid bool
	cmdConfig   *cmdutil.BaseCommandConfig
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

var parseImdbFunc = ParseImdb

// ParseImdbWithParams allows calling imdb parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseImdbWithParams(
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
	// Set the global variables that ParseImdb expects
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
		ConfigKey:  "imdb",
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
	return parseImdbFunc()
}
