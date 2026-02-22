package goodreads

import (
	"context"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseGoodreadsWithParams_ConfigFallback(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)

	env.WriteFileString("books.csv", "id,title\n1,Test\n")
	csv := env.Path("books.csv")

	var capturedParams ParseParams
	mockParseGoodreads := func(params ParseParams) error {
		capturedParams = params
		return nil
	}

	err := ParseGoodreadsWithParams(ParseParams{
		CSVPath:    csv,
		OutputDir:  env.Path("output"),
		WriteJSON:  true,
		JSONOutput: env.Path("books.json"),
	}, mockParseGoodreads, DefaultDownloadGoodreadsCSVFunc)

	require.NoError(t, err, "ParseGoodreadsWithParams should not error")

	// Verify parameters were passed correctly
	require.Equal(t, csv, capturedParams.CSVPath)
	require.True(t, capturedParams.WriteJSON)
	require.Equal(t, env.Path("books.json"), capturedParams.JSONOutput)
	require.NotEmpty(t, capturedParams.OutputDir)
}

func TestParseGoodreadsWithParams_AutomationFlow(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	downloaded := env.Path("automated.csv")

	var capturedOpts AutomationOptions
	mockDownloadGoodreadsCSV := func(_ context.Context, opts AutomationOptions) (string, error) {
		capturedOpts = opts
		return downloaded, nil
	}

	var capturedParams ParseParams
	mockParseGoodreads := func(params ParseParams) error {
		capturedParams = params
		require.Equal(t, downloaded, params.CSVPath, "Should use downloaded CSV path")
		require.True(t, params.Automated, "Should mark as automated")
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
	}, mockParseGoodreads, mockDownloadGoodreadsCSV)

	require.NoError(t, err)

	// Verify automation options were passed correctly
	require.Equal(t, "user@example.com", capturedOpts.Email)
	require.Equal(t, "secret", capturedOpts.Password)
	require.Equal(t, time.Minute, capturedOpts.Timeout)

	// Verify parse was called with automated flag
	require.True(t, capturedParams.Automated)
}

func TestGoodreadsCmdRun_ParameterPassing(t *testing.T) {
	t.Parallel()

	testInputFile := "test-input.csv"
	testOutputDir := "test-output"
	testJSONOutput := "test-json.json"
	testEmail := "test@example.com"
	testPassword := "test-password"

	var capturedParams ParseParams
	mockParseGoodreads := func(params ParseParams) error {
		capturedParams = params
		return nil
	}

	mockDownloadGoodreadsCSV := func(_ context.Context, opts AutomationOptions) (string, error) {
		return "", nil
	}

	cmd := GoodreadsCmd{
		Input:             testInputFile,
		Output:            testOutputDir,
		JSON:              true,
		JSONOutput:        testJSONOutput,
		Automated:         false,
		GoodreadsEmail:    testEmail,
		GoodreadsPassword: testPassword,
		Headful:           false,
		DownloadDir:       "exports",
		AutomationTimeout: 5 * time.Minute,
		DryRun:            false,
	}
	cmd.Init(mockParseGoodreads, mockDownloadGoodreadsCSV)

	err := cmd.Run()
	require.NoError(t, err)

	// Verify all parameters were passed correctly
	require.Equal(t, testInputFile, capturedParams.CSVPath)
	require.Equal(t, "markdown/"+testOutputDir, capturedParams.OutputDir)
	require.True(t, capturedParams.WriteJSON)
	require.Equal(t, testJSONOutput, capturedParams.JSONOutput)
	require.False(t, capturedParams.Automated)
	require.False(t, capturedParams.DryRun)
	require.Equal(t, testEmail, capturedParams.AutomationOptions.Email)
	require.Equal(t, testPassword, capturedParams.AutomationOptions.Password)
	require.True(t, capturedParams.AutomationOptions.Headless)
	require.Equal(t, "exports", capturedParams.AutomationOptions.DownloadDir)
	require.Equal(t, 5*time.Minute, capturedParams.AutomationOptions.Timeout)
}
