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

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/lepinkainen/hermes/internal/automation"
)

const (
	downloadPollInterval = 2 * time.Second

	letterboxdSignInURL     = "https://letterboxd.com/sign-in/"
	letterboxdDataExportURL = "https://letterboxd.com/data/export/"
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

	downloadDir, cleanup, err := automation.PrepareDownloadDir(opts.DownloadDir, "hermes-letterboxd-*")
	if err != nil {
		return "", err
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()
	slog.Info("Prepared Letterboxd download directory", "path", downloadDir, "headless", opts.Headless)

	session, err := automation.NewBrowser(automation.AutomationOptions{Headless: opts.Headless})
	if err != nil {
		return "", err
	}
	defer session.Close()

	page, err := automation.NavigatePage(session.Browser, letterboxdSignInURL)
	if err != nil {
		return "", fmt.Errorf("failed to open Letterboxd login page: %w", err)
	}

	if err := automation.ConfigurePageDownloadDirectory(page, downloadDir); err != nil {
		return "", err
	}

	if err := performLetterboxdLogin(ctx, page, opts); err != nil {
		return "", err
	}

	if err := triggerLetterboxdExport(page); err != nil {
		return "", err
	}

	zipPath, err := waitForDownload(ctx, downloadDir)
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

func performLetterboxdLogin(ctx context.Context, page *rod.Page, opts AutomationOptions) error {
	slog.Info("Logging in to Letterboxd", "username", opts.Username)

	// Wait for username field to be present
	_, usernameEl, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//input[@id="field-username"]`,
		`//input[@type="text" and @autocomplete="username"]`,
		`//input[@name="username"]`,
	}, "username field", 10*time.Second)
	if err != nil {
		return err
	}

	// Remove 'disabled' attribute via JS if present
	slog.Debug("Removing disabled attribute from login fields")
	_, evalErr := page.Eval(`() => {
		const username = document.querySelector('#field-username');
		const password = document.querySelector('#field-password');
		if (username) username.removeAttribute('disabled');
		if (password) password.removeAttribute('disabled');
	}`)
	if evalErr != nil {
		slog.Debug("Failed to remove disabled attribute", "error", evalErr)
	}

	// Small wait to ensure fields are ready
	time.Sleep(500 * time.Millisecond)

	// Fill username
	if err := usernameEl.Input(opts.Username); err != nil {
		return fmt.Errorf("failed to enter username: %w", err)
	}

	// Fill password
	_, passwordEl, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//input[@id="field-password"]`,
		`//input[@type="password"]`,
		`//input[@name="password"]`,
	}, "password field", 10*time.Second)
	if err != nil {
		return err
	}

	if err := passwordEl.Input(opts.Password); err != nil {
		return fmt.Errorf("failed to enter password: %w", err)
	}

	// Submit form
	_, buttonEl, err := automation.WaitForSelectorWithContext(ctx, page, []string{
		`//form[contains(@class, 'js-sign-in-form')]//button[@type='submit']`,
		`//button[@type='submit']//span[contains(text(), 'Sign')]`,
		`//button[@type='submit' and contains(@class, 'standalone-flow-button')]`,
		`.standalone-flow-form button[type=submit]`,
	}, "sign in button", 10*time.Second)
	if err != nil {
		return err
	}

	slog.Info("Clicking sign in button")
	if err := buttonEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click sign in: %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := waitForLoginSuccess(ctx, page); err != nil {
		return err
	}

	slog.Info("Letterboxd login completed")
	return nil
}

func waitForLoginSuccess(ctx context.Context, page *rod.Page) error {
	start := time.Now()
	slog.Info("Waiting for Letterboxd login to complete")

	err := automation.WaitForURLChange(
		ctx,
		func() (string, error) {
			currentURL, err := automation.GetPageURL(page)
			if err != nil {
				return "", err
			}

			// Check for error messages on the page
			result, evalErr := page.Eval(`() => {
				const errorMsg = document.querySelector('.error, .errormessage, .form-error');
				return errorMsg !== null;
			}`)

			if evalErr == nil && result.Value.Bool() {
				errorResult, _ := page.Eval(`() => {
					const errorMsg = document.querySelector('.error, .errormessage, .form-error');
					return errorMsg ? errorMsg.textContent.trim() : '';
				}`)
				if errorResult != nil {
					return "", fmt.Errorf("login error: %s", errorResult.Value.Str())
				}
			}

			return currentURL, nil
		},
		[]string{"/sign-in/", "/user/login.do"},
		30*time.Second,
	)

	if err != nil {
		return err
	}

	slog.Info("Successfully logged in to Letterboxd", "elapsed", time.Since(start))
	return nil
}

func triggerLetterboxdExport(page *rod.Page) error {
	slog.Info("Navigating directly to Letterboxd export URL to trigger download")

	// Navigate to export URL - this may abort because it triggers a download
	err := page.Navigate(letterboxdDataExportURL)
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

	path, err := automation.PollWithTimeout(
		ctx,
		downloadPollInterval,
		3*time.Minute,
		"Letterboxd export download",
		func() (string, bool, error) {
			path, err := findDownloadedZip(downloadDir, start)
			if err != nil {
				// Not found yet, continue polling
				return "", false, nil
			}
			return path, true, nil
		},
	)

	if err != nil {
		return "", err
	}

	slog.Info("Letterboxd export download completed", "path", path, "waited", time.Since(start))
	return path, nil
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
			if copyErr := automation.CopyFile(csvPath, targetCSVPath); copyErr != nil {
				return "", fmt.Errorf("failed to move CSV: %v (copy error: %w)", err, copyErr)
			}
			_ = os.Remove(csvPath)
		}
	}

	// Also copy the ZIP file to exports directory (per user decision to keep it)
	zipFilename := filepath.Base(zipPath)
	targetZipPath := filepath.Join(targetDir, zipFilename)

	if zipPath != targetZipPath {
		if err := automation.CopyFile(zipPath, targetZipPath); err != nil {
			slog.Warn("Failed to copy ZIP to exports", "error", err)
		} else {
			slog.Info("Copied ZIP to exports", "path", targetZipPath)
		}
	}

	return targetCSVPath, nil
}
