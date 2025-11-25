package imdb

import (
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseImdbWithParams(t *testing.T) {
	env := testutil.NewTestEnv(t)

	env.WriteFileString("ratings.csv", "const,yourRating\n1,10\n")
	csv := env.Path("ratings.csv")
	dir := env.RootDir()

	var called bool
	parseImdbFunc = func() error {
		called = true
		require.Equal(t, csv, csvFile, "csvFile mismatch")
		require.True(t, strings.Contains(outputDir, dir), "outputDir should contain %s", dir)
		require.True(t, tmdbEnabled, "tmdbEnabled should be true")
		require.True(t, tmdbDownloadCover, "tmdbDownloadCover should be true")
		require.False(t, tmdbInteractive, "tmdbInteractive should be false")
		require.Equal(t, "overview", tmdbContentSections[0], "tmdbContentSections not set")
		return nil
	}
	defer func() { parseImdbFunc = ParseImdb }()

	err := ParseImdbWithParams(csv, dir, true, env.Path("imdb.json"), true, true, true, true, false, []string{"overview"}, false, "")
	require.NoError(t, err, "ParseImdbWithParams should not error")
	require.True(t, called, "expected parseImdbFunc to be called")
}
