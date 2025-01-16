package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/chtc/chtc-go-logger/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	loggerConfig   *config.Config // Global configuration for the logger
	consoleHandler slog.Handler   // Shared console handler
	fileHandler    slog.Handler   // Shared file handler
)

// LoggerOptions defines options for enabling/disabling handlers
type LoggerOptions struct {
	EnableConsole bool // Whether to enable console logging
	EnableFile    bool // Whether to enable file logging
}

// Define a custom type for context keys
type contextKey string

// Define a constant for the log attributes key
const LogAttrsKey contextKey = "logAttrs"

func init() {
	var err error
	loggerConfig, err = config.LoadConfig("", nil) // Load defaults
	if err != nil {
		panic("Failed to load logger configuration: " + err.Error())
	}

	// Initialize shared console handler
	if loggerConfig.ConsoleOutput.Enabled {
		if loggerConfig.ConsoleOutput.JSONOutput {
			consoleHandler = slog.NewJSONHandler(os.Stdout, nil)
		} else if loggerConfig.ConsoleOutput.Colors {
			consoleHandler = &ColorConsoleHandler{output: os.Stdout}
		} else {
			consoleHandler = slog.NewTextHandler(os.Stdout, nil)
		}
	}

	// Initialize shared file handler
	if loggerConfig.FileOutput.Enabled {
		fileHandler = slog.NewJSONHandler(&lumberjack.Logger{
			Filename:   loggerConfig.FileOutput.FilePath,
			MaxSize:    loggerConfig.FileOutput.MaxFileSize,
			MaxBackups: loggerConfig.FileOutput.MaxBackups,
			MaxAge:     loggerConfig.FileOutput.MaxAgeDays,
			Compress:   true,
		}, nil)
	}
}

// --- Non-Context Logger ---

// NewLogger creates a logger based on LoggerOptions
func NewLogger(options ...LoggerOptions) *slog.Logger {
	// Set default options
	opts := LoggerOptions{
		EnableConsole: true,
		EnableFile:    true,
	}

	// If options are provided, override the defaults
	if len(options) > 0 {
		opts = options[0]
	}

	var handlers []slog.Handler

	// Add console handler if enabled
	if opts.EnableConsole && consoleHandler != nil {
		handlers = append(handlers, consoleHandler)
	}

	// Add file handler if enabled
	if opts.EnableFile && fileHandler != nil {
		handlers = append(handlers, fileHandler)
	}

	return slog.New(&LogDispatcher{handlers: handlers})
}

// --- Context-Aware Logger ---

// ContextAwareLogger wraps slog.Logger to support context-based logging
type ContextAwareLogger struct {
	logger *slog.Logger
}

// NewContextAwareLogger creates a logger with context support by internally calling NewLogger
func NewContextAwareLogger(options ...LoggerOptions) *ContextAwareLogger {
	return &ContextAwareLogger{logger: NewLogger(options...)}
}

// Log logs a message at the specified level with context attributes and additional attributes
func (l *ContextAwareLogger) Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	// Extract attributes from context
	contextAttrs := extractContextAttributes(ctx)

	// Merge context attributes with additional attributes
	finalAttrs := append(contextAttrs, attrs...)

	// Convert []slog.Attr to []any for slog.Log
	anyAttrs := make([]any, len(finalAttrs))
	for i, attr := range finalAttrs {
		anyAttrs[i] = attr
	}

	// Log the message
	l.logger.Log(ctx, level, msg, anyAttrs...)
}

// Convenience methods for log levels
func (l *ContextAwareLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Log(ctx, slog.LevelInfo, msg, attrs...)
}

func (l *ContextAwareLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Log(ctx, slog.LevelDebug, msg, attrs...)
}

func (l *ContextAwareLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Log(ctx, slog.LevelWarn, msg, attrs...)
}

func (l *ContextAwareLogger) Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Log(ctx, slog.LevelError, msg, attrs...)
}

// --- Utilities ---

// extractContextAttributes extracts key-value pairs from a context.Context
func extractContextAttributes(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}

	// Assume attributes are stored in a map[string]string under "logAttrs"
	contextData, ok := ctx.Value(LogAttrsKey).(map[string]string)
	if !ok {
		return nil
	}

	// Convert map to slog.Attr
	attrs := make([]slog.Attr, 0, len(contextData))
	for k, v := range contextData {
		attrs = append(attrs, slog.String(k, v))
	}
	return attrs
}

// --- Handlers ---

// LogDispatcher forwards logs to multiple handlers
type LogDispatcher struct {
	handlers []slog.Handler
}

// Required by slog.Handler interface: Determines if this dispatcher processes a log record at the given level
func (d *LogDispatcher) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range d.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Required by slog.Handler interface: Processes and forwards a log record to all handlers
func (d *LogDispatcher) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, handler := range d.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Required by slog.Handler interface: Groups attributes under a namespace for all handlers
func (d *LogDispatcher) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(d.handlers))
	for i, handler := range d.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &LogDispatcher{handlers: newHandlers}
}

// Required by slog.Handler interface: Adds attributes to all handlers
func (d *LogDispatcher) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(d.handlers))
	for i, handler := range d.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &LogDispatcher{handlers: newHandlers}
}

// ColorConsoleHandler provides color-coded console logging
type ColorConsoleHandler struct {
	output io.Writer
}

// Required by slog.Handler interface: Determines if this handler processes a log record at the given level
func (h *ColorConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// Required by slog.Handler interface: Processes and outputs a log record
func (h *ColorConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	// Fetch log level color
	levelColor := levelColors[r.Level]
	if levelColor == "" {
		levelColor = ColorReset
	}

	// Collect attributes
	attrs := []string{}
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
		return true
	})

	// Format and write the log
	message := fmt.Sprintf("%s%s\033[0m: %s [%s]\n", levelColor, r.Level.String(), r.Message, strings.Join(attrs, ", "))
	_, err := h.output.Write([]byte(message))
	return err
}

// Required by slog.Handler interface: Adds attributes to the handler
func (h *ColorConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// Required by slog.Handler interface: Groups attributes under a namespace
func (h *ColorConsoleHandler) WithGroup(name string) slog.Handler {
	return h
}
