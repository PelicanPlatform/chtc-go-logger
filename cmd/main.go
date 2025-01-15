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

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chtc/chtc-go-logger/logger"
)

// Set up baisc error handling (fall back to stdout) -
// it might make sense to provide an in-api wrapper for this?
func logErrorPoller(done <-chan bool) {
	for {
		select {
		case err := <-logger.GetLogErrorWatcher():
			fmt.Printf("An error occurred during logging! Falling back to stdout: %v\n", err.Err)
		case <-done:
			return
		}
	}
}
func main() {

	doneChan := make(chan bool)
	go logErrorPoller(doneChan)

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
	doneChan <- true
}
