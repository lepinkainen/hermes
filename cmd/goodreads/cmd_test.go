package goodreads

import (
	"context"
	"testing"
	"time"

	"github.com/lepinkainen/hermes/internal/automation"
	"github.com/lepinkainen/hermes/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseGoodreadsWithParams(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)

	env.WriteFileString("books.csv", "id,title\n1,Test\n")
	csv := env.Path("books.csv")

	var called bool
	mockParseGoodreads := func(params ParseParams) error {
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
	}, mockParseGoodreads, DefaultDownloadGoodreadsCSVFunc, &automation.DefaultCDPRunner{})

	require.NoError(t, err, "ParseGoodreadsWithParams should not error")
	require.True(t, called, "expected mockParseGoodreads to be called")
}

func TestParseGoodreadsWithAutomation(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	downloaded := env.Path("automated.csv")

	var automationOpts AutomationOptions
	mockDownloadGoodreadsCSV := func(_ context.Context, _ automation.CDPRunner, opts AutomationOptions) (string, error) {
		automationOpts = opts
		return downloaded, nil
	}

	var called bool
	mockParseGoodreads := func(params ParseParams) error {
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
	}, mockParseGoodreads, mockDownloadGoodreadsCSV, &automation.DefaultCDPRunner{})

	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, "user@example.com", automationOpts.Email)
	require.Equal(t, "secret", automationOpts.Password)
	require.Equal(t, time.Minute, automationOpts.Timeout)
}

func TestGoodreadsCmdRun(t *testing.T) {
	t.Parallel()

	testInputFile := "test-input.csv"
	testOutputDir := "test-output"
	testJSONOutput := "test-json.json"
	testEmail := "test@example.com"
	testPassword := "test-password"

	var receivedParams ParseParams
	mockParseGoodreads := func(params ParseParams) error {
		receivedParams = params
		return nil
	}

	mockDownloadGoodreadsCSV := func(_ context.Context, _ automation.CDPRunner, opts AutomationOptions) (string, error) {
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
	cmd.Init(mockParseGoodreads, mockDownloadGoodreadsCSV, &mockCDPRunner{})

	t.Logf("cmd.Automated before Run: %v", cmd.Automated)

	err := cmd.Run()
	require.NoError(t, err)

	require.Equal(t, testInputFile, receivedParams.CSVPath)
	require.Equal(t, "markdown/"+testOutputDir, receivedParams.OutputDir)
	require.True(t, receivedParams.WriteJSON)
	require.Equal(t, testJSONOutput, receivedParams.JSONOutput)
	require.False(t, receivedParams.Automated)
	require.False(t, receivedParams.DryRun)
	require.Equal(t, testEmail, receivedParams.AutomationOptions.Email)
	require.Equal(t, testPassword, receivedParams.AutomationOptions.Password)
	require.True(t, receivedParams.AutomationOptions.Headless)
	require.Equal(t, "exports", receivedParams.AutomationOptions.DownloadDir)
	require.Equal(t, 5*time.Minute, receivedParams.AutomationOptions.Timeout)
}
