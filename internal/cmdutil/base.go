package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// BaseCommandConfig holds common configuration for import commands
type BaseCommandConfig struct {
	OutputDir     string
	OutputDirFlag string
	ConfigKey     string
	JSONOutput    string
	WriteJSON     bool
	Overwrite     bool
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

	// If JSON output is enabled but no path specified, use default in json directory
	if cfg.WriteJSON && cfg.JSONOutput == "" {
		// Get the base JSON directory from config or use default
		jsonBaseDir := viper.GetString("jsonoutputdir")
		if jsonBaseDir == "" {
			jsonBaseDir = "json"
		}
		// Create filename based on parser name
		jsonFile := cfg.ConfigKey + ".json"
		cfg.JSONOutput = filepath.Join(jsonBaseDir, jsonFile)
	}

	// Create directories if they don't exist
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if cfg.WriteJSON {
		// Create JSON directory if it doesn't exist
		jsonDir := filepath.Dir(cfg.JSONOutput)
		if err := os.MkdirAll(jsonDir, 0755); err != nil {
			return fmt.Errorf("failed to create JSON output directory: %w", err)
		}
	}

	return nil
}
