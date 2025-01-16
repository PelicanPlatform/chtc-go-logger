package main

import (
	"context"
	"log/slog"

	"github.com/chtc/chtc-go-logger/logger"
)

func main() {
	// Example 1: Logging Without Context
	nonContextLogger := logger.NewLogger()
	nonContextLogger.Info("Hello, world!",
		slog.String("status", "success"),
		slog.String("module", "main"),
		slog.String("env", "production"),
	)
	nonContextLogger.Warn("Potential issue detected",
		slog.String("code", "123"),
		slog.String("severity", "high"),
	)

	// Example 2: Logging With Context
	contextLogger := logger.NewContextAwareLogger()

	// Create a context with attributes using the custom context key
	ctx := context.WithValue(context.Background(), logger.LogAttrsKey, map[string]string{
		"userID":    "12345",
		"operation": "dataProcessing",
		"requestID": "abc-123",
	})

	// Log messages with multiple additional key-value pairs
	contextLogger.Info(ctx, "Operation completed",
		slog.String("status", "success"),
		slog.String("elapsedTime", "34ms"),
		slog.String("result", "ok"),
	)
	contextLogger.Warn(ctx, "Potential issue detected",
		slog.String("code", "123"),
		slog.String("severity", "high"),
		slog.String("retryable", "false"),
	)
	contextLogger.Error(ctx, "Operation failed",
		slog.String("error", "timeout"),
		slog.String("endpoint", "/api/v1/data"),
		slog.String("method", "POST"),
	)
}
