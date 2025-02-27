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
	"log/slog"
	"path"
	"time"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/chtc/chtc-go-logger/logger/handlers"
	"golang.org/x/sys/unix"
)

// LogError is a data structure for an error that occured
// within one of the logger's sub-handlers
type LogError struct {
	// Error that occurred while logging
	Err error
	// Log record that caused the error
	Record slog.Record
	// Sub-handler in which the error occured
	Handler handlers.NamedHandler
}

// LogStats is a data structure that reports resource usage
// and errors that may have occured after each log message
type LogStats struct {
	// The time that the log message took to produce
	Duration time.Duration
	// If file-based output is available, the remaining storage space
	// on the file output's storage device
	DiskAvail uint64
	// An array of errors that occured in each of the logger's
	// sub-handlers
	Errors []LogError
	// The most recent remote health-check result for this logger
	HealthCheck HealthCheckStatus
}

// LogStatsCallback is a function type for a callback that accepts a LogStats
type LogStatsCallback func(stats LogStats)

// Wrapper for the slog.Handler interface that also provides data access
// to the LogStats that are collected during logging
type LogStatHandler interface {
	slog.Handler
	// Get the latest LogStats produced by the logger
	GetLatestStats() LogStats
	// Set a callback function that will be called whenever a new LogStats is produced
	SetStatsCallbackHandler(LogStatsCallback)
}

// Handler that wraps another set of slog handlers, and implements LogStatHandler
type logDispatchStatHandler struct {
	handlers      []handlers.NamedHandler
	logConfig     config.Config
	latestStats   LogStats
	statsCallback LogStatsCallback
}

func (s *logDispatchStatHandler) GetLatestStats() LogStats {
	return s.latestStats
}

func (s *logDispatchStatHandler) SetStatsCallbackHandler(callback LogStatsCallback) {
	s.statsCallback = callback
}

// NewLogStatsHandler constructs a new metrics-collecting log handler
// LogStatsHandler wraps the handler given in the constructor, collecting
// info such as log message duration and disk usage with each log message
func NewLogStatsHandler(logConfig config.Config, handlers []handlers.NamedHandler) LogStatHandler {
	handler := logDispatchStatHandler{
		handlers:  handlers,
		logConfig: logConfig,
	}

	return &handler
}

// slog.Handler implementation
func (s *logDispatchStatHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range s.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (s *logDispatchStatHandler) statLogFS() (uint64, error) {
	fs := path.Dir(s.logConfig.FileOutput.FilePath)

	stat := unix.Statfs_t{}

	if err := unix.Statfs(fs, &stat); err != nil {
		return 0, err
	}

	// Via stackoverflow, available blocks * blocksize
	return stat.Bavail * uint64(stat.Bsize), nil
}

// Required by slog.Handler interface: Processes a log via each sub-handler,
// collecting all the errors that occured during logging and exporting externally
// via the logger's set LogStatsCallback
func (s *logDispatchStatHandler) Handle(ctx context.Context, r slog.Record) error {
	stats := LogStats{}
	start := time.Now()

	// Call into the actual log handler, checking for errors on result
	errs := make([]LogError, 0, len(s.handlers))
	for _, handler := range s.handlers {
		err := handler.Handle(ctx, r)
		if err != nil {
			errs = append(errs, LogError{
				Err:     err,
				Record:  r,
				Handler: handler,
			})
		}
	}

	// If filesystem logging is enabled, check usage
	// This is probably a pretty big performance bottleneck
	if s.logConfig.FileOutput.Enabled {
		usage, err := s.statLogFS()
		stats.DiskAvail = usage
		if err != nil {
			errs = append(errs, LogError{
				Err:    err,
				Record: r,
			})
		}
	}

	// Measure duration of logging + log metadata acquisition
	elapsed := time.Since(start)

	stats.Duration = elapsed
	stats.Errors = errs
	if heathCheck := lastHealthCheck.Load(); heathCheck != nil {
		stats.HealthCheck = *heathCheck
	}

	s.latestStats = stats

	if s.statsCallback != nil {
		s.statsCallback(stats)
	}

	if len(errs) == 0 {
		return nil
	}

	// return the errors.join of all logging errors that occurred
	allErrs := make([]error, len(errs))
	for idx, err := range errs {
		allErrs[idx] = err.Err
	}
	return errors.Join(allErrs...)
}

// Required by slog.Handler interface: Groups attributes under a namespace for the writing handler
func (s *logDispatchStatHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]handlers.NamedHandler, len(s.handlers))
	for i, handler := range s.handlers {
		newHandlers[i] = handlers.NamedHandler{
			Handler:     handler.WithGroup(name),
			HandlerType: handler.HandlerType,
		}
	}
	return &logDispatchStatHandler{
		handlers:      newHandlers,
		statsCallback: s.statsCallback,
		logConfig:     s.logConfig,
	}
}

// Required by slog.Handler interface: Adds attributes to the writing handler
func (s *logDispatchStatHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]handlers.NamedHandler, len(s.handlers))
	for i, handler := range s.handlers {
		newHandlers[i] = handlers.NamedHandler{
			Handler:     handler.WithAttrs(attrs),
			HandlerType: handler.HandlerType,
		}
	}
	return &logDispatchStatHandler{
		handlers:      newHandlers,
		statsCallback: s.statsCallback,
		logConfig:     s.logConfig,
	}
}
