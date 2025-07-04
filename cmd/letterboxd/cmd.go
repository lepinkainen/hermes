package letterboxd

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
	skipEnrich  bool
	cmdConfig   *cmdutil.BaseCommandConfig
)

// Define the command
var importCmd = &cobra.Command{
	Use:   "letterboxd",
	Short: "Parse Letterboxd export",
	Long: `Parse Letterboxd export files into JSON and Markdown formats.
Supports watched movies exports from Letterboxd.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// If flags weren't provided, try to get values from config
		if csvFile == "" {
			csvFile = viper.GetString("letterboxd.csvfile")
		}

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
		outputDir = cmdConfig.OutputDir
		jsonOutput = cmdConfig.JSONOutput

		if csvFile == "" {
			return cmd.MarkFlagRequired("input")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Update the overwrite flag from the global config right before running
		overwrite = config.OverwriteFiles
		slog.Info("Processing Letterboxd export", "overwrite", overwrite)
		return ParseLetterboxd()
	},
}

func init() {
	importCmd.Flags().StringVarP(&csvFile, "input", "f", "", "Input CSV file path")
	cmdutil.AddOutputFlag(importCmd, &outputDir, "letterboxd", "Subdirectory under markdown output directory for Letterboxd files")
	cmdutil.AddJSONFlags(importCmd, &writeJSON, &jsonOutput)
	importCmd.Flags().BoolVar(&skipInvalid, "skip-invalid", false, "Skip invalid entries instead of failing")
	importCmd.Flags().BoolVar(&skipEnrich, "skip-enrich", false, "Skip enriching data with OMDB API")

	// Use the global overwrite flag by default
	overwrite = config.OverwriteFiles
}

// GetCommand returns the letterboxd command
func GetCommand() *cobra.Command {
	return importCmd
}

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
