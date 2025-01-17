package config

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestMergeConfigs(t *testing.T) {
	loggerConfig, err := LoadConfig("", nil) // Load defaults with no external file or overrides
	if err != nil {
		t.Fatal("Failed to load logger configuration: " + err.Error())
	}

	expectedConfig := Config{
		// Set several non-default options
		LogLevel: "DEBUG",
		ConsoleOutput: &ConsoleOutputConfig{
			Enabled: false,
		},
		FileOutput: &FileOutputConfig{
			Enabled: false,
		},
	}
	ApplyOverrides(loggerConfig, &expectedConfig)
	assert.Equal(t, loggerConfig.ConsoleOutput.Enabled, expectedConfig.ConsoleOutput.Enabled, "Config.ConsoleOutput.Enabled should match override")
	assert.Equal(t, loggerConfig.FileOutput.Enabled, expectedConfig.FileOutput.Enabled, "Config.FileOutput.Enabled should match override")

}
