package content

import (
	"fmt"
)

// BuildCoverImageEmbed generates Obsidian embed syntax for a cover image.
// Returns: ![[filename|250]]
func BuildCoverImageEmbed(coverFilename string) string {
	if coverFilename == "" {
		return ""
	}
	return fmt.Sprintf("![[%s|250]]", coverFilename)
}
