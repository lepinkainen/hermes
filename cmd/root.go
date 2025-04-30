package cmd

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lepinkainen/hermes/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hermes",
	Short: "A tool to import data from various sources",
	Long:  `A tool to import data from various sources into a unified format.`,
}

// Global flags
var (
	overwriteFiles bool
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	initLogging()

	viper.SetDefault("MarkdownOutputDir", "./markdown/")
	viper.SetDefault("JSONOutputDir", "./json/")
	viper.SetDefault("OverwriteFiles", false)

	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found, writing default config file...")
			if err := viper.SafeWriteConfig(); err != nil {
				slog.Error("Error writing config file", "error", err)
			}
			os.Exit(0)
		} else {
			slog.Error("Fatal error config file", "error", err)
			os.Exit(1)
		}
	}

	// Initialize global config
	config.InitConfig()

	// Add global flags
	rootCmd.PersistentFlags().BoolVar(&overwriteFiles, "overwrite", viper.GetBool("OverwriteFiles"), "Overwrite existing markdown files when processing")

	// Update the config when the flag changes
	cobra.OnInitialize(func() {
		config.SetOverwriteFiles(overwriteFiles)
	})
}

// LogrusLikeHandler is a custom slog.Handler that formats logs similar to logrus
type LogrusLikeHandler struct {
	level     slog.Level
	writer    io.Writer
	formatter slog.Handler
}

// Enabled reports whether the handler handles records at the given level.
func (h *LogrusLikeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle handles the Record.
func (h *LogrusLikeHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.formatter.Handle(ctx, r)
}

// WithAttrs returns a new Handler whose attributes consist of h's attributes followed by attrs.
func (h *LogrusLikeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogrusLikeHandler{
		level:     h.level,
		writer:    h.writer,
		formatter: h.formatter.WithAttrs(attrs),
	}
}

// WithGroup returns a new Handler with the given group name.
func (h *LogrusLikeHandler) WithGroup(name string) slog.Handler {
	return &LogrusLikeHandler{
		level:     h.level,
		writer:    h.writer,
		formatter: h.formatter.WithGroup(name),
	}
}

// LogrusLikeHandlerOptions contains options for the LogrusLikeHandler.
type LogrusLikeHandlerOptions struct {
	Level slog.Level
}

// NewLogrusLikeHandler creates a new LogrusLikeHandler.
func NewLogrusLikeHandler(w io.Writer, opts *LogrusLikeHandlerOptions) *LogrusLikeHandler {
	if opts == nil {
		opts = &LogrusLikeHandlerOptions{
			Level: slog.LevelInfo,
		}
	}

	// Create a custom text handler with our formatting
	textHandler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: opts.Level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Format the time to match logrus
			if a.Key == "time" {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}
			}
			// Format the level to match logrus (e.g., "INFO   ")
			if a.Key == "level" {
				level := a.Value.String()
				switch strings.ToUpper(level) {
				case "INFO":
					a.Value = slog.StringValue("INFO   ")
				case "WARN":
					a.Value = slog.StringValue("WARNING")
				case "ERROR":
					a.Value = slog.StringValue("ERROR  ")
				case "DEBUG":
					a.Value = slog.StringValue("DEBUG  ")
				}
			}
			return a
		},
	})

	return &LogrusLikeHandler{
		level:     opts.Level,
		writer:    w,
		formatter: textHandler,
	}
}

func initLogging() {
	// Create a custom handler that formats logs similar to logrus
	handler := NewLogrusLikeHandler(os.Stdout, &LogrusLikeHandlerOptions{
		Level: slog.LevelInfo,
	})

	// Set the default logger
	slog.SetDefault(slog.New(handler))
}
