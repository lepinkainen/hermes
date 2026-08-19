package enrichment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

// withRAWGSearchServer redirects the RAWG HTTP seams to an httptest.Server
// for the duration of the test, mirroring withSteamSearchServer.
func withRAWGSearchServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	prevURL := rawgSearchURL
	prevClient := rawgHTTPClient
	rawgSearchURL = server.URL
	rawgHTTPClient = server.Client()
	t.Cleanup(func() {
		rawgSearchURL = prevURL
		rawgHTTPClient = prevClient
	})
}

func TestFetchRAWGSearchSuccess(t *testing.T) {
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Uncharted 2", r.URL.Query().Get("search"))
		require.Equal(t, "testkey", r.URL.Query().Get("key"))
		require.Equal(t, "5", r.URL.Query().Get("page_size"))

		_ = json.NewEncoder(w).Encode(RAWGSearchResponse{
			Results: []RAWGGameResult{
				{ID: 1, Name: "Uncharted 2: Among Thieves", Metacritic: new(96)},
			},
		})
	})

	results, err := fetchRAWGSearch(t.Context(), "testkey", "Uncharted 2")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, 1, results[0].ID)
	require.Equal(t, "Uncharted 2: Among Thieves", results[0].Name)
	require.NotNil(t, results[0].Metacritic)
	require.Equal(t, 96, *results[0].Metacritic)
}

func TestFetchRAWGSearchEmptyResults(t *testing.T) {
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(RAWGSearchResponse{Results: []RAWGGameResult{}})
	})

	results, err := fetchRAWGSearch(t.Context(), "testkey", "Some Obscure Title")
	require.NoError(t, err)
	require.Empty(t, results)
}

// TestSearchRAWGGame_EmptyResultsReturnsNilNil exercises the pure
// fetch+select path (bypassing the cache layer, which persists state across
// test runs and made an earlier version of this test flaky): an empty
// search response should select to nil without error.
func TestSearchRAWGGame_EmptyResultsReturnsNilNil(t *testing.T) {
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(RAWGSearchResponse{Results: []RAWGGameResult{}})
	})

	results, err := fetchRAWGSearch(t.Context(), "testkey", "Some Obscure Title")
	require.NoError(t, err)
	require.Empty(t, results)

	match, err := selectRAWGResult(results, "Some Obscure Title", false)
	require.NoError(t, err)
	require.Nil(t, match)
}

func TestFetchRAWGSearchHTTPError(t *testing.T) {
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	})

	_, err := fetchRAWGSearch(t.Context(), "testkey", "foo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 500")
}

func TestFetchRAWGSearchInvalidJSON(t *testing.T) {
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})

	_, err := fetchRAWGSearch(t.Context(), "testkey", "foo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse RAWG search response")
}

func TestFetchRAWGGameDetailsSuccess(t *testing.T) {
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/123", r.URL.Path)
		require.Equal(t, "testkey", r.URL.Query().Get("key"))

		_ = json.NewEncoder(w).Encode(RAWGGameResult{
			ID:             123,
			Name:           "Shadow of the Colossus",
			DescriptionRaw: "A boy and his horse.",
			Developers:     []RAWGCompany{{Name: "SIE Japan Studio"}},
			Publishers:     []RAWGCompany{{Name: "Sony Interactive Entertainment"}},
		})
	})

	details, err := fetchRAWGGameDetails(t.Context(), "testkey", 123)
	require.NoError(t, err)
	require.Equal(t, 123, details.ID)
	require.Equal(t, "A boy and his horse.", details.DescriptionRaw)
	require.Equal(t, []RAWGCompany{{Name: "SIE Japan Studio"}}, details.Developers)
}

func TestFetchRAWGGameDetailsHTTPError(t *testing.T) {
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, err := fetchRAWGGameDetails(t.Context(), "testkey", 999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 404")
}

func TestGetRAWGAPIKey_EnvFallback(t *testing.T) {
	testutil.ResetConfig(t)
	t.Setenv("RAWG_API_KEY", "env-key")
	require.Equal(t, "env-key", getRAWGAPIKey())
}

func TestGetRAWGAPIKey_Empty(t *testing.T) {
	testutil.ResetConfig(t)
	t.Setenv("RAWG_API_KEY", "")
	require.Equal(t, "", getRAWGAPIKey())
}

func TestEnrichFromRAWG_NoAPIKeyReturnsNilNil(t *testing.T) {
	testutil.ResetConfig(t)
	t.Setenv("RAWG_API_KEY", "")

	game, err := EnrichFromRAWG(t.Context(), "Some Game", SteamEnrichmentOptions{})
	require.NoError(t, err)
	require.Nil(t, game)
}

// --- Selection logic ---

func TestSelectRAWGResult_SingleResult(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Test Game"},
	}

	selected, err := selectRAWGResult(results, "Test Game", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 1, selected.ID)
}

func TestSelectRAWGResult_EmptyResults(t *testing.T) {
	selected, err := selectRAWGResult(nil, "Test Game", false)
	require.NoError(t, err)
	require.Nil(t, selected)
}

func TestSelectRAWGResult_ExactMatchNonInteractive(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Half-Life"},
		{ID: 2, Name: "Half-Life 2"},
		{ID: 3, Name: "Half-Life: Source"},
	}

	selected, err := selectRAWGResult(results, "Half-Life 2", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 2, selected.ID, "should select exact match")
}

// TestSelectRAWGResult_MetacriticRankedSelection verifies the junk-first
// case that made naive "pick the first result" unsafe: a low-quality fan
// entry ranks first, but the real game (higher Metacritic) should win.
func TestSelectRAWGResult_MetacriticRankedSelection(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Uncharted 2 Fan Compilation", Metacritic: nil},
		{ID: 2, Name: "Uncharted 2: Among Thieves", Metacritic: new(96)},
		{ID: 3, Name: "Uncharted 2 Multiplayer Mod", Metacritic: new(40)},
	}

	selected, err := selectRAWGResult(results, "Uncharted 2", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 2, selected.ID, "should pick the highest-Metacritic result, not the first")
}

func TestSelectRAWGResult_MetacriticTiebreakKeepsFirst(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Game Remaster", Metacritic: new(80)},
		{ID: 2, Name: "Game Remake", Metacritic: new(80)},
	}

	selected, err := selectRAWGResult(results, "Game", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 1, selected.ID, "should keep the earlier result on a metacritic tie")
}

func TestSelectRAWGResult_NoMetacriticFallsBackToFirst(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Some Game"},
		{ID: 2, Name: "Some Other Game"},
	}

	selected, err := selectRAWGResult(results, "Totally Different Title", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 1, selected.ID, "should fall back to the first result when nothing has a Metacritic score")
}

// TestSelectRAWGResult_FranchiseContaminationGuard is a regression test for
// a real bug found during e2e verification: searching "Uncharted Drake's
// Fortune" against RAWG surfaces "Uncharted 3: Drake's Deception"
// (Metacritic 92) in the same top-5 page as the actual "Uncharted: Drake's
// Fortune" (88). A pure highest-Metacritic pick wrongly chose the sequel. It
// must be excluded by the title-compatibility filter before Metacritic
// ranking is applied. Fixture data is the real RAWG response captured
// during that run.
func TestSelectRAWGResult_FranchiseContaminationGuard(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 4340, Name: "Uncharted: Drake's Fortune", Metacritic: new(88)},
		{ID: 5703, Name: "Uncharted 3: Drake's Deception", Metacritic: new(92)},
		{ID: 102541, Name: "Uncharted: Drake's Equation"},
		{ID: 909144, Name: "Drake's Fortune"},
		{ID: 47953, Name: "UNCHARTED: Fortune Hunter", Metacritic: new(77)},
	}

	selected, err := selectRAWGResult(results, "Uncharted Drake's Fortune", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 4340, selected.ID, "should pick the actual Drake's Fortune, not the higher-Metacritic Uncharted 3")
}

// TestSelectRAWGResult_FranchiseContaminationGuard_LostLegacy is the same
// regression as above for "Uncharted The Lost Legacy", which also surfaced
// "Uncharted 3: Drake's Deception" in its top-5.
func TestSelectRAWGResult_FranchiseContaminationGuard_LostLegacy(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 21926, Name: "Uncharted: The Lost Legacy", Metacritic: new(85)},
		{ID: 704634, Name: "Uncharted: Legacy of Thieves Collection"},
		{ID: 4475, Name: "Uncharted: Golden Abyss", Metacritic: new(80)},
		{ID: 19256, Name: "Abyss Raiders: Uncharted"},
		{ID: 5703, Name: "Uncharted 3: Drake's Deception", Metacritic: new(92)},
	}

	selected, err := selectRAWGResult(results, "Uncharted The Lost Legacy", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 21926, selected.ID, "should pick the actual Lost Legacy, not the higher-Metacritic Uncharted 3")
}

// TestSelectRAWGResult_Uncharted2PicksReviewedOverBogusExactMatch is a
// regression test for the inverse case: an unreviewed, literally-exact-match
// entry ("Uncharted 2") ranks alongside the real, well-reviewed
// "Uncharted 2: Among Thieves". Metacritic ranking among title-compatible
// candidates should still surface the real game. Fixture data is the real
// RAWG response captured during e2e verification.
func TestSelectRAWGResult_Uncharted2PicksReviewedOverBogusExactMatch(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 670945, Name: "Uncharted 2"},
		{ID: 22513, Name: "Uncharted 2: Among Thieves", Metacritic: new(96)},
		{ID: 4475, Name: "Uncharted: Golden Abyss", Metacritic: new(80)},
		{ID: 19256, Name: "Abyss Raiders: Uncharted"},
		{ID: 5703, Name: "Uncharted 3: Drake's Deception", Metacritic: new(92)},
	}

	selected, err := selectRAWGResult(results, "Uncharted 2", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 22513, selected.ID, "should pick the reviewed Among Thieves over the bogus exact-name-match entry")
}

// TestSelectRAWGResult_TiedMetacriticPrefersExactMatch is a regression test
// for a second real bug found during e2e verification (after the first
// franchise-contamination fix): "Shadow of the Colossus" search results
// include the original, a 2011 remaster, and a 2018 remake — all scored 91
// on Metacritic. A pure "keep first on tie" rule would have picked whichever
// dated variant RAWG happened to list first, not the plain title the note
// actually refers to. Fixture data is the real RAWG response.
func TestSelectRAWGResult_TiedMetacriticPrefersExactMatch(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 4491, Name: "Shadow of the Colossus (2011)", Metacritic: new(91)},
		{ID: 59248, Name: "Shadow of the Colossus", Metacritic: new(91)},
		{ID: 52368, Name: "Shadow of the Colossus  (2018)", Metacritic: new(91)},
		{ID: 5683, Name: "The ICO & Shadow of the Colossus Collection"},
		{ID: 30180, Name: "Shadow of Destiny", Metacritic: new(74)},
	}

	selected, err := selectRAWGResult(results, "Shadow of the Colossus", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 59248, selected.ID, "should pick the plain title over dated variants tied on Metacritic")
}

// TestSelectRAWGResult_RomanNumeralMatchesArabicDigit is a regression test
// for a third real bug: RAWG lists "Diablo III" (Roman numeral) while the
// Obsidian note is titled "Diablo 3" (Arabic digit). Without numeral
// normalization, the title-compatibility filter would have wrongly excluded
// the real game and let an unrelated, unreviewed "diablo 3-13" entry
// through as the only "compatible" candidate. Fixture data is the real
// RAWG response.
func TestSelectRAWGResult_RomanNumeralMatchesArabicDigit(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 23600, Name: "Diablo III", Metacritic: new(88)},
		{ID: 44093, Name: "Diablo III: Eternal Collection", Metacritic: new(88)},
		{ID: 28418, Name: "Diablo III: Reaper of Souls", Metacritic: new(87)},
		{ID: 42457, Name: "Diablo III: Ultimate Evil Edition", Metacritic: new(88)},
		{ID: 239642, Name: "diablo 3-13"},
	}

	selected, err := selectRAWGResult(results, "Diablo 3", false)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 23600, selected.ID, "should pick the base game, matching Roman numeral III to Arabic 3")
}

func TestIsTitlePrefixRelevant_TokenBoundary(t *testing.T) {
	// "uncharted 2" must not be considered relevant to "uncharted 20xx" via
	// naive substring matching; token-level comparison should catch this.
	require.False(t, isTitlePrefixRelevant(
		normalizeGameTitleTokens("Uncharted 20XX"),
		normalizeGameTitleTokens("Uncharted 2"),
	))
	require.True(t, isTitlePrefixRelevant(
		normalizeGameTitleTokens("Uncharted 2: Among Thieves"),
		normalizeGameTitleTokens("Uncharted 2"),
	))
}

func TestNormalizeGameTitleTokens_PunctuationVariants(t *testing.T) {
	expected := []string{"uncharted", "drakes", "fortune"}
	require.Equal(t, expected, normalizeGameTitleTokens("Uncharted: Drake's Fortune"))
	require.Equal(t, expected, normalizeGameTitleTokens("Uncharted Drake's Fortune"))
	require.Equal(t, expected, normalizeGameTitleTokens("Uncharted - Drake's Fortune"))
}

// TestNormalizeGameTitleTokens_TrademarkSymbols is a regression test: Steam
// store titles commonly carry trademark/registered/copyright symbols (e.g.
// "STAR WARS Jedi: Fallen Order™") that neither RAWG nor Obsidian note
// titles ever have. Without stripping them, a legitimate Steam match would
// be wrongly rejected by the title-compatibility gate.
func TestNormalizeGameTitleTokens_TrademarkSymbols(t *testing.T) {
	expected := []string{"star", "wars", "jedi", "fallen", "order"}
	require.Equal(t, expected, normalizeGameTitleTokens("STAR WARS Jedi: Fallen Order™"))
	require.Equal(t, []string{"diablo", "3"}, normalizeGameTitleTokens("Diablo III®"))
	require.Equal(t, []string{"portal", "2"}, normalizeGameTitleTokens("Portal 2©"))
}

func TestIsGameTitleCompatible(t *testing.T) {
	require.True(t, isGameTitleCompatible("STAR WARS Jedi: Fallen Order™", "Star Wars Jedi Fallen Order"))
	require.True(t, isGameTitleCompatible("Portal 2", "Portal 2"))
	require.True(t, isGameTitleCompatible("Half-Life", "Half-Life 2"))
	require.True(t, isGameTitleCompatible("The Witcher 3", "The Witcher 3: Wild Hunt"))
	// Leading-article difference must not break compatibility: a note titled
	// "Witcher 3" against Steam's "The Witcher 3: Wild Hunt".
	require.True(t, isGameTitleCompatible("The Witcher 3: Wild Hunt", "Witcher 3"))
	require.True(t, isGameTitleCompatible("The Last of Us", "Last of Us"))
	require.True(t, isGameTitleCompatible("Grand Theft Auto V", "Grand Theft Auto 5"))
	require.False(t, isGameTitleCompatible("Some Indie Citybuilder", "Uncharted 2"))
	require.False(t, isGameTitleCompatible("", "Uncharted 2"))
	require.False(t, isGameTitleCompatible("Uncharted 2", ""))
}

func TestFindExactRAWGMatch_Found(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Half-Life"},
		{ID: 2, Name: "Half-Life 2"},
	}

	match := findExactRAWGMatch(results, "half-life 2")
	require.NotNil(t, match)
	require.Equal(t, 2, match.ID)
}

func TestFindExactRAWGMatch_NotFound(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Half-Life"},
	}

	match := findExactRAWGMatch(results, "Portal")
	require.Nil(t, match)
}

func TestFindExactRAWGMatch_Ambiguous(t *testing.T) {
	results := []RAWGGameResult{
		{ID: 1, Name: "Game"},
		{ID: 2, Name: "Game"},
	}

	match := findExactRAWGMatch(results, "Game")
	require.Nil(t, match, "should return nil for ambiguous matches")
}

// --- Query normalization ---

// TestSearchRAWGGame_SpacedDashNormalization verifies the search term sent
// to RAWG has Obsidian's " - " (substituted for ":") collapsed to a plain
// space, same as Steam. This exercises fetchRAWGSearch directly (bypassing
// the cache layer used by searchRAWGGame/searchRAWGStore) to keep the test
// hermetic; steamSearchTitle's collapsing behavior itself is covered by
// TestSteamSearchTitle.
func TestSearchRAWGGame_SpacedDashNormalization(t *testing.T) {
	var seenSearch string
	withRAWGSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		seenSearch = r.URL.Query().Get("search")
		_ = json.NewEncoder(w).Encode(RAWGSearchResponse{
			Results: []RAWGGameResult{{ID: 1, Name: "Star Wars Jedi Fallen Order"}},
		})
	})

	// Obsidian filenames replace ":" with " - "; RAWG search should receive
	// the collapsed form, same as Steam.
	query := steamSearchTitle("Star Wars Jedi - Fallen Order")
	_, err := fetchRAWGSearch(t.Context(), "testkey", query)
	require.NoError(t, err)
	require.Equal(t, "Star Wars Jedi Fallen Order", seenSearch)
}

func TestNormalizeRAWGQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase and spaces to underscores",
			input:    "Half-Life 2",
			expected: "half-life_2",
		},
		{
			name:     "remove special characters",
			input:    "Grand Theft Auto: V",
			expected: "grand_theft_auto__v",
		},
		{
			name:     "trim whitespace",
			input:    "  Portal  ",
			expected: "portal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRAWGQuery(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
