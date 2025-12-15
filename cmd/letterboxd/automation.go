package letterboxd

import (
	"archive/zip"
	"context"
	"encoding/csv"
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

	letterboxdSignInURL     = "https://letterboxd.com/sign-in/"
	letterboxdDataExportURL = "https://letterboxd.com/data/export/"
)

var (
	chromedpExecAllocator = chromedp.NewExecAllocator
	chromedpContext       = chromedp.NewContext
	chromedpRunner        = chromedp.Run
)

// AutomationOptions holds configuration for Letterboxd automation
type AutomationOptions struct {
	Username    string
	Password    string
	DownloadDir string
	Headless    bool
	Timeout     time.Duration
}

var downloadLetterboxdZip = AutomateLetterboxdExport

// AutomateLetterboxdExport orchestrates the full automation workflow
func AutomateLetterboxdExport(parentCtx context.Context, opts AutomationOptions) (string, error) {
	if opts.Username == "" || opts.Password == "" {
		return "", errors.New("letterboxd automation requires both username and password")
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
	slog.Info("Prepared Letterboxd download directory", "path", downloadDir, "headless", opts.Headless)

	allocCtx, cancelAllocator := chromedpExecAllocator(ctx, buildExecAllocatorOptions(opts)...)
	defer cancelAllocator()

	browserCtx, cancelBrowser := chromedpContext(allocCtx)
	defer cancelBrowser()

	if err := configureDownloadDirectory(browserCtx, downloadDir); err != nil {
		return "", err
	}

	if err := performLetterboxdLogin(browserCtx, opts); err != nil {
		return "", err
	}

	if err := triggerLetterboxdExport(browserCtx); err != nil {
		return "", err
	}

	zipPath, err := waitForDownload(browserCtx, downloadDir)
	if err != nil {
		return "", err
	}

	// Extract CSVs from the ZIP
	watchedPath, ratingsPath, err := extractLetterboxdCSVs(zipPath, downloadDir)
	if err != nil {
		return "", err
	}

	// Merge watched and ratings CSVs
	mergedPath := filepath.Join(downloadDir, "letterboxd_merged.csv")
	if err := mergeWatchedAndRatings(watchedPath, ratingsPath, mergedPath); err != nil {
		return "", err
	}

	// Move merged CSV to final destination
	finalPath, err := moveDownloadedCSV(mergedPath, zipPath, opts.DownloadDir)
	if err != nil {
		return "", err
	}

	slog.Info("Letterboxd export completed", "csv_path", finalPath)
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

	tmpDir, err := os.MkdirTemp("", "hermes-letterboxd-*")
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

func performLetterboxdLogin(ctx context.Context, opts AutomationOptions) error {
	slog.Info("Logging in to Letterboxd", "username", opts.Username)

	if err := chromedpRunner(ctx, chromedp.Navigate(letterboxdSignInURL)); err != nil {
		return fmt.Errorf("failed to open Letterboxd login page: %w", err)
	}

	// Wait for username field to be present
	usernameSelector, err := waitForSelector(ctx, []string{
		`//input[@id="field-username"]`,
		`//input[@type="text" and @autocomplete="username"]`,
		`//input[@name="username"]`,
	}, "username field")
	if err != nil {
		return err
	}

	// Remove 'disabled' attribute via JS if present
	slog.Debug("Removing disabled attribute from login fields")
	if err := chromedpRunner(ctx, chromedp.Evaluate(`
		(function() {
			const username = document.querySelector('#field-username');
			const password = document.querySelector('#field-password');
			if (username) username.removeAttribute('disabled');
			if (password) password.removeAttribute('disabled');
		})()
	`, nil)); err != nil {
		slog.Debug("Failed to remove disabled attribute", "error", err)
	}

	// Small wait to ensure fields are ready
	_ = chromedpRunner(ctx, chromedp.Sleep(500*time.Millisecond))

	// Fill username
	if err := chromedpRunner(ctx, chromedp.SendKeys(usernameSelector, opts.Username, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to enter username: %w", err)
	}

	// Fill password
	passwordSelector, err := waitForSelector(ctx, []string{
		`//input[@id="field-password"]`,
		`//input[@type="password"]`,
		`//input[@name="password"]`,
	}, "password field")
	if err != nil {
		return err
	}

	if err := chromedpRunner(ctx, chromedp.SendKeys(passwordSelector, opts.Password, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to enter password: %w", err)
	}

	// Submit form
	buttonSelector, err := waitForSelector(ctx, []string{
		`//form[contains(@class, 'js-sign-in-form')]//button[@type='submit']`,
		`//button[@type='submit']//span[contains(text(), 'Sign')]`,
		`//button[@type='submit' and contains(@class, 'standalone-flow-button')]`,
		`.standalone-flow-form button[type=submit]`,
	}, "sign in button")
	if err != nil {
		return err
	}

	slog.Info("Clicking sign in button", "selector", buttonSelector)
	if err := chromedpRunner(ctx, chromedp.Click(buttonSelector, chromedp.BySearch)); err != nil {
		return fmt.Errorf("failed to click sign in: %w", err)
	}

	_ = chromedpRunner(ctx, chromedp.Sleep(2*time.Second))

	if err := waitForLoginSuccess(ctx); err != nil {
		return err
	}

	slog.Info("Letterboxd login completed")
	return nil
}

func waitForLoginSuccess(ctx context.Context) error {
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		// Check current URL - if we're not on sign-in page, we're logged in
		var currentURL string
		_ = chromedpRunner(ctx, chromedp.Location(&currentURL))

		slog.Debug("Checking login status", "url", currentURL, "elapsed", time.Since(deadline.Add(-timeout)))

		// Check if we've navigated away from sign-in page
		if !strings.Contains(currentURL, "/sign-in/") && !strings.Contains(currentURL, "/user/login.do") {
			slog.Info("Successfully logged in to Letterboxd", "url", currentURL)
			return nil
		}

		// Also check for error messages on the page
		var hasError bool
		_ = chromedpRunner(ctx, chromedp.Evaluate(`
			(function() {
				const errorMsg = document.querySelector('.error, .errormessage, .form-error');
				return errorMsg !== null;
			})()
		`, &hasError))

		if hasError {
			var errorText string
			_ = chromedpRunner(ctx, chromedp.Evaluate(`
				(function() {
					const errorMsg = document.querySelector('.error, .errormessage, .form-error');
					return errorMsg ? errorMsg.textContent.trim() : '';
				})()
			`, &errorText))
			return fmt.Errorf("login error: %s", errorText)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("login canceled: %w", ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return errors.New("timeout waiting for Letterboxd login")
			}
		}
	}
}

func waitForSelector(ctx context.Context, selectors []string, description string) (string, error) {
	slog.Debug("Waiting for selector", "desc", description, "selectors", strings.Join(selectors, " | "))

	timeout := 10 * time.Second
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Check each selector
		for _, sel := range selectors {
			var exists bool

			// For XPath selectors (starting with //)
			if strings.HasPrefix(sel, "//") {
				checkScript := fmt.Sprintf(`!!document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue`, sel)
				if err := chromedpRunner(ctx, chromedp.Evaluate(checkScript, &exists)); err == nil && exists {
					slog.Debug("Found selector", "desc", description, "selector", sel)
					return sel, nil
				}
			} else {
				// For CSS selectors
				checkScript := fmt.Sprintf(`!!document.querySelector(%q)`, sel)
				if err := chromedpRunner(ctx, chromedp.Evaluate(checkScript, &exists)); err == nil && exists {
					slog.Debug("Found selector", "desc", description, "selector", sel)
					return sel, nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("selector wait canceled for %s: %w", description, ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				// Dump page content for debugging
				var htmlContent string
				var currentURL string
				_ = chromedpRunner(ctx, chromedp.Location(&currentURL))
				_ = chromedpRunner(ctx, chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery))
				slog.Debug("Selector timeout", "desc", description, "url", currentURL, "html_length", len(htmlContent))
				return "", fmt.Errorf("timeout waiting for %s", description)
			}
		}
	}
}

func triggerLetterboxdExport(ctx context.Context) error {
	slog.Info("Navigating directly to Letterboxd export URL to trigger download")

	// Navigate to export URL - this may abort because it triggers a download
	err := chromedpRunner(ctx, chromedp.Navigate(letterboxdDataExportURL))
	if err != nil {
		// ERR_ABORTED is expected when the page immediately triggers a download
		if !strings.Contains(err.Error(), "ERR_ABORTED") && !strings.Contains(err.Error(), "net::ERR_ABORTED") {
			return fmt.Errorf("failed to navigate to export URL: %w", err)
		}
		slog.Debug("Navigation aborted (expected - download triggered)", "error", err)
	}

	// Wait for download to start
	time.Sleep(2 * time.Second)

	slog.Info("Export download triggered")
	return nil
}

func waitForDownload(ctx context.Context, downloadDir string) (string, error) {
	start := time.Now()
	ticker := time.NewTicker(downloadPollInterval)
	defer ticker.Stop()

	tries := 0
	for {
		path, err := findDownloadedZip(downloadDir, start)
		if err == nil {
			slog.Info("Letterboxd export download completed", "path", path, "waited", time.Since(start))
			return path, nil
		}

		if tries%5 == 0 {
			slog.Info("Waiting for Letterboxd export download", "elapsed", time.Since(start))
		}
		tries++

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timed out waiting for Letterboxd export download: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func findDownloadedZip(downloadDir string, startTime time.Time) (string, error) {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return "", fmt.Errorf("failed to read download directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Match letterboxd-*.zip but not partial downloads
		if strings.HasPrefix(name, "letterboxd-") &&
			strings.HasSuffix(name, ".zip") &&
			!strings.HasSuffix(name, ".crdownload") {
			// Check file modification time to avoid stale files
			info, err := entry.Info()
			if err != nil {
				slog.Debug("Failed to get file info", "name", name, "error", err)
				continue
			}

			if info.ModTime().After(startTime) {
				slog.Debug("Found downloaded ZIP file", "name", name, "modTime", info.ModTime())
				return filepath.Join(downloadDir, name), nil
			}
			slog.Debug("Skipping stale ZIP file", "name", name, "modTime", info.ModTime(), "startTime", startTime)
		}
	}

	return "", errors.New("ZIP file not found yet")
}

func extractLetterboxdCSVs(zipPath, targetDir string) (watchedPath, ratingsPath string, err error) {
	slog.Info("Extracting CSVs from ZIP", "zip", zipPath)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer func() { _ = reader.Close() }()

	for _, file := range reader.File {
		var targetPath string
		if strings.HasSuffix(file.Name, "watched.csv") {
			targetPath = filepath.Join(targetDir, "watched.csv")
		} else if strings.HasSuffix(file.Name, "ratings.csv") {
			targetPath = filepath.Join(targetDir, "ratings.csv")
		} else {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return "", "", fmt.Errorf("failed to open %s in ZIP: %w", file.Name, err)
		}
		defer func() { _ = rc.Close() }()

		out, err := os.Create(targetPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to create %s: %w", targetPath, err)
		}
		defer func() { _ = out.Close() }()

		if _, err := io.Copy(out, rc); err != nil {
			return "", "", fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}

		if strings.HasSuffix(file.Name, "watched.csv") {
			watchedPath = targetPath
		} else {
			ratingsPath = targetPath
		}
	}

	if watchedPath == "" {
		return "", "", errors.New("watched.csv not found in ZIP")
	}

	if ratingsPath == "" {
		slog.Warn("ratings.csv not found in ZIP, continuing without ratings")
	}

	slog.Info("Extracted CSVs", "watched", watchedPath, "ratings", ratingsPath)
	return watchedPath, ratingsPath, nil
}

func mergeWatchedAndRatings(watchedPath, ratingsPath, outputPath string) error {
	slog.Info("Merging watched and ratings CSVs")

	// Read ratings into map: URI -> Rating
	ratingsMap := make(map[string]string)
	if ratingsPath != "" {
		ratingsFile, err := os.Open(ratingsPath)
		if err != nil {
			// Only log warning if file doesn't exist (not critical)
			if os.IsNotExist(err) {
				slog.Warn("Ratings file not found, continuing without ratings", "path", ratingsPath)
			} else {
				return fmt.Errorf("failed to open ratings.csv: %w", err)
			}
		} else {
			defer func() { _ = ratingsFile.Close() }()
			reader := csv.NewReader(ratingsFile)
			records, err := reader.ReadAll()
			if err != nil {
				return fmt.Errorf("failed to parse ratings.csv: %w", err)
			}

			for i, record := range records {
				if i == 0 {
					continue
				} // Skip header
				if len(record) >= 5 {
					uri := record[3]    // Letterboxd URI
					rating := record[4] // Rating
					ratingsMap[uri] = rating
				}
			}
		}
	}

	// Read watched.csv and add Rating column
	watchedFile, err := os.Open(watchedPath)
	if err != nil {
		return fmt.Errorf("failed to open watched.csv: %w", err)
	}
	defer func() { _ = watchedFile.Close() }()

	reader := csv.NewReader(watchedFile)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read watched.csv: %w", err)
	}

	// Create output with Rating column
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create merged CSV: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	for i, record := range records {
		if i == 0 {
			// Add Rating header
			if err := writer.Write(append(record, "Rating")); err != nil {
				return fmt.Errorf("failed to write CSV header: %w", err)
			}
			continue
		}

		if len(record) >= 4 {
			uri := record[3]
			rating := ratingsMap[uri] // Empty string if not rated
			if err := writer.Write(append(record, rating)); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
	}

	// Check for flush errors
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	slog.Info("Merged CSV created", "path", outputPath, "ratings_added", len(ratingsMap))
	return nil
}

func moveDownloadedCSV(csvPath, zipPath, requestedDir string) (string, error) {
	targetDir := requestedDir
	if targetDir == "" {
		targetDir = "exports"
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	// Move CSV to target directory
	csvFilename := "letterboxd_merged.csv"
	targetCSVPath := filepath.Join(targetDir, csvFilename)

	if csvPath != targetCSVPath {
		if err := os.Rename(csvPath, targetCSVPath); err != nil {
			if copyErr := copyFile(csvPath, targetCSVPath); copyErr != nil {
				return "", fmt.Errorf("failed to move CSV: %v (copy error: %w)", err, copyErr)
			}
			_ = os.Remove(csvPath)
		}
	}

	// Also copy the ZIP file to exports directory (per user decision to keep it)
	zipFilename := filepath.Base(zipPath)
	targetZipPath := filepath.Join(targetDir, zipFilename)

	if zipPath != targetZipPath {
		if err := copyFile(zipPath, targetZipPath); err != nil {
			slog.Warn("Failed to copy ZIP to exports", "error", err)
		} else {
			slog.Info("Copied ZIP to exports", "path", targetZipPath)
		}
	}

	return targetCSVPath, nil
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
