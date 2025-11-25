package goodreads

import (
	"context"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseGoodreadsWithParams(t *testing.T) {
	t.Cleanup(func() { parseGoodreadsFunc = ParseGoodreads })

	env := testutil.NewTestEnv(t)

	env.WriteFileString("books.csv", "id,title\n1,Test\n")
	csv := env.Path("books.csv")

	var called bool
	parseGoodreadsFunc = func(params ParseParams) error {
		called = true
		require.Equal(t, csv, params.CSVPath)
		require.True(t, params.WriteJSON)
		require.Equal(t, env.Path("books.json"), params.JSONOutput)
		require.NotEmpty(t, params.OutputDir)
		return nil
	}

	err := ParseGoodreadsWithParams(ParseParams{
		CSVPath:    csv,
		OutputDir:  env.Path("output"),
		WriteJSON:  true,
		JSONOutput: env.Path("books.json"),
		Overwrite:  true,
	})

	require.NoError(t, err, "ParseGoodreadsWithParams should not error")
	require.True(t, called, "expected parseGoodreadsFunc to be called")
}

func TestParseGoodreadsWithAutomation(t *testing.T) {
	t.Cleanup(func() {
		parseGoodreadsFunc = ParseGoodreads
		downloadGoodreadsCSV = AutomateGoodreadsExport
	})

	env := testutil.NewTestEnv(t)
	downloaded := env.Path("automated.csv")

	var automationOpts AutomationOptions
	downloadGoodreadsCSV = func(_ context.Context, opts AutomationOptions) (string, error) {
		automationOpts = opts
		return downloaded, nil
	}

	var called bool
	parseGoodreadsFunc = func(params ParseParams) error {
		called = true
		require.Equal(t, downloaded, params.CSVPath)
		require.True(t, params.Automated)
		return nil
	}

	err := ParseGoodreadsWithParams(ParseParams{
		Automated: true,
		OutputDir: env.Path("output"),
		AutomationOptions: AutomationOptions{
			Email:    "user@example.com",
			Password: "secret",
			Headless: true,
			Timeout:  time.Minute,
		},
	})

	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, "user@example.com", automationOpts.Email)
	require.Equal(t, "secret", automationOpts.Password)
	require.Equal(t, time.Minute, automationOpts.Timeout)
}
