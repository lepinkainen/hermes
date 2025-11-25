package goodreads

import (
	"strings"
	"testing"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseGoodreadsWithParams(t *testing.T) {
	env := testutil.NewTestEnv(t)

	env.WriteFileString("books.csv", "id,title\n1,Test\n")
	csv := env.Path("books.csv")
	dir := env.RootDir()

	var called bool
	parseGoodreadsFunc = func() error {
		called = true
		require.Equal(t, csv, csvFile, "csvFile mismatch")
		require.True(t, strings.Contains(outputDir, dir), "outputDir should contain %s", dir)
		require.True(t, writeJSON, "writeJSON should be true")
		require.NotEmpty(t, jsonOutput, "jsonOutput should be set")
		return nil
	}
	defer func() { parseGoodreadsFunc = ParseGoodreads }()

	err := ParseGoodreadsWithParams(csv, dir, true, env.Path("books.json"), true)
	require.NoError(t, err, "ParseGoodreadsWithParams should not error")
	require.True(t, called, "expected parseGoodreadsFunc to be called")
}
