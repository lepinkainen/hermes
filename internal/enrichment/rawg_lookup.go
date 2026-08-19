package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	hermeserrors "github.com/lepinkainen/hermes/internal/errors"
	"github.com/lepinkainen/hermes/internal/tui"
	"github.com/spf13/viper"
)

// HTTP seams — package vars so tests can redirect to an httptest.Server.
var (
	rawgSearchURL  = "https://api.rawg.io/api/games"
	rawgHTTPClient = &http.Client{Timeout: 30 * time.Second}
)

// RAWGGenre represents a genre entry in RAWG API responses.
type RAWGGenre struct {
	Name string `json:"name"`
}

// RAWGCompany represents a developer/publisher entry in RAWG API responses.
type RAWGCompany struct {
	Name string `json:"name"`
}

// RAWGGameResult represents a game entry from RAWG's search or details
// endpoints (only the fields hermes uses). Search rows carry genres,
// released, metacritic and background_image directly; developers,
// publishers and description_raw are only populated by the details
// endpoint (/api/games/{id}).
type RAWGGameResult struct {
	ID              int           `json:"id"`
	Slug            string        `json:"slug"`
	Name            string        `json:"name"`
	Released        string        `json:"released"`
	BackgroundImage string        `json:"background_image"`
	Metacritic      *int          `json:"metacritic"`
	DescriptionRaw  string        `json:"description_raw"`
	Genres          []RAWGGenre   `json:"genres"`
	Developers      []RAWGCompany `json:"developers"`
	Publishers      []RAWGCompany `json:"publishers"`
}

// RAWGSearchResponse is the API response from RAWG's game search endpoint.
type RAWGSearchResponse struct {
	Results []RAWGGameResult `json:"results"`
}

// CachedRAWGSearchResults wraps search results for caching.
type CachedRAWGSearchResults struct {
	Results []RAWGGameResult `json:"results"`
}

// getRAWGAPIKey resolves the RAWG API key from config (rawg.api_key) or the
// RAWG_API_KEY environment variable. An empty return means RAWG is not
// configured; callers no-op silently, matching the ISBNdb enricher pattern.
func getRAWGAPIKey() string {
	if key := viper.GetString("rawg.api_key"); key != "" {
		return key
	}
	return os.Getenv("RAWG_API_KEY")
}

// searchRAWGGame resolves the best RAWG match for title, or nil if RAWG has
// no results.
func searchRAWGGame(ctx context.Context, apiKey, title string, opts SteamEnrichmentOptions) (*RAWGGameResult, error) {
	// Obsidian filenames replace ":" with " - "; collapse it the same way
	// searchSteamAppID does so search terms match cleanly.
	query := steamSearchTitle(title)

	results, err := searchRAWGStore(ctx, apiKey, query)
	if err != nil {
		return nil, fmt.Errorf("RAWG search failed: %w", err)
	}

	if len(results) == 0 {
		slog.Debug("No RAWG results found", "title", title)
		return nil, nil
	}

	return selectRAWGResult(results, title, opts.Interactive)
}

// searchRAWGStore searches RAWG for games matching the query with caching.
func searchRAWGStore(ctx context.Context, apiKey, query string) ([]RAWGGameResult, error) {
	cacheKey := normalizeRAWGQuery(query)

	// Use cached search with policy to avoid caching empty results
	cached, _, err := cache.GetOrFetchWithPolicy(
		"rawg_search_cache",
		cacheKey,
		func() (*CachedRAWGSearchResults, error) {
			results, fetchErr := fetchRAWGSearch(ctx, apiKey, query)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return &CachedRAWGSearchResults{Results: results}, nil
		},
		func(result *CachedRAWGSearchResults) bool {
			// Only cache if we got results
			return result != nil && len(result.Results) > 0
		},
	)

	if err != nil {
		return nil, err
	}

	return cached.Results, nil
}

// fetchRAWGSearch performs the actual RAWG game search API call.
func fetchRAWGSearch(ctx context.Context, apiKey, query string) ([]RAWGGameResult, error) {
	params := url.Values{}
	params.Set("key", apiKey)
	params.Set("search", query)
	params.Set("page_size", "5")
	searchURL := rawgSearchURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rawgHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RAWG search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RAWG search returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp RAWGSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse RAWG search response: %w", err)
	}

	return searchResp.Results, nil
}

// fetchRAWGDetails fetches full game details (description, developers,
// publishers) for a RAWG game id, with caching keyed by id.
func fetchRAWGDetails(ctx context.Context, apiKey string, id int) (*RAWGGameResult, error) {
	cacheKey := strconv.Itoa(id)

	details, _, err := cache.GetOrFetch("rawg_cache", cacheKey, func() (*RAWGGameResult, error) {
		return fetchRAWGGameDetails(ctx, apiKey, id)
	})
	if err != nil {
		return nil, err
	}

	return details, nil
}

// fetchRAWGGameDetails performs the actual RAWG game details API call.
func fetchRAWGGameDetails(ctx context.Context, apiKey string, id int) (*RAWGGameResult, error) {
	params := url.Values{}
	params.Set("key", apiKey)
	detailsURL := fmt.Sprintf("%s/%d?%s", rawgSearchURL, id, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailsURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := rawgHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RAWG game details: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RAWG details returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result RAWGGameResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse RAWG details response: %w", err)
	}

	return &result, nil
}

// normalizeRAWGQuery normalizes a query string for use as a cache key.
func normalizeRAWGQuery(query string) string {
	// Convert to lowercase and replace spaces with underscores
	normalized := strings.ToLower(strings.TrimSpace(query))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	// Remove special characters that might cause issues
	normalized = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, normalized)
	return normalized
}

// selectRAWGResult picks the best RAWG match from a page of search results.
//
// An exact (case-insensitive) name match wins when unambiguous. Otherwise,
// naively picking the first result surfaces junk fan/asset entries ahead of
// the real game — verified against "Uncharted 2" (the real Among Thieves,
// Metacritic 96, ranks 2nd) and "Shadow of the Colossus" (the 2018 remake
// ranks 1st ahead of variants). So the non-exact fallback prefers the
// highest non-null Metacritic score among the top page_size results,
// tiebreaking by original relevance order.
func selectRAWGResult(results []RAWGGameResult, title string, interactive bool) (*RAWGGameResult, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// If only one result, auto-select it
	if len(results) == 1 {
		slog.Debug("Auto-selected single RAWG result", "title", title, "rawg_id", results[0].ID)
		return &results[0], nil
	}

	if interactive {
		return selectRAWGInteractive(title, results)
	}

	// Restrict to results whose name is title-compatible with the query
	// before ranking by Metacritic. RAWG's fuzzy search returns other
	// entries from the same franchise, which can otherwise out-score the
	// intended title — e.g. searching "Uncharted Drake's Fortune" surfaces
	// "Uncharted 3: Drake's Deception" (Metacritic 92) ahead of the actual
	// "Uncharted: Drake's Fortune" (88); a pure highest-Metacritic pick would
	// wrongly select the sequel. Falls back to the full result set if
	// nothing is title-compatible (unexpected, but keeps behavior sane).
	candidates := filterTitleCompatibleRAWGResults(results, title)
	if len(candidates) == 0 {
		candidates = results
	}

	// Prefer a reviewed (non-null Metacritic) result over an unreviewed one,
	// even when the unreviewed one is a literal exact title match — RAWG's
	// crowdsourced DB often has bare, unreviewed duplicate listings
	// alongside the real game's full subtitled entry (e.g. a plain
	// "Uncharted 2" listing with no score, next to the real
	// "Uncharted 2: Among Thieves" at 96). Naively picking the first result,
	// or blindly trusting an exact match, both surface that junk entry
	// instead of the real game — verified against "Uncharted 2" and "Shadow
	// of the Colossus" (where the remake similarly outranks the original by
	// naive ordering). Ties broken by original relevance order.
	if hasAnyMetacritic(candidates) {
		best := bestRAWGByMetacritic(candidates, title)
		slog.Debug("Auto-selected RAWG result by metacritic", "title", title, "rawg_id", best.ID, "metacritic", best.Metacritic)
		return best, nil
	}

	// Nobody has review data to rank by: fall back to an exact
	// (punctuation/case insensitive) name match, else the first
	// (most-relevant, per RAWG's own search ordering) candidate.
	if exact := findExactRAWGMatch(candidates, title); exact != nil {
		slog.Debug("Auto-selected exact RAWG match", "title", title, "rawg_id", exact.ID)
		return exact, nil
	}

	slog.Debug("Auto-selected first title-compatible RAWG result", "title", title, "rawg_id", candidates[0].ID)
	return &candidates[0], nil
}

// hasAnyMetacritic reports whether any result carries a non-null Metacritic
// score.
func hasAnyMetacritic(results []RAWGGameResult) bool {
	for i := range results {
		if results[i].Metacritic != nil {
			return true
		}
	}
	return false
}

// romanNumerals maps lowercase Roman numerals (as used in game titles like
// "Diablo III" or "Final Fantasy VII") to their Arabic digit equivalent, so
// they normalize to the same token as an Obsidian note titled "Diablo 3".
// Covers I-XV, comfortably past the highest sequel number seen in practice.
var romanNumerals = map[string]string{
	"i": "1", "ii": "2", "iii": "3", "iv": "4", "v": "5",
	"vi": "6", "vii": "7", "viii": "8", "ix": "9", "x": "10",
	"xi": "11", "xii": "12", "xiii": "13", "xiv": "14", "xv": "15",
}

// normalizeGameTitleTokens lowercases a title, treats ':'/'-'/','/'.' as word
// separators, drops apostrophes entirely (so "Drake's" and "Drakes" match,
// and "Uncharted: X" / "Uncharted - X" / "Uncharted X" all normalize the
// same way), maps Roman numeral tokens to Arabic digits, and splits on
// whitespace.
func normalizeGameTitleTokens(s string) []string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"'", "",
		"’", "", // right single quotation mark (curly apostrophe)
		"™", "", // trademark symbol (common in Steam titles)
		"®", "", // registered trademark symbol
		"©", "", // copyright symbol
		":", " ",
		"-", " ",
		",", " ",
		".", " ",
	)
	fields := strings.Fields(replacer.Replace(s))
	tokens := make([]string, len(fields))
	for i, f := range fields {
		if arabic, ok := romanNumerals[f]; ok {
			tokens[i] = arabic
		} else {
			tokens[i] = f
		}
	}
	return tokens
}

// isTitlePrefixRelevant reports whether one token sequence is a prefix of
// the other. Comparing whole tokens (rather than raw substrings) avoids
// false positives like "uncharted 2" matching "uncharted 20xx".
func isTitlePrefixRelevant(a, b []string) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return false
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// leadingArticles are dropped from the front of a title before compatibility
// comparison, so a note titled "Witcher 3" stays compatible with a store
// listing named "The Witcher 3: Wild Hunt" (and vice versa). Only a *leading*
// article is stripped — articles elsewhere are meaningful.
var leadingArticles = map[string]bool{"the": true, "a": true, "an": true}

func dropLeadingArticle(tokens []string) []string {
	if len(tokens) > 1 && leadingArticles[tokens[0]] {
		return tokens[1:]
	}
	return tokens
}

// isGameTitleCompatible reports whether name is plausibly the same game as
// title, by comparing their normalized token sequences with
// isTitlePrefixRelevant. Shared by both the RAWG and Steam auto-selection
// paths so "is this search result actually the game we're looking for?"
// means the same thing everywhere. A leading article ("The"/"A"/"An") is
// ignored on either side. An empty token sequence on either side (e.g. a
// blank name) can never be considered compatible.
func isGameTitleCompatible(name, title string) bool {
	nameTokens := dropLeadingArticle(normalizeGameTitleTokens(name))
	titleTokens := dropLeadingArticle(normalizeGameTitleTokens(title))
	if len(nameTokens) == 0 || len(titleTokens) == 0 {
		return false
	}
	return isTitlePrefixRelevant(nameTokens, titleTokens)
}

// filterTitleCompatibleRAWGResults returns the subset of results whose name
// shares a leading-token prefix with title, in either direction (handles
// both "RAWG name has extra subtitle words" and "query has extra words").
func filterTitleCompatibleRAWGResults(results []RAWGGameResult, title string) []RAWGGameResult {
	var relevant []RAWGGameResult
	for _, r := range results {
		if isGameTitleCompatible(r.Name, title) {
			relevant = append(relevant, r)
		}
	}
	return relevant
}

// findExactRAWGMatch looks for an unambiguous, punctuation-insensitive exact
// name match (e.g. Obsidian's "Uncharted Drake's Fortune" against RAWG's
// "Uncharted: Drake's Fortune").
func findExactRAWGMatch(results []RAWGGameResult, title string) *RAWGGameResult {
	normalizedTitle := strings.Join(normalizeGameTitleTokens(title), " ")

	var match *RAWGGameResult
	matchCount := 0

	for i := range results {
		result := &results[i]
		if strings.Join(normalizeGameTitleTokens(result.Name), " ") == normalizedTitle {
			match = result
			matchCount++
			if matchCount > 1 {
				return nil // Ambiguous
			}
		}
	}

	return match
}

// bestRAWGByMetacritic returns the result with the highest non-null
// Metacritic score. On a tie, an exact (punctuation/case insensitive) title
// match among the tied results wins — e.g. "Shadow of the Colossus" search
// results include the original, a 2011 remaster, and a 2018 remake all
// scored 91; the plain, un-suffixed title should win over the dated
// variants. Otherwise ties keep the earliest (most relevant) result.
func bestRAWGByMetacritic(results []RAWGGameResult, title string) *RAWGGameResult {
	best := &results[0]
	for i := 1; i < len(results); i++ {
		candidate := &results[i]
		if candidate.Metacritic == nil {
			continue
		}
		if best.Metacritic == nil || *candidate.Metacritic > *best.Metacritic {
			best = candidate
		}
	}

	if best.Metacritic == nil {
		return best
	}

	var tied []RAWGGameResult
	for i := range results {
		if results[i].Metacritic != nil && *results[i].Metacritic == *best.Metacritic {
			tied = append(tied, results[i])
		}
	}
	if len(tied) <= 1 {
		return best
	}
	if exact := findExactRAWGMatch(tied, title); exact != nil {
		return exact
	}
	return best
}

// selectRAWGInteractive presents a TUI for RAWG game selection, reusing the
// Steam selection TUI (it only displays AppID/Name/HeaderImage, all of which
// map cleanly onto a RAWG result).
func selectRAWGInteractive(title string, results []RAWGGameResult) (*RAWGGameResult, error) {
	tuiResults := make([]tui.SteamSearchResult, len(results))
	for i, r := range results {
		tuiResults[i] = tui.SteamSearchResult{
			AppID:       r.ID,
			Name:        r.Name,
			HeaderImage: r.BackgroundImage,
		}
	}

	selection, err := tui.SelectSteam(title, tuiResults, nil)
	if err != nil {
		return nil, fmt.Errorf("TUI selection failed: %w", err)
	}

	switch selection.Action {
	case tui.ActionSelected:
		if selection.SteamSelection == nil {
			return nil, nil
		}
		for i := range results {
			if results[i].ID == selection.SteamSelection.AppID {
				return &results[i], nil
			}
		}
		return nil, nil
	case tui.ActionStopped:
		return nil, hermeserrors.NewStopProcessingError("RAWG selection stopped by user")
	default:
		slog.Debug("User skipped RAWG selection")
		return nil, nil
	}
}
