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

// ParseSteamWithParams allows calling steam parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseSteamWithParams(steamIDParam, apiKeyParam, outputDir string, writeJSON bool, jsonOutput string, overwrite bool) error {
	// Set the global variables that ParseSteam expects
	steamID = steamIDParam
	apiKey = apiKeyParam

	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDir,
		ConfigKey:  "steam",
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
	return ParseSteam()
}
