package cmd

import (
	"fmt"
	"os"

	"github.com/lepinkainen/hermes/internal/config"
	log "github.com/sirupsen/logrus"
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

	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Info("Config file not found, writing default config file...")
			if err := viper.SafeWriteConfig(); err != nil {
				fmt.Printf("Error writing config file: %v\n", err)
			}
			os.Exit(0)
		} else {
			log.Panicf("fatal error config file: %v", err)
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
	// Set default log level
	log.SetLevel(log.InfoLevel)

	// Set up custom formatter
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
		PadLevelText:           true,
		DisableColors:          false,
	})
}
