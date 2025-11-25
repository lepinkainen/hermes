package goodreads

import (
	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
)

func writeBookToJson(books []Book, filename string) error {
	_, err := fileutil.WriteJSONFile(books, filename, config.OverwriteFiles)
	return err
}
