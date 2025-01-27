/***************************************************************
 *
 * Copyright (C) 2024, Pelican Project, Morgridge Institute for Research
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you
 * may not use this file except in compliance with the License.  You may
 * obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 ***************************************************************/

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
	handler "github.com/chtc/chtc-go-logger/logger/handlers"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log *slog.Logger // Global logger
)

// Define a custom type for context keys
type contextKey string

// Define a constant for the log attributes key
const LogAttrsKey contextKey = "logAttrs"

// LogInit initializes the global logger.
// Accepts optional parameters: string (configFile) and *config.Config/config.Config (overrides).
func LogInit(params ...interface{}) error {
	// Parse the parameters
	cfg, err := parseParams(params...)
	if err != nil {
		return err
	}

	// Create and assign the global logger
	log, err = createLogger(cfg)
	return err
}

// NewLogger creates and returns a new logger.
// Accepts optional parameters: string (configFile) and *config.Config/config.Config (overrides).
func NewLogger(params ...interface{}) (*slog.Logger, error) {
	// Parse the parameters
	cfg, err := parseParams(params...)
	if err != nil {
		return nil, err
	}

	// Create and return a new logger
	return createLogger(cfg)
}

// parseParams parses the variadic parameters and loads the configuration.
func parseParams(params ...interface{}) (*config.Config, error) {
	var configFile string
	var overrides *config.Config

	// Process the parameters
	for _, param := range params {
		switch v := param.(type) {
		case string:
			configFile = v
		case *config.Config:
			overrides = v
		case config.Config:
			overrides = &v
		default:
			return nil, errors.New("invalid parameter type")
		}
	}

	// Load the configuration
	return config.LoadConfig(configFile, overrides)
}

// createLogger creates a logger using the provided configuration.
func createLogger(cfg *config.Config) (*slog.Logger, error) {
	var handlers []slog.Handler

	// Console handler
	if cfg.ConsoleOutput.Enabled {
		if cfg.ConsoleOutput.JSONOutput {
			handlers = append(handlers, slog.NewJSONHandler(os.Stdout, nil))
		} else if cfg.ConsoleOutput.Colors {
			handlers = append(handlers, &ColorConsoleHandler{output: os.Stdout})
		} else {
			handlers = append(handlers, slog.NewTextHandler(os.Stdout, nil))
		}
	}

	// File handler
	if cfg.FileOutput.Enabled {
		if cfg.FileOutput.FilePath == "" {
			panic("File output enabled but file path is empty")
		}
		handlers = append(handlers, slog.NewJSONHandler(&lumberjack.Logger{
			Filename:   cfg.FileOutput.FilePath,
			MaxSize:    cfg.FileOutput.MaxFileSize,
			MaxBackups: cfg.FileOutput.MaxBackups,
			MaxAge:     cfg.FileOutput.MaxAgeDays,
			Compress:   true,
		}, nil))
	}

	// Syslog handler
	if cfg.SyslogOutput.Enabled {
		var (
			syslogHandler slog.Handler
			err           error
		)
		if cfg.SyslogOutput.JSONOutput {
			syslogHandler, err = handler.NewSyslogHandler(cfg.SyslogOutput, func(w io.Writer) slog.Handler {
				return slog.NewJSONHandler(w, nil)
			})
		} else {
			syslogHandler, err = handler.NewSyslogHandler(cfg.SyslogOutput, func(w io.Writer) slog.Handler {
				return slog.NewTextHandler(w, nil)
			})
		}
		if err != nil {
			return nil, err
		}

		handlers = append(handlers, syslogHandler)
	}

	// Fallback to a basic console logger if no handlers are configured
	if len(handlers) == 0 {
		handlers = append(handlers, slog.NewTextHandler(os.Stdout, nil))
	}

	return slog.New(&LogDispatcher{handlers: handlers}), nil
}

// GetLogger returns the global logger. If `LogInit` is not called, it initializes the logger with default settings.
func GetLogger() *slog.Logger {
	if log == nil {
		// Initialize with defaults if LogInit is not called
		if err := LogInit("", nil); err != nil {
			panic("Failed to initialize default logger: " + err.Error())
		}
	}
	return log
}

// --- Context-Aware Logger ---

// ContextAwareLogger wraps slog.Logger to support context-based logging
type ContextAwareLogger struct {
	logger *slog.Logger
}

// GetContextLogger returns the global context logger. If `LogInit` is not called, it initializes the logger with default settings.
func GetContextLogger() *ContextAwareLogger {
	if log == nil {
		// Initialize with defaults if LogInit is not called
		if err := LogInit("", nil); err != nil {
			panic("Failed to initialize default logger: " + err.Error())
		}
	}
	return &ContextAwareLogger{logger: log}
}

// NewContextAwareLogger creates a logger with context support by internally calling NewLogger
func NewContextAwareLogger(params ...interface{}) (*ContextAwareLogger, error) {
	newLogger, err := NewLogger(params...)
	if err != nil {
		return nil, err
	}
	return &ContextAwareLogger{logger: newLogger}, err
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
