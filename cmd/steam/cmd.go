package steam

import (
	"fmt"

	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/spf13/viper"
)

// SteamCmd represents the steam import command
type SteamCmd struct {
	SteamID    string `help:"Steam ID to fetch data for"`
	APIKey     string `help:"Steam API key"`
	Output     string `short:"o" help:"Subdirectory under markdown output directory for Steam files" default:"steam"`
	JSON       bool   `help:"Write data to JSON format"`
	JSONOutput string `help:"Path to JSON output file (defaults to json/steam.json)"`
}

func (s *SteamCmd) Run() error {
	// Read from config if values not provided via flags
	steamID := s.SteamID
	if steamID == "" {
		steamID = viper.GetString("steam.steamid")
	}

	apiKey := s.APIKey
	if apiKey == "" {
		apiKey = viper.GetString("steam.apikey")
	}

	// Check if required values are still missing
	if steamID == "" {
		return fmt.Errorf("steam ID is required (provide via --steamid flag or steam.steamid in config)")
	}
	if apiKey == "" {
		return fmt.Errorf("steam API key is required (provide via --apikey flag or steam.apikey in config)")
	}

	return ParseSteamWithParamsFunc(steamID, apiKey, s.Output, s.JSON, s.JSONOutput, false)
}

var (
	steamID    string
	apiKey     string
	outputDir  string
	writeJSON  bool
	jsonOutput string
	overwrite  bool
	cmdConfig  *cmdutil.BaseCommandConfig
)

// ParseSteamFuncType is the signature of the low-level steam parser function (ParseSteam)
type ParseSteamFuncType func() error

// ParseSteamFunc is a variable that can be overridden for testing purposes
var ParseSteamFunc ParseSteamFuncType = ParseSteam

// ParseSteamWithParamsFuncType is the signature of the ParseSteamWithParams function
type ParseSteamWithParamsFuncType func(steamIDParam, apiKeyParam, outputDirParam string, writeJSONParam bool, jsonOutputParam string, overwriteParam bool) error

// ParseSteamWithParamsFunc is a variable that can be overridden for testing purposes
var ParseSteamWithParamsFunc ParseSteamWithParamsFuncType = ParseSteamWithParams


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
	return ParseSteamFunc()
}
