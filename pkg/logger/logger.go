package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	baseSlogger  *slog.Logger
	initSlogOnce sync.Once
	errChan      chan LogError
	doneChan     chan bool
	errHandlers  []ErrHandler
)

type ErrHandler func(LogError)

type LogError struct {
	Record slog.Record
	Err    error
}

// TODO viper config or some such thing

type TeeHandlerConfig struct {
	fileLogRoot string
	consoleLog  io.Writer
	logOpts     slog.HandlerOptions
}

func NewTeeHandlerConfig() *TeeHandlerConfig {
	rootDir, exists := os.LookupEnv("LOG_ROOT")
	if !exists {
		rootDir = "/tmp"
	}

	return &TeeHandlerConfig{
		fileLogRoot: rootDir,
		consoleLog:  os.Stdout,
		logOpts:     FatalPrintOpts,
	}
}

type TeeHandler struct {
	handlers []slog.Handler
	errChan  chan LogError
}

const FatalLevel = slog.Level(12)

var FatalPrintOpts = slog.HandlerOptions{
	// Need to do a custom string representation of fatal
	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey {
			if a.Value.Any().(slog.Level) == FatalLevel {
				a.Value = slog.StringValue("FATAL")
			}
		}
		return a
	},
}

// Preconfigure the desired "child" loggers to CHTC standards
// TODO develop CHTC standards
func NewConsoleFileTeeHandler(config *TeeHandlerConfig, errChan chan LogError) *TeeHandler {
	consoleLogger := slog.New(slog.NewTextHandler(config.consoleLog, &config.logOpts))
	logrotate := &lumberjack.Logger{
		Filename:   filepath.Join(config.fileLogRoot, "log.log"),
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
	fileLogger := slog.New(slog.NewJSONHandler(logrotate, &config.logOpts))

	return &TeeHandler{
		handlers: []slog.Handler{
			consoleLogger.Handler(),
			fileLogger.Handler(),
		},
		errChan: errChan,
	}
}

// Return whether any child handler is able to handle the message
func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Pass the record down to both child loggers for handling
func (h *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	errs := make([]error, 0)
	for _, handler := range h.handlers {
		errs = append(errs, handler.Handle(ctx, r))
	}
	err := errors.Join(errs...)
	if err != nil {
		h.errChan <- LogError{
			Err:    err,
			Record: r,
		}
	}
	return err
}

// Return a new struct that contains copies of both handlers
func (h *TeeHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, 0)
	for _, handler := range h.handlers {
		newHandlers = append(newHandlers, handler.WithGroup(name))
	}
	// TODO does it make sense to share the error channel among all children
	// of the base logger?
	return &TeeHandler{handlers: newHandlers, errChan: h.errChan}
}

// Return a new struct that contains copies of both handlers
func (h *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, 0)
	for _, handler := range h.handlers {
		newHandlers = append(newHandlers, handler.WithAttrs(attrs))
	}
	return &TeeHandler{handlers: newHandlers, errChan: h.errChan}
}

func pollForLogErrors() {
	for {
		select {
		case err := <-errChan:
			if err.Err != nil {
				for _, handler := range errHandlers {
					handler(err)
				}
			}
		case <-doneChan:
		}
	}
}

func LogBase() *slog.Logger {
	initSlogOnce.Do(func() {
		fmt.Println("Base Slogger initializing...")
		errChan = make(chan LogError)
		doneChan = make(chan bool)
		go pollForLogErrors()
		baseSlogger = slog.New(NewConsoleFileTeeHandler(NewTeeHandlerConfig(), errChan))
	})

	if baseSlogger == nil {
		// Return the default logger if unable to initialize
		return slog.Default()
	}

	return baseSlogger
}

// Add a listener to the list of logging error handlers
func AddErrHandler(handler ErrHandler) {
	// TODO remove errHandlers
	errHandlers = append(errHandlers, handler)
}

// End the goroutine that delegates log error handling work
func StopErrorHandlers() {
	doneChan <- true
}

func LogWith(args ...any) *slog.Logger {
	return LogBase().With(args...)
}

// One big issue, no default fatal level
func LogFatal(log *slog.Logger, msg string, err error, args ...any) {
	args = append([]any{Error(err)}, args...)
	log.Log(context.Background(), FatalLevel, msg, args...)
	os.Exit(1)
}

// Helper function that adds an error-typed key/value pair to an slog message
func Error(err error) slog.Attr {
	return slog.Any("error", err)
}
