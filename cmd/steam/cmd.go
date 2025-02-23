package steam

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	steamID   string
	apiKey    string
	outputDir string
)

var importCmd = &cobra.Command{
	Use:   "steam",
	Short: "Import games from Steam library",
	Long:  `Import games from your Steam library and create markdown files with detailed information.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// If flags weren't provided, try to get values from config
		if steamID == "" {
			steamID = viper.GetString("steam.steamid")
		}
		if apiKey == "" {
			apiKey = viper.GetString("steam.apikey")
		}
		if outputDir == "" {
			outputDir = viper.GetString("steam.output")
		}

		// Combine the base markdown directory with the steam subdirectory
		baseDir := viper.GetString("markdownoutputdir")
		outputDir = filepath.Join(baseDir, outputDir)

		// Still require the values to be present somewhere
		if steamID == "" {
			return cmd.MarkFlagRequired("steamid")
		}
		if apiKey == "" {
			return cmd.MarkFlagRequired("apikey")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return ParseSteam()
	},
}

func init() {
	importCmd.Flags().StringVarP(&steamID, "steamid", "s", "", "Steam ID of the user (required if not in config)")
	importCmd.Flags().StringVarP(&apiKey, "apikey", "k", "", "Steam API key (required if not in config)")
	importCmd.Flags().StringVarP(&outputDir, "output", "o", "steam", "Subdirectory under markdown output directory for Steam files")
}

func GetCommand() *cobra.Command {
	return importCmd
}
