package fileutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCoverFilename(t *testing.T) {
	testCases := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "simple title",
			title:    "Test Movie",
			expected: "Test Movie - cover.jpg",
		},
		{
			name:     "title with colon",
			title:    "Movie: Subtitle",
			expected: "Movie - Subtitle - cover.jpg",
		},
		{
			name:     "title with slash",
			title:    "Movie/Part",
			expected: "Movie-Part - cover.jpg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildCoverFilename(tc.title)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDownloadCover_EmptyURL(t *testing.T) {
	result, err := DownloadCover(CoverDownloadOptions{
		URL:       "",
		OutputDir: "/tmp",
		Filename:  "test.jpg",
	})
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestDownloadCover_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake image data"))
	}))
	defer server.Close()

	// Create temp directory
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	result, err := DownloadCover(CoverDownloadOptions{
		URL:          server.URL,
		OutputDir:    tempDir,
		Filename:     "test-cover.jpg",
		UpdateCovers: false,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Downloaded)
	assert.Equal(t, "test-cover.jpg", result.Filename)
	assert.Equal(t, filepath.Join("attachments", "test-cover.jpg"), result.RelativePath)
	assert.Equal(t, filepath.Join(tempDir, "attachments", "test-cover.jpg"), result.LocalPath)

	// Verify file was created
	assert.True(t, FileExists(result.LocalPath))
}

func TestDownloadCover_SkipsExisting(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("new image data"))
	}))
	defer server.Close()

	// Create temp directory with existing file
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	attachmentsDir := filepath.Join(tempDir, "attachments")
	err := os.MkdirAll(attachmentsDir, 0755)
	require.NoError(t, err)

	existingFile := filepath.Join(attachmentsDir, "existing-cover.jpg")
	err = os.WriteFile(existingFile, []byte("old image data"), 0644)
	require.NoError(t, err)

	result, err := DownloadCover(CoverDownloadOptions{
		URL:          server.URL,
		OutputDir:    tempDir,
		Filename:     "existing-cover.jpg",
		UpdateCovers: false,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Downloaded, "Should not download when file exists and UpdateCovers is false")

	// Verify original content is preserved
	content, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, "old image data", string(content))
}

func TestDownloadCover_OverwritesExisting(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("new image data"))
	}))
	defer server.Close()

	// Create temp directory with existing file
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	attachmentsDir := filepath.Join(tempDir, "attachments")
	err := os.MkdirAll(attachmentsDir, 0755)
	require.NoError(t, err)

	existingFile := filepath.Join(attachmentsDir, "existing-cover.jpg")
	err = os.WriteFile(existingFile, []byte("old image data"), 0644)
	require.NoError(t, err)

	result, err := DownloadCover(CoverDownloadOptions{
		URL:          server.URL,
		OutputDir:    tempDir,
		Filename:     "existing-cover.jpg",
		UpdateCovers: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Downloaded, "Should download when UpdateCovers is true")

	// Verify new content
	content, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, "new image data", string(content))
}

func TestDownloadCover_HTTPError(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create temp directory
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	result, err := DownloadCover(CoverDownloadOptions{
		URL:       server.URL,
		OutputDir: tempDir,
		Filename:  "test-cover.jpg",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status 404")
}

func TestAddCoverToMarkdown_TMDBCoverPreferred(t *testing.T) {
	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")

	AddCoverToMarkdown(mb, AddCoverOptions{
		TMDBCoverPath:     "attachments/tmdb-cover.jpg",
		TMDBCoverFilename: "tmdb-cover.jpg",
		FallbackURL:       "https://example.com/fallback.jpg",
		Title:             "Test Movie",
		Width:             300,
	})

	result := mb.Build()

	// TMDB cover should be used
	assert.Contains(t, result, "cover: \"attachments/tmdb-cover.jpg\"")
	assert.Contains(t, result, "![[tmdb-cover.jpg|300]]")
	// Fallback URL should not appear
	assert.NotContains(t, result, "example.com/fallback.jpg")
}

func TestAddCoverToMarkdown_FallbackDownloadSuccess(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake image data"))
	}))
	defer server.Close()

	// Create temp directory
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")

	AddCoverToMarkdown(mb, AddCoverOptions{
		FallbackURL: server.URL,
		Title:       "Test Movie",
		Directory:   tempDir,
		Width:       300,
	})

	result := mb.Build()

	// Should use local path from download
	assert.Contains(t, result, "cover: \"attachments/Test Movie - cover.jpg\"")
	assert.Contains(t, result, "![[Test Movie - cover.jpg|300]]")

	// Verify file was downloaded
	coverPath := filepath.Join(tempDir, "attachments", "Test Movie - cover.jpg")
	assert.True(t, FileExists(coverPath))
}

func TestAddCoverToMarkdown_FallbackDownloadFailure(t *testing.T) {
	// Create test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create temp directory
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")

	AddCoverToMarkdown(mb, AddCoverOptions{
		FallbackURL: server.URL,
		Title:       "Test Movie",
		Directory:   tempDir,
		Width:       300,
	})

	result := mb.Build()

	// Should fall back to URL when download fails
	assert.Contains(t, result, "cover: \""+server.URL+"\"")
	assert.Contains(t, result, "![]("+server.URL+")")
}

func TestAddCoverToMarkdown_NoCover(t *testing.T) {
	mb := NewMarkdownBuilder()
	mb.AddTitle("Test Movie")

	AddCoverToMarkdown(mb, AddCoverOptions{
		Title: "Test Movie",
	})

	result := mb.Build()

	// Should not have any cover field
	assert.NotContains(t, result, "cover:")
	assert.NotContains(t, result, "![[")
	assert.NotContains(t, result, "![](")
}

func TestAddCoverToMarkdown_UpdateCoversFlag(t *testing.T) {
	// Create test server
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("new image data"))
	}))
	defer server.Close()

	// Create temp directory with existing cover
	env := testutil.NewTestEnv(t)
	tempDir := env.RootDir()

	attachmentsDir := filepath.Join(tempDir, "attachments")
	err := os.MkdirAll(attachmentsDir, 0755)
	require.NoError(t, err)

	existingFile := filepath.Join(attachmentsDir, "Test Movie - cover.jpg")
	err = os.WriteFile(existingFile, []byte("old image data"), 0644)
	require.NoError(t, err)

	// First call without UpdateCovers
	mb1 := NewMarkdownBuilder()
	AddCoverToMarkdown(mb1, AddCoverOptions{
		FallbackURL:  server.URL,
		Title:        "Test Movie",
		Directory:    tempDir,
		UpdateCovers: false,
	})

	assert.Equal(t, 0, requestCount, "Should not make request when file exists and UpdateCovers is false")

	// Second call with UpdateCovers
	mb2 := NewMarkdownBuilder()
	AddCoverToMarkdown(mb2, AddCoverOptions{
		FallbackURL:  server.URL,
		Title:        "Test Movie",
		Directory:    tempDir,
		UpdateCovers: true,
	})

	assert.Equal(t, 1, requestCount, "Should make request when UpdateCovers is true")
}

func TestAddCoverToMarkdown_WidthVariations(t *testing.T) {
	testCases := []struct {
		name           string
		width          int
		expectedFormat string
	}{
		{
			name:           "with width",
			width:          300,
			expectedFormat: "![[tmdb-cover.jpg|300]]",
		},
		{
			name:           "without width",
			width:          0,
			expectedFormat: "![[tmdb-cover.jpg]]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mb := NewMarkdownBuilder()
			mb.AddTitle("Test Movie")

			AddCoverToMarkdown(mb, AddCoverOptions{
				TMDBCoverPath:     "attachments/tmdb-cover.jpg",
				TMDBCoverFilename: "tmdb-cover.jpg",
				Width:             tc.width,
			})

			result := mb.Build()
			assert.Contains(t, result, tc.expectedFormat)
		})
	}
}
