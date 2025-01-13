package main

import (
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chtc/chtc-go-logger/logger"
)

// ContinuousLogStream generates a continuous stream of logs.
func ContinuousLogStream(log *slog.Logger) {
	logLevels := []string{"info", "error", "warn", "debug"}
	messages := []string{
		"Processing request",
		"Connecting to database",
		"Performing health check",
		"Restarting service",
		"Critical system failure",
	}

	// Create a dedicated random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		// Random log level and message
		level := logLevels[r.Intn(len(logLevels))]
		message := messages[r.Intn(len(messages))]
		additionalData := map[string]interface{}{
			"userID":  r.Intn(1000),
			"request": r.Intn(500),
			"status":  r.Intn(2) == 0, // Simulate success/failure
		}

		// Log with random level and attributes
		switch level {
		case "info":
			log.Info(message, "data", additionalData)
		case "error":
			log.Error(message, "error", "random error occurred", "data", additionalData)
		case "warn":
			log.Warn(message, "warning", "random warning", "data", additionalData)
		case "debug":
			log.Debug(message, "debugData", additionalData)
		}

		// Simulate delay between log entries
		time.Sleep(time.Millisecond * time.Duration(100+r.Intn(400)))
	}
}

func main() {
	// Initialize the logger
	baseLogger := logger.LogWith(nil, slog.String("package", "logsimulator"))

	// Set up graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		ContinuousLogStream(baseLogger)
	}()

	// Wait for termination signal
	<-stopChan
	baseLogger.Info("Shutting down log stream")
}
