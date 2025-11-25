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

	if err := chromedpRunner(browserCtx, chromedp.Navigate(exportLink)); err != nil {
		return "", fmt.Errorf("failed to start Goodreads export download: %w", err)
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

	tasks := chromedp.Tasks{
		chromedp.Navigate("https://www.goodreads.com/user/sign_in"),
		// Expose the email/password form (hidden behind the Amazon SSO options)
		chromedp.WaitVisible(`//button[contains(., "Sign in with email")]`, chromedp.BySearch),
		chromedp.Click(`//button[contains(., "Sign in with email")]`, chromedp.BySearch),
		// Form lives on a redirected sign-in page (selectors differ from the initial modal)
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, opts.Email, chromedp.ByQuery),
		chromedp.WaitVisible(`input[name="password"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, opts.Password, chromedp.ByQuery),
		chromedp.Click(`//button[@type="submit" or contains(., "Sign in")] | //input[@type="submit" and (@name="signIn" or @id="signInSubmit")]`, chromedp.BySearch),
		chromedp.WaitVisible(`.siteHeader__topLevelItem--profile`, chromedp.ByQuery),
	}

	if err := chromedpRunner(ctx, tasks...); err != nil {
		return fmt.Errorf("failed to log in to Goodreads: %w", err)
	}

	slog.Info("Goodreads login completed")
	return nil
}

func triggerGoodreadsExport(ctx context.Context) error {
	slog.Info("Requesting Goodreads export")

	tasks := chromedp.Tasks{
		chromedp.Navigate("https://www.goodreads.com/review/import"),
		chromedp.WaitVisible(`//input[@value='Export Library']`, chromedp.BySearch),
		chromedp.Click(`//input[@value='Export Library']`, chromedp.BySearch),
	}

	if err := chromedpRunner(ctx, tasks...); err != nil {
		return fmt.Errorf("failed to request Goodreads export: %w", err)
	}

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
				const link = document.querySelector('a[href*="goodreads_library_export.csv"]');
				return link ? link.href : "";
			})()
		`, &exportLink)); err != nil {
			return "", fmt.Errorf("failed to check Goodreads export link: %w", err)
		}

		if exportLink != "" {
			slog.Info("Found Goodreads export link", "waited", time.Since(start))
			return exportLink, nil
		}

		if tries%5 == 0 {
			slog.Info("Waiting for Goodreads export link", "elapsed", time.Since(start))
		}
		tries++

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timed out waiting for Goodreads export link: %w", ctx.Err())
		case <-ticker.C:
		}

		if err := chromedpRunner(ctx, chromedp.Reload()); err != nil {
			slog.Debug("Failed to refresh Goodreads export page", "error", err)
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
		if strings.Contains(name, exportFileName) && strings.HasSuffix(name, ".csv") && !strings.HasSuffix(name, ".crdownload") {
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
