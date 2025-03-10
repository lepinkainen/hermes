package letterboxd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

const movieTemplate = `---
title: {{.Name}}
year: {{.Year}}
date_watched: {{.Date}}
letterboxd_uri: {{.LetterboxdURI}}
letterboxd_id: {{.LetterboxdID}}
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
	// Create filename from movie title and year
	sanitizedTitle := sanitizeFilename(movie.Name)
	filename := fmt.Sprintf("%s (%d).md", sanitizedTitle, movie.Year)
	filepath := filepath.Join(directory, filename)

	// Check if file exists and we should not overwrite
	if !overwrite {
		if _, err := os.Stat(filepath); err == nil {
			log.Infof("File %s already exists, skipping", filepath)
			return nil
		}
	}

	// Parse template
	tmpl, err := template.New("movie").Parse(movieTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, movie); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	log.Infof("Wrote %s", filepath)
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
