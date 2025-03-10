package letterboxd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lepinkainen/hermes/internal/fileutil"
	log "github.com/sirupsen/logrus"
)

// writeJSONFile writes the movies to a JSON file
func writeJSONFile(movies []Movie, filename string) error {
	// Check if file exists and we shouldn't overwrite
	if fileutil.FileExists(filename) && !overwrite {
		log.Infof("JSON file %s already exists, skipping (overwrite=%v)", filename, overwrite)
		return nil
	}

	data, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		return err
	}

	// Ensure the output directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	log.Infof("Writing JSON file to %s (overwrite=%v)", filename, overwrite)
	return os.WriteFile(filename, data, 0644)
}
