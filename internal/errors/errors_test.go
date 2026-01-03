package errors

import (
	stdErrors "errors"
	"testing"
	"time"
)

func TestRateLimitError(t *testing.T) {
	err := NewRateLimitError("slow down")

	if err.Error() != "slow down" {
		t.Fatalf("Error message = %q, want %q", err.Error(), "slow down")
	}

	if !IsRateLimitError(err) {
		t.Fatalf("IsRateLimitError returned false for RateLimitError")
	}

	wrapped := stdErrors.Join(err)
	if !IsRateLimitError(wrapped) {
		t.Fatalf("IsRateLimitError returned false for wrapped RateLimitError")
	}
}

func TestStopProcessingError(t *testing.T) {
	err := NewStopProcessingError("user stopped")

	if err.Error() != "user stopped" {
		t.Fatalf("Error message = %q, want %q", err.Error(), "user stopped")
	}

	if !IsStopProcessingError(err) {
		t.Fatalf("IsStopProcessingError returned false for StopProcessingError")
	}

	wrapped := stdErrors.Join(err)
	if !IsStopProcessingError(wrapped) {
		t.Fatalf("IsStopProcessingError returned false for wrapped StopProcessingError")
	}
}

func TestRateLimitErrorWithRetry(t *testing.T) {
	err := NewRateLimitErrorWithRetry("too many requests", 2*time.Minute)

	expected := "too many requests (retry after 2m0s)"
	if err.Error() != expected {
		t.Fatalf("Error message = %q, want %q", err.Error(), expected)
	}

	if !IsRateLimitError(err) {
		t.Fatalf("IsRateLimitError returned false for RateLimitErrorWithRetry")
	}

	if err.RetryAfter.Minutes() != 2.0 {
		t.Fatalf("RetryAfter = %v, want 2 minutes", err.RetryAfter)
	}
}

func TestRateLimitErrorWithRetry_ZeroDuration(t *testing.T) {
	err := NewRateLimitErrorWithRetry("rate limited", 0)

	// When RetryAfter is 0, the implementation only adds retry info if > 0
	expected := "rate limited"
	if err.Error() != expected {
		t.Fatalf("Error message = %q, want %q", err.Error(), expected)
	}

	if err.RetryAfter != 0 {
		t.Fatalf("RetryAfter = %v, want 0", err.RetryAfter)
	}
}

func TestRateLimitErrorWithRetry_VariousDurations(t *testing.T) {
	tests := []struct {
		name            string
		duration        time.Duration
		expectedMessage string
	}{
		{
			name:            "1 second",
			duration:        1 * time.Second,
			expectedMessage: "rate limited (retry after 1s)",
		},
		{
			name:            "30 seconds",
			duration:        30 * time.Second,
			expectedMessage: "rate limited (retry after 30s)",
		},
		{
			name:            "1 hour",
			duration:        1 * time.Hour,
			expectedMessage: "rate limited (retry after 1h0m0s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewRateLimitErrorWithRetry("rate limited", tt.duration)
			if err.Error() != tt.expectedMessage {
				t.Fatalf("Error message = %q, want %q", err.Error(), tt.expectedMessage)
			}
		})
	}
}

func TestSteamProfileError_403Private(t *testing.T) {
	err := NewSteamProfileError(403, "Profile is private")

	expected := "Steam profile is private or inaccessible (HTTP 403): Profile is private"
	if err.Error() != expected {
		t.Fatalf("Error message = %q, want %q", err.Error(), expected)
	}

	if !IsSteamProfileError(err) {
		t.Fatalf("IsSteamProfileError returned false for SteamProfileError")
	}

	if err.StatusCode != 403 {
		t.Fatalf("StatusCode = %d, want 403", err.StatusCode)
	}
}

func TestSteamProfileError_403Forbidden(t *testing.T) {
	err := NewSteamProfileError(403, "Access denied")

	expected := "Access forbidden - check API key and profile settings (HTTP 403): Access denied"
	if err.Error() != expected {
		t.Fatalf("Error message = %q, want %q", err.Error(), expected)
	}

	if err.Message != "Access forbidden - check API key and profile settings" {
		t.Fatalf("Message = %q, want 'Access forbidden - check API key and profile settings'", err.Message)
	}
}

func TestSteamProfileError_401(t *testing.T) {
	err := NewSteamProfileError(401, "Invalid key")

	expected := "Invalid Steam API key (HTTP 401): Invalid key"
	if err.Error() != expected {
		t.Fatalf("Error message = %q, want %q", err.Error(), expected)
	}

	if err.Message != "Invalid Steam API key" {
		t.Fatalf("Message = %q, want 'Invalid Steam API key'", err.Message)
	}
}

func TestSteamProfileError_OtherStatusCode(t *testing.T) {
	err := NewSteamProfileError(500, "Server error")

	expected := "Steam API access error (HTTP 500): Server error"
	if err.Error() != expected {
		t.Fatalf("Error message = %q, want %q", err.Error(), expected)
	}

	if err.Message != "Steam API access error" {
		t.Fatalf("Message = %q, want 'Steam API access error'", err.Message)
	}
}

func TestSteamProfileError_CaseInsensitivePrivate(t *testing.T) {
	testCases := []string{
		"Profile is PRIVATE",
		"profile is private",
		"This profile contains PRIVATE information",
	}

	for _, apiMsg := range testCases {
		err := NewSteamProfileError(403, apiMsg)
		if err.Message != "Steam profile is private or inaccessible" {
			t.Fatalf("For message %q, expected private profile message, got %q", apiMsg, err.Message)
		}
	}
}

func TestSteamProfileError_EmptyAPIMessage(t *testing.T) {
	err := NewSteamProfileError(403, "")

	expected := "Access forbidden - check API key and profile settings (HTTP 403)"
	if err.Error() != expected {
		t.Fatalf("Error message = %q, want %q", err.Error(), expected)
	}
}

func TestSteamProfileError_Wrapped(t *testing.T) {
	err := NewSteamProfileError(403, "private profile")
	wrapped := stdErrors.Join(err, stdErrors.New("additional context"))

	if !IsSteamProfileError(wrapped) {
		t.Fatalf("IsSteamProfileError returned false for wrapped SteamProfileError")
	}
}
