package letterboxd

import (
	"github.com/lepinkainen/hermes/internal/cmdutil"
)

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
)

// ParseLetterboxdWithParams allows calling letterboxd parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseLetterboxdWithParams(inputFile, outputDir string, writeJSON bool, jsonOutput string, overwrite bool) error {
	// Set the global variables that ParseLetterboxd expects
	csvFile = inputFile
	skipInvalid = false // Default value
	skipEnrich = false  // Default value

	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDir,
		ConfigKey:  "letterboxd",
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
	return ParseLetterboxd()
}
