package errors

// RateLimitError represents a rate limit error from any API
type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// NewRateLimitError creates a new RateLimitError with the given message
func NewRateLimitError(message string) *RateLimitError {
	return &RateLimitError{Message: message}
}
