package logger

import "log/slog"

// Map of log levels to their corresponding ANSI color codes
var levelColors = map[slog.Level]string{
	slog.LevelDebug: "\033[36m", // Cyan
	slog.LevelInfo:  "\033[32m", // Green
	slog.LevelWarn:  "\033[33m", // Yellow
	slog.LevelError: "\033[31m", // Red
}

// Reset color
const ColorReset = "\033[0m"
