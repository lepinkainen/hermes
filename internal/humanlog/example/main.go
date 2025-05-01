package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lepinkainen/hermes/internal/humanlog"
)

func main() {
	// Create a new human-readable handler with default options
	handler := humanlog.NewHandler(os.Stdout, nil)
	logger := slog.New(handler)

	// Set the default logger
	slog.SetDefault(logger)

	// Log messages with different levels and attributes
	logger.Info("Short message")
	logger.Info("This is a message that will be truncated because it's too long for the fixed-width field")

	logger.Debug("Debug message with attributes", "count", 42, "enabled", true)
	logger.Info("Info message with attributes", "user", "john", "action", "login")
	logger.Warn("Warning message with attributes", "latency", 150*time.Millisecond, "threshold", 100*time.Millisecond)
	logger.Error("Error message with attributes", "error", "connection refused", "retries", 3)

	// Log with structured attributes
	logger.LogAttrs(context.Background(), slog.LevelInfo, "Structured attributes",
		slog.String("service", "auth"),
		slog.Int("status", 200),
		slog.Duration("response_time", 30*time.Millisecond),
	)

	// Log with a group
	logger = logger.WithGroup("request")
	logger.Info("Request received",
		"method", "GET",
		"path", "/api/v1/users",
		"remote_addr", "192.168.1.1",
	)

	// Log with pre-registered attributes
	logger = logger.With("request_id", "abc-123", "trace_id", "xyz-789")
	logger.Info("Processing request")
	logger.Error("Request failed", "error", "database connection error")
}
