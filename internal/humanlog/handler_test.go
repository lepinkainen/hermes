package humanlog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestHandler_Enabled(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  bool
	}{
		{
			name:  "Debug level with Info handler",
			level: slog.LevelDebug,
			want:  false,
		},
		{
			name:  "Info level with Info handler",
			level: slog.LevelInfo,
			want:  true,
		},
		{
			name:  "Warn level with Info handler",
			level: slog.LevelWarn,
			want:  true,
		},
		{
			name:  "Error level with Info handler",
			level: slog.LevelError,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(new(bytes.Buffer), &Options{Level: slog.LevelInfo})
			if got := h.Enabled(context.Background(), tt.level); got != tt.want {
				t.Errorf("Handler.Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name       string
		level      slog.Level
		message    string
		attrs      []slog.Attr
		wantPrefix string
		wantAttrs  []string
	}{
		{
			name:       "Info message without attributes",
			level:      slog.LevelInfo,
			message:    "Test message",
			attrs:      nil,
			wantPrefix: "INFO",
			wantAttrs:  nil,
		},
		{
			name:       "Error message with string attribute",
			level:      slog.LevelError,
			message:    "Error occurred",
			attrs:      []slog.Attr{slog.String("error", "file not found")},
			wantPrefix: "ERROR",
			wantAttrs:  []string{`error="file not found"`},
		},
		{
			name:       "Warn message with multiple attributes",
			level:      slog.LevelWarn,
			message:    "Warning",
			attrs:      []slog.Attr{slog.Int("count", 42), slog.Bool("critical", false)},
			wantPrefix: "WARN",
			wantAttrs:  []string{"count=42", "critical=false"},
		},
		{
			name:       "Debug message with complex attributes",
			level:      slog.LevelDebug,
			message:    "Debug info",
			attrs:      []slog.Attr{slog.Time("timestamp", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)), slog.Duration("elapsed", 5*time.Second)},
			wantPrefix: "DEBUG",
			wantAttrs:  []string{"timestamp=", "elapsed=5s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			h := NewHandler(buf, &Options{Level: slog.LevelDebug, DisableColor: true})
			logger := slog.New(h)

			if len(tt.attrs) > 0 {
				logger.LogAttrs(context.Background(), tt.level, tt.message, tt.attrs...)
			} else {
				logger.Log(context.Background(), tt.level, tt.message)
			}

			got := buf.String()

			// Check level prefix
			if !strings.Contains(got, tt.wantPrefix) {
				t.Errorf("Handler.Handle() output = %v, should contain level %v", got, tt.wantPrefix)
			}

			// Check message
			if !strings.Contains(got, tt.message) {
				t.Errorf("Handler.Handle() output = %v, should contain message %v", got, tt.message)
			}

			// Check attributes if any
			for _, attr := range tt.wantAttrs {
				if !strings.Contains(got, attr) {
					t.Errorf("Handler.Handle() output = %v, should contain attribute %v", got, attr)
				}
			}

			// Check that the message is padded to the fixed width
			// This is a simple check to ensure the fixed-width formatting is applied
			if len(tt.message) < messageWidth {
				// If the message is shorter than messageWidth, there should be spaces after it
				expectedPadding := messageWidth - len(tt.message)
				if !strings.Contains(got, tt.message+strings.Repeat(" ", expectedPadding)) {
					t.Errorf("Handler.Handle() output = %v, should contain padded message", got)
				}
			}
		})
	}
}

func TestHandler_WithAttrs(t *testing.T) {
	buf := new(bytes.Buffer)
	h := NewHandler(buf, &Options{Level: slog.LevelInfo, DisableColor: true})

	// Create a handler with pre-defined attributes
	h2 := h.WithAttrs([]slog.Attr{slog.String("component", "test"), slog.Int("id", 123)})
	logger := slog.New(h2)

	// Log a message
	logger.Info("Test with attrs")

	got := buf.String()

	// Check that pre-defined attributes are included
	if !strings.Contains(got, "component=test") {
		t.Errorf("WithAttrs() output = %v, should contain component=test", got)
	}

	if !strings.Contains(got, "id=123") {
		t.Errorf("WithAttrs() output = %v, should contain id=123", got)
	}
}

func TestHandler_WithGroup(t *testing.T) {
	buf := new(bytes.Buffer)
	h := NewHandler(buf, &Options{Level: slog.LevelInfo, DisableColor: true})

	// Create a handler with a group
	h2 := h.WithGroup("request")

	// Add attributes to the group
	h3 := h2.WithAttrs([]slog.Attr{slog.String("method", "GET"), slog.String("path", "/api/v1/users")})

	logger := slog.New(h3)

	// Log a message
	logger.Info("Received request")

	got := buf.String()

	// Check that grouped attributes are included with the correct prefix
	if !strings.Contains(got, "request.method=GET") {
		t.Errorf("WithGroup() output = %v, should contain request.method=GET", got)
	}

	// The path contains a slash, which may or may not be quoted depending on the implementation
	if !strings.Contains(got, "request.path=/api/v1/users") && !strings.Contains(got, "request.path=\"/api/v1/users\"") {
		t.Errorf("WithGroup() output = %v, should contain request.path=/api/v1/users or request.path=\"/api/v1/users\"", got)
	}
}

func TestHandler_CustomTimeFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	h := NewHandler(buf, &Options{
		Level:        slog.LevelInfo,
		TimeFormat:   "2006-01-02",
		DisableColor: true,
	})

	logger := slog.New(h)
	logger.Info("Custom time format test")

	got := buf.String()

	// Get today's date in the format YYYY-MM-DD
	today := time.Now().Format("2006-01-02")

	// Check that the log contains today's date
	if !strings.Contains(got, today) {
		t.Errorf("Custom time format output = %v, should contain date %v", got, today)
	}
}

func TestHandler_AttributeFormatting(t *testing.T) {
	tests := []struct {
		name     string
		attr     slog.Attr
		expected string
	}{
		{
			name:     "Empty string",
			attr:     slog.String("key", ""),
			expected: `key=""`,
		},
		{
			name:     "String with spaces",
			attr:     slog.String("key", "hello world"),
			expected: `key="hello world"`,
		},
		{
			name:     "String without spaces",
			attr:     slog.String("key", "hello"),
			expected: `key=hello`,
		},
		{
			name:     "Integer",
			attr:     slog.Int("key", 42),
			expected: `key=42`,
		},
		{
			name:     "Boolean true",
			attr:     slog.Bool("key", true),
			expected: `key=true`,
		},
		{
			name:     "Boolean false",
			attr:     slog.Bool("key", false),
			expected: `key=false`,
		},
		{
			name:     "Error",
			attr:     slog.Any("key", errors.New("test error")),
			expected: `key="test error"`,
		},
		{
			name:     "String with special characters",
			attr:     slog.String("key", "value{with}braces"),
			expected: `key="value{with}braces"`,
		},
		{
			name:     "String that looks like Go keyword",
			attr:     slog.String("key", "nil"),
			expected: `key="nil"`,
		},
		{
			name:     "String that looks like boolean literal",
			attr:     slog.String("key", "true"),
			expected: `key="true"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			h := NewHandler(buf, &Options{Level: slog.LevelInfo, DisableColor: true})
			logger := slog.New(h)

			logger.LogAttrs(context.Background(), slog.LevelInfo, "Test", tt.attr)

			got := buf.String()

			if !strings.Contains(got, tt.expected) {
				t.Errorf("Attribute formatting output = %v, should contain %v", got, tt.expected)
			}
		})
	}
}

func TestHandler_ColorOutput(t *testing.T) {
	// Test with color enabled
	bufWithColor := new(bytes.Buffer)
	hWithColor := NewHandler(bufWithColor, &Options{
		Level:        slog.LevelInfo,
		DisableColor: false,
	})

	// Test with color disabled
	bufNoColor := new(bytes.Buffer)
	hNoColor := NewHandler(bufNoColor, &Options{
		Level:        slog.LevelInfo,
		DisableColor: true,
	})

	// Log the same message with both handlers
	loggerWithColor := slog.New(hWithColor)
	loggerNoColor := slog.New(hNoColor)

	loggerWithColor.Error("Test error message")
	loggerNoColor.Error("Test error message")

	gotWithColor := bufWithColor.String()
	gotNoColor := bufNoColor.String()

	// The colored output should contain ANSI escape codes
	if !strings.Contains(gotWithColor, colorRed) {
		t.Errorf("Color output = %v, should contain ANSI color code", gotWithColor)
	}

	// The non-colored output should not contain ANSI escape codes
	if strings.Contains(gotNoColor, colorRed) {
		t.Errorf("Non-color output = %v, should not contain ANSI color code", gotNoColor)
	}
}
