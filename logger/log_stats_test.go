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
package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/chtc/chtc-go-logger/logger/handlers"
	"golang.org/x/sys/unix"
)

// TODO make these configurable?
const (
	SMALL_TEST_VOL    = "/dev/shm"
	MAX_TEST_VOL_SIZE = 1024 * 1024 // Bail out if the small test volume is > 1MB
	// The default labels for each handler
	HandlerConsole string = "console_output"
	HandlerFile           = "file_output"
	HandlerSyslog         = "syslog_output"
)

// Crete a log directory to which we don't have write access,
// Then attempt to write a log to it. Confirm that the
// error callback is invoked.
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
	log, err := NewContextAwareLogger(cfg)
	if err != nil {
		t.Fatalf("Unable to create logger: %v", err)
	}

	var lastStats LogStats
	log.SetErrorCallback(func(stats LogStats) {
		lastStats = stats
	})

	log.Info(context.Background(), "Test msg")

	if lastStats.Errors == nil || len(lastStats.Errors) != 1 {
		t.Fatalf("Expected 1 error to occur during logging, got %v", len(lastStats.Errors))
	}

	// Confirm that the correct handler failed
	failedHandler := lastStats.Errors[0].Handler.HandlerType
	if failedHandler != HandlerFile {
		t.Fatalf("Expected test failure in %v, got %v", HandlerFile, failedHandler)
	}
}

func getTestDirSpace() (uint64, error) {
	fs := SMALL_TEST_VOL

	stat := unix.Statfs_t{}

	if err := unix.Statfs(fs, &stat); err != nil {
		return 0, err
	}

	// Via stackoverflow, available blocks * blocksize
	avail := stat.Bavail * uint64(stat.Bsize)
	if avail > MAX_TEST_VOL_SIZE {
		return 0, errors.New("Too much space available in /dev/shm! Please set a smaller shm_size")
	}
	return avail, nil

}

func fillTestDirSpace(name string, size int) (string, error) {
	dataPath := path.Join(SMALL_TEST_VOL, name)

	f, err := os.Create(dataPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// set a non-zero value to stop linux from creating a sparse file
	for i := 0; i < size; i++ {
		if _, err = f.WriteString("1"); err != nil {
			return "", err
		}
	}

	return dataPath, nil
}

// Fill all the open space in the small test directory,
// Then attempt to write additional log lines in it.
// Confirm that the error callback is invoked
func TestFileOutputZeroSpaceError(t *testing.T) {
	avail, err := getTestDirSpace()
	if err != nil {
		t.Fatalf("Unable to stat small test volume: %v", err)
	}
	// Fill the entire test volume
	dataPath, err := fillTestDirSpace("data", int(avail))
	if err != nil {
		t.Fatalf("Unable to fill space on small test volume: %v", err)
	}

	logPath := path.Join(SMALL_TEST_VOL, "out.log")

	defer os.Remove(dataPath)
	defer os.Remove(logPath)

	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: logPath,
			Enabled:  true,
		},
	}
	log, err := NewContextAwareLogger(cfg)
	if err != nil {
		t.Fatalf("Unable to create logger: %v", err)
	}

	var lastStats LogStats
	log.SetErrorCallback(func(stats LogStats) {
		lastStats = stats
	})

	log.Info(context.Background(), "Test msg")

	if lastStats.Errors == nil || len(lastStats.Errors) != 1 {
		t.Fatalf("Expected 1 error to occur during logging, got %v", len(lastStats.Errors))
	}

	// Confirm that the correct handler failed
	failedHandler := lastStats.Errors[0].Handler.HandlerType
	if failedHandler != HandlerFile {
		t.Fatalf("Expected test failure in %v, got %v", HandlerFile, failedHandler)
	}
}

// Fill most of the open space in the small test directory,
// Then attempt to write additional log lines in it.
// Confirm that the log callback reports a low disk
// availability metric
func TestFileOutputLowSpaceWarning(t *testing.T) {
	const OpenSpace = 8 * 1024
	avail, err := getTestDirSpace()
	if err != nil {
		t.Fatalf("Unable to stat small test volume: %v", err)
	}
	// Fill the entire test volume, minus some amount of space
	dataPath, err := fillTestDirSpace("data", int(avail)-OpenSpace)
	if err != nil {
		t.Fatalf("Unable to fill space on small test volume: %v", err)
	}

	logPath := path.Join(SMALL_TEST_VOL, "out.log")

	defer os.Remove(dataPath)
	defer os.Remove(logPath)

	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: logPath,
			Enabled:  true,
		},
	}
	log, err := NewContextAwareLogger(cfg)
	if err != nil {
		t.Fatalf("Unable to create logger: %v", err)
	}

	var lastStats LogStats
	log.SetErrorCallback(func(stats LogStats) {
		lastStats = stats
	})

	log.Info(context.Background(), "Test msg")

	if len(lastStats.Errors) != 0 {
		t.Fatalf("Unexpected error occurred during logging: %v", lastStats.Errors[0].Err)
	}

	diskAvail := lastStats.DiskAvail

	if diskAvail > OpenSpace {
		t.Fatalf("Expected to detect <=%v available disk space, got %v", OpenSpace, diskAvail)
	}

	if diskAvail == 0 {
		t.Fatal("Expected to detect >0 available disk space")
	}
}

// Redirect stdout to a new file handler, close it to trigger an error when
// writing via the console logger, then check that those errors are reported
func TestCloseStdoutTextLogError(t *testing.T) {
	// TODO is there a better way to forcibly trigger a console write error?
	testDir := t.TempDir()

	f, err := os.Create(path.Join(testDir, "stdout"))
	if err != nil {
		t.Fatalf("Unable to create test stdout")
	}
	realStdout := os.Stdout
	defer (func() { os.Stdout = realStdout })()
	os.Stdout = f
	f.Close()

	// Attempt to write to stdout, expect a "file closed" error
	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: path.Join(testDir, "out.log"),
		},
		ConsoleOutput: config.ConsoleOutputConfig{
			Enabled: true,
		},
	}
	log, err := NewContextAwareLogger(cfg)
	if err != nil {
		t.Fatalf("Unable to create logger: %v", err)
	}

	var lastStats LogStats
	log.SetErrorCallback(func(stats LogStats) {
		lastStats = stats
	})

	log.Info(context.Background(), "Test msg")

	if lastStats.Errors == nil || len(lastStats.Errors) != 1 {
		t.Fatalf("Expected 1 error to occur during logging, got %v", len(lastStats.Errors))
	}

	// Confirm that the correct handler failed
	failedHandler := lastStats.Errors[0].Handler.HandlerType
	if failedHandler != HandlerConsole {
		t.Fatalf("Expected test failure in %v, got %v", HandlerConsole, failedHandler)
	}
}

// Test that each reported log stat contins the most recently set
// value of the log destination health check
func TestLogStatsContainHealthCheck(t *testing.T) {
	testDir := t.TempDir()
	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: path.Join(testDir, "out.log"),
			Enabled:  true,
		},
		HealthCheck: config.HealthCheckConfig{
			Enabled: true,
		},
	}
	log, err := NewContextAwareLogger(cfg)

	if err != nil {
		t.Fatalf("Unable to create logger: %v", err)
	}

	var lastStats LogStats
	log.SetErrorCallback(func(stats LogStats) {
		lastStats = stats
	})

	// Set a test health check value to retrieve
	sampleHealthCheck := HealthCheckStatus{
		Timestamp: time.Now(),
		Err:       errors.New("Sample Health Check Error"),
	}
	lastHealthCheck.Store(&sampleHealthCheck)

	log.Info(context.Background(), "Test msg")

	if len(lastStats.Errors) != 0 {
		t.Fatalf("Expected 0 errors to occur during logging, got %v", len(lastStats.Errors))
	}

	// Confirm that the set value for the last health check is returned
	healthCheck := lastStats.HealthCheck

	if healthCheck.Timestamp != sampleHealthCheck.Timestamp {
		t.Fatalf("Expected health check with timestamp %v, got %v", sampleHealthCheck.Timestamp, healthCheck.Timestamp)
	}
	if !errors.Is(healthCheck.Err, sampleHealthCheck.Err) {
		t.Fatalf("Expected health check to have error %v, got %v", sampleHealthCheck.Err, healthCheck.Err)
	}
}

// Dummy struct that waits an expected duration on write, used to
type testDelayWriter struct {
	delay time.Duration
}

func (t *testDelayWriter) Write(p []byte) (int, error) {
	time.Sleep(t.delay)
	return len(p), nil
}

// Test that each reported log stat contins the amount of time
// that that log message was expected to produce
func TestLogStatsElapsed(t *testing.T) {
	// Create a test directory, then set it to read-only
	testDir := t.TempDir()
	// Attempt to write to that directory, expect a permissions error to occur
	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: path.Join(testDir, "out.log"),
			Enabled:  true,
		},
	}
	// Create a log handler that uses the "wait" output stream
	expectedDelay := 10 * time.Millisecond
	delayHandler := slog.NewJSONHandler(&testDelayWriter{delay: expectedDelay}, nil)
	statsHandler := NewLogStatsHandler(cfg, []handlers.NamedHandler{{
		Handler:     delayHandler,
		HandlerType: HandlerFile,
	}})
	log := slog.New(statsHandler)

	var lastStats LogStats
	statsHandler.SetStatsCallbackHandler(func(stats LogStats) {
		lastStats = stats
	})

	log.InfoContext(context.Background(), "Test msg")

	delay := lastStats.Duration

	if delay < expectedDelay {
		t.Fatalf("Expected log duration of at least %v, got %v", expectedDelay, delay)
	}

	if delay > 2*expectedDelay {
		t.Fatalf("Expected log duration of less than %v, got %v", 2*expectedDelay, delay)
	}
}

// Create a large number of logs in parallel, then
// verify that every sequence number appears in the
// resulting output
func TestLogSequencing(t *testing.T) {
	testDir := t.TempDir()

	messageCount := 500

	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: path.Join(testDir, "out.log"),
			Enabled:  true,
		},
		SequenceInfo: config.SequenceConfig{
			Enabled: true,
		},
	}

	log, err := NewContextAwareLogger(cfg)

	if err != nil {
		t.Fatalf("Unable to create logger: %v", err)
	}

	var wg sync.WaitGroup

	for i := 0; i < messageCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info(context.Background(), "Test msg")
		}()
	}
	wg.Wait()

	// Confirm that every sequence number is included exactly once
	file_contents, err := os.ReadFile(cfg.FileOutput.FilePath)
	if err != nil {
		t.Fatalf("Unable to read logger output: %v", err)
	}
	file_str := string(file_contents)

	for i := 1; i <= messageCount; i++ {
		match, err := regexp.MatchString(fmt.Sprintf("\"sequence_no\":%v", i), file_str)
		if err != nil || !match {
			t.Fatalf("Sequence #%v not found in output!", i)
		}
	}

}
