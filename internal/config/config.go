package config

import (
	"github.com/spf13/viper"
)

// Global configuration variables
var (
	// OverwriteFiles controls whether existing markdown files should be overwritten
	OverwriteFiles bool
	// TMDBAPIKey is the API key for TheMovieDB
	TMDBAPIKey string
	// OMDBAPIKey is the API key for OMDB (Open Movie Database)
	OMDBAPIKey string
)

// InitConfig initializes the global configuration
func InitConfig() {
	// Set default values
	viper.SetDefault("MarkdownOutputDir", "./markdown/")
	viper.SetDefault("JSONOutputDir", "./json/")
	viper.SetDefault("OverwriteFiles", false)

	// Get values from viper
	OverwriteFiles = viper.GetBool("OverwriteFiles")
	TMDBAPIKey = viper.GetString("TMDBAPIKey")
	OMDBAPIKey = viper.GetString("OMDBAPIKey")
}

// SetOverwriteFiles sets the OverwriteFiles flag
func SetOverwriteFiles(overwrite bool) {
	OverwriteFiles = overwrite
}
