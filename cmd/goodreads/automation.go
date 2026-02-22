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

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
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

func AutomateGoodreadsExport(parentCtx context.Context, opts AutomationOptions) (string, error) {
	if opts.Email == "" || opts.Password == "" {
		return "", errors.New("goodreads automation requires both email and password")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultAutomationTimeout
	}

	ctx, cancel := context.WithTimeout(parentCtx, timeout)
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

	session, err := automation.NewBrowser(automation.AutomationOptions{Headless: opts.Headless})
	if err != nil {
		return "", err
	}
	defer session.Close()

	page, err := automation.NavigatePage(session.Browser, "https://www.goodreads.com/user/sign_in")
	if err != nil {
		return "", fmt.Errorf("failed to open login page: %w", err)
	}

	if err := automation.ConfigurePageDownloadDirectory(page, downloadDir); err != nil {
		return "", err
	}

	if err := performGoodreadsLogin(ctx, page, opts); err != nil {
		return "", err
	}

	if err := triggerGoodreadsExport(ctx, page); err != nil {
		return "", err
	}

	exportLink, err := waitForExportLink(ctx, page)
	if err != nil {
		return "", err
	}

	slog.Info("Initiating download of export file")

	// Try clicking the link instead of navigating directly
	clickSelector := fmt.Sprintf(`//a[@href="%s"]`, strings.TrimPrefix(exportLink, "https://www.goodreads.com"))
	els, err := page.ElementsX(clickSelector)
	if err == nil && len(els) > 0 {
		if clickErr := els[0].Click(proto.InputMouseButtonLeft, 1); clickErr != nil {
			slog.Info("Click failed, trying direct navigation", "error", clickErr)
			if navErr := page.Navigate(exportLink); navErr != nil {
				return "", fmt.Errorf("failed to start Goodreads export download: %w", navErr)
			}
		}
	} else {
		// Fallback to navigation if selector not found
		slog.Info("Export link element not found, trying direct navigation")
		if navErr := page.Navigate(exportLink); navErr != nil {
			return "", fmt.Errorf("failed to start Goodreads export download: %w", navErr)
		}
	}

	csvPath, err := waitForDownload(ctx, downloadDir)
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

func performGoodreadsLogin(ctx context.Context, page *rod.Page, opts AutomationOptions) error {
	slog.Info("Logging in to Goodreads", "email", opts.Email)

	// Wait for email login button
	_, emailBtn, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//button[contains(., "Sign in with email")]`,
	}, "email login button", 10*time.Second)
	if err != nil {
		return fmt.Errorf("email login button not visible: %w", err)
	}
	if err := emailBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click email login button: %w", err)
	}

	// Wait for navigation to complete (may redirect to Amazon sign-in page)
	slog.Debug("Waiting for page to load after clicking email sign-in button")
	time.Sleep(2 * time.Second)

	url, _ := automation.GetPageURL(page)
	slog.Info("Current page after clicking email sign-in", "url", url)

	_, emailEl, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//input[@type="email" or @name="email" or @id="ap_email"]`,
		`//input[@name="user[email]"]`,
	}, "email field", 10*time.Second)
	if err != nil {
		return err
	}
	if err := emailEl.Input(opts.Email); err != nil {
		return fmt.Errorf("failed to fill email: %w", err)
	}

	_, passwordEl, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//input[@type="password" or @name="password" or @id="ap_password"]`,
		`//input[@name="user[password]"]`,
	}, "password field", 10*time.Second)
	if err != nil {
		return err
	}
	if err := passwordEl.Input(opts.Password); err != nil {
		return fmt.Errorf("failed to fill password: %w", err)
	}

	_, submitEl, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//button[@type="submit" or contains(., "Sign in")]`,
		`//input[@type="submit" and (@name="signIn" or @id="signInSubmit")]`,
	}, "sign-in submit button", 10*time.Second)
	if err != nil {
		return err
	}
	if err := submitEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to submit login form: %w", err)
	}

	if err := waitForLoginSuccess(ctx, page); err != nil {
		return err
	}

	slog.Info("Goodreads login completed")
	return nil
}

func waitForLoginSuccess(ctx context.Context, page *rod.Page) error {
	start := time.Now()
	slog.Info("Waiting for Goodreads login to complete")

	err := automation.WaitForURLChange(
		ctx,
		func() (string, error) {
			return automation.GetPageURL(page)
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

func triggerGoodreadsExport(ctx context.Context, page *rod.Page) error {
	slog.Info("Navigating to Goodreads export page")

	if err := page.Navigate("https://www.goodreads.com/review/import"); err != nil {
		return fmt.Errorf("failed to navigate to import page: %w", err)
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Check if an export link already exists (from a previous export)
	existingLink, err := page.Eval(`() => {
		const fileList = document.getElementById('exportFile');
		if (fileList) {
			const link = fileList.querySelector('a');
			if (link && link.href) {
				return link.href;
			}
		}
		return "";
	}`)
	if err == nil && existingLink.Value.Str() != "" {
		slog.Info("Found existing export link, skipping export button click", "link", existingLink.Value.Str())
		return nil
	}

	// No existing link, need to trigger export
	slog.Info("No existing export link found, triggering new export")

	// Try multiple selectors for the export button
	_, exportBtn, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//button[contains(., 'Export Library')]`,
		`//input[@value='Export Library']`,
		`//input[@type='submit' and contains(@value, 'Export')]`,
	}, "export library button", 10*time.Second)
	if err != nil {
		return err
	}

	slog.Info("Found export button, clicking")

	if err := exportBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	slog.Info("Clicked export button")
	return nil
}

func waitForExportLink(ctx context.Context, page *rod.Page) (string, error) {
	start := time.Now()
	ticker := time.NewTicker(exportPollInterval)
	defer ticker.Stop()

	tries := 0
	for {
		result, err := page.Eval(`() => {
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
		}`)
		if err != nil {
			return "", fmt.Errorf("failed to check Goodreads export link: %w", err)
		}

		exportLink := result.Value.Str()
		if exportLink != "" {
			slog.Info("Found Goodreads export link", "link", exportLink, "waited", time.Since(start))
			return exportLink, nil
		}

		if tries%5 == 0 {
			slog.Info("Waiting for Goodreads export link", "elapsed", time.Since(start))

			// Debug: check specifically for the export file div
			divResult, divErr := page.Eval(`() => {
				const div = document.getElementById('exportFile');
				return div ? div.innerHTML : 'exportFile div not found';
			}`)
			if divErr == nil {
				slog.Info("Export file div content", "html", divResult.Value.Str())
			}
		}
		tries++

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timed out waiting for Goodreads export link: %w", ctx.Err())
		case <-ticker.C:
		}

		// Only reload if we've tried several times without success
		if tries > 3 {
			slog.Debug("Reloading page to check for export link", "tries", tries)
			if reloadErr := page.Reload(); reloadErr != nil {
				slog.Debug("Failed to refresh Goodreads export page", "error", reloadErr)
			}
		}
	}
}

func waitForDownload(ctx context.Context, downloadDir string) (string, error) {
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
