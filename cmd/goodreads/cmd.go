package goodreads

import (
	"github.com/lepinkainen/hermes/internal/cmdutil"
)

var (
	csvFile    string
	outputDir  string
	writeJSON  bool
	jsonOutput string
	overwrite  bool
	cmdConfig  *cmdutil.BaseCommandConfig
)

// ParseGoodreadsWithParams allows calling goodreads parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseGoodreadsWithParams(inputFile, outputDirParam string, writeJSONParam bool, jsonOutputParam string, overwriteParam bool) error {
	// Set the global variables that ParseGoodreads expects
	csvFile = inputFile

	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDirParam,
		ConfigKey:  "goodreads",
		WriteJSON:  writeJSONParam,
		JSONOutput: jsonOutputParam,
		Overwrite:  overwriteParam,
	}

	if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
		return err
	}

	// Update package-level globals with resolved paths and flags
	outputDir = cmdConfig.OutputDir
	writeJSON = cmdConfig.WriteJSON
	jsonOutput = cmdConfig.JSONOutput
	overwrite = cmdConfig.Overwrite

	// Call the existing parser
	return ParseGoodreads()
}
