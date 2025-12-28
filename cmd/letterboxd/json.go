package letterboxd

import (
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

// writeJSONFile writes the movies to a JSON file
func writeJSONFile(movies []Movie, filename string) error {
	_, err := fileutil.WriteJSONFile(movies, filename, config.OverwriteFiles)
	return err
}
