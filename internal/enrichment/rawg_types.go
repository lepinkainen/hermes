package enrichment

// GameEnrichment holds game enrichment data sourced from RAWG (rawg.io).
// RAWG is used as the fallback source in cmd/enhance when EnrichFromSteam
// finds nothing — Steam is PC-only, so console/handheld exclusives (e.g.
// Uncharted 2, Shadow of the Colossus) never resolve there. RAWG's
// platform-agnostic database covers those titles.
//
// The shape mirrors SteamEnrichment so cmd/enhance can treat both sources
// uniformly; SteamEnrichmentOptions is reused for options since its fields
// (DownloadCover, GenerateContent, AttachmentsDir, NoteDir, Interactive,
// Force) are source-agnostic.
type GameEnrichment struct {
	// RAWGID is the RAWG numeric game identifier
	RAWGID int
	// CoverPath is the relative path to the downloaded cover image
	CoverPath string
	// CoverFilename is just the filename of the cover
	CoverFilename string
	// GenreTags are the RAWG genre tags (genre/<slug> convention)
	GenreTags []string
	// Developers is the list of game developers
	Developers []string
	// Publishers is the list of game publishers
	Publishers []string
	// ReleaseDate is the game release date string
	ReleaseDate string
	// MetacriticScore is the Metacritic score (0-100)
	MetacriticScore int
	// Description is the RAWG plain-text game description
	Description string
	// ContentMarkdown is the generated RAWG content section
	ContentMarkdown string
}
