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
	IMDbLetterboxd IMDbLetterboxdCmd `cmd:"" name:"imdb-letterboxd" help:"Diff IMDb and Letterboxd movie imports"`
}

// IMDbLetterboxdCmd generates a diff report between IMDb and Letterboxd movies.
type IMDbLetterboxdCmd struct {
	Output     string `short:"o" help:"Path to output Markdown file"`
	HTMLOutput string `short:"H" name:"html" help:"Path to output HTML file"`
	DBFile     string `help:"Path to main SQLite database file"`
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

	// Build the diff report (shared data for both outputs)
	report, err := BuildDiffReport(mainDB, cacheDB, now)
	if err != nil {
		return err
	}

	// Generate markdown output (default if no --html flag, or if -o is specified)
	if c.HTMLOutput == "" || c.Output != "" {
		outputPath := c.Output
		if outputPath == "" {
			baseDir := viper.GetString("markdownoutputdir")
			if baseDir == "" {
				baseDir = "markdown"
			}
			filename := fmt.Sprintf("imdb_letterboxd_diff-%s.md", now.Format("2006-01-02"))
			outputPath = filepath.Join(baseDir, "diffs", filename)
		}

		note := buildDiffNote(report.ImdbOnly, report.LetterboxdOnly, report.Stats, report.GeneratedAt, report.MainDBPath, report.CacheDBPath)
		content, err := note.Build()
		if err != nil {
			return err
		}

		if err := fileutil.WriteMarkdownFile(outputPath, string(content), config.OverwriteFiles); err != nil {
			return err
		}
	}

	// Generate HTML output if --html flag is specified
	if c.HTMLOutput != "" {
		htmlContent, err := renderDiffHTML(report)
		if err != nil {
			return fmt.Errorf("failed to render HTML: %w", err)
		}

		if _, err := fileutil.WriteFileWithOverwrite(c.HTMLOutput, htmlContent, 0644, config.OverwriteFiles); err != nil {
			return fmt.Errorf("failed to write HTML file: %w", err)
		}
	}

	return nil
}
