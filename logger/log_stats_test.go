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
	"os"
	"path"
	"testing"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/chtc/chtc-go-logger/logger/handlers"
	"golang.org/x/sys/unix"
)

// TODO make these configurable?
const (
	SMALL_TEST_VOL    = "/dev/shm"
	MAX_TEST_VOL_SIZE = 1024 * 1024 // Bail out if the small test volume is > 1MB
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
	if failedHandler != handlers.HandlerFile {
		t.Fatalf("Expected test failure in %v, got %v", handlers.HandlerFile, failedHandler)
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
	if failedHandler != handlers.HandlerFile {
		t.Fatalf("Expected test failure in %v, got %v", handlers.HandlerFile, failedHandler)
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
	// Fill the entire test volume
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

	// Attempt to write to that directory, expect a permissions error to occur
	cfg := config.Config{
		FileOutput: config.FileOutputConfig{
			FilePath: path.Join(testDir, "out.log"),
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
	if failedHandler != handlers.HandlerConsole {
		t.Fatalf("Expected test failure in %v, got %v", handlers.HandlerConsole, failedHandler)
	}
}
