package logger

import (
	"context"
	"log/slog"
	"sync"

	"github.com/chtc/chtc-go-logger/logger/handlers"
)

// Wrapper for ContextAwareErrorLogger that returns the latest log info with each logger call
type ContextAwareErrorLogger struct {
	mu sync.Mutex
	ContextAwareLogger
}

func (l *ContextAwareErrorLogger) Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) handlers.LogStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.ContextAwareLogger.Log(ctx, level, msg, attrs...)
	return l.statHandler.GetLatestStats()
}

// Convenience methods for log levels
func (l *ContextAwareErrorLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) handlers.LogStats {
	return l.Log(ctx, slog.LevelInfo, msg, attrs...)
}

func (l *ContextAwareErrorLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) handlers.LogStats {
	return l.Log(ctx, slog.LevelDebug, msg, attrs...)
}

func (l *ContextAwareErrorLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) handlers.LogStats {
	return l.Log(ctx, slog.LevelWarn, msg, attrs...)
}

func (l *ContextAwareErrorLogger) Error(ctx context.Context, msg string, attrs ...slog.Attr) handlers.LogStats {
	return l.Log(ctx, slog.LevelError, msg, attrs...)
}
