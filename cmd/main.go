/***************************************************************
 *
 * Copyright (C) 2025, Pelican Project, Morgridge Institute for Research
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

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/chtc/chtc-go-logger/logger"
)

func main() {
	log := logger.GetLogger()

	// Determine execution mode (default: "burst")
	mode := "burst"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	log.Info("Application started",
		slog.String("mode", mode),
	)

	switch mode {
	case "stream":
		log.Info("Running in STREAM mode")
		runStreamMode()
	case "burst":
		log.Info("Running in BURST mode (default)")
		runBurstMode()
	default:
		log.Error("Invalid mode provided",
			slog.String("expected", "stream | burst"),
			slog.String("received", mode),
		)
	}
}

// **STREAM MODE: Starts server + multiple clients with graceful shutdown**
func runStreamMode() {
	log := logger.GetLogger()
	var wg sync.WaitGroup
	numClients := 5
	portChan := make(chan int, 1)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Handle Ctrl+C for clean shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM) // Catch SIGINT (Ctrl+C) and SIGTERM
		<-sigChan
		log.Info("Shutdown signal received, stopping clients and server...")
		cancel() // Signal all goroutines to stop
	}()

	// Start the server in a separate goroutine and get its port
	wg.Add(1)
	go func() {
		defer wg.Done()
		StartServer(portChan)
	}()

	// Wait for the server to provide its port
	serverPort, ok := <-portChan
	if !ok {
		log.Error("Failed to retrieve server port. Exiting...")
		return
	}

	log.Info("Server started successfully", slog.Int("port", serverPort))

	// Give the server a moment to start
	time.Sleep(1 * time.Second)

	// Start clients with the correct port
	log.Info("Starting clients in STREAM mode", slog.Int("num_clients", numClients))
	StartMockClients(ctx, numClients, serverPort, &wg)

	// Wait for all goroutines (server + clients) to exit
	wg.Wait()
	log.Info("All clients and server have exited.")
}

// **BURST MODE: Runs a few quick log examples**
func runBurstMode() {
	log := logger.GetLogger()

	// Log the burst mode execution
	log.Info("Running quick BURST logging example",
		slog.String("mode", "burst"),
	)

	// Example 1: Logging Without Context
	log.Info("Hello, world!",
		slog.String("status", "success"),
		slog.String("module", "main"),
		slog.String("env", "production"),
	)
	log.Warn("Potential issue detected",
		slog.String("code", "123"),
		slog.String("severity", "high"),
	)

	// Example 2: Logging With Context
	contextLogger := logger.GetContextLogger()
	ctx := context.WithValue(context.Background(), logger.LogAttrsKey, map[string]string{
		"userID":    "12345",
		"operation": "dataProcessing",
		"requestID": "abc-123",
	})

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

func init() {
	overrideConfig := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: "/workspaces/logger_config_cleanup/chtc-logger.log",
		},
		HealthCheck: config.HealthCheckConfig{
			Enabled:                  true,
			LogPeriodicity:           10 * time.Second,
			ElasticsearchPeriodicity: 10 * time.Second,
			ElasticsearchIndex:       "my-app-logs",
			ElasticsearchURL:         "http://host.docker.internal:9200",
		},
	}

	// Initialize the global logger and suppress error
	_ = logger.LogInit(&overrideConfig)
	LoadConfig()
}
