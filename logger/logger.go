package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/chtc/chtc-go-logger/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	baseSlogger  *slog.Logger
	initSlogOnce sync.Once
	loggerConfig *config.Config // Global configuration for the logger
)

// init is called when the package is imported.
func init() {
	// Automatically load the configuration
	var err error
	loggerConfig, err = config.LoadConfig("", nil) // Load defaults with no external file or overrides
	if err != nil {
		panic("Failed to load logger configuration: " + err.Error())
	}

	// Initialize the base logger
	initSlogOnce.Do(func() {
		baseSlogger = slog.New(ConsoleFileLogger(loggerConfig))
	})
}

func mergeConfigs(baseConfig, partialConfig *config.Config) *config.Config {
	merged := *baseConfig // Start with a copy of the base configuration

	if partialConfig != nil {
		config.ApplyOverrides(&merged, partialConfig)
	}

	return &merged
}

// ConsoleFileLogger initializes the logger with console and file output based on config
func ConsoleFileLogger(partialConfig *config.Config) *LogDispatcher {
	// Merge the partial configuration with the global configuration
	finalConfig := mergeConfigs(loggerConfig, partialConfig)

	var handlers []slog.Handler

	// Console Handler
	if finalConfig.ConsoleOutput.Enabled {
		if finalConfig.ConsoleOutput.JSONOutput {
			handlers = append(handlers, slog.NewJSONHandler(os.Stdout, nil))
		} else if finalConfig.ConsoleOutput.Colors {
			handlers = append(handlers, &ColorConsoleHandler{output: os.Stdout})
		} else {
			handlers = append(handlers, slog.NewTextHandler(os.Stdout, nil))
		}
	}

	// File Handler
	if finalConfig.FileOutput.Enabled {
		logrotate := &lumberjack.Logger{
			Filename:   finalConfig.FileOutput.FilePath,
			MaxSize:    finalConfig.FileOutput.MaxFileSize,
			MaxBackups: finalConfig.FileOutput.MaxBackups,
			MaxAge:     finalConfig.FileOutput.MaxAgeDays,
			Compress:   true,
		}
		handlers = append(handlers, slog.NewJSONHandler(logrotate, nil))
	}

	return &LogDispatcher{handlers: handlers}
}

// ColorConsoleHandler provides color-coded console logging
type ColorConsoleHandler struct {
	output io.Writer
}

func (h *ColorConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *ColorConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	// Fetch the color for the log level; default to reset if not found
	levelColor, ok := levelColors[r.Level]
	if !ok {
		levelColor = ColorReset
	}

	// Collect attributes as key-value pairs
	attrs := []string{}
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
		return true
	})

	// Format and write the log message with color
	_, err := h.output.Write([]byte(fmt.Sprintf(
		"%s%s\033[0m: %s [%s]\n",
		levelColor, r.Level.String(), r.Message, strings.Join(attrs, ", "),
	)))
	return err
}

func (h *ColorConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *ColorConsoleHandler) WithGroup(name string) slog.Handler {
	return h
}

// LogDispatcher forwards logs to multiple handlers
type LogDispatcher struct {
	handlers []slog.Handler
}

func (d *LogDispatcher) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range d.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (d *LogDispatcher) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, handler := range d.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (d *LogDispatcher) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(d.handlers))
	for i, handler := range d.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &LogDispatcher{handlers: newHandlers}
}

func (d *LogDispatcher) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(d.handlers))
	for i, handler := range d.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &LogDispatcher{handlers: newHandlers}
}

// LogBase initializes the base logger based on the configuration
func LogBase(config *config.Config) *slog.Logger {
	initSlogOnce.Do(func() {
		baseSlogger = slog.New(ConsoleFileLogger(config))
	})

	if baseSlogger == nil {
		return slog.Default()
	}
	return baseSlogger
}

// LogWith provides a logger with additional context
func LogWith(config *config.Config, args ...any) *slog.Logger {
	return LogBase(config).With(args...)
}
