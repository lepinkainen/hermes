package letterboxd

import (
	"context"
	"time"

	"github.com/lepinkainen/hermes/internal/cmdutil"
)

const defaultAutomationTimeout = 3 * time.Minute

// Define package-level variables for flags
var (
	csvFile     string
	skipInvalid bool
	skipEnrich  bool
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

// ParseLetterboxdWithParams allows calling letterboxd parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseLetterboxdWithParams(
	inputFile, outputDirParam string,
	writeJSONFlag bool,
	jsonOutputPath string,
	overwriteFlag bool,
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
	skipEnrich = false  // Default value

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
		Overwrite:  overwriteFlag,
	}

	if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
		return err
	}

	// Update package-level global variables with processed paths for parser usage
	outputDir = cmdConfig.OutputDir
	writeJSON = cmdConfig.WriteJSON
	jsonOutput = cmdConfig.JSONOutput
	overwrite = cmdConfig.Overwrite

	// Call the existing parser
	return parseLetterboxdFunc()
}
