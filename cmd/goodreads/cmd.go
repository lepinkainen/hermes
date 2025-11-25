package goodreads

import (
	"context"
	"fmt"
	"time"

	"github.com/lepinkainen/hermes/internal/cmdutil"
)

const defaultAutomationTimeout = 3 * time.Minute

type ParseParams struct {
	CSVPath           string
	OutputDir         string
	WriteJSON         bool
	JSONOutput        string
	Overwrite         bool
	Automated         bool
	AutomationOptions AutomationOptions
}

var parseGoodreadsFunc = ParseGoodreads

// ParseGoodreadsWithParams allows calling goodreads parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseGoodreadsWithParams(params ParseParams) error {
	cmdConfig := &cmdutil.BaseCommandConfig{
		OutputDir:  params.OutputDir,
		ConfigKey:  "goodreads",
		WriteJSON:  params.WriteJSON,
		JSONOutput: params.JSONOutput,
		Overwrite:  params.Overwrite,
	}

	if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
		return err
	}

	params.OutputDir = cmdConfig.OutputDir
	params.WriteJSON = cmdConfig.WriteJSON
	params.JSONOutput = cmdConfig.JSONOutput
	params.Overwrite = cmdConfig.Overwrite

	if params.Automated {
		if params.AutomationOptions.Timeout == 0 {
			params.AutomationOptions.Timeout = defaultAutomationTimeout
		}

		ctx, cancel := context.WithTimeout(context.Background(), params.AutomationOptions.Timeout)
		defer cancel()

		csvPath, err := downloadGoodreadsCSV(ctx, params.AutomationOptions)
		if err != nil {
			return err
		}
		params.CSVPath = csvPath
	}

	if params.CSVPath == "" {
		return fmt.Errorf("input CSV file is required (provide via --input flag or goodreads.csvfile in config)")
	}

	// Call the existing parser
	return parseGoodreadsFunc(params)
}
