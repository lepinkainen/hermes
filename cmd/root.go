package cmd

import (
	"log/slog"
	"os"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/humanlog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hermes",
	Short: "A tool to import data from various sources",
	Long:  `A tool to import data from various sources into a unified format.`,
}

// Global flags
var (
	overwriteFiles bool
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	initLogging()

	viper.SetDefault("MarkdownOutputDir", "./markdown/")
	viper.SetDefault("JSONOutputDir", "./json/")
	viper.SetDefault("OverwriteFiles", false)

	// Datasette defaults
	viper.SetDefault("datasette.enabled", false)
	viper.SetDefault("datasette.mode", "local")
	viper.SetDefault("datasette.dbfile", "./hermes.db")
	viper.SetDefault("datasette.remote_url", "")
	viper.SetDefault("datasette.api_token", "")

	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory

	// Add Datasette flags
	rootCmd.PersistentFlags().Bool("datasette", false, "Enable Datasette output")
	rootCmd.PersistentFlags().String("datasette-mode", "local", "Datasette mode (local or remote)")
	rootCmd.PersistentFlags().String("datasette-dbfile", "./hermes.db", "Path to local SQLite database file")
	rootCmd.PersistentFlags().String("datasette-url", "", "URL of remote Datasette instance")
	rootCmd.PersistentFlags().String("datasette-token", "", "API token for remote Datasette instance")

	// Bind Datasette flags to viper
	viper.BindPFlag("datasette.enabled", rootCmd.PersistentFlags().Lookup("datasette"))
	viper.BindPFlag("datasette.mode", rootCmd.PersistentFlags().Lookup("datasette-mode"))
	viper.BindPFlag("datasette.dbfile", rootCmd.PersistentFlags().Lookup("datasette-dbfile"))
	viper.BindPFlag("datasette.remote_url", rootCmd.PersistentFlags().Lookup("datasette-url"))
	viper.BindPFlag("datasette.api_token", rootCmd.PersistentFlags().Lookup("datasette-token"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found, writing default config file...")
			if err := viper.SafeWriteConfig(); err != nil {
				slog.Error("Error writing config file", "error", err)
			}
			os.Exit(0)
		} else {
			slog.Error("Fatal error config file", "error", err)
			os.Exit(1)
		}
	}

	// Initialize global config
	config.InitConfig()

	// Add global flags
	rootCmd.PersistentFlags().BoolVar(&overwriteFiles, "overwrite", viper.GetBool("OverwriteFiles"), "Overwrite existing markdown files when processing")

	// Update the config when the flag changes
	cobra.OnInitialize(func() {
		config.SetOverwriteFiles(overwriteFiles)
	})
}

func initLogging() {
	// Create a human-readable handler for logging
	handler := humanlog.NewHandler(os.Stdout, &humanlog.Options{
		Level:        slog.LevelInfo,
		TimeFormat:   "15:04:05",
		DisableColor: false,
	})

	// Set the default logger
	slog.SetDefault(slog.New(handler))
}
