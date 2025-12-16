package goodreads

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/lepinkainen/hermes/internal/automation"
)

const (
	exportPollInterval   = 3 * time.Second
	downloadPollInterval = 2 * time.Second
	exportFileName       = "goodreads_library_export.csv"
)

type AutomationOptions struct {
	Email       string
	Password    string
	DownloadDir string
	Headless    bool
	Timeout     time.Duration
}

func AutomateGoodreadsExport(parentCtx context.Context, runner automation.CDPRunner, opts AutomationOptions) (string, error) {
	if opts.Email == "" || opts.Password == "" {
		return "", errors.New("goodreads automation requires both email and password")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultAutomationTimeout
	}

	_, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	downloadDir, cleanup, err := automation.PrepareDownloadDir(opts.DownloadDir, "hermes-goodreads-*")
	if err != nil {
		return "", err
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()
	slog.Info("Prepared Goodreads download directory", "path", downloadDir, "headless", opts.Headless)

	browserCtx, cancelBrowser := automation.NewBrowser(runner, automation.AutomationOptions{Headless: opts.Headless})
	defer cancelBrowser()

	if err := automation.ConfigureDownloadDirectory(browserCtx, runner, downloadDir); err != nil {
		return "", err
	}

	if err := performGoodreadsLogin(browserCtx, runner, opts); err != nil {
		return "", err
	}

	if err := triggerGoodreadsExport(browserCtx, runner); err != nil {
		return "", err
	}

	exportLink, err := waitForExportLink(browserCtx, runner)
	if err != nil {
		return "", err
	}

	slog.Info("Initiating download of export file")

	// Try clicking the link instead of navigating directly
	// This is more natural and might avoid HTTP response issues
	clickSelector := fmt.Sprintf(`//a[@href="%s"]`, strings.TrimPrefix(exportLink, "https://www.goodreads.com"))
	if err := runner.Run(browserCtx, chromedp.Click(clickSelector, chromedp.BySearch)); err != nil {
		// Fallback to navigation if click fails
		slog.Info("Click failed, trying direct navigation", "error", err)
		if err := runner.Run(browserCtx, chromedp.Navigate(exportLink)); err != nil {
			return "", fmt.Errorf("failed to start Goodreads export download: %w", err)
		}
	}

	csvPath, err := waitForDownload(browserCtx, runner, downloadDir)
	if err != nil {
		return "", err
	}

	finalPath, err := moveDownloadedCSV(csvPath, opts.DownloadDir)
	if err != nil {
		return "", err
	}

	slog.Info("Goodreads export downloaded", "path", finalPath)
	return finalPath, nil
}

func performGoodreadsLogin(ctx context.Context, runner automation.CDPRunner, opts AutomationOptions) error {
	slog.Info("Logging in to Goodreads", "email", opts.Email)

	if err := runner.Run(ctx, chromedp.Navigate("https://www.goodreads.com/user/sign_in")); err != nil {
		return fmt.Errorf("failed to open login page: %w", err)
	}

	if err := runner.Run(ctx, chromedp.WaitVisible(`//button[contains(., "Sign in with email")]`, chromedp.BySearch)); err != nil {
		return fmt.Errorf("email login button not visible: %w", err)
	}
	if err := runner.Run(ctx, chromedp.Click(`//button[contains(., "Sign in with email")]`, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to click email login button: %w", err)
	}

	// Wait for navigation to complete (may redirect to Amazon sign-in page)
	slog.Debug("Waiting for page to load after clicking email sign-in button")
	var currentURL string
	if err := runner.Run(ctx, chromedp.Sleep(2*time.Second)); err != nil {
		return fmt.Errorf("failed to wait after clicking email button: %w", err)
	}
	if err := runner.Run(ctx, chromedp.Location(&currentURL)); err == nil {
		slog.Info("Current page after clicking email sign-in", "url", currentURL)
	}

	emailSelector, err := automation.WaitForSelector(ctx, runner, []string{
		`//input[@type="email" or @name="email" or @id="ap_email"]`,
		`//input[@name="user[email]"]`,
	}, "email field", 10*time.Second)
	if err != nil {
		return err
	}
	if err := runner.Run(ctx, chromedp.SendKeys(emailSelector, opts.Email, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to fill email: %w", err)
	}

	passwordSelector, err := automation.WaitForSelector(ctx, runner, []string{
		`//input[@type="password" or @name="password" or @id="ap_password"]`,
		`//input[@name="user[password]"]`,
	}, "password field", 10*time.Second)
	if err != nil {
		return err
	}
	if err := runner.Run(ctx, chromedp.SendKeys(passwordSelector, opts.Password, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to fill password: %w", err)
	}

	submitSelector, err := automation.WaitForSelector(ctx, runner, []string{
		`//button[@type="submit" or contains(., "Sign in")]`,
		`//input[@type="submit" and (@name="signIn" or @id="signInSubmit")]`,
	}, "sign-in submit button", 10*time.Second)
	if err != nil {
		return err
	}
	if err := runner.Run(ctx, chromedp.Click(submitSelector, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to submit login form: %w", err)
	}

	if err := waitForLoginSuccess(ctx, runner); err != nil {
		return err
	}

	slog.Info("Goodreads login completed")
	return nil
}

func waitForLoginSuccess(ctx context.Context, runner automation.CDPRunner) error {
	start := time.Now()
	slog.Info("Waiting for Goodreads login to complete")

	err := automation.WaitForURLChange(
		ctx,
		runner,
		func() (string, error) {
			var location string
			err := runner.Run(ctx, chromedp.Location(&location))
			return location, err
		},
		[]string{"/sign_in", "ap/signin"},
		30*time.Second,
	)

	if err != nil {
		return fmt.Errorf("timed out waiting for Goodreads login to complete: %w", err)
	}

	slog.Info("Goodreads login completed", "elapsed", time.Since(start))
	return nil
}

func triggerGoodreadsExport(ctx context.Context, runner automation.CDPRunner) error {
	slog.Info("Navigating to Goodreads export page")

	if err := runner.Run(ctx, chromedp.Navigate("https://www.goodreads.com/review/import")); err != nil {
		return fmt.Errorf("failed to navigate to import page: %w", err)
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Check if an export link already exists (from a previous export)
	var existingLink string
	_ = runner.Run(ctx, chromedp.Evaluate(`
		(() => {
			const fileList = document.getElementById('exportFile');
			if (fileList) {
				const link = fileList.querySelector('a');
				if (link && link.href) {
					return link.href;
				}
			}
			return "";
		})()
	`, &existingLink))

	if existingLink != "" {
		slog.Info("Found existing export link, skipping export button click", "link", existingLink)
		return nil
	}

	// No existing link, need to trigger export
	slog.Info("No existing export link found, triggering new export")

	// Try multiple selectors for the export button
	exportButtonSelector, err := automation.WaitForSelector(ctx, runner, []string{
		`//button[contains(., 'Export Library')]`,
		`//input[@value='Export Library']`,
		`//input[@type='submit' and contains(@value, 'Export')]`,
	}, "export library button", 10*time.Second)
	if err != nil {
		return err
	}

	slog.Info("Found export button", "selector", exportButtonSelector)

	if err := runner.Run(ctx, chromedp.Click(exportButtonSelector, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	slog.Info("Clicked export button")
	return nil
}

func waitForExportLink(ctx context.Context, runner automation.CDPRunner) (string, error) {
	start := time.Now()
	ticker := time.NewTicker(exportPollInterval)
	defer ticker.Stop()

	tries := 0
	for {
		var exportLink string
		if err := runner.Run(ctx, chromedp.Evaluate(`
			(() => {
				// Try looking in the fileList div first (most reliable)
				const fileList = document.getElementById('exportFile');
				if (fileList) {
					const link = fileList.querySelector('a');
					if (link && link.href) {
						return link.href;
					}
				}

				// Fallback: try multiple possible link patterns
				let link = document.querySelector('a[href*="review_porter/export"]');
				if (!link) {
					link = document.querySelector('a[href*="goodreads_export.csv"]');
				}
				if (!link) {
					link = document.querySelector('a[href*="goodreads_library_export.csv"]');
				}

				// Return the full absolute URL
				return link ? link.href : "";
			})()
		`, &exportLink)); err != nil {
			return "", fmt.Errorf("failed to check Goodreads export link: %w", err)
		}

		if exportLink != "" {
			slog.Info("Found Goodreads export link", "link", exportLink, "waited", time.Since(start))
			return exportLink, nil
		}

		if tries%5 == 0 {
			slog.Info("Waiting for Goodreads export link", "elapsed", time.Since(start))

			// Debug: check specifically for the export file div
			var exportHTML string
			_ = runner.Run(ctx, chromedp.Evaluate(`
				(() => {
					const div = document.getElementById('exportFile');
					return div ? div.innerHTML : 'exportFile div not found';
				})()
			`, &exportHTML))
			slog.Info("Export file div content", "html", exportHTML)
		}
		tries++

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timed out waiting for Goodreads export link: %w", ctx.Err())
		case <-ticker.C:
		}

		// Only reload if we've tried several times without success
		// Don't reload immediately as it might clear the just-generated link
		if tries > 3 {
			slog.Debug("Reloading page to check for export link", "tries", tries)
			if err := runner.Run(ctx, chromedp.Reload()); err != nil {
				slog.Debug("Failed to refresh Goodreads export page", "error", err)
			}
		}
	}
}

func waitForDownload(ctx context.Context, runner automation.CDPRunner, downloadDir string) (string, error) {
	start := time.Now()

	path, err := automation.PollWithTimeout(
		ctx,
		downloadPollInterval,
		3*time.Minute,
		"Goodreads export download",
		func() (string, bool, error) {
			return findDownloadedCSV(downloadDir)
		},
	)

	if err != nil {
		return "", err
	}

	slog.Info("Goodreads export download completed", "path", path, "waited", time.Since(start))
	return path, nil
}

func findDownloadedCSV(downloadDir string) (string, bool, error) {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return "", false, fmt.Errorf("failed to read download directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Match both goodreads_library_export.csv and goodreads_export.csv
		if (strings.Contains(name, "goodreads_export.csv") || strings.Contains(name, exportFileName)) &&
			strings.HasSuffix(name, ".csv") &&
			!strings.HasSuffix(name, ".crdownload") {
			slog.Debug("Found downloaded CSV file", "name", name)
			return filepath.Join(downloadDir, name), true, nil
		}
	}

	return "", false, nil
}

func moveDownloadedCSV(downloadedPath, requestedDir string) (string, error) {
	targetDir := requestedDir
	if targetDir == "" {
		targetDir = "exports"
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, exportFileName)

	if downloadedPath == targetPath {
		return targetPath, nil
	}

	if err := os.Rename(downloadedPath, targetPath); err != nil {
		if copyErr := automation.CopyFile(downloadedPath, targetPath); copyErr != nil {
			return "", fmt.Errorf("failed to move downloaded export: %v (copy error: %w)", err, copyErr)
		}
		_ = os.Remove(downloadedPath)
	}

	return targetPath, nil
}
