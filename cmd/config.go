package main

import (
	"log/slog"

	"github.com/chtc/chtc-go-logger/logger"
	"github.com/spf13/viper"
)

type Config struct {
	HTTPResponseWeights struct {
		Response200 int `mapstructure:"response_200"`
		Response400 int `mapstructure:"response_400"`
		Response500 int `mapstructure:"response_500"`
	} `mapstructure:"http_response_weights"`

	Logging struct {
		MinDiskSpaceRequired int `mapstructure:"min_disk_space_required"`
	} `mapstructure:"logging"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set environment variable prefix
	v.SetEnvPrefix("SAMPLE_LOGGER")
	v.AutomaticEnv()

	// Set default values
	v.SetDefault("http_response_weights.response_200", 1)
	v.SetDefault("http_response_weights.response_400", 1)
	v.SetDefault("http_response_weights.response_500", 1)
	v.SetDefault("logging.min_disk_space_required", 500) // Example default in MB

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	log := logger.GetLogger()
	// Log the full config dump
	log.Debug("Loaded configuration",
		slog.Group("http_response_weights",
			slog.Int("response_200", config.HTTPResponseWeights.Response200),
			slog.Int("response_400", config.HTTPResponseWeights.Response400),
			slog.Int("response_500", config.HTTPResponseWeights.Response500),
		),
		slog.Group("logging",
			slog.Int("min_disk_space_required", config.Logging.MinDiskSpaceRequired),
		),
	)

	return &config, nil
}
