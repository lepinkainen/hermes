package letterboxd

import (
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseLetterboxdWithParams(t *testing.T) {
	env := testutil.NewTestEnv(t)

	env.WriteFileString("log.csv", "Date,Name,Year\n")
	csv := env.Path("log.csv")
	dir := env.RootDir()

	var called bool
	parseLetterboxdFunc = func() error {
		called = true
		require.Equal(t, csv, csvFile, "csvFile mismatch")
		require.True(t, strings.Contains(outputDir, dir), "outputDir should contain %s", dir)
		require.True(t, tmdbEnabled, "tmdbEnabled should be true")
		require.True(t, tmdbGenerateContent, "tmdbGenerateContent should be true")
		return nil
	}
	defer func() { parseLetterboxdFunc = ParseLetterboxd }()

	err := ParseLetterboxdWithParams(csv, dir, false, "", true, true, false, true, true, []string{"overview"}, false, "")
	require.NoError(t, err, "ParseLetterboxdWithParams should not error")
	require.True(t, called, "expected parser to run")
}
