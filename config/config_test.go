package config

import (
	"reflect"
	"testing"
)

func TestApplyOverrides(t *testing.T) {
	// Define default config values
	defaultConfig := &Config{
		LogLevel: "INFO",
		ConsoleOutput: ConsoleOutputConfig{
			Enabled:    true,
			JSONOutput: false,
			Colors:     true,
		},
		FileOutput: FileOutputConfig{
			Enabled:     true,
			FilePath:    "/var/log/chtc/app.log",
			MaxFileSize: 100,
			MaxBackups:  5,
			MaxAgeDays:  30,
		},
	}

	// Define override values
	overrides := &Config{
		LogLevel: "DEBUG",
		FileOutput: FileOutputConfig{
			FilePath:    "/custom/path/logfile.log",
			MaxFileSize: 200,
		},
	}

	// Expected config after applying overrides
	expectedConfig := &Config{
		LogLevel: "DEBUG", // Overridden
		ConsoleOutput: ConsoleOutputConfig{
			Enabled:    true,  // Default retained
			JSONOutput: false, // Default retained
			Colors:     true,  // Default retained
		},
		FileOutput: FileOutputConfig{
			Enabled:     true,                       // Default retained
			FilePath:    "/custom/path/logfile.log", // Overridden
			MaxFileSize: 200,                        // Overridden
			MaxBackups:  5,                          // Default retained
			MaxAgeDays:  30,                         // Default retained
		},
	}

	// Apply overrides
	ApplyOverrides(defaultConfig, overrides)

	// Verify the results
	if !reflect.DeepEqual(defaultConfig, expectedConfig) {
		t.Errorf("ApplyOverrides failed.\nGot: %+v\nWant: %+v", defaultConfig, expectedConfig)
	}
}
