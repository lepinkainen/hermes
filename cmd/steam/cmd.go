package steam

import (
	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		outputDir = cmdConfig.OutputDir
		jsonOutput = cmdConfig.JSONOutput

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
		err := ParseSteam()
		if err != nil {
			if rateLimitErr, ok := err.(*errors.RateLimitError); ok {
				log.Error(rateLimitErr.Message)
				return nil // Return nil to prevent Cobra from showing help text
			}
			return err
		}
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&steamID, "steamid", "s", "", "Steam ID of the user (required if not in config)")
	importCmd.Flags().StringVarP(&apiKey, "apikey", "k", "", "Steam API key (required if not in config)")
	cmdutil.AddOutputFlag(importCmd, &outputDir, "steam", "Subdirectory under markdown output directory for Steam files")
	cmdutil.AddJSONFlags(importCmd, &writeJSON, &jsonOutput)

	// Use the global overwrite flag by default
	overwrite = config.OverwriteFiles
}

func GetCommand() *cobra.Command {
	return importCmd
}
