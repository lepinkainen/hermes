package diff

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/spf13/viper"
)

// DiffCmd groups diff-related subcommands.
type DiffCmd struct {
	IMDbLetterboxd IMDbLetterboxdCmd `cmd:"" help:"Diff IMDb and Letterboxd movie imports"`
}

// IMDbLetterboxdCmd generates a diff report between IMDb and Letterboxd movies.
type IMDbLetterboxdCmd struct {
	Output string `short:"o" help:"Path to output Markdown file"`
	DBFile string `help:"Path to main SQLite database file"`
}

func (c *IMDbLetterboxdCmd) Run() error {
	mainDB := c.DBFile
	if mainDB == "" {
		mainDB = viper.GetString("datasette.dbfile")
	}
	if mainDB == "" {
		mainDB = "./hermes.db"
	}

	cacheDB := viper.GetString("cache.dbfile")
	if cacheDB == "" {
		cacheDB = "./cache.db"
	}

	now := time.Now()
	outputPath := c.Output
	if outputPath == "" {
		baseDir := viper.GetString("markdownoutputdir")
		if baseDir == "" {
			baseDir = "markdown"
		}
		filename := fmt.Sprintf("imdb_letterboxd_diff-%s.md", now.Format("2006-01-02"))
		outputPath = filepath.Join(baseDir, "diffs", filename)
	}

	note, err := BuildIMDbLetterboxdReport(mainDB, cacheDB, now)
	if err != nil {
		return err
	}

	content, err := note.Build()
	if err != nil {
		return err
	}

	return fileutil.WriteMarkdownFile(outputPath, string(content), config.OverwriteFiles)
}
