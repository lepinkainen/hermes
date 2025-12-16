package automation

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

type CDPRunner interface {
	NewExecAllocator(ctx context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc)
	NewContext(parent context.Context, opts ...chromedp.ContextOption) (context.Context, context.CancelFunc)
	Run(ctx context.Context, actions ...chromedp.Action) error
}

// DefaultCDPRunner is the default implementation using chromedp functions directly.
type DefaultCDPRunner struct{}

func (d *DefaultCDPRunner) NewExecAllocator(ctx context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc) {
	return chromedp.NewExecAllocator(ctx, opts...)
}

func (d *DefaultCDPRunner) NewContext(parent context.Context, opts ...chromedp.ContextOption) (context.Context, context.CancelFunc) {
	return chromedp.NewContext(parent, opts...)
}

func (d *DefaultCDPRunner) Run(ctx context.Context, actions ...chromedp.Action) error {
	return chromedp.Run(ctx, actions...)
}

// AutomationOptions holds common configuration for browser automation
type AutomationOptions struct {
	Headless bool
}

// NewBrowser creates a new chromedp browser context with the given options and runner.
func NewBrowser(runner CDPRunner, opts AutomationOptions) (context.Context, context.CancelFunc) {
	allocOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoDefaultBrowserCheck,
		chromedp.NoFirstRun,
		chromedp.Flag("headless", opts.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("no-default-browser-check", true),
	}
	allocCtx, cancelAllocator := runner.NewExecAllocator(context.Background(), allocOpts...)
	browserCtx, cancelBrowser := runner.NewContext(allocCtx)

	combinedCancel := func() {
		cancelBrowser()
		cancelAllocator()
	}
	return browserCtx, combinedCancel
}

// ConfigureDownloadDirectory sets the download behavior for the browser
func ConfigureDownloadDirectory(ctx context.Context, runner CDPRunner, downloadDir string) error {
	action := browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
		WithDownloadPath(downloadDir).
		WithEventsEnabled(true)
	slog.Debug("Configuring download directory", "path", downloadDir)
	if err := runner.Run(ctx, action); err != nil {
		return fmt.Errorf("failed to configure download directory: %w", err)
	}
	return nil
}

// WaitForSelector waits for one of the given selectors to become visible on the page
func WaitForSelector(ctx context.Context, runner CDPRunner, selectors []string, description string, timeout time.Duration) (string, error) {
	slog.Debug("Waiting for selector", "desc", description, "selectors", strings.Join(selectors, " | "))

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
				if err := runner.Run(ctx, chromedp.Evaluate(checkScript, &exists)); err == nil && exists {
					slog.Debug("Found selector", "desc", description, "selector", sel)
					return sel, nil
				}
			} else {
				// For CSS selectors
				checkScript := fmt.Sprintf(`!!document.querySelector(%q)`, sel)
				if err := runner.Run(ctx, chromedp.Evaluate(checkScript, &exists)); err == nil && exists {
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
				_ = runner.Run(ctx, chromedp.Location(&currentURL))
				_ = runner.Run(ctx, chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery))
				slog.Debug("Selector timeout", "desc", description, "url", currentURL, "html_length", len(htmlContent))
				return "", fmt.Errorf("timeout waiting for %s", description)
			}
		}
	}
}

// PrepareDownloadDir prepares a directory for browser downloads.
// If downloadDir is empty, creates a temporary directory with the given pattern.
// Returns the directory path and a cleanup function (nil if using a persistent directory).
func PrepareDownloadDir(downloadDir, pattern string) (string, func(), error) {
	if downloadDir == "" {
		tempDir, err := os.MkdirTemp("", pattern)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create temporary download directory: %w", err)
		}
		cleanup := func() {
			_ = os.RemoveAll(tempDir)
		}
		return tempDir, cleanup, nil
	}

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create download directory: %w", err)
	}
	return downloadDir, nil, nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
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

// PollWithTimeout polls a condition function at regular intervals until it succeeds, times out, or context is canceled.
// The checkFunc returns (result, found, error). If found is true, polling stops and result is returned.
// If checkFunc returns an error, polling stops and the error is returned.
// The description is used in timeout error messages.
func PollWithTimeout[T any](ctx context.Context, interval, timeout time.Duration, description string, checkFunc func() (T, bool, error)) (T, error) {
	var zero T
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	tries := 0
	for {
		result, found, err := checkFunc()
		if err != nil {
			return zero, err
		}
		if found {
			return result, nil
		}

		tries++
		if tries%5 == 0 {
			slog.Debug("Polling", "description", description, "tries", tries, "elapsed", time.Since(deadline.Add(-timeout)))
		}

		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("polling canceled for %s: %w", description, ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return zero, fmt.Errorf("timeout waiting for %s", description)
			}
		}
	}
}

// WaitForURLChange polls the current URL until it no longer contains any of the specified patterns.
// This is useful for detecting when authentication completes and redirects away from login pages.
// The getURL function should return the current URL.
// The excludePatterns are URL substrings that indicate we're still on the login page.
func WaitForURLChange(ctx context.Context, runner CDPRunner, getURL func() (string, error), excludePatterns []string, timeout time.Duration) error {
	_, err := PollWithTimeout(
		ctx,
		500*time.Millisecond,
		timeout,
		"URL to change from login page",
		func() (struct{}, bool, error) {
			url, err := getURL()
			if err != nil {
				return struct{}{}, false, err
			}

			// Check if URL still contains any login patterns
			for _, pattern := range excludePatterns {
				if strings.Contains(url, pattern) {
					return struct{}{}, false, nil
				}
			}

			// URL has changed away from login page
			slog.Debug("Login successful - URL changed", "url", url)
			return struct{}{}, true, nil
		},
	)
	return err
}
