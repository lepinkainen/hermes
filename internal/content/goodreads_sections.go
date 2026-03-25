package content

import (
	"fmt"
	"strings"
)

// GoodreadsBookDetails contains the information needed to generate Goodreads content sections
type GoodreadsBookDetails struct {
	Title                   string
	Subtitle                string
	Authors                 []string
	Publisher               string
	Pages                   int
	YearPublished           int
	OriginalPublicationYear int
	MyRating                float64
	AverageRating           float64
	ISBN                    string
	ISBN13                  string
	Binding                 string
	GoodreadsID             string
	Description             string
	Subjects                []string
	SubjectPeople           []string
}

// BuildGoodreadsContent generates Goodreads content sections based on the provided book details
// sections can include: "info", "description", "subjects"
func BuildGoodreadsContent(details *GoodreadsBookDetails, sections []string) string {
	if details == nil {
		return ""
	}

	sectionMap := make(map[string]bool)
	for _, s := range sections {
		sectionMap[s] = true
	}

	var builder strings.Builder
	first := true

	// Build sections in order
	if sectionMap["info"] {
		if !first {
			builder.WriteString("\n")
		}
		builder.WriteString(buildGoodreadsInfoSection(details))
		first = false
	}

	if sectionMap["description"] && details.Description != "" {
		if !first {
			builder.WriteString("\n")
		}
		builder.WriteString(buildGoodreadsDescriptionSection(details))
		first = false
	}

	if sectionMap["subjects"] && len(details.Subjects) > 0 {
		if !first {
			builder.WriteString("\n")
		}
		builder.WriteString(buildGoodreadsSubjectsSection(details))
	}

	return builder.String()
}

// buildGoodreadsInfoSection creates the book info table
func buildGoodreadsInfoSection(details *GoodreadsBookDetails) string {
	var builder strings.Builder
	builder.WriteString("## Book Info\n\n")
	builder.WriteString("| | |\n")
	builder.WriteString("|---|---|\n")

	// Title (with subtitle)
	titleLine := details.Title
	if details.Subtitle != "" {
		titleLine = fmt.Sprintf("%s: %s", details.Title, details.Subtitle)
	}
	if details.YearPublished > 0 {
		titleLine = fmt.Sprintf("%s (%d)", titleLine, details.YearPublished)
	}
	fmt.Fprintf(&builder, "| **Title** | %s |\n", titleLine)

	// Authors
	if len(details.Authors) > 0 {
		fmt.Fprintf(&builder, "| **Author** | %s |\n", strings.Join(details.Authors, ", "))
	}

	// Publisher
	if details.Publisher != "" {
		fmt.Fprintf(&builder, "| **Publisher** | %s |\n", details.Publisher)
	}

	// Pages
	if details.Pages > 0 {
		fmt.Fprintf(&builder, "| **Pages** | %d |\n", details.Pages)
	}

	// Binding
	if details.Binding != "" {
		fmt.Fprintf(&builder, "| **Binding** | %s |\n", details.Binding)
	}

	// Original publication year (if different from published year)
	if details.OriginalPublicationYear > 0 && details.OriginalPublicationYear != details.YearPublished {
		fmt.Fprintf(&builder, "| **Original Publication** | %d |\n", details.OriginalPublicationYear)
	}

	// Ratings
	if details.MyRating > 0 {
		stars := buildStarRating5(details.MyRating)
		fmt.Fprintf(&builder, "| **My Rating** | %s (%.1f/5) |\n", stars, details.MyRating)
	}
	if details.AverageRating > 0 {
		fmt.Fprintf(&builder, "| **Average Rating** | %.2f/5 |\n", details.AverageRating)
	}

	// ISBNs
	if details.ISBN != "" {
		fmt.Fprintf(&builder, "| **ISBN** | %s |\n", details.ISBN)
	}
	if details.ISBN13 != "" {
		fmt.Fprintf(&builder, "| **ISBN-13** | %s |\n", details.ISBN13)
	}

	// Goodreads link
	if details.GoodreadsID != "" {
		fmt.Fprintf(&builder, "| **Goodreads** | [goodreads.com/book/show/%s](https://www.goodreads.com/book/show/%s) |\n",
			details.GoodreadsID, details.GoodreadsID)
	}

	return builder.String()
}

// buildGoodreadsDescriptionSection creates the description section
func buildGoodreadsDescriptionSection(details *GoodreadsBookDetails) string {
	var builder strings.Builder
	builder.WriteString("## Description\n\n")
	builder.WriteString(details.Description)
	builder.WriteString("\n")
	return builder.String()
}

// buildGoodreadsSubjectsSection creates the subjects section
func buildGoodreadsSubjectsSection(details *GoodreadsBookDetails) string {
	var builder strings.Builder
	builder.WriteString("## Subjects\n\n")

	if len(details.Subjects) > 0 {
		builder.WriteString("**Topics**: ")
		builder.WriteString(strings.Join(details.Subjects, ", "))
		builder.WriteString("\n")
	}

	if len(details.SubjectPeople) > 0 {
		if len(details.Subjects) > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("**People**: ")
		builder.WriteString(strings.Join(details.SubjectPeople, ", "))
	}

	return builder.String()
}

// buildStarRating5 converts a 1-5 rating to star emojis
func buildStarRating5(rating float64) string {
	fullStars := int(rating)
	hasHalf := (rating - float64(fullStars)) >= 0.5

	var builder strings.Builder
	for range fullStars {
		builder.WriteString("⭐")
	}
	if hasHalf {
		builder.WriteString("½")
	}

	return builder.String()
}
