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
		statusCode, response := generateRandomStatusResponse() // Now status and response are correctly paired

		// Extract context
		ctx := c.Request.Context()
		logger := logger.GetContextLogger()

		// Log messages only based on status codes
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

// Generate a weighted random status code and its corresponding response
func generateRandomStatusResponse() (int, string) {
	cfg := GetConfig()
	// Define valid status codes and corresponding response options
	statusOptions := []struct {
		statusCode int
		responses  []string
		weight     int
	}{
		{200, []string{"Success", "Processing"}, cfg.HTTPResponseWeights.Response200},
		{400, []string{"Failure"}, cfg.HTTPResponseWeights.Response400},
		{500, []string{"Timeout", "Error"}, cfg.HTTPResponseWeights.Response500},
	}

	// Extract weights and pick a status code
	statusCodes := make([]int, len(statusOptions))
	weights := make([]int, len(statusOptions))
	for i, opt := range statusOptions {
		statusCodes[i] = opt.statusCode
		weights[i] = opt.weight
	}

	selectedStatus := weightedRandomChoice(statusCodes, weights)

	// Select a response for the chosen status code
	var selectedResponse string
	for _, opt := range statusOptions {
		if opt.statusCode == selectedStatus {
			selectedResponse = opt.responses[rand.Intn(len(opt.responses))]
			break
		}
	}

	return selectedStatus, selectedResponse
}

// Helper function for weighted random selection
func weightedRandomChoice[T any](items []T, weights []int) T {
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}

	randVal := rand.Intn(totalWeight)
	for i, w := range weights {
		if randVal < w {
			return items[i]
		}
		randVal -= w
	}

	// Fallback (should not be reached)
	return items[len(items)-1]
}
