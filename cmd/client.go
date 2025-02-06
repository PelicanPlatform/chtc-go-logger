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
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/chtc/chtc-go-logger/logger"
)

// StartClient runs a single client in its own goroutine
func StartClient(ctx context.Context, clientID string, serverPort int, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{Timeout: 3 * time.Second}

	contextLogger := logger.GetContextLogger()

	for {
		select {
		case <-ctx.Done(): // Check if termination is requested
			contextLogger.Info(ctx, "Client stopping due to shutdown signal",
				slog.String("clientID", clientID),
			)
			return
		default:
			time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)

			jobID := fmt.Sprintf("job-%d", rand.Intn(100000))
			requestID := fmt.Sprintf("req-%d", rand.Intn(100000))

			ctx := context.WithValue(ctx, logger.LogAttrsKey, map[string]string{
				"clientID":  clientID,
				"jobID":     jobID,
				"requestID": requestID,
			})

			path := generateRandomRequestPath()
			// Use dynamic server port
			url := fmt.Sprintf("http://localhost:%d/%v", serverPort, path)
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

			resp, err := client.Do(req)
			if err != nil {
				contextLogger.Error(ctx, "Failed to send request",
					slog.String("error", err.Error()),
				)
				continue
			}

			contextLogger.Info(ctx, "Request sent",
				slog.String("clientID", clientID),
				slog.String("jobID", jobID),
				slog.String("status", fmt.Sprintf("%d", resp.StatusCode)),
			)
			resp.Body.Close()
		}
	}
}

// StartMockClients spawns multiple independent clients
func StartMockClients(ctx context.Context, numClients int, serverPort int, wg *sync.WaitGroup) {
	for i := 0; i < numClients; i++ {
		clientID := fmt.Sprintf("client-%d", i+1)
		wg.Add(1)
		go StartClient(ctx, clientID, serverPort, wg)
	}
}

// Generate a weighted random status code and its corresponding response
func generateRandomRequestPath() string {
	cfg := GetConfig()
	// Extract weights and pick a request path
	paths := []string{
		"test",
		"staging",
		"development",
	}
	weights := []int{
		cfg.ClientPathWeights.Test,
		cfg.ClientPathWeights.Staging,
		cfg.ClientPathWeights.Development,
	}

	selectedPath := weightedRandomChoice(paths, weights)
	return selectedPath
}
