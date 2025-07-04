package imdb

import (
	"log/slog"

	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

// GetCommand returns the imdb command
var importCmd = &cobra.Command{
	Use:   "imdb",
	Short: "Parse IMDB export",
	Long: `Parse IMDB export files into JSON and Markdown formats.
Supports both ratings and watchlist exports.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// If flags weren't provided, try to get values from config
		if csvFile == "" {
			csvFile = viper.GetString("imdb.csvfile")
		}

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
		outputDir = cmdConfig.OutputDir
		jsonOutput = cmdConfig.JSONOutput

		if csvFile == "" {
			return cmd.MarkFlagRequired("input")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("Processing imdb export...")
		return ParseImdb()
	},
}

func init() {
	importCmd.Flags().StringVarP(&csvFile, "input", "f", "", "Input CSV file path")
	cmdutil.AddOutputFlag(importCmd, &outputDir, "imdb", "Subdirectory under markdown output directory for IMDB files")
	cmdutil.AddJSONFlags(importCmd, &writeJSON, &jsonOutput)
	importCmd.Flags().BoolVar(&skipInvalid, "skip-invalid", false, "Skip invalid entries instead of failing")

	// Use the global overwrite flag by default
	overwrite = config.OverwriteFiles
}

func GetCommand() *cobra.Command {
	return importCmd
}

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
