package imdb

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Define package-level variables for flags
var (
	inputFile   string
	outputJson  string
	outputDir   string
	skipInvalid bool
	logLevel    string
)

// GetCommand returns the imdb command
var importCmd = &cobra.Command{
	Use:   "imdb",
	Short: "Parse IMDB export",
	Long: `Parse IMDB export files into JSON and Markdown formats.
Supports both ratings and watchlist exports.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// If flags weren't provided, try to get values from config
		if inputFile == "" {
			inputFile = viper.GetString("imdb.csvfile")
		}

		if outputDir == "" {
			outputDir = viper.GetString("goodreads.output")
		}

		// Combine the base markdown directory with the imdb subdirectory
		baseDir := viper.GetString("markdownoutputdir")
		outputDir = filepath.Join(baseDir, outputDir)

		if inputFile == "" {
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
	importCmd.Flags().StringVarP(&inputFile, "input", "f", "", "Input CSV file path")
	importCmd.Flags().StringVarP(&outputDir, "output", "o", "imdb", "Subdirectory under markdown output directory for IMDB files")
	importCmd.Flags().StringVar(&outputJson, "output-json", "movies.json", "Output JSON file path")
	importCmd.Flags().BoolVar(&skipInvalid, "skip-invalid", false, "Skip invalid entries instead of failing")
	importCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
}

func GetCommand() *cobra.Command {
	return importCmd
}
