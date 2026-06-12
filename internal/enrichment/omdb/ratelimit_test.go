package omdb

import (
	"fmt"
	"testing"

	"github.com/lepinkainen/hermes/internal/errors"
)

func TestRateLimitHandling(t *testing.T) {
	// Reset state before test
	ResetRateLimit()

	// Initially requests should be allowed
	if !RequestsAllowed() {
		t.Error("Expected requests to be allowed initially")
	}

	// Mark rate limit reached
	MarkRateLimitReached()

	// Now requests should not be allowed
	if RequestsAllowed() {
		t.Error("Expected requests to be blocked after rate limit reached")
	}

	// Calling mark again should be a no-op (idempotent)
	MarkRateLimitReached()

	// Still should not be allowed
	if RequestsAllowed() {
		t.Error("Expected requests to still be blocked")
	}

	// Reset and verify
	ResetRateLimit()
	if !RequestsAllowed() {
		t.Error("Expected requests to be allowed after reset")
	}
}

func TestGetCached_RateLimitBlocked(t *testing.T) {
	// Mark rate limit as reached
	MarkRateLimitReached()
	defer ResetRateLimit()

	fetcher := func() (*string, error) {
		s := "should not be called"
		return &s, nil
	}

	_, _, err := GetCached("test_key", fetcher)
	if err == nil {
		t.Error("Expected error when rate limit is reached")
	}
	if !errors.IsRateLimitError(err) {
		t.Errorf("Expected rate limit error, got: %v", err)
	}
}

func TestGetCached_FetcherError(t *testing.T) {
	ResetRateLimit()

	fetcher := func() (*string, error) {
		return nil, fmt.Errorf("fetch failed")
	}

	_, _, err := GetCached("test_key", fetcher)
	if err == nil {
		t.Error("Expected error from fetcher")
	}
}

func TestSeedCacheByID_EmptyID(t *testing.T) {
	err := SeedCacheByID("", "some data")
	if err != nil {
		t.Errorf("Expected no error for empty ID, got: %v", err)
	}
}
