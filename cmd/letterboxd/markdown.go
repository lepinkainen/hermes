package letterboxd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/lepinkainen/hermes/internal/fileutil"
	log "github.com/sirupsen/logrus"
)

const movieTemplate = `---
title: {{.Name}}
year: {{.Year}}
date_watched: {{.Date}}
letterboxd_uri: {{.LetterboxdURI}}
letterboxd_id: {{.LetterboxdID}}
{{- if .ImdbID}}
imdb_id: {{.ImdbID}}
{{- end}}
{{- if .Director}}
director: {{.Director}}
{{- end}}
{{- if .Cast}}
cast:
{{- range .Cast}}
  - {{.}}
{{- end}}
{{- end}}
{{- if .Genres}}
genres:
{{- range .Genres}}
  - {{.}}
{{- end}}
{{- end}}
{{- if .Runtime}}
runtime: {{.Runtime}}
{{- end}}
{{- if .Rating}}
rating: {{.Rating}}
{{- end}}
{{- if .PosterURL}}
poster: {{.PosterURL}}
{{- end}}
---
{{if .Description}}
{{.Description}}
{{end}}
`

// writeMovieToMarkdown writes a single movie to a markdown file
func writeMovieToMarkdown(movie Movie, directory string) error {
	// Create a sanitized title
	sanitizedTitle := sanitizeFilename(movie.Name)

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

	// Parse the template
	tmpl, err := template.New("movie").Parse(movieTemplate)
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

// sanitizeFilename replaces invalid filename characters with underscores
func sanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name

	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	return result
}
