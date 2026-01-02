package errors

import (
	stdErrors "errors"
	"fmt"
	"strings"
)

// SteamProfileError represents a Steam profile access error (private profile, invalid key, etc.)
type SteamProfileError struct {
	Message    string
	StatusCode int
	APIMessage string // Error message from Steam API if available
}

func (e *SteamProfileError) Error() string {
	if e.APIMessage != "" {
		return fmt.Sprintf("%s (HTTP %d): %s", e.Message, e.StatusCode, e.APIMessage)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.Message, e.StatusCode)
}

// NewSteamProfileError creates a new profile access error
func NewSteamProfileError(statusCode int, apiMessage string) *SteamProfileError {
	var message string
	apiLower := strings.ToLower(apiMessage)

	switch statusCode {
	case 403:
		if strings.Contains(apiLower, "private") {
			message = "Steam profile is private or inaccessible"
		} else {
			message = "Access forbidden - check API key and profile settings"
		}
	case 401:
		message = "Invalid Steam API key"
	default:
		message = "Steam API access error"
	}

	return &SteamProfileError{
		Message:    message,
		StatusCode: statusCode,
		APIMessage: apiMessage,
	}
}

// IsSteamProfileError checks if error is a SteamProfileError
func IsSteamProfileError(err error) bool {
	var profileErr *SteamProfileError
	return stdErrors.As(err, &profileErr)
}
