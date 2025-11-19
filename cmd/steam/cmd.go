package steam

import (
	"github.com/lepinkainen/hermes/internal/cmdutil"
)

var (
	steamID    string
	apiKey     string
	outputDir  string
	writeJSON  bool
	jsonOutput string
	overwrite  bool
	cmdConfig  *cmdutil.BaseCommandConfig
)

var parseSteamFunc = ParseSteam

// ParseSteamWithParams allows calling steam parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseSteamWithParams(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string, overwriteParam bool) error {
	// Set the global variables that ParseSteam expects
	steamID = steamIDParam
	apiKey = apiKeyParam

	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDirParam,
		ConfigKey:  "steam",
		WriteJSON:  writeJSONParam,
		JSONOutput: jsonOutputParam,
		Overwrite:  overwriteParam,
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
	return parseSteamFunc()
}
