package errors

import "errors"

// StopProcessingError represents a user-driven stop signal (e.g., from TUI).
type StopProcessingError struct {
	Reason string
}

func (e *StopProcessingError) Error() string {
	return e.Reason
}

// NewStopProcessingError creates a StopProcessingError with the provided reason.
func NewStopProcessingError(reason string) *StopProcessingError {
	return &StopProcessingError{Reason: reason}
}

// IsStopProcessingError reports whether err is a StopProcessingError (even when wrapped).
func IsStopProcessingError(err error) bool {
	var stopErr *StopProcessingError
	return errors.As(err, &stopErr)
}
