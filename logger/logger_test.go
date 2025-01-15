package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/slogtest"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/magiconair/properties/assert"
)

// Files will not exist until after slogtest.TestHandler runs, defer opening them
// until it's time to evaluate the contents
type EvaluatesToFile func() *os.File

// TODO the fact that we can't redirect the stdout of the console logger in-config is a code smell
type redirectStdout struct {
	old          *os.File
	redirectPath string
}

func (r *redirectStdout) Redirect(t *testing.T) error {
	tmpDir := t.TempDir()
	r.redirectPath = filepath.Join(tmpDir, "stout-redirect")
	r.old = os.Stdout
	writer, err := os.Create(r.redirectPath)
	if err != nil {
		return err
	}
	os.Stdout = writer
	return nil
}
func (r *redirectStdout) Close() {
	os.Stdout = r.old
}

func (r *redirectStdout) Read() *os.File {
	reader, _ := os.Open(r.redirectPath)
	return reader
}

// Parse the attributes back out of a json-formatted slog message
// via https://pkg.go.dev/testing/slogtest#example-package-Parsing
func defaultJSONParser(t *testing.T, outReader EvaluatesToFile) []map[string]any {
	var buf bytes.Buffer
	io.Copy(&buf, outReader())
	var ms []map[string]any
	for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatal(err)
		}
		ms = append(ms, m)
	}
	return ms
}

// Test that the expected slog output is created at the file target when
// file logging is enabled
func TestJSONFileLoggerEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	testConfig := config.Config{
		ConsoleOutput: &config.ConsoleOutputConfig{
			Enabled: false,
		},
		FileOutput: &config.FileOutputConfig{
			Enabled:  true,
			FilePath: filepath.Join(tmpDir, "test.log"),
		},
	}

	logger := NewLogger()

	err := slogtest.TestHandler(logger.Handler(), func() []map[string]any {
		return defaultJSONParser(t, func() *os.File {
			reader, err := os.Open(testConfig.FileOutput.FilePath)
			if err != nil {
				t.Fatal(err)
			}
			return reader
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Test that the expected slog output is created at os.Stdout when
// console logging is enabled in JSON format
func TestJSONConsoleLoggerEnabled(t *testing.T) {
	_ = config.Config{
		ConsoleOutput: &config.ConsoleOutputConfig{
			Enabled:    true,
			JSONOutput: true,
		},
		FileOutput: &config.FileOutputConfig{
			Enabled: false,
		},
	}
	redirect := redirectStdout{}
	if err := redirect.Redirect(t); err != nil {
		t.Fatal(err)
	}
	defer redirect.Close()

	logger := NewLogger()

	err := slogtest.TestHandler(logger.Handler(), func() []map[string]any {
		return defaultJSONParser(t, redirect.Read)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Test that nothing is written to a test file when the file logger is disabled
func TestJSONFileLoggerDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	testConfig := config.Config{
		FileOutput: &config.FileOutputConfig{
			Enabled:  false,
			FilePath: filepath.Join(tmpDir, "test.log"),
		},
	}

	logger := NewLogger()

	logger.Info("Test Message")

	_, err := os.Stat(testConfig.FileOutput.FilePath)
	assert.Equal(t, errors.Is(err, os.ErrNotExist), true)
}

// Test that nothing is written to stdout when the console logger is disabled
func TestConsoleLoggerDisabled(t *testing.T) {
	_ = config.Config{
		ConsoleOutput: &config.ConsoleOutputConfig{
			Enabled: false,
		},
		FileOutput: &config.FileOutputConfig{
			Enabled: false,
		},
	}
	redirect := redirectStdout{}
	if err := redirect.Redirect(t); err != nil {
		t.Fatal(err)
	}
	defer redirect.Close()

	logger := NewLogger()
	logger.Info("Test Message")

	stat, err := os.Stat(redirect.redirectPath)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, stat.Size(), int64(0), "Expected empty log contents")
}
