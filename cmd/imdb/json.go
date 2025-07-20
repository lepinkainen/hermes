package imdb

import (
	"github.com/lepinkainen/hermes/internal/fileutil"
)

func writeMovieToJson(movies []MovieSeen, filename string) error {
	_, err := fileutil.WriteJSONFile(movies, filename, overwrite)
	return err
}
