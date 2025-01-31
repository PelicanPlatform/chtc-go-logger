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
package handlers

import (
	"context"
	"log/slog"
	"path"
	"time"

	"github.com/chtc/chtc-go-logger/config"
	"golang.org/x/sys/unix"
)

type LogError struct {
	Err    error
	Record slog.Record
}

type LogStats struct {
	Duration  time.Duration
	DiskAvail uint64
	Error     *LogError
}

// Handler that wraps another slog handler, forwarding its output to syslog
type LogStatsHandler struct {
	handler     slog.Handler
	logConfig   config.Config
	latestStats LogStats
}

func (s *LogStatsHandler) GetLatestStats() LogStats {
	return s.latestStats
}

// NewLogStatsHandler constructs a new metrics-collecting log handler
// LogStatsHandler wraps the handler given in the constructor, collecting
// info such as log message duration and disk usage with each log message
func NewLogStatsHandler(logConfig config.Config, baseHandler slog.Handler) slog.Handler {
	handler := LogStatsHandler{
		handler:   baseHandler,
		logConfig: logConfig,
	}

	return &handler
}

func (s *LogStatsHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.handler.Enabled(ctx, level)
}

func (s *LogStatsHandler) statLogFS() (uint64, error) {
	fs := path.Dir(s.logConfig.FileOutput.FilePath)

	stat := unix.Statfs_t{}

	if err := unix.Statfs(fs, &stat); err != nil {
		return 0, err
	}

	// Via stackoverflow, available blocks * blocksize
	return stat.Bavail * uint64(stat.Bsize), nil
}

// Required by slog.Handler interface: Processes a log via the writing handler, then
// forward to syslog
func (s *LogStatsHandler) Handle(ctx context.Context, r slog.Record) error {
	stats := LogStats{}
	start := time.Now()

	// Call into the actual log handler, checking for errors on result
	err := s.handler.Handle(ctx, r)
	if err != nil {
		stats.Error = &LogError{
			Err:    err,
			Record: r,
		}
	}

	// If filesystem logging is enabled, check usage
	// This is probably a pretty big performance bottleneck
	if s.logConfig.FileOutput.Enabled {
		usage, err := s.statLogFS()
		stats.DiskAvail = usage
		if err != nil {
			stats.Error = &LogError{
				Err:    err,
				Record: r,
			}
		}
	}

	// Measure duration of logging + log metadata acquisition
	elapsed := time.Since(start)
	stats.Duration = elapsed

	s.latestStats = stats
	return err
}

// Required by slog.Handler interface: Groups attributes under a namespace for the writing handler
func (s *LogStatsHandler) WithGroup(name string) slog.Handler {
	return &LogStatsHandler{handler: s.handler.WithGroup(name)}
}

// Required by slog.Handler interface: Adds attributes to the writing handler
func (s *LogStatsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogStatsHandler{handler: s.handler.WithAttrs(attrs)}
}
