package imdb

import (
	"github.com/lepinkainen/hermes/internal/cmdutil"
)

// Define package-level variables for flags
var (
	csvFile     string
	outputDir   string
	writeJSON   bool
	jsonOutput  string
	skipInvalid bool
	overwrite   bool
	cmdConfig   *cmdutil.BaseCommandConfig
	// TMDB enrichment options
	tmdbEnabled         bool
	tmdbDownloadCover   bool
	tmdbGenerateContent bool
	tmdbInteractive     bool
	tmdbContentSections []string
)

// ParseImdbWithParams allows calling imdb parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseImdbWithParams(
	inputFile, outputDirParam string,
	writeJSONFlag bool,
	jsonOutputPath string,
	overwriteFlag bool,
	tmdbEnabledFlag bool,
	tmdbDownloadCoverFlag bool,
	tmdbGenerateContentFlag bool,
	tmdbInteractiveFlag bool,
	tmdbContentSectionsFlag []string,
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

	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDirParam,
		ConfigKey:  "imdb",
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

	// Call the existing parser
	return ParseImdb()
}
