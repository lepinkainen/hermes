package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	var lastErr error
	for attempt := 1; attempt <= c.retryAttempts; attempt++ {
		if err := c.doJSONRequest(ctx, endpoint, target); err != nil {
			lastErr = err
			if !isRetryable(err) || attempt == c.retryAttempts {
				return err
			}
			time.Sleep(backoffDelay(attempt))
			continue
		}
		return nil
	}
	return lastErr
}

func (c *Client) getJSONMap(ctx context.Context, endpoint string) (map[string]any, error) {
	var data map[string]any
	if err := c.getJSON(ctx, endpoint, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) doJSONRequest(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("tmdb: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func isRetryable(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		// Network errors (connection resets etc.)
		if strings.Contains(urlErr.Error(), "connection") {
			return true
		}
	}
	return false
}

func backoffDelay(attempt int) time.Duration {
	// exponential backoff capped at 10 seconds
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	if delay > 10*time.Second {
		return 10 * time.Second
	}
	return delay
}
