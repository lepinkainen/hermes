package goodreads

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lepinkainen/hermes/internal/cmdutil"
	"github.com/spf13/viper"
)

const defaultAutomationTimeout = 3 * time.Minute

type ParseParams struct {
	CSVPath           string
	OutputDir         string
	WriteJSON         bool
	JSONOutput        string
	Automated         bool
	DryRun            bool
	AutomationOptions AutomationOptions
}

type ParseGoodreadsFuncType func(params ParseParams) error
type DownloadGoodreadsCSVFuncType func(ctx context.Context, opts AutomationOptions) (string, error)

var DefaultParseGoodreadsFunc ParseGoodreadsFuncType = ParseGoodreads
var DefaultDownloadGoodreadsCSVFunc DownloadGoodreadsCSVFuncType = AutomateGoodreadsExport

// ParseGoodreadsWithParams allows calling goodreads parsing with specific parameters
// This is used by the Kong-based CLI implementation
func ParseGoodreadsWithParams(params ParseParams, parseFunc ParseGoodreadsFuncType, downloadFunc DownloadGoodreadsCSVFuncType) error {
	cmdConfig := &cmdutil.BaseCommandConfig{
		OutputDir:  params.OutputDir,
		ConfigKey:  "goodreads",
		WriteJSON:  params.WriteJSON,
		JSONOutput: params.JSONOutput,
	}

	if err := cmdutil.SetupOutputDir(cmdConfig); err != nil {
		return err
	}

	params.OutputDir = cmdConfig.OutputDir
	params.WriteJSON = cmdConfig.WriteJSON
	params.JSONOutput = cmdConfig.JSONOutput

	if params.Automated {
		if params.AutomationOptions.Timeout == 0 {
			params.AutomationOptions.Timeout = defaultAutomationTimeout
		}

		ctx, cancel := context.WithTimeout(context.Background(), params.AutomationOptions.Timeout)
		defer cancel()

		csvPath, err := downloadFunc(ctx, params.AutomationOptions)
		if err != nil {
			return err
		}
		params.CSVPath = csvPath

		if params.DryRun {
			slog.Info("Dry-run mode: automation completed successfully", "csv_path", csvPath)
			return nil
		}
	}

	if params.CSVPath == "" {
		return fmt.Errorf("input CSV file is required (provide via --input flag or goodreads.csvfile in config)")
	}

	// Call the existing parser
	return parseFunc(params)
}

// GoodreadsCmd represents the goodreads import command
type GoodreadsCmd struct {
	Input             string        `short:"f" help:"Path to Goodreads library export CSV file"`
	Output            string        `short:"o" help:"Subdirectory under markdown output directory for Goodreads files" default:"goodreads"`
	JSON              bool          `help:"Write data to JSON format"`
	JSONOutput        string        `help:"Path to JSON output file (defaults to json/goodreads.json)"`
	Automated         bool          `help:"Automatically download Goodreads export via browser automation"`
	GoodreadsEmail    string        `help:"Goodreads account email for automation"`
	GoodreadsPassword string        `help:"Goodreads account password for automation"`
	Headful           bool          `help:"Run automation with a visible browser window (default is headless)"`
	DownloadDir       string        `help:"Directory for automated Goodreads export download (defaults to exports/)"`
	AutomationTimeout time.Duration `help:"Timeout for Goodreads automation flow" default:"3m"`
	DryRun            bool          `help:"Test automation without running the import (automation only)"`

	// Dependencies
	parseFunc    ParseGoodreadsFuncType
	downloadFunc DownloadGoodreadsCSVFuncType
}

// Init initializes the GoodreadsCmd with its dependencies.
// This method is used for dependency injection in tests or when manually constructing the command.
func (g *GoodreadsCmd) Init(parseFn ParseGoodreadsFuncType, downloadFn DownloadGoodreadsCSVFuncType) {
	g.parseFunc = parseFn
	g.downloadFunc = downloadFn
}

func (g *GoodreadsCmd) Run() error {
	slog.Info("GoodreadsCmd.Run() called", "g.Automated", g.Automated)
	// Read from config if value not provided via flag
	input := g.Input
	if input == "" && !g.Automated {
		input = viper.GetString("goodreads.csvfile")
	}

	automationTimeout := g.AutomationTimeout
	if automationTimeout == 0 {
		automationTimeout = viper.GetDuration("goodreads.automation.timeout")
		if automationTimeout == 0 {
			automationTimeout = 3 * time.Minute
		}
	}

	headful := g.Headful
	if !headful && viper.IsSet("goodreads.automation.headful") {
		headful = viper.GetBool("goodreads.automation.headful")
	}

	downloadDir := g.DownloadDir
	if downloadDir == "" {
		downloadDir = viper.GetString("goodreads.automation.download_dir")
	}

	email := g.GoodreadsEmail
	if email == "" {
		email = viper.GetString("goodreads.automation.email")
	}
	password := g.GoodreadsPassword
	if password == "" {
		password = viper.GetString("goodreads.automation.password")
	}

	// Check if required value is still missing when not automated
	if input == "" && !g.Automated {
		return fmt.Errorf("input CSV file is required (provide via --input flag or goodreads.csvfile in config)")
	}

	params := ParseParams{
		CSVPath:    input,
		OutputDir:  g.Output,
		JSONOutput: g.JSONOutput,
		WriteJSON:  g.JSON,
		Automated:  g.Automated,
		DryRun:     g.DryRun,
		AutomationOptions: AutomationOptions{
			Email:       email,
			Password:    password,
			DownloadDir: downloadDir,
			Headless:    !headful,
			Timeout:     automationTimeout,
		},
	}

	if params.Automated && (params.AutomationOptions.Email == "" || params.AutomationOptions.Password == "") {
		return fmt.Errorf("goodreads automation requires email and password (use flags, environment variables, or config)")
	}

	parseFunc := g.parseFunc
	if parseFunc == nil {
		parseFunc = DefaultParseGoodreadsFunc
	}
	downloadFunc := g.downloadFunc
	if downloadFunc == nil {
		downloadFunc = DefaultDownloadGoodreadsCSVFunc
	}

	return ParseGoodreadsWithParams(params, parseFunc, downloadFunc)
}
