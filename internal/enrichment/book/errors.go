package book

import "errors"

var (
	// ErrBookNotFound is returned when a book cannot be found by the given identifier.
	ErrBookNotFound = errors.New("book not found")

	// ErrInvalidISBN is returned when the provided ISBN is invalid.
	ErrInvalidISBN = errors.New("invalid ISBN")

	// ErrAPIUnavailable is returned when the external API is unavailable.
	ErrAPIUnavailable = errors.New("API unavailable")
)
