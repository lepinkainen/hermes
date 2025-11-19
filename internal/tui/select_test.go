package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lepinkainen/hermes/internal/tmdb"
	"github.com/stretchr/testify/require"
)

func TestTmdbItemFormatting(t *testing.T) {
	item := tmdbItem{SearchResult: tmdb.SearchResult{
		Title:       "Heat",
		Overview:    "A Los Angeles crime saga.",
		ReleaseDate: "1995-12-15",
	}}

	require.Equal(t, "HEAT (1995)", item.Title())
	require.Equal(t, "Heat", item.FilterValue())
	require.Equal(t, "A Los Angeles crime saga.", item.Description())
}

func TestModelUpdateHandlesKeyMessages(t *testing.T) {
	item := tmdbItem{SearchResult: tmdb.SearchResult{ID: 1, Title: "Heat", ReleaseDate: "1995-12-15"}}

	t.Run("enter selects item", func(t *testing.T) {
		m := newModel("Heat", []tmdbItem{item})
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		typed := updated.(*model)

		require.Equal(t, ActionSelected, typed.result.Action)
		require.NotNil(t, typed.result.Selection)
		require.Equal(t, 1, typed.result.Selection.ID)
	})

	t.Run("skip and stop actions", func(t *testing.T) {
		m := newModel("Heat", []tmdbItem{item})
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		require.Equal(t, ActionSkipped, updated.(*model).result.Action)

		m = newModel("Heat", []tmdbItem{item})
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		require.Equal(t, ActionStopped, updated.(*model).result.Action)

		m = newModel("Heat", []tmdbItem{item})
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		require.Equal(t, ActionSkipped, updated.(*model).result.Action)
	})
}

func TestFormattingHelpers(t *testing.T) {
	meta := formatMetadata(tmdb.SearchResult{
		Runtime:      170,
		OriginalLang: "en",
		VoteCount:    12345,
		Popularity:   8.9,
	}, 100)
	require.Contains(t, meta, "170m")
	require.Contains(t, meta, "EN")
	require.Contains(t, meta, "12.3K votes")
	require.Contains(t, meta, "📊8.9")

	short := formatMetadata(tmdb.SearchResult{
		VoteCount: 9999,
	}, 5)
	require.Equal(t, truncate("10.0K votes", 5), short)

	require.Equal(t, "abc", truncate("abcdef", 3))
	require.Equal(t, "abc", truncate("abc", 10))

	require.Equal(t, "5 votes", formatVoteCount(5))
	require.Equal(t, "1.2K votes", formatVoteCount(1200))

	require.Equal(t, 50, clamp(50, 0, 40))
	require.Equal(t, 10, clamp(20, 10, 5))
	require.Equal(t, 8, clamp(10, 5, 8))
}

func TestSelectFiltersLowVoteResults(t *testing.T) {
	originalRunner := runProgram
	defer func() { runProgram = originalRunner }()

	runProgram = func(m tea.Model) (tea.Model, error) {
		typed := m.(*model)
		// Should only contain the high-vote result after filtering
		require.Equal(t, 1, len(typed.list.Items()))
		item := typed.list.Items()[0].(tmdbItem)
		typed.result = SelectionResult{
			Action:    ActionSelected,
			Selection: &item.SearchResult,
		}
		return typed, nil
	}

	results := []tmdb.SearchResult{
		{ID: 1, Title: "Low", VoteCount: 50},
		{ID: 2, Title: "High", VoteCount: 200},
	}

	res, err := Select("Title", results)
	require.NoError(t, err)
	require.Equal(t, ActionSelected, res.Action)
	require.NotNil(t, res.Selection)
	require.Equal(t, 2, res.Selection.ID)
}
