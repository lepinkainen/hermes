package humanlog

import (
	"io"
	"log/slog"
)

// Options configures the human-readable slog.Handler.
type Options struct {
	// Level is the minimum level to log.
	Level slog.Level

	// Writer is where the logs are written to.
	Writer io.Writer

	// TimeFormat is the format used for timestamps.
	// Default: "15:04:05" (hour:minute:second)
	TimeFormat string

	// DisableColor disables colored output for log levels.
	// When true, no ANSI color codes will be used.
	DisableColor bool
}

// DefaultOptions returns a new Options with default values.
func DefaultOptions() *Options {
	return &Options{
		Level:        slog.LevelInfo,
		TimeFormat:   "15:04:05",
		DisableColor: false,
	}
}
