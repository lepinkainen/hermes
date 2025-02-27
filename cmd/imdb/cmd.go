package imdb

import (
	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/config"
	log "github.com/sirupsen/logrus"
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
		log.Info("Processing imdb export...")
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
