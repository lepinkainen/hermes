package enrichers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/lepinkainen/hermes/internal/enrichment/book"
	"github.com/lepinkainen/hermes/internal/parseutil"
	"github.com/lepinkainen/hermes/internal/ratelimit"
)

const (
	finnaBaseURL  = "https://api.finna.fi"
	finnaPriority = 4
)

// finnaSearchFields enumerates the record fields requested from the Finna
// search API. Limiting the projection keeps responses small.
var finnaSearchFields = []string{
	"title", "authors", "publishers", "year",
	"languages", "subjects", "id", "cleanIsbn",
}

// FinnaEnricher implements the book.Enricher interface for the Finna API
// (https://api.finna.fi), the Finnish national library discovery service.
// It's most useful for Finnish-language titles where OpenLibrary / Google Books
// have weak coverage of Finnish publishers and subject headings.
type FinnaEnricher struct {
	getHTTPClient  func() *http.Client
	getRateLimiter func() *ratelimit.Limiter
}

var _ book.Enricher = (*FinnaEnricher)(nil)

// NewFinnaEnricher creates a new Finna enricher.
func NewFinnaEnricher() *FinnaEnricher {
	return &FinnaEnricher{
		getHTTPClient: sync.OnceValue(func() *http.Client {
			return &http.Client{Timeout: 15 * time.Second}
		}),
		getRateLimiter: sync.OnceValue(func() *ratelimit.Limiter {
			return ratelimit.New("Finna", 1)
		}),
	}
}

// Name returns the human-readable name of this enricher.
func (e *FinnaEnricher) Name() string { return "Finna" }

// Priority returns the priority for merging data (lower = higher precedence).
func (e *FinnaEnricher) Priority() int { return finnaPriority }

// Ping tests the connection to Finna.
func (e *FinnaEnricher) Ping(ctx context.Context) error {
	client := e.getHTTPClient()
	pingURL := finnaBaseURL + "/api/v1/search?lookfor=test&type=AllFields&limit=0"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("finna ping failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("finna returned status %d", resp.StatusCode)
	}
	return nil
}

// Enrich fetches book data from Finna by ISBN.
func (e *FinnaEnricher) Enrich(ctx context.Context, isbn string) (*book.EnrichmentData, error) {
	if isbn == "" {
		return nil, book.ErrInvalidISBN
	}

	normalizedISBN := parseutil.NormalizeISBN(isbn)
	cached, _, err := cache.GetOrFetchWithTTL("finna_cache", normalizedISBN, func() (*cachedFinnaResult, error) {
		return e.fetchFromAPI(ctx, normalizedISBN)
	}, cache.SelectNegativeCacheTTL(func(r *cachedFinnaResult) bool {
		return r.NotFound
	}))
	if err != nil {
		return nil, err
	}
	if cached.NotFound {
		return nil, nil
	}
	return cached.Data, nil
}

// cachedFinnaResult wraps EnrichmentData with metadata for caching.
type cachedFinnaResult struct {
	Data     *book.EnrichmentData `json:"data"`
	NotFound bool                 `json:"not_found"`
}

type finnaSearchResponse struct {
	ResultCount int           `json:"resultCount"`
	Records     []finnaRecord `json:"records"`
	Status      string        `json:"status"`
}

type finnaRecord struct {
	Title      string       `json:"title"`
	Authors    finnaAuthors `json:"authors"`
	Publishers []string     `json:"publishers"`
	Year       string       `json:"year"`
	Languages  []string     `json:"languages"`
	Subjects   [][]string   `json:"subjects"`
	ID         string       `json:"id"`
	CleanISBN  string       `json:"cleanIsbn"`
}

// finnaAuthors holds primary author names from a Finna record. Secondary and
// corporate entries are skipped: their keys embed contributor roles
// ("Lastname, Firstname, kirjoittaja") and publisher names that pollute the
// author list. The custom decoder is needed because Finna serializes empty
// author sub-maps as JSON arrays (`"primary": []`) rather than objects.
type finnaAuthors struct {
	Primary []string
}

func (a *finnaAuthors) UnmarshalJSON(data []byte) error {
	var raw struct {
		Primary json.RawMessage `json:"primary"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	primary := bytes.TrimSpace(raw.Primary)
	switch {
	case len(primary) == 0, bytes.Equal(primary, []byte("null")):
		return nil
	case primary[0] == '[':
		return nil // Finna quirk: empty author sub-maps come back as `[]`.
	case primary[0] != '{':
		return fmt.Errorf("decoding finna authors.primary: unexpected JSON value %s", primary)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(primary, &m); err != nil {
		return fmt.Errorf("decoding finna authors.primary: %w", err)
	}
	a.Primary = make([]string, 0, len(m))
	for name := range m {
		if name = strings.TrimSpace(name); name != "" {
			a.Primary = append(a.Primary, name)
		}
	}
	return nil
}

func (e *FinnaEnricher) fetchFromAPI(ctx context.Context, isbn string) (*cachedFinnaResult, error) {
	limiter := e.getRateLimiter()
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	client := e.getHTTPClient()

	q := url.Values{}
	q.Set("lookfor", isbn)
	q.Set("type", "ISN")
	q.Set("limit", "1")
	for _, f := range finnaSearchFields {
		q.Add("field[]", f)
	}
	searchURL := finnaBaseURL + "/api/v1/search?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("finna API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finna API returned status %d", resp.StatusCode)
	}

	var searchResp finnaSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decoding finna response: %w", err)
	}

	if searchResp.ResultCount == 0 || len(searchResp.Records) == 0 {
		return &cachedFinnaResult{NotFound: true}, nil
	}

	return &cachedFinnaResult{Data: extractFinnaEnrichmentData(&searchResp.Records[0])}, nil
}

// extractFinnaEnrichmentData maps a Finna record onto book.EnrichmentData.
// Description and CoverURL are deliberately omitted: Finna search responses
// almost always return empty summaries and images, and covers must be fetched
// via a separate /Cover/Show endpoint that's better left to higher-priority
// enrichers.
func extractFinnaEnrichmentData(rec *finnaRecord) *book.EnrichmentData {
	data := &book.EnrichmentData{}

	if title := strings.TrimSpace(rec.Title); title != "" {
		data.Title = &title
	}
	if len(rec.Publishers) > 0 {
		if pub := strings.TrimSpace(rec.Publishers[0]); pub != "" {
			data.Publisher = &pub
		}
	}
	if year := strings.TrimSpace(rec.Year); year != "" {
		data.PublishDate = &year
	}
	if len(rec.Languages) > 0 {
		if lang := strings.TrimSpace(rec.Languages[0]); lang != "" {
			data.Language = &lang
		}
	}
	if subjects := flattenFinnaSubjects(rec.Subjects); len(subjects) > 0 {
		data.Subjects = subjects
	}
	if len(rec.Authors.Primary) > 0 {
		data.Authors = rec.Authors.Primary
	}
	return data
}

// flattenFinnaSubjects flattens Finna's nested subject arrays and trims
// trailing periods that come from MARC heading punctuation.
func flattenFinnaSubjects(subjects [][]string) []string {
	if len(subjects) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(subjects))
	for _, group := range subjects {
		for _, s := range group {
			s = strings.TrimSpace(s)
			s = strings.TrimRight(s, ".")
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
