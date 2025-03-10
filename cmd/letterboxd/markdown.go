package letterboxd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/lepinkainen/hermes/internal/fileutil"
	log "github.com/sirupsen/logrus"
)

const movieTemplate = `---
title: "{{.Name}}"
type: movie
year: {{.Year}}
date_watched: {{.Date}}
letterboxd_uri: {{.LetterboxdURI}}
letterboxd_id: {{.LetterboxdID}}
{{- if .ImdbID}}
imdb_id: {{.ImdbID}}
{{- end}}
{{- if .Rating}}
letterboxd_rating: {{.Rating}}
{{- end}}
{{- if .Runtime}}
runtime_mins: {{.Runtime}}
duration: {{formatDuration .Runtime}}
{{- end}}
{{- if .Genres}}
genres:
{{- range .Genres}}
  - "{{.}}"
{{- end}}
{{- end}}
{{- if .Director}}
directors:
  - "{{.Director}}"
{{- end}}
tags:
  - letterboxd/movie
  {{- if .Rating}}
  - rating/{{.Rating}}
  {{- end}}
  - year/{{if ge .Year 2020}}2020s{{else if ge .Year 2010}}2010s{{else if ge .Year 2000}}2000s{{else if ge .Year 1990}}1990s{{else if ge .Year 1980}}1980s{{else if ge .Year 1970}}1970s{{else if ge .Year 1960}}1960s{{else if ge .Year 1950}}1950s{{else}}pre-1950s{{end}}
---

{{- if .PosterURL}}
![][{{.PosterURL}}]
{{- end}}

{{- if .Description}}

>[!summary]- Plot
> {{.Description}}
{{- end}}

{{- if .Cast}}

>[!cast]- Cast
> {{- range .Cast}}
> - {{.}}
{{- end}}
{{- end}}

>[!info]- Letterboxd
> [View on Letterboxd]({{.LetterboxdURI}})
{{- if .ImdbID}}
> [View on IMDb](https://www.imdb.com/title/{{.ImdbID}})
{{- end}}
`

// Helper function to format runtime into hours and minutes
func formatDuration(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// writeMovieToMarkdown writes a single movie to a markdown file
func writeMovieToMarkdown(movie Movie, directory string) error {
	// Create a sanitized title
	sanitizedTitle := fileutil.SanitizeFilename(movie.Name)

	// For year-specific filenames, we need to customize the path
	filename := fmt.Sprintf("%s (%d)", sanitizedTitle, movie.Year)
	filePath := filepath.Join(directory, filename+".md")

	// Log the current overwrite setting
	log.Debugf("Processing %s with overwrite=%v", filename, overwrite)

	// Check if file exists and if we should skip it
	if fileutil.FileExists(filePath) {
		if !overwrite {
			log.Infof("File %s already exists, skipping (overwrite=%v)", filePath, overwrite)
			return nil
		} else {
			log.Infof("File %s already exists, overwriting", filePath)
		}
	}

	// Parse the template with custom functions
	funcMap := template.FuncMap{
		"formatDuration": formatDuration,
		"ge":             func(a, b int) bool { return a >= b },
	}

	tmpl, err := template.New("movie").Funcs(funcMap).Parse(movieTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, movie); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	log.Infof("Wrote %s", filePath)
	return nil
}
