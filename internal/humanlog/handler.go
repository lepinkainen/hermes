package humanlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

// Handler implements slog.Handler for human-readable logging output.
type Handler struct {
	opts   Options
	mu     sync.Mutex
	attrs  []slog.Attr
	groups []string
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

// Handle handles the Record.
// Format: [TIME] LEVEL: Message [key=value key2=value2 ...]
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Format time
	timeStr := r.Time.Format(h.opts.TimeFormat)

	// Format level
	levelStr := formatLevel(r.Level, h.opts.DisableColor)

	// Build the log line
	var sb strings.Builder

	// [TIME] LEVEL: Message
	sb.WriteString("[")
	sb.WriteString(timeStr)
	sb.WriteString("] ")
	sb.WriteString(levelStr)
	sb.WriteString(" ")
	sb.WriteString(r.Message)

	// Collect and format attributes
	var attrs []string

	// Add pre-registered attributes from WithAttrs
	for _, attr := range h.attrs {
		attrs = append(attrs, formatAttr(attr, h.opts.DisableColor))
	}

	// Add attributes from the record
	r.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, formatAttr(attr, h.opts.DisableColor))
		return true
	})

	// Add attributes if any
	if len(attrs) > 0 {
		sb.WriteString(" [")
		sb.WriteString(strings.Join(attrs, " "))
		sb.WriteString("]")
	}

	// Add newline and write to output
	sb.WriteString("\n")
	_, err := io.WriteString(h.opts.Writer, sb.String())
	return err
}

// WithAttrs returns a new Handler whose attributes consist of h's attributes followed by attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.attrs = make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	h2.attrs = append(h2.attrs, h.attrs...)

	// Apply groups to attributes if any
	if len(h.groups) > 0 {
		group := strings.Join(h.groups, ".")
		for _, attr := range attrs {
			h2.attrs = append(h2.attrs, slog.Attr{
				Key:   group + "." + attr.Key,
				Value: attr.Value,
			})
		}
	} else {
		h2.attrs = append(h2.attrs, attrs...)
	}

	return &h2
}

// WithGroup returns a new Handler with the given group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h2 := *h
	h2.groups = make([]string, 0, len(h.groups)+1)
	h2.groups = append(h2.groups, h.groups...)
	h2.groups = append(h2.groups, name)
	return &h2
}

// formatLevel returns a fixed-width, uppercase level string with optional color.
func formatLevel(level slog.Level, disableColor bool) string {
	var levelStr string
	var colorCode string

	switch {
	case level >= slog.LevelError:
		levelStr = "ERROR"
		colorCode = colorRed
	case level >= slog.LevelWarn:
		levelStr = "WARN "
		colorCode = colorYellow
	case level >= slog.LevelInfo:
		levelStr = "INFO "
		colorCode = colorBlue
	default:
		levelStr = "DEBUG"
		colorCode = colorGray
	}

	if disableColor {
		return levelStr
	}

	return colorCode + levelStr + colorReset
}

// formatAttr formats a single attribute as "key=value".
func formatAttr(attr slog.Attr, disableColor bool) string {
	if attr.Equal(slog.Attr{}) {
		return ""
	}

	key := attr.Key
	val := attr.Value

	// Handle special cases
	switch val.Kind() {
	case slog.KindString:
		// Quote strings if they contain spaces or special characters
		s := val.String()
		if needsQuoting(s) {
			return fmt.Sprintf("%s=%q", key, s)
		}
		return fmt.Sprintf("%s=%s", key, s)

	case slog.KindTime:
		// Format time values
		t := val.Time()
		return fmt.Sprintf("%s=%s", key, t.Format(time.RFC3339))

	case slog.KindDuration:
		// Format duration values
		d := val.Duration()
		return fmt.Sprintf("%s=%s", key, d.String())

	case slog.KindAny:
		// Handle error values specially
		if err, ok := val.Any().(error); ok {
			return fmt.Sprintf("%s=%q", key, err.Error())
		}
		fallthrough

	default:
		// Use the default string representation for other types
		return fmt.Sprintf("%s=%s", key, val.String())
	}
}

// needsQuoting returns true if the string contains spaces or special characters.
func needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	// Check if it's a valid number
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return false
	}

	// Check for spaces or special characters
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' || r == '\'' || r == '`' || r == '[' || r == ']' {
			return true
		}
	}

	return false
}
