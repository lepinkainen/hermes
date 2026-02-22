package automation

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// AutomationOptions holds common configuration for browser automation
type AutomationOptions struct {
	Headless bool
}

// BrowserSession wraps a rod browser and page for automation workflows.
type BrowserSession struct {
	Browser *rod.Browser
	cleanup func()
}

// Close cleans up the browser session.
func (s *BrowserSession) Close() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// NewBrowser creates a new rod browser with the given options.
func NewBrowser(opts AutomationOptions) (*BrowserSession, error) {
	l := launcher.New().
		Headless(opts.Headless).
		Set("disable-gpu").
		Set("disable-sync").
		Set("mute-audio").
		Set("disable-default-apps").
		Set("no-default-browser-check")

	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		l.Kill()
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	cleanup := func() {
		_ = browser.Close()
		l.Kill()
	}

	return &BrowserSession{
		Browser: browser,
		cleanup: cleanup,
	}, nil
}

// NavigatePage creates a new page and navigates to the given URL.
func NavigatePage(browser *rod.Browser, url string) (*rod.Page, error) {
	page, err := browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load at %s: %w", url, err)
	}
	return page, nil
}

// WaitForSelector waits for one of the given selectors to become visible on the page.
// Supports both CSS selectors and XPath selectors (prefixed with //).
func WaitForSelector(page *rod.Page, selectors []string, description string, timeout time.Duration) (string, *rod.Element, error) {
	slog.Debug("Waiting for selector", "desc", description, "selectors", strings.Join(selectors, " | "))

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		for _, sel := range selectors {
			var el *rod.Element
			var err error

			if strings.HasPrefix(sel, "//") {
				// XPath selector
				els, findErr := page.ElementsX(sel)
				if findErr == nil && len(els) > 0 {
					el = els[0]
					err = nil
				}
			} else {
				// CSS selector
				has, _, findErr := page.Has(sel)
				if findErr == nil && has {
					el, err = page.Element(sel)
				}
			}

			if err == nil && el != nil {
				slog.Debug("Found selector", "desc", description, "selector", sel)
				return sel, el, nil
			}
		}

		if time.Now().After(deadline) {
			url := page.MustInfo().URL
			slog.Debug("Selector timeout", "desc", description, "url", url)
			return "", nil, fmt.Errorf("timeout waiting for %s", description)
		}

		<-ticker.C
	}
}

// WaitForSelectorWithContext is like WaitForSelector but also respects context cancellation.
func WaitForSelectorWithContext(ctx context.Context, page *rod.Page, selectors []string, description string, timeout time.Duration) (string, *rod.Element, error) {
	slog.Debug("Waiting for selector", "desc", description, "selectors", strings.Join(selectors, " | "))

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		for _, sel := range selectors {
			var el *rod.Element
			var err error

			if strings.HasPrefix(sel, "//") {
				els, findErr := page.ElementsX(sel)
				if findErr == nil && len(els) > 0 {
					el = els[0]
					err = nil
				}
			} else {
				has, _, findErr := page.Has(sel)
				if findErr == nil && has {
					el, err = page.Element(sel)
				}
			}

			if err == nil && el != nil {
				slog.Debug("Found selector", "desc", description, "selector", sel)
				return sel, el, nil
			}
		}

		if time.Now().After(deadline) {
			url := page.MustInfo().URL
			slog.Debug("Selector timeout", "desc", description, "url", url)
			return "", nil, fmt.Errorf("timeout waiting for %s", description)
		}

		select {
		case <-ctx.Done():
			return "", nil, fmt.Errorf("selector wait canceled for %s: %w", description, ctx.Err())
		case <-ticker.C:
			continue
		}
	}
}

// ConfigureDownloadDirectory sets the download behavior for the browser.
func ConfigureDownloadDirectory(browser *rod.Browser, downloadDir string) error {
	slog.Debug("Configuring download directory", "path", downloadDir)

	// Get the first page/target to set download behavior on
	pages, err := browser.Pages()
	if err != nil {
		return fmt.Errorf("failed to get browser pages: %w", err)
	}

	for _, page := range pages {
		err := proto.BrowserSetDownloadBehavior{
			Behavior:      proto.BrowserSetDownloadBehaviorBehaviorAllow,
			DownloadPath:  downloadDir,
			EventsEnabled: true,
		}.Call(page)
		if err != nil {
			return fmt.Errorf("failed to configure download directory: %w", err)
		}
	}

	return nil
}

// ConfigurePageDownloadDirectory sets the download behavior for a specific page.
func ConfigurePageDownloadDirectory(page *rod.Page, downloadDir string) error {
	slog.Debug("Configuring download directory for page", "path", downloadDir)

	err := proto.BrowserSetDownloadBehavior{
		Behavior:      proto.BrowserSetDownloadBehaviorBehaviorAllow,
		DownloadPath:  downloadDir,
		EventsEnabled: true,
	}.Call(page)
	if err != nil {
		return fmt.Errorf("failed to configure download directory: %w", err)
	}

	return nil
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
func WaitForURLChange(ctx context.Context, getURL func() (string, error), excludePatterns []string, timeout time.Duration) error {
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

// GetPageURL returns the current URL of a rod page.
func GetPageURL(page *rod.Page) (string, error) {
	info, err := page.Info()
	if err != nil {
		return "", err
	}
	return info.URL, nil
}
