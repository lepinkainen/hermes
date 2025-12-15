package goodreads

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/cache"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestGetGoogleBooksHTTPClientSingleton(t *testing.T) {
	t.Cleanup(func() {
		googleBooksHTTPClient = nil
		googleBooksClientOnce = sync.Once{}
	})

	googleBooksClientOnce = sync.Once{}
	googleBooksHTTPClient = nil
	origFactory := googleBooksHTTPClientNew
	defer func() { googleBooksHTTPClientNew = origFactory }()

	var builds int
	googleBooksHTTPClientNew = func() *http.Client {
		builds++
		return &http.Client{}
	}

	first := getGoogleBooksHTTPClient()
	second := getGoogleBooksHTTPClient()
	require.Equal(t, first, second)
	require.Equal(t, 1, builds)
}

func TestFetchBookDataFromGoogleBooksSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		require.Equal(t, "isbn:9780316769488", query)

		response := `{
			"totalItems": 1,
			"items": [{
				"volumeInfo": {
					"title": "The Catcher in the Rye",
					"subtitle": "A Novel",
					"authors": ["J.D. Salinger"],
					"publisher": "Little, Brown Books for Young Readers",
					"publishedDate": "1991-05-01",
					"description": "The hero-narrator of The Catcher in the Rye...",
					"pageCount": 277,
					"categories": ["Fiction", "Classics"],
					"averageRating": 3.8,
					"ratingsCount": 6789,
					"imageLinks": {
						"thumbnail": "http://books.google.com/books/content?id=PCDengEACAAJ&printsec=frontcover&img=1&zoom=1&source=gbs_api",
						"smallThumbnail": "http://books.google.com/books/content?id=PCDengEACAAJ&printsec=frontcover&img=1&zoom=5&source=gbs_api"
					},
					"language": "en",
					"infoLink": "https://books.google.com/books?id=PCDengEACAAJ&dq=isbn:9780316769488"
				}
			}]
		}`
		_, _ = w.Write([]byte(response))
	})

	server := newIPv4TestServer(t, mux)

	t.Cleanup(func() {
		googleBooksHTTPClient = nil
		googleBooksClientOnce = sync.Once{}
		googleBooksHTTPClientNew = func() *http.Client { return &http.Client{Timeout: 10 * time.Second} }
		googleBooksBaseURL = "https://www.googleapis.com/books/v1"
	})

	googleBooksClientOnce = sync.Once{}
	googleBooksHTTPClient = nil
	googleBooksHTTPClientNew = func() *http.Client { return server.Client() }
	googleBooksBaseURL = server.URL

	book, err := fetchBookDataFromGoogleBooks("9780316769488")
	require.NoError(t, err)
	require.Equal(t, "The Catcher in the Rye", book.VolumeInfo.Title)
	require.Equal(t, "A Novel", book.VolumeInfo.Subtitle)
	require.Equal(t, []string{"J.D. Salinger"}, book.VolumeInfo.Authors)
	require.Equal(t, "Little, Brown Books for Young Readers", book.VolumeInfo.Publisher)
	require.Equal(t, 277, book.VolumeInfo.PageCount)
	require.Equal(t, []string{"Fiction", "Classics"}, book.VolumeInfo.Categories)
	require.Contains(t, book.VolumeInfo.ImageLinks.Thumbnail, "books.google.com")
}

func TestFetchBookDataFromGoogleBooksEmptyResults(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		response := `{"totalItems": 0, "items": []}`
		_, _ = w.Write([]byte(response))
	})

	server := newIPv4TestServer(t, mux)

	t.Cleanup(func() {
		googleBooksHTTPClient = nil
		googleBooksClientOnce = sync.Once{}
		googleBooksHTTPClientNew = func() *http.Client { return &http.Client{Timeout: 10 * time.Second} }
		googleBooksBaseURL = "https://www.googleapis.com/books/v1"
	})

	googleBooksClientOnce = sync.Once{}
	googleBooksHTTPClient = nil
	googleBooksHTTPClientNew = func() *http.Client { return server.Client() }
	googleBooksBaseURL = server.URL

	book, err := fetchBookDataFromGoogleBooks("0000000000")
	require.Error(t, err)
	require.Nil(t, book)
	require.Contains(t, err.Error(), "no data found in Google Books")
}

func TestFetchBookDataFromGoogleBooksHTTPError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})

	server := newIPv4TestServer(t, mux)

	t.Cleanup(func() {
		googleBooksHTTPClient = nil
		googleBooksClientOnce = sync.Once{}
		googleBooksHTTPClientNew = func() *http.Client { return &http.Client{Timeout: 10 * time.Second} }
		googleBooksBaseURL = "https://www.googleapis.com/books/v1"
	})

	googleBooksClientOnce = sync.Once{}
	googleBooksHTTPClient = nil
	googleBooksHTTPClientNew = func() *http.Client { return server.Client() }
	googleBooksBaseURL = server.URL

	book, err := fetchBookDataFromGoogleBooks("1234567890")
	require.Error(t, err)
	require.Nil(t, book)
	require.Contains(t, err.Error(), "non-200 status code")
}

func TestFetchBookDataFromGoogleBooksMalformedJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{invalid json`))
	})

	server := newIPv4TestServer(t, mux)

	t.Cleanup(func() {
		googleBooksHTTPClient = nil
		googleBooksClientOnce = sync.Once{}
		googleBooksHTTPClientNew = func() *http.Client { return &http.Client{Timeout: 10 * time.Second} }
		googleBooksBaseURL = "https://www.googleapis.com/books/v1"
	})

	googleBooksClientOnce = sync.Once{}
	googleBooksHTTPClient = nil
	googleBooksHTTPClientNew = func() *http.Client { return server.Client() }
	googleBooksBaseURL = server.URL

	book, err := fetchBookDataFromGoogleBooks("1234567890")
	require.Error(t, err)
	require.Nil(t, book)
	require.Contains(t, err.Error(), "failed to decode")
}

func TestNormalizeISBN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ISBN with hyphens",
			input:    "978-0-316-76948-8",
			expected: "9780316769488",
		},
		{
			name:     "ISBN with spaces",
			input:    "978 0 316 76948 8",
			expected: "9780316769488",
		},
		{
			name:     "ISBN with hyphens and spaces",
			input:    "978-0-316 76948-8",
			expected: "9780316769488",
		},
		{
			name:     "ISBN already clean",
			input:    "9780316769488",
			expected: "9780316769488",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeISBN(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestEnrichBookFromGoogleBooksDataMerge(t *testing.T) {
	// Test that Google Books doesn't overwrite OpenLibrary data
	book := &Book{
		Title:       "Test Book",
		ISBN13:      "9780316769488",
		Description: "OpenLibrary description",
		Publisher:   "OpenLibrary Publisher",
	}

	// Mock Google Books to return different data
	mux := http.NewServeMux()
	mux.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("HTTP handler called for data merge test: %s", r.URL.String())
		response := `{
			"totalItems": 1,
			"items": [{
				"volumeInfo": {
					"title": "Test Book",
					"description": "Google Books description (should not overwrite)",
					"publisher": "Google Books Publisher (should not overwrite)",
					"subtitle": "Subtitle from Google Books",
					"pageCount": 300
				}
			}]
		}`
		_, _ = w.Write([]byte(response))
	})

	server := newIPv4TestServer(t, mux)

	// Use a temporary cache database for this test
	tmpDB := t.TempDir() + "/test_cache.db"
	viper.Set("cache.dbfile", tmpDB)

	t.Cleanup(func() {
		googleBooksHTTPClient = nil
		googleBooksClientOnce = sync.Once{}
		googleBooksHTTPClientNew = func() *http.Client { return &http.Client{Timeout: 10 * time.Second} }
		googleBooksBaseURL = "https://www.googleapis.com/books/v1"
		_ = cache.ResetGlobalCache()
		viper.Set("cache.dbfile", "./cache.db")
	})

	googleBooksClientOnce = sync.Once{}
	googleBooksHTTPClient = nil
	googleBooksHTTPClientNew = func() *http.Client { return server.Client() }
	googleBooksBaseURL = server.URL

	// Reset cache before test to ensure fresh start
	_ = cache.ResetGlobalCache()

	t.Logf("About to call enrichBookFromGoogleBooks with ISBN13: %s", book.ISBN13)
	err := enrichBookFromGoogleBooks(book)
	t.Logf("enrichBookFromGoogleBooks returned error: %v", err)
	t.Logf("Book subtitle after enrichment: %s", book.Subtitle)
	require.NoError(t, err)

	// Verify OpenLibrary data is preserved
	require.Equal(t, "OpenLibrary description", book.Description, "Description should not be overwritten")
	require.Equal(t, "OpenLibrary Publisher", book.Publisher, "Publisher should not be overwritten")

	// Verify Google Books filled empty fields
	require.Equal(t, "Subtitle from Google Books", book.Subtitle, "Subtitle should be filled from Google Books")
	require.Equal(t, 300, book.NumberOfPages, "Page count should be filled from Google Books")
}

func TestEnrichBookFromGoogleBooksNoISBN(t *testing.T) {
	book := &Book{
		Title: "Test Book",
	}

	err := enrichBookFromGoogleBooks(book)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no ISBN available")
}
