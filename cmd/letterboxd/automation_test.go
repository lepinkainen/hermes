package letterboxd

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestPrepareDownloadDirCreatesTemp(t *testing.T) {
	dir, cleanup, err := prepareDownloadDir("")
	require.NoError(t, err)
	require.DirExists(t, dir)
	require.NotNil(t, cleanup)

	cleanup()
	_, statErr := os.Stat(dir)
	require.True(t, os.IsNotExist(statErr), "temp dir should be removed after cleanup")
}

func TestPrepareDownloadDirCreatesCustom(t *testing.T) {
	env := testutil.NewTestEnv(t)
	customDir := env.Path("custom-downloads")

	dir, cleanup, err := prepareDownloadDir(customDir)
	require.NoError(t, err)
	require.DirExists(t, dir)
	require.Nil(t, cleanup) // No cleanup for custom dirs
	require.Equal(t, customDir, dir)
}

func TestExtractLetterboxdCSVs(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Create test ZIP with both CSVs
	zipPath := env.Path("test.zip")
	zf, err := os.Create(zipPath)
	require.NoError(t, err)

	zw := zip.NewWriter(zf)

	w1, err := zw.Create("letterboxd-user-2025-01-01/watched.csv")
	require.NoError(t, err)
	_, err = w1.Write([]byte("Date,Name,Year,URI\n2025-01-01,Test Movie,2025,https://boxd.it/abc\n"))
	require.NoError(t, err)

	w2, err := zw.Create("letterboxd-user-2025-01-01/ratings.csv")
	require.NoError(t, err)
	_, err = w2.Write([]byte("Date,Name,Year,URI,Rating\n2025-01-01,Test Movie,2025,https://boxd.it/abc,5\n"))
	require.NoError(t, err)

	require.NoError(t, zw.Close())
	require.NoError(t, zf.Close())

	// Extract
	watchedPath, ratingsPath, err := extractLetterboxdCSVs(zipPath, env.RootDir())
	require.NoError(t, err)
	require.FileExists(t, watchedPath)
	require.FileExists(t, ratingsPath)

	// Verify content
	watchedContent, err := os.ReadFile(watchedPath)
	require.NoError(t, err)
	require.Contains(t, string(watchedContent), "Test Movie")

	ratingsContent, err := os.ReadFile(ratingsPath)
	require.NoError(t, err)
	require.Contains(t, string(ratingsContent), "Rating")
}

func TestExtractLetterboxdCSVsWithoutRatings(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Create test ZIP with only watched.csv
	zipPath := env.Path("test.zip")
	zf, err := os.Create(zipPath)
	require.NoError(t, err)

	zw := zip.NewWriter(zf)
	w1, err := zw.Create("letterboxd-user-2025-01-01/watched.csv")
	require.NoError(t, err)
	_, err = w1.Write([]byte("Date,Name,Year,URI\n2025-01-01,Test Movie,2025,https://boxd.it/abc\n"))
	require.NoError(t, err)

	require.NoError(t, zw.Close())
	require.NoError(t, zf.Close())

	// Extract
	watchedPath, ratingsPath, err := extractLetterboxdCSVs(zipPath, env.RootDir())
	require.NoError(t, err)
	require.FileExists(t, watchedPath)
	require.Empty(t, ratingsPath) // ratings.csv not present
}

func TestMergeWatchedAndRatings(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Create test files
	watchedPath := env.Path("watched.csv")
	watchedContent := "Date,Name,Year,URI\n" +
		"2025-01-01,Movie1,2025,https://boxd.it/abc\n" +
		"2025-01-02,Movie2,2025,https://boxd.it/def\n" +
		"2025-01-03,Movie3,2025,https://boxd.it/ghi\n"
	require.NoError(t, os.WriteFile(watchedPath, []byte(watchedContent), 0644))

	ratingsPath := env.Path("ratings.csv")
	ratingsContent := "Date,Name,Year,URI,Rating\n" +
		"2025-01-01,Movie1,2025,https://boxd.it/abc,5\n" +
		"2025-01-03,Movie3,2025,https://boxd.it/ghi,3.5\n"
	require.NoError(t, os.WriteFile(ratingsPath, []byte(ratingsContent), 0644))

	outputPath := env.Path("merged.csv")
	err := mergeWatchedAndRatings(watchedPath, ratingsPath, outputPath)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")
	require.GreaterOrEqual(t, len(lines), 4)

	// Check header has Rating column
	require.Contains(t, lines[0], "Rating")

	// Movie1 should have rating 5
	require.Contains(t, lines[1], ",5")

	// Movie2 should have empty rating
	require.Contains(t, lines[2], "Movie2")
	require.NotContains(t, lines[2], ",5")
	require.NotContains(t, lines[2], ",3.5")

	// Movie3 should have rating 3.5
	require.Contains(t, lines[3], ",3.5")
}

func TestMergeWatchedAndRatingsWithoutRatingsFile(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Create only watched.csv
	watchedPath := env.Path("watched.csv")
	watchedContent := "Date,Name,Year,URI\n2025-01-01,Movie1,2025,https://boxd.it/abc\n"
	require.NoError(t, os.WriteFile(watchedPath, []byte(watchedContent), 0644))

	outputPath := env.Path("merged.csv")
	err := mergeWatchedAndRatings(watchedPath, "", outputPath)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")
	require.Contains(t, lines[0], "Rating") // Header should have Rating
	require.Contains(t, lines[1], "Movie1")
	// Rating column should be empty
	fields := strings.Split(lines[1], ",")
	require.Equal(t, 5, len(fields)) // Date, Name, Year, URI, Rating
	require.Equal(t, "", fields[4])  // Rating should be empty
}

func TestFindDownloadedZipSkipsPartialFiles(t *testing.T) {
	env := testutil.NewTestEnv(t)

	startTime := time.Now().Add(-1 * time.Hour) // Start time in the past

	// Create partial download
	require.NoError(t, os.WriteFile(env.Path("letterboxd-export.zip.crdownload"), []byte("partial"), 0644))

	_, err := findDownloadedZip(env.RootDir(), startTime)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ZIP file not found")

	// Create complete download
	require.NoError(t, os.WriteFile(env.Path("letterboxd-user-2025.zip"), []byte("complete"), 0644))

	path, err := findDownloadedZip(env.RootDir(), startTime)
	require.NoError(t, err)
	require.Contains(t, path, "letterboxd")
	require.Contains(t, filepath.Base(path), "letterboxd-user-2025.zip")
}

func TestFindDownloadedZipIgnoresNonLetterboxdFiles(t *testing.T) {
	env := testutil.NewTestEnv(t)

	startTime := time.Now().Add(-1 * time.Hour) // Start time in the past

	// Create non-letterboxd zip files
	require.NoError(t, os.WriteFile(env.Path("other-export.zip"), []byte("data"), 0644))
	require.NoError(t, os.WriteFile(env.Path("random.zip"), []byte("data"), 0644))

	_, err := findDownloadedZip(env.RootDir(), startTime)
	require.Error(t, err)

	// Add a valid letterboxd ZIP
	require.NoError(t, os.WriteFile(env.Path("letterboxd-test-2025-01-01.zip"), []byte("data"), 0644))

	path, err := findDownloadedZip(env.RootDir(), startTime)
	require.NoError(t, err)
	require.Contains(t, filepath.Base(path), "letterboxd-")
}

func TestCopyFile(t *testing.T) {
	env := testutil.NewTestEnv(t)

	src := env.Path("source.txt")
	dst := env.Path("dest.txt")

	testContent := "test file content"
	require.NoError(t, os.WriteFile(src, []byte(testContent), 0644))

	err := copyFile(src, dst)
	require.NoError(t, err)
	require.FileExists(t, dst)

	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, testContent, string(content))
}

func TestFindDownloadedZipRespectsStartTime(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Create old ZIP file
	oldZipPath := env.Path("letterboxd-old-2024-12-31.zip")
	require.NoError(t, os.WriteFile(oldZipPath, []byte("old"), 0644))

	// Set file mod time to yesterday
	yesterday := time.Now().Add(-24 * time.Hour)
	require.NoError(t, os.Chtimes(oldZipPath, yesterday, yesterday))

	// Start time is now
	startTime := time.Now()

	// Should not find old file
	_, err := findDownloadedZip(env.RootDir(), startTime)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ZIP file not found")

	// Create new ZIP file after start time
	time.Sleep(10 * time.Millisecond) // Ensure file is newer
	newZipPath := env.Path("letterboxd-new-2025-01-27.zip")
	require.NoError(t, os.WriteFile(newZipPath, []byte("new"), 0644))

	// Should find new file
	path, err := findDownloadedZip(env.RootDir(), startTime)
	require.NoError(t, err)
	require.Equal(t, newZipPath, path)
}

func TestMergeWatchedAndRatingsCorruptedRatingsFile(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Create valid watched.csv
	watchedPath := env.Path("watched.csv")
	watchedContent := "Date,Name,Year,URI\n2025-01-01,Movie1,2025,https://boxd.it/abc\n"
	require.NoError(t, os.WriteFile(watchedPath, []byte(watchedContent), 0644))

	// Create corrupted ratings.csv (invalid CSV format)
	ratingsPath := env.Path("ratings.csv")
	ratingsContent := "Date,Name,Year,URI,Rating\n\"unclosed quote,Movie1,2025,https://boxd.it/abc,5\n"
	require.NoError(t, os.WriteFile(ratingsPath, []byte(ratingsContent), 0644))

	outputPath := env.Path("merged.csv")
	err := mergeWatchedAndRatings(watchedPath, ratingsPath, outputPath)

	// Should return error when ratings.csv is corrupted
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse ratings.csv")
}

func TestMergeWatchedAndRatingsMissingWatchedFile(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Don't create watched.csv
	watchedPath := env.Path("watched.csv")
	ratingsPath := env.Path("ratings.csv")
	outputPath := env.Path("merged.csv")

	err := mergeWatchedAndRatings(watchedPath, ratingsPath, outputPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open watched.csv")
}

func TestWaitForLoginSuccessRespectsContext(t *testing.T) {
	t.Skip("Requires chromedp integration - will be tested with fix implementation")
}

func TestWaitForSelectorRespectsContext(t *testing.T) {
	t.Skip("Requires chromedp integration - will be tested with fix implementation")
}

func TestMergeWatchedAndRatings_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		watchedContent string
		ratingsContent string
		wantLines      []string
	}{
		{
			name: "movies with same name and year but different URIs are matched correctly",
			watchedContent: `Date,Name,Year,Letterboxd URI
2019-03-09,The Thing,1982,https://boxd.it/1abc
2019-03-09,The Thing,1982,https://boxd.it/2xyz`,
			ratingsContent: `Date,Name,Year,Letterboxd URI,Rating
2019-03-09,The Thing,1982,https://boxd.it/1abc,5
2019-03-09,The Thing,1982,https://boxd.it/2xyz,3`,
			wantLines: []string{
				"Date,Name,Year,Letterboxd URI,Rating",
				"2019-03-09,The Thing,1982,https://boxd.it/1abc,5",
				"2019-03-09,The Thing,1982,https://boxd.it/2xyz,3",
			},
		},
		{
			name: "duplicate URIs in watched both get same rating",
			watchedContent: `Date,Name,Year,Letterboxd URI
2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA
2019-03-10,Captain Marvel,2019,https://boxd.it/9vSA`,
			ratingsContent: `Date,Name,Year,Letterboxd URI,Rating
2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA,4`,
			wantLines: []string{
				"Date,Name,Year,Letterboxd URI,Rating",
				"2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA,4",
				"2019-03-10,Captain Marvel,2019,https://boxd.it/9vSA,4",
			},
		},
		{
			name: "duplicate URIs in ratings uses last rating",
			watchedContent: `Date,Name,Year,Letterboxd URI
2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA`,
			ratingsContent: `Date,Name,Year,Letterboxd URI,Rating
2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA,4
2019-03-10,Captain Marvel,2019,https://boxd.it/9vSA,5`,
			wantLines: []string{
				"Date,Name,Year,Letterboxd URI,Rating",
				"2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA,5",
			},
		},
		{
			name: "movies watched on same date are preserved in order",
			watchedContent: `Date,Name,Year,Letterboxd URI
2019-03-09,Movie A,2019,https://boxd.it/aaa
2019-03-09,Movie B,2019,https://boxd.it/bbb
2019-03-09,Movie C,2019,https://boxd.it/ccc`,
			ratingsContent: `Date,Name,Year,Letterboxd URI,Rating
2019-03-09,Movie A,2019,https://boxd.it/aaa,5
2019-03-09,Movie C,2019,https://boxd.it/ccc,3`,
			wantLines: []string{
				"Date,Name,Year,Letterboxd URI,Rating",
				"2019-03-09,Movie A,2019,https://boxd.it/aaa,5",
				"2019-03-09,Movie B,2019,https://boxd.it/bbb,",
				"2019-03-09,Movie C,2019,https://boxd.it/ccc,3",
			},
		},
		{
			name: "URIs are case-sensitive for matching",
			watchedContent: `Date,Name,Year,Letterboxd URI
2019-03-09,Movie1,2019,https://boxd.it/AbC`,
			ratingsContent: `Date,Name,Year,Letterboxd URI,Rating
2019-03-09,Movie1,2019,https://boxd.it/abc,4`,
			wantLines: []string{
				"Date,Name,Year,Letterboxd URI,Rating",
				"2019-03-09,Movie1,2019,https://boxd.it/AbC,",
			},
		},
		{
			name: "empty ratings file results in all empty ratings",
			watchedContent: `Date,Name,Year,Letterboxd URI
2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA`,
			ratingsContent: `Date,Name,Year,Letterboxd URI,Rating
`,
			wantLines: []string{
				"Date,Name,Year,Letterboxd URI,Rating",
				"2019-03-09,Captain Marvel,2019,https://boxd.it/9vSA,",
			},
		},
		{
			name: "ratings with half-star values are preserved",
			watchedContent: `Date,Name,Year,Letterboxd URI
2019-03-09,Movie1,2019,https://boxd.it/abc`,
			ratingsContent: `Date,Name,Year,Letterboxd URI,Rating
2019-03-09,Movie1,2019,https://boxd.it/abc,4.5`,
			wantLines: []string{
				"Date,Name,Year,Letterboxd URI,Rating",
				"2019-03-09,Movie1,2019,https://boxd.it/abc,4.5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testutil.NewTestEnv(t)

			watchedPath := env.Path("watched.csv")
			require.NoError(t, os.WriteFile(watchedPath, []byte(tt.watchedContent), 0644))

			ratingsPath := env.Path("ratings.csv")
			require.NoError(t, os.WriteFile(ratingsPath, []byte(tt.ratingsContent), 0644))

			outputPath := env.Path("merged.csv")
			err := mergeWatchedAndRatings(watchedPath, ratingsPath, outputPath)
			require.NoError(t, err)

			content, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			require.Equal(t, len(tt.wantLines), len(lines), "Expected %d lines, got %d\nContent:\n%s", len(tt.wantLines), len(lines), string(content))

			for i, wantLine := range tt.wantLines {
				require.Equal(t, wantLine, lines[i], "Line %d mismatch", i)
			}
		})
	}
}
