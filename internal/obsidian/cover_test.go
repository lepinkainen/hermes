package obsidian

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useCoverServer points cover downloads at an httptest server that answers
// every request with the given status, and returns the server's URL.
func useCoverServer(t *testing.T, status int) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte("fake-cover-bytes"))
	}))
	t.Cleanup(server.Close)
	t.Cleanup(fileutil.SetCoverHTTPClient(server.Client()))
	return server.URL
}

func TestResolveCover_PrefersPreDownloadedCover(t *testing.T) {
	// A pre-downloaded cover must win without touching the network.
	url := useCoverServer(t, http.StatusInternalServerError)
	dir := t.TempDir()

	got := ResolveCover(CoverSource{
		TMDBCoverPath:     "attachments/Title - cover.jpg",
		TMDBCoverFilename: "Title - cover.jpg",
		FallbackURL:       url + "/poster.jpg",
	}, "Title", dir)

	require.NotNil(t, got)
	assert.Equal(t, "Title - cover.jpg", got.Filename)
	assert.Equal(t, "attachments/Title - cover.jpg", got.FrontmatterValue)
	assert.Empty(t, got.RemoteURL)
	assert.Equal(t, "![[Title - cover.jpg|250]]\n\n", got.Embed(250))
	assert.NoFileExists(t, filepath.Join(dir, "attachments", "Title - cover.jpg"))
}

func TestResolveCover_DownloadsFallbackURL(t *testing.T) {
	url := useCoverServer(t, http.StatusOK)
	dir := t.TempDir()

	got := ResolveCover(CoverSource{FallbackURL: url + "/poster.jpg"}, "My: Title", dir)

	require.NotNil(t, got)
	assert.Equal(t, "My - Title - cover.jpg", got.Filename)
	assert.Equal(t, filepath.Join("attachments", "My - Title - cover.jpg"), got.FrontmatterValue)
	assert.Empty(t, got.RemoteURL)
	assert.Equal(t, "![[My - Title - cover.jpg|500]]\n\n", got.Embed(500))

	data, err := os.ReadFile(filepath.Join(dir, "attachments", "My - Title - cover.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "fake-cover-bytes", string(data))
}

func TestResolveCover_NoSources(t *testing.T) {
	assert.Nil(t, ResolveCover(CoverSource{}, "Title", t.TempDir()))
}

func TestResolveCover_DownloadFailure(t *testing.T) {
	url := useCoverServer(t, http.StatusNotFound)

	t.Run("without URL fallback returns nil", func(t *testing.T) {
		got := ResolveCover(CoverSource{FallbackURL: url + "/missing.jpg"}, "Title", t.TempDir())
		assert.Nil(t, got)
	})

	t.Run("with URL fallback references the remote image", func(t *testing.T) {
		src := CoverSource{FallbackURL: url + "/missing.jpg", FallbackToURLOnError: true}
		got := ResolveCover(src, "Title", t.TempDir())

		require.NotNil(t, got)
		assert.Empty(t, got.Filename)
		assert.Equal(t, src.FallbackURL, got.FrontmatterValue)
		assert.Equal(t, src.FallbackURL, got.RemoteURL)
		assert.Equal(t, "![]("+src.FallbackURL+")\n\n", got.Embed(250))
	})
}
