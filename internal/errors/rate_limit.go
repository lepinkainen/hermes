package errors

import (
	stdErrors "errors"
	"fmt"
	"time"
)

// RateLimitError represents a rate limit error from any API
type RateLimitError struct {
	Message    string
	RetryAfter time.Duration // 0 means unknown
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s (retry after %v)", e.Message, e.RetryAfter)
	}
	return e.Message
}

// NewRateLimitError creates a new RateLimitError with the given message
func NewRateLimitError(message string) *RateLimitError {
	return &RateLimitError{
		Message:    message,
		RetryAfter: 0,
	}
}

// NewRateLimitErrorWithRetry creates a RateLimitError with retry timing
func NewRateLimitErrorWithRetry(message string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		Message:    message,
		RetryAfter: retryAfter,
	}
}

// IsRateLimitError reports if err is a RateLimitError, even when wrapped.
func IsRateLimitError(err error) bool {
	var rateLimitErr *RateLimitError
	return stdErrors.As(err, &rateLimitErr)
}
