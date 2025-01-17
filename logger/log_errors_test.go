package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chtc/chtc-go-logger/config"
)

// Test that an error occurs and is caught when logs are created
// file logging is enabled
// TODO what other log errors can we reliably trigger?
func TestJSONFileLogPermissionDenied(t *testing.T) {
	// TODO set this config!
	tmpDir := t.TempDir()
	_ = config.Config{
		ConsoleOutput: &config.ConsoleOutputConfig{
			Enabled: false,
		},
		FileOutput: &config.FileOutputConfig{
			Enabled:  true,
			FilePath: filepath.Join(tmpDir, "test.log"),
		},
	}

	// Make tmpdir read only
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Fatal(err)
	}
	defer (func() {
		os.Chmod(tmpDir, 0644)
	})()

	logger := NewLogger()

	logger.Info("You should not be able to write this.")

	select {
	case <-GetLogErrorWatcher():
		// error occured as expected
	case <-time.After(1 * time.Second):
		t.Fatal("Expected error did not occur while logging")
	}
}

// Logger should handle invalid context being passed gracefully
func TestLogNilContext(t *testing.T) {
	tmpDir := t.TempDir()
	// TODO set this config!
	_ = config.Config{
		ConsoleOutput: &config.ConsoleOutputConfig{
			Enabled: true,
		},
		FileOutput: &config.FileOutputConfig{
			Enabled:  true,
			FilePath: filepath.Join(tmpDir, "test.log"),
		},
	}

	logger := NewContextAwareLogger()
	logger.Info(nil, "Hello, world!")

	// Make tmpdir read only
	select {
	case <-GetLogErrorWatcher():
		t.Fatal("Error occured when not expected")
	case <-time.After(1 * time.Second):
		// No error expected
	}
}
