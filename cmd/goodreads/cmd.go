package goodreads

import (
	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	csvFile    string
	outputDir  string
	writeJSON  bool
	jsonOutput string
	overwrite  bool
	cmdConfig  *cmdutil.BaseCommandConfig
)

var importCmd = &cobra.Command{
	Use:   "goodreads",
	Short: "Import books from Goodreads library export",
	Long: `Import books from your Goodreads library export CSV file and create markdown files with detailed information.
The CSV file can be exported from your Goodreads account settings.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// If flags weren't provided, try to get values from config
		if csvFile == "" {
			csvFile = viper.GetString("goodreads.csvfile")
		}

		cmdConfig = &cmdutil.BaseCommandConfig{
			OutputDir:  outputDir,
			ConfigKey:  "goodreads",
			WriteJSON:  writeJSON,
			JSONOutput: jsonOutput,
			Overwrite:  overwrite,
		}
		if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
			return err
		}
		outputDir = cmdConfig.OutputDir
		jsonOutput = cmdConfig.JSONOutput

		// Still require the values to be present somewhere
		if csvFile == "" {
			return cmd.MarkFlagRequired("input")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return ParseGoodreads()
	},
}

func init() {
	importCmd.Flags().StringVarP(&csvFile, "input", "f", "", "Path to Goodreads library export CSV file (required if not in config)")
	cmdutil.AddOutputFlag(importCmd, &outputDir, "goodreads", "Subdirectory under markdown output directory for Goodreads files")
	cmdutil.AddJSONFlags(importCmd, &writeJSON, &jsonOutput)

	// Use the global overwrite flag by default
	overwrite = config.OverwriteFiles
}

func GetCommand() *cobra.Command {
	return importCmd
}

// ParseGoodreadsWithParams allows calling goodreads parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseGoodreadsWithParams(inputFile, outputDir string, writeJSON bool, jsonOutput string, overwrite bool) error {
	// Set the global variables that ParseGoodreads expects
	csvFile = inputFile
	
	// Set up command config similar to PreRunE logic
	cmdConfig = &cmdutil.BaseCommandConfig{
		OutputDir:  outputDir,
		ConfigKey:  "goodreads",
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
	return ParseGoodreads()
}
