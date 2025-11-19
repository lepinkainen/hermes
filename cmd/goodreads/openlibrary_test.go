package goodreads

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetHTTPClientSingleton(t *testing.T) {
	t.Cleanup(func() {
		httpClient = nil
		clientOnce = sync.Once{}
	})

	clientOnce = sync.Once{}
	httpClient = nil
	origFactory := httpClientNew
	defer func() { httpClientNew = origFactory }()

	var builds int
	httpClientNew = func() *http.Client {
		builds++
		return &http.Client{}
	}

	first := getHTTPClient()
	second := getHTTPClient()
	require.Equal(t, first, second)
	require.Equal(t, 1, builds)
}

func TestFetchBookAndEditionData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/books", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ISBN:123":{"title":"Test Book","subtitle":"Sub","publishers":[]}}`))
	})
	mux.HandleFunc("/isbn/123.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"number_of_pages":321,"publishers":["Edition Pub"]}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Cleanup(func() {
		httpClient = nil
		clientOnce = sync.Once{}
		httpClientNew = func() *http.Client { return &http.Client{Timeout: 10 * time.Second} }
		openLibraryBaseURL = "https://openlibrary.org"
	})

	clientOnce = sync.Once{}
	httpClient = nil
	httpClientNew = func() *http.Client { return server.Client() }
	openLibraryBaseURL = server.URL

	book, olBook, err := fetchBookData("123")
	require.NoError(t, err)
	require.Equal(t, "Test Book", book.Title)
	require.Equal(t, "Sub", book.Subtitle)
	require.Equal(t, 321, book.NumberOfPages)
	require.Equal(t, "Edition Pub", book.Publisher)
	require.Equal(t, "Test Book", olBook.Title)
}

func TestFetchEditionDataError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/isbn/000.json", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Cleanup(func() {
		httpClient = nil
		clientOnce = sync.Once{}
		httpClientNew = func() *http.Client { return &http.Client{Timeout: 10 * time.Second} }
		openLibraryBaseURL = "https://openlibrary.org"
	})

	clientOnce = sync.Once{}
	httpClient = nil
	httpClientNew = func() *http.Client { return server.Client() }
	openLibraryBaseURL = server.URL

	edition, err := fetchEditionData("000")
	require.Error(t, err)
	require.Nil(t, edition)
}

func TestFetchCoverImage(t *testing.T) {
	url, err := fetchCoverImage(123)
	require.NoError(t, err)
	require.Equal(t, "https://covers.openlibrary.org/b/id/123-L.jpg", url)

	_, err = fetchCoverImage(0)
	require.Error(t, err)
}
