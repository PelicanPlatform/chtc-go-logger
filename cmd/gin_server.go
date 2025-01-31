package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chtc/chtc-go-logger/logger"
	"github.com/gin-gonic/gin"
)

// StartServer initializes the Gin HTTP server with a dynamic port
func StartServer(portChan chan<- int) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(GinLoggerMiddleware(), GinRecoveryMiddleware())

	// Define a test endpoint (same as before)
	r.GET("/test", func(c *gin.Context) {
		response := generateRandomResponse()
		statusCode := generateRandomStatusCode()

		// Extract context
		ctx := c.Request.Context()
		logger := logger.GetContextLogger()

		// Log appropriately based on status and response
		switch {
		case statusCode >= 500:
			logger.Error(ctx, "Internal server error",
				slog.String("status_code", fmt.Sprintf("%d", statusCode)),
				slog.String("response", response),
			)
		case statusCode >= 400:
			logger.Warn(ctx, "Client request resulted in an error",
				slog.String("status_code", fmt.Sprintf("%d", statusCode)),
				slog.String("response", response),
			)
		case response == "Timeout":
			logger.Warn(ctx, "Request processing took too long",
				slog.String("status_code", fmt.Sprintf("%d", statusCode)),
				slog.String("warning", "timeout"),
			)
		case response == "Processing":
			logger.Debug(ctx, "Request is still being processed",
				slog.String("status_code", fmt.Sprintf("%d", statusCode)),
			)
		default:
			logger.Info(ctx, "Request processed successfully",
				slog.String("status_code", fmt.Sprintf("%d", statusCode)),
				slog.String("response", response),
			)
		}

		c.JSON(statusCode, gin.H{"message": response})
	})

	// Dynamically find an available port
	listener, err := net.Listen("tcp", ":0") // Bind to any available port
	if err != nil {
		logger.GetLogger().Error("Failed to bind server to a port", slog.String("error", err.Error()))
		close(portChan)
		return
	}
	port := listener.Addr().(*net.TCPAddr).Port
	portChan <- port // Send the chosen port to the channel

	// Log server start
	logger := logger.GetLogger()
	logger.Info("Server started successfully", slog.Int("port", port))

	// Create HTTP server
	srv := &http.Server{Handler: r}

	// Run server in a goroutine
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("Server encountered an unexpected error", slog.String("error", err.Error()))
		}
	}()

	// Handle graceful shutdown
	waitForShutdown(srv)
}

// Handles graceful shutdown of the server
func waitForShutdown(srv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger := logger.GetLogger()
	logger.Info("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", slog.String("error", err.Error()))
	} else {
		logger.Info("Server stopped cleanly.")
	}
}

// Generate a random response message
func generateRandomResponse() string {
	responses := []string{"Success", "Failure", "Timeout", "Error", "Processing"}
	return responses[rand.Intn(len(responses))]
}

// Generate a random status code
func generateRandomStatusCode() int {
	statusCodes := []int{200, 400, 403, 500, 503}
	return statusCodes[rand.Intn(len(statusCodes))]
}
