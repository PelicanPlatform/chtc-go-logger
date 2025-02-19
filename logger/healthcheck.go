package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/elastic/go-elasticsearch/v8"
)

// lastHealthCheckStatus stores the last known health check timestamp and any query errors
type lastHealthCheckStatus struct {
	Timestamp time.Time
	Err       error
}

// Atomic pointer to store the last health check status
var lastHealthCheck atomic.Pointer[lastHealthCheckStatus]

// Global Elasticsearch client (initialized once)
var esClient *elasticsearch.Client

// StartHealthCheckMonitor starts the health check monitoring
func StartHealthCheckMonitor(ctx context.Context, cfg *config.Config) {
	log := GetLogger()

	// Initialize atomic pointer with a default value
	lastHealthCheck.Store(&lastHealthCheckStatus{})

	// Initialize Elasticsearch client
	if err := initElasticsearchClient(cfg); err != nil {
		log.Error("Failed to initialize Elasticsearch client",
			slog.String("component", "healthcheck"),
			slog.String("error", err.Error()),
		)
		return
	}

	// Debug log indicating the goroutines are starting
	log.Debug("Starting goroutines for health check monitoring",
		slog.String("component", "healthcheck"),
	)

	go logHealthChecks(ctx, cfg, log)
	go queryElasticsearch(ctx, cfg, log)
}

// Initialize Elasticsearch client once
func initElasticsearchClient(cfg *config.Config) error {
	var err error
	esClient, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.HealthCheck.ElasticsearchURL}, // Fetch from config
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Elasticsearch client: %w", err)
	}
	return nil
}

// logHealthChecks periodically logs health check status
func logHealthChecks(ctx context.Context, cfg *config.Config, log *slog.Logger) {
	ticker := time.NewTicker(cfg.HealthCheck.LogPeriodicity)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("logHealthChecks exiting")
			return
		case t := <-ticker.C:
			status := lastHealthCheck.Load()
			if status.Err != nil {
				log.Warn("Health check issue",
					slog.String("component", "healthcheck"),
					slog.Time("timestamp", t),
					slog.Time("last_received", status.Timestamp),
					slog.String("error", status.Err.Error()),
				)
			} else {
				log.Info("Health check log",
					slog.String("component", "healthcheck"),
					slog.Time("timestamp", t),
					slog.Time("last_received", status.Timestamp),
				)
			}
		}
	}
}

// queryElasticsearch periodically fetches the last received health check timestamp
func queryElasticsearch(ctx context.Context, cfg *config.Config, log *slog.Logger) {
	ticker := time.NewTicker(cfg.HealthCheck.ElasticsearchPeriodicity)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("queryElasticsearch exiting")
			return
		case <-ticker.C:
			timestamp, err := fetchLastLogTimestamp(ctx, cfg, log)
			newStatus := &lastHealthCheckStatus{Timestamp: timestamp, Err: err}

			// Atomically update last health check status
			lastHealthCheck.Store(newStatus)

			if err != nil {
				log.Error("Failed to fetch last log timestamp",
					slog.String("component", "healthcheck"),
					slog.String("error", err.Error()),
				)
			}
		}
	}
}

// fetchLastLogTimestamp queries Elasticsearch for the latest health check log timestamp
func fetchLastLogTimestamp(ctx context.Context, cfg *config.Config, log *slog.Logger) (time.Time, error) {
	query := `{
		"size": 1,
		"sort": [{ "timestamp": "desc" }],
		"_source": ["timestamp"]
	}`

	res, err := esClient.Search(
		esClient.Search.WithContext(ctx),
		esClient.Search.WithIndex(cfg.HealthCheck.ElasticsearchIndex),
		esClient.Search.WithBody(strings.NewReader(query)),
		esClient.Search.WithFilterPath("hits.hits._source.timestamp"),
		esClient.Search.WithSize(1),
	)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to execute Elasticsearch query: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return time.Time{}, fmt.Errorf("elasticsearch query failed: %s", res.String())
	}

	var esResp struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Timestamp string `json:"timestamp"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return time.Time{}, fmt.Errorf("failed to decode Elasticsearch response: %w", err)
	}

	if len(esResp.Hits.Hits) == 0 {
		return time.Time{}, fmt.Errorf("no health check logs found")
	}

	parsedTime, err := time.Parse(time.RFC3339, esResp.Hits.Hits[0].Source.Timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	log.Debug("Successfully retrieved last health check timestamp",
		slog.String("component", "healthcheck"),
		slog.Time("last_timestamp", parsedTime),
	)

	return parsedTime, nil
}
