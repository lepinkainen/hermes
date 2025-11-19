package errors

import (
	stdErrors "errors"
	"testing"
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
