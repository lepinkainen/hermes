package automation

import (
	"context"
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