package config

import (
	"bytes"
	_ "embed"
	"os"
	"reflect"
	"strings"
	"time"

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
type SyslogOutputConfig struct {
	Enabled    bool   `mapstructure:"enabled"`     // Enable or disable syslog output
	Network    string `mapstructure:"network"`     // Network over which to connect to syslog, default empty for local daemon
	Addr       string `mapstructure:"addr"`        // Address of remote syslog server, if any
	JSONOutput bool   `mapstructure:"json_object"` // If true, output JSON objects
}

type HealthCheckConfig struct {
	Enabled                  bool          `mapstructure:"enabled"`
	LogPeriodicity           time.Duration `mapstructure:"log_periodicity"`
	ElasticsearchPeriodicity time.Duration `mapstructure:"elasticsearch_periodicity"`
	ElasticsearchIndex       string        `mapstructure:"elasticsearch_index"`
	ElasticsearchURL         string        `mapstructure:"elasticsearch_url"`
}

type Config struct {
	LogLevel      string              `mapstructure:"log_level"`      // Log level (e.g., DEBUG, INFO, WARN, ERROR)
	ConsoleOutput ConsoleOutputConfig `mapstructure:"console_output"` // Console output settings
	FileOutput    FileOutputConfig    `mapstructure:"file_output"`    // File output settings
	SyslogOutput  SyslogOutputConfig  `mapstructure:"syslog_output"`  // Syslog output settings
	HealthCheck   HealthCheckConfig   `mapstructure:"health_check"`   // Health Check Settings
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

	// Manually load environment variables
	ManuallyLoadEnvVariables(v, "LOGGER")

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

// ApplyOverrides dynamically applies non-zero override values to a config, including nested structs.
func ApplyOverrides(config, overrides interface{}) {
	// Get reflection values of the structs
	overrideVal := reflect.ValueOf(overrides).Elem()
	configVal := reflect.ValueOf(config).Elem()

	for i := 0; i < overrideVal.NumField(); i++ {
		field := overrideVal.Type().Field(i)
		overrideField := overrideVal.Field(i)
		configField := configVal.FieldByName(field.Name)

		if overrideField.Kind() == reflect.Struct {
			// If the field is a struct, recurse
			ApplyOverrides(configField.Addr().Interface(), overrideField.Addr().Interface())
		} else if !overrideField.IsZero() {
			// If the field is not zero, override the value
			configField.Set(overrideField)
		}
	}
}

// ManuallyLoadEnvVariables scans and loads all environment variables with the given prefix into Viper.
func ManuallyLoadEnvVariables(v *viper.Viper, prefix string) {
	prefix = strings.ToUpper(prefix) + "__" // Ensure prefix is uppercase

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		key, value := parts[0], parts[1]

		if strings.HasPrefix(key, prefix) {
			// Remove the prefix
			keyWithoutPrefix := strings.TrimPrefix(key, prefix)

			// Convert key to Viper-compatible format:
			// - Replace "__" with "." for nested structures
			// - Convert to lowercase (Viper's default behavior)
			viperKey := strings.ToLower(strings.ReplaceAll(keyWithoutPrefix, "__", "."))

			v.Set(viperKey, value)

		}
	}
}
