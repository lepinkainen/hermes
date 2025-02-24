package goodreads

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	csvFile   string
	outputDir string
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
		if outputDir == "" {
			outputDir = viper.GetString("goodreads.output")
		}

		// Combine the base markdown directory with the goodreads subdirectory
		baseDir := viper.GetString("markdownoutputdir")
		outputDir = filepath.Join(baseDir, outputDir)

		// Still require the values to be present somewhere
		if csvFile == "" {
			return cmd.MarkFlagRequired("csvfile")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return ParseGoodreads()
	},
}

func init() {
	importCmd.Flags().StringVarP(&csvFile, "csvfile", "f", "", "Path to Goodreads library export CSV file (required if not in config)")
	importCmd.Flags().StringVarP(&outputDir, "output", "o", "goodreads", "Subdirectory under markdown output directory for Goodreads files")
}

func GetCommand() *cobra.Command {
	return importCmd
}
