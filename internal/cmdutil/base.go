package cmdutil

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// BaseCommandConfig holds common configuration for import commands
type BaseCommandConfig struct {
	OutputDir     string
	OutputDirFlag string
	ConfigKey     string
}

// SetupOutputDir handles the common output directory setup logic
func SetupOutputDir(cfg *BaseCommandConfig) error {
	// If flag wasn't provided, try to get value from config
	if cfg.OutputDir == "" {
		cfg.OutputDir = viper.GetString(cfg.ConfigKey + ".output")
	}

	// Combine the base markdown directory with the specific subdirectory
	baseDir := viper.GetString("markdownoutputdir")
	cfg.OutputDir = filepath.Join(baseDir, cfg.OutputDir)
	return nil
}

// AddOutputFlag adds the common output directory flag to a command
func AddOutputFlag(cmd *cobra.Command, outputDir *string, defaultValue, flagDesc string) {
	cmd.Flags().StringVarP(outputDir, "output", "o", defaultValue, flagDesc)
}
