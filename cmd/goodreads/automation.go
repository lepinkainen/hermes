package goodreads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

const (
	exportPollInterval   = 3 * time.Second
	downloadPollInterval = 2 * time.Second
	exportFileName       = "goodreads_library_export.csv"
)

var (
	chromedpExecAllocator = chromedp.NewExecAllocator
	chromedpContext       = chromedp.NewContext
	chromedpRunner        = chromedp.Run
)

type AutomationOptions struct {
	Email       string
	Password    string
	DownloadDir string
	Headless    bool
	Timeout     time.Duration
}

var downloadGoodreadsCSV = AutomateGoodreadsExport

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

	downloadDir, cleanup, err := prepareDownloadDir(opts.DownloadDir)
	if err != nil {
		return "", err
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()
	slog.Info("Prepared Goodreads download directory", "path", downloadDir, "headless", opts.Headless)

	allocCtx, cancelAllocator := chromedpExecAllocator(ctx, buildExecAllocatorOptions(opts)...)
	defer cancelAllocator()

	browserCtx, cancelBrowser := chromedpContext(allocCtx)
	defer cancelBrowser()

	if err := configureDownloadDirectory(browserCtx, downloadDir); err != nil {
		return "", err
	}

	if err := performGoodreadsLogin(browserCtx, opts); err != nil {
		return "", err
	}

	if err := triggerGoodreadsExport(browserCtx); err != nil {
		return "", err
	}

	exportLink, err := waitForExportLink(browserCtx)
	if err != nil {
		return "", err
	}

	slog.Info("Initiating download of export file")

	// Try clicking the link instead of navigating directly
	// This is more natural and might avoid HTTP response issues
	clickSelector := fmt.Sprintf(`//a[@href="%s"]`, strings.TrimPrefix(exportLink, "https://www.goodreads.com"))
	if err := chromedpRunner(browserCtx, chromedp.Click(clickSelector, chromedp.BySearch)); err != nil {
		// Fallback to navigation if click fails
		slog.Info("Click failed, trying direct navigation", "error", err)
		if err := chromedpRunner(browserCtx, chromedp.Navigate(exportLink)); err != nil {
			return "", fmt.Errorf("failed to start Goodreads export download: %w", err)
		}
	}

	csvPath, err := waitForDownload(browserCtx, downloadDir)
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

func buildExecAllocatorOptions(opts AutomationOptions) []chromedp.ExecAllocatorOption {
	return []chromedp.ExecAllocatorOption{
		chromedp.NoDefaultBrowserCheck,
		chromedp.NoFirstRun,
		chromedp.Flag("headless", opts.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("no-default-browser-check", true),
	}
}

func prepareDownloadDir(path string) (string, func(), error) {
	if path != "" {
		if err := os.MkdirAll(path, 0755); err != nil {
			return "", nil, fmt.Errorf("failed to create download directory: %w", err)
		}
		return filepath.Clean(path), nil, nil
	}

	tmpDir, err := os.MkdirTemp("", "hermes-goodreads-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary download directory: %w", err)
	}

	return tmpDir, func() { _ = os.RemoveAll(tmpDir) }, nil
}

func configureDownloadDirectory(ctx context.Context, downloadDir string) error {
	action := browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
		WithDownloadPath(downloadDir).
		WithEventsEnabled(true)
	slog.Debug("Configuring download directory", "path", downloadDir)
	if err := chromedpRunner(ctx, action); err != nil {
		return fmt.Errorf("failed to configure download directory: %w", err)
	}
	return nil
}

func performGoodreadsLogin(ctx context.Context, opts AutomationOptions) error {
	slog.Info("Logging in to Goodreads", "email", opts.Email)

	if err := chromedpRunner(ctx, chromedp.Navigate("https://www.goodreads.com/user/sign_in")); err != nil {
		return fmt.Errorf("failed to open login page: %w", err)
	}

	if err := chromedpRunner(ctx, chromedp.WaitVisible(`//button[contains(., "Sign in with email")]`, chromedp.BySearch)); err != nil {
		return fmt.Errorf("email login button not visible: %w", err)
	}
	if err := chromedpRunner(ctx, chromedp.Click(`//button[contains(., "Sign in with email")]`, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to click email login button: %w", err)
	}

	// Wait for navigation to complete (may redirect to Amazon sign-in page)
	slog.Debug("Waiting for page to load after clicking email sign-in button")
	var currentURL string
	if err := chromedpRunner(ctx, chromedp.Sleep(2*time.Second)); err != nil {
		return fmt.Errorf("failed to wait after clicking email button: %w", err)
	}
	if err := chromedpRunner(ctx, chromedp.Location(&currentURL)); err == nil {
		slog.Info("Current page after clicking email sign-in", "url", currentURL)
	}

	emailSelector, err := waitForSelector(ctx, []string{
		`//input[@type="email" or @name="email" or @id="ap_email"]`,
		`//input[@name="user[email]"]`,
	}, "email field")
	if err != nil {
		return err
	}
	if err := chromedpRunner(ctx, chromedp.SendKeys(emailSelector, opts.Email, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to fill email: %w", err)
	}

	passwordSelector, err := waitForSelector(ctx, []string{
		`//input[@type="password" or @name="password" or @id="ap_password"]`,
		`//input[@name="user[password]"]`,
	}, "password field")
	if err != nil {
		return err
	}
	if err := chromedpRunner(ctx, chromedp.SendKeys(passwordSelector, opts.Password, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to fill password: %w", err)
	}

	submitSelector, err := waitForSelector(ctx, []string{
		`//button[@type="submit" or contains(., "Sign in")]`,
		`//input[@type="submit" and (@name="signIn" or @id="signInSubmit")]`,
	}, "sign-in submit button")
	if err != nil {
		return err
	}
	if err := chromedpRunner(ctx, chromedp.Click(submitSelector, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to submit login form: %w", err)
	}

	if err := waitForLoginSuccess(ctx); err != nil {
		return err
	}

	slog.Info("Goodreads login completed")
	return nil
}

func waitForLoginSuccess(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		var hasProfile bool
		if err := chromedpRunner(ctx, chromedp.Evaluate(`!!document.querySelector('.siteHeader__topLevelItem--profile')`, &hasProfile)); err == nil && hasProfile {
			return nil
		}

		var location string
		if err := chromedpRunner(ctx, chromedp.Location(&location)); err == nil {
			if strings.Contains(location, "goodreads.com") && !strings.Contains(location, "/sign_in") && !strings.Contains(location, "ap/signin") {
				return nil
			}
		}

		slog.Info("Waiting for Goodreads login to complete", "elapsed", time.Since(start))

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for Goodreads login to complete: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForSelector(ctx context.Context, selectors []string, description string) (string, error) {
	// Build a combined XPath expression that matches any of the selectors
	// This allows chromedp to wait for any of them to appear
	var combinedSelector string
	if len(selectors) == 1 {
		combinedSelector = selectors[0]
	} else {
		// Combine selectors with OR: (selector1) | (selector2) | (selector3)
		combinedSelector = "(" + strings.Join(selectors, ") | (") + ")"
	}

	slog.Debug("Waiting for selector", "desc", description, "combined", combinedSelector)

	// Wait for the combined selector (any match wins)
	if err := chromedpRunner(ctx, chromedp.WaitVisible(combinedSelector, chromedp.BySearch)); err != nil {
		// Dump page content for debugging
		var htmlContent string
		var currentURL string
		_ = chromedpRunner(ctx, chromedp.Location(&currentURL))
		_ = chromedpRunner(ctx, chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery))
		slog.Debug("Selector not found", "desc", description, "url", currentURL, "html_length", len(htmlContent))
		return "", fmt.Errorf("could not find %s on page: %w", description, err)
	}

	// Figure out which selector actually matched by checking each one
	for _, sel := range selectors {
		// Try to evaluate if this specific selector exists
		var exists bool
		checkScript := fmt.Sprintf(`!!document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue`, sel)
		if err := chromedpRunner(ctx, chromedp.Evaluate(checkScript, &exists)); err == nil && exists {
			slog.Info("Found selector", "desc", description, "selector", sel)
			return sel, nil
		}
	}

	// Fallback to combined selector if we can't determine which one matched
	return combinedSelector, nil
}

func triggerGoodreadsExport(ctx context.Context) error {
	slog.Info("Navigating to Goodreads export page")

	if err := chromedpRunner(ctx, chromedp.Navigate("https://www.goodreads.com/review/import")); err != nil {
		return fmt.Errorf("failed to navigate to import page: %w", err)
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Check if an export link already exists (from a previous export)
	var existingLink string
	_ = chromedpRunner(ctx, chromedp.Evaluate(`
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
	exportButtonSelector, err := waitForSelector(ctx, []string{
		`//button[contains(., 'Export Library')]`,
		`//input[@value='Export Library']`,
		`//input[@type='submit' and contains(@value, 'Export')]`,
	}, "export library button")
	if err != nil {
		return err
	}

	slog.Info("Found export button", "selector", exportButtonSelector)

	if err := chromedpRunner(ctx, chromedp.Click(exportButtonSelector, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	slog.Info("Clicked export button")
	return nil
}

func waitForExportLink(ctx context.Context) (string, error) {
	start := time.Now()
	ticker := time.NewTicker(exportPollInterval)
	defer ticker.Stop()

	tries := 0
	for {
		var exportLink string
		if err := chromedpRunner(ctx, chromedp.Evaluate(`
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
			_ = chromedpRunner(ctx, chromedp.Evaluate(`
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
			if err := chromedpRunner(ctx, chromedp.Reload()); err != nil {
				slog.Debug("Failed to refresh Goodreads export page", "error", err)
			}
		}
	}
}

func waitForDownload(ctx context.Context, downloadDir string) (string, error) {
	start := time.Now()
	ticker := time.NewTicker(downloadPollInterval)
	defer ticker.Stop()

	tries := 0
	for {
		path, found, err := findDownloadedCSV(downloadDir)
		if err != nil {
			return "", err
		}

		if found {
			slog.Info("Goodreads export download completed", "path", path, "waited", time.Since(start))
			return path, nil
		}

		if tries%5 == 0 {
			slog.Info("Waiting for Goodreads export download", "elapsed", time.Since(start))
		}
		tries++

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timed out waiting for Goodreads export download: %w", ctx.Err())
		case <-ticker.C:
		}
	}
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
		if copyErr := copyFile(downloadedPath, targetPath); copyErr != nil {
			return "", fmt.Errorf("failed to move downloaded export: %v (copy error: %w)", err, copyErr)
		}
		_ = os.Remove(downloadedPath)
	}

	return targetPath, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
