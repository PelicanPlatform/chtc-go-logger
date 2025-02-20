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
package handlers_test

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/chtc/chtc-go-logger/logger"
	"github.com/chtc/chtc-go-logger/logger/handlers"
)

func TestFileOutputPermissionDeniedError(t *testing.T) {
	// Create a test directory, then set it to read-only
	testDir := t.TempDir()
	err := os.Chmod(testDir, 0o400)
	if err != nil {
		t.Fatalf("Unable to create read-only test dir: %v", err)
	}

	// Attempt to write to that directory, expect a permissions error to occur
	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: path.Join(testDir, "out.log"),
			Enabled:  true,
		},
	}
	log, err := logger.NewContextAwareLogger(cfg)
	if err != nil {
		t.Fatalf("Unable to create logger: %v", err)
	}

	var lastStats handlers.LogStats
	log.SetErrorCallback(func(stats handlers.LogStats) {
		lastStats = stats
	})

	log.Info(context.Background(), "Test msg")

	if lastStats.Errors == nil || len(lastStats.Errors) != 1 {
		t.Fatalf("Expected 1 error to occur during logging, got %v", len(lastStats.Errors))
	}

	// Confirm that the correct handler failed
	failedHandler := lastStats.Errors[0].Handler.HandlerType
	if failedHandler != handlers.HandlerFile {
		t.Fatalf("Expected test failure in %v, got %v", handlers.HandlerFile, failedHandler)
	}
}
