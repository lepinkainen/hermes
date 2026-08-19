package enhance

import (
	"testing"

	"github.com/lepinkainen/hermes/internal/enrichment"
	"github.com/lepinkainen/hermes/internal/obsidian"
	"github.com/stretchr/testify/assert"
)

// TestAddGameData_ClearsStaleSteamData covers the poisoned-note repair path:
// a note that was previously (wrongly) matched to a Steam AppID is re-enriched
// via the RAWG fallback. The stale steam_appid must be removed, and the old
// genre/* tags replaced rather than accumulated.
func TestAddGameData_ClearsStaleSteamData(t *testing.T) {
	fm := obsidian.NewFrontmatter()
	fm.Set("title", "Uncharted 2")
	fm.Set("steam_appid", 1155880) // junk match from a prior run
	fm.Set("tags", []string{"game", "genre/Indie", "genre/Simulation", "to-watch"})

	note := &Note{Type: "game", SteamAppID: 1155880, Frontmatter: fm}

	note.AddGameData(&enrichment.GameEnrichment{
		RAWGID:          22513,
		GenreTags:       []string{"genre/Action", "genre/Shooter"},
		Developers:      []string{"Naughty Dog"},
		MetacriticScore: 96,
	})

	// steam_appid fully removed (not just zeroed)
	_, hasSteam := fm.Get("steam_appid")
	assert.False(t, hasSteam, "stale steam_appid should be deleted")
	assert.Equal(t, 0, note.SteamAppID)

	assert.Equal(t, 22513, note.RAWGID)
	assert.Equal(t, 22513, fm.GetInt("rawg_id"))

	// Genre tags replaced, non-genre tags preserved, no stale genres left.
	assert.ElementsMatch(t,
		[]string{"game", "to-watch", "genre/Action", "genre/Shooter"},
		fm.GetStringArray("tags"),
	)
}

func TestStripGenreTags(t *testing.T) {
	got := stripGenreTags([]string{"game", "genre/Action", "to-watch", "genre/RPG"})
	assert.Equal(t, []string{"game", "to-watch"}, got)

	// No genre tags: unchanged content.
	assert.Equal(t, []string{"game", "to-watch"}, stripGenreTags([]string{"game", "to-watch"}))

	// Empty input.
	assert.Empty(t, stripGenreTags(nil))
}
