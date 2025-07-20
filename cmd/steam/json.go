package steam

import (
	"github.com/lepinkainen/hermes/internal/fileutil"
)

func writeGameToJson(games []GameDetails, filename string) error {
	_, err := fileutil.WriteJSONFile(games, filename, overwrite)
	return err
}
