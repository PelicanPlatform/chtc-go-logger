package config

import (
	"bytes"
	_ "embed"

	"github.com/spf13/viper"
)

// Embed the default.yaml file into the binary
//
//go:embed resources/default.yaml
var defaultYAML []byte

type ConsoleOutputConfig struct {
	Enabled    bool `mapstructure:"enabled"`     // Enable or disable console output
	JSONOutput bool `mapstructure:"json_object"` // If true, output JSON objects; disables colors
	Colors     bool `mapstructure:"colors"`      // Enable color-coded logs (ignored if JSONOutput is true)
}

type FileOutputConfig struct {
	Enabled     bool   `mapstructure:"enabled"`       // Enable or disable file output
	FilePath    string `mapstructure:"file_path"`     // Path to the log file
	MaxFileSize int    `mapstructure:"max_file_size"` // Max file size in MB
	MaxBackups  int    `mapstructure:"max_backups"`   // Number of backups to retain
	MaxAgeDays  int    `mapstructure:"max_age_days"`  // Maximum age of log files in days
}

type Config struct {
	LogLevel      string               `mapstructure:"log_level"`      // Log level (e.g., DEBUG, INFO, WARN, ERROR)
	ConsoleOutput *ConsoleOutputConfig `mapstructure:"console_output"` // Console output settings
	FileOutput    *FileOutputConfig    `mapstructure:"file_output"`    // File output settings
}

// LoadConfig loads and merges the configuration in this order:
// 1. Defaults from default.yaml (embedded).
// 2. Configurations from a file (if provided).
// 3. Environment variables (LOGGER_ prefix).
// 4. Overrides provided programmatically.
func LoadConfig(configFile string, overrides *Config) (*Config, error) {
	v := viper.New()

	// Load embedded default.yaml
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(defaultYAML)); err != nil {
		return nil, err
	}

	// Load from config file if provided
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.MergeInConfig(); err != nil {
			return nil, err
		}
	}

	// Load environment variables
	v.SetEnvPrefix("LOGGER")
	v.AutomaticEnv()

	// Parse into Config struct
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}

	// Apply overrides if provided
	if overrides != nil {
		ApplyOverrides(config, overrides)
	}

	return config, nil
}

// Apply programmatic overrides to the config
func ApplyOverrides(config, overrides *Config) {
	if overrides.LogLevel != "" {
		config.LogLevel = overrides.LogLevel
	}
	if overrides.ConsoleOutput != nil {
		config.ConsoleOutput.Enabled = overrides.ConsoleOutput.Enabled
		config.ConsoleOutput.JSONOutput = overrides.ConsoleOutput.JSONOutput
		config.ConsoleOutput.Colors = overrides.ConsoleOutput.Colors
	}

	if overrides.FileOutput != nil {
		config.FileOutput.Enabled = overrides.FileOutput.Enabled
		config.FileOutput.FilePath = overrides.FileOutput.FilePath
		config.FileOutput.MaxFileSize = overrides.FileOutput.MaxFileSize
		config.FileOutput.MaxBackups = overrides.FileOutput.MaxBackups
		config.FileOutput.MaxAgeDays = overrides.FileOutput.MaxAgeDays
	}
}
