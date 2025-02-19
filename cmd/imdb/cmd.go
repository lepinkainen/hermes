package imdb

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Define package-level variables for flags
var (
	inputFile   string
	outputJson  string
	skipInvalid bool
	logLevel    string
)

// GetCommand returns the imdb command
func GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "imdb",
		Short: "Parse IMDB export",
		Long: `Parse IMDB export files into JSON and Markdown formats.
Supports both ratings and watchlist exports.`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Processing imdb export...")
			parse_imdb()
		},
	}

	// Define flags
	cmd.Flags().StringVar(&inputFile, "input", "imdb_export.csv", "Input CSV file path")
	cmd.Flags().StringVar(&outputJson, "output-json", "movies.json", "Output JSON file path")
	cmd.Flags().BoolVar(&skipInvalid, "skip-invalid", false, "Skip invalid entries instead of failing")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	return cmd
}
