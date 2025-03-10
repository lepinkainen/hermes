package letterboxd

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
)

// writeJSONFile writes the movies to a JSON file
func writeJSONFile(movies []Movie, filename string) error {
	data, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		return err
	}

	log.Infof("Writing JSON file to %s\n", filename)
	return os.WriteFile(filename, data, 0644)
}
