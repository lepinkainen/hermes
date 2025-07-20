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
)

// ParseImdbWithParams allows calling imdb parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseImdbWithParams(inputFile, outputDir string, writeJSON bool, jsonOutput string, overwrite bool) error {
	// Set the global variables that ParseImdb expects
	csvFile = inputFile
	skipInvalid = false // Default value

	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDir,
		ConfigKey:  "imdb",
		WriteJSON:  writeJSON,
		JSONOutput: jsonOutput,
		Overwrite:  overwrite,
	}

	if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
		return err
	}

	// Update package-level global variables with processed paths for parser usage
	// Need to work around parameter shadowing by creating local vars with different names
	globalOutputDir := &outputDir
	globalJSONOutput := &jsonOutput
	*globalOutputDir = cmdConfig.OutputDir
	*globalJSONOutput = cmdConfig.JSONOutput

	// Call the existing parser
	return ParseImdb()
}
