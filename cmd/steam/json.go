package steam

import (
	"encoding/json"
	"fmt"
	"os"
)

func writeGameToJson(games []GameDetails, filename string) error {
	jsonData, err := json.MarshalIndent(games, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	jsonFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create JSON file '%s': %v", filename, err)
	}
	defer jsonFile.Close()

	_, err = jsonFile.Write(jsonData)
	return err
}
