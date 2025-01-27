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
	"bytes"
	"context"
	"log/slog"
	"log/syslog"
	"sync"

	"github.com/chtc/chtc-go-logger/config"
)

type SyslogHandler struct {
	buf     bytes.Buffer
	handler slog.Handler
	writer  *syslog.Writer
	mu      *sync.Mutex
}

func NewSyslogHandler(syslogOpts config.SyslogOutputConfig, logOpts *slog.HandlerOptions) (slog.Handler, error) {
	handler := SyslogHandler{
		mu:  &sync.Mutex{},
		buf: bytes.Buffer{},
	}

	handler.handler = slog.NewJSONHandler(&handler.buf, logOpts)
	writer, err := syslog.Dial(syslogOpts.Network, syslogOpts.Addr, syslog.LOG_DEBUG, syslogOpts.Tag)
	if err != nil {
		return nil, err
	}
	handler.writer = writer

	return &handler, nil
}

func (s *SyslogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.handler.Enabled(ctx, level)
}

// Required by slog.Handler interface: Processes a log via the writing handler, then
// forward to syslog
func (s *SyslogHandler) Handle(ctx context.Context, r slog.Record) (err error) {
	// Must be thread-safe, need to write to a buffer then immediately read back
	s.mu.Lock()
	defer s.mu.Unlock()
	// Write the log message via the child handler to the internal buffer
	if err = s.handler.Handle(ctx, r); err != nil {
		return err
	}
	// Read the logged contents back out of the buffer, then forward to syslog, converting
	// the slog level as appropriate
	switch lvl := r.Level; lvl {
	case slog.LevelDebug:
		err = s.writer.Debug(s.buf.String())
	case slog.LevelInfo:
		err = s.writer.Info(s.buf.String())
	case slog.LevelWarn:
		err = s.writer.Warning(s.buf.String())
	case slog.LevelError:
		err = s.writer.Err(s.buf.String())
	}
	return err
}

// Required by slog.Handler interface: Groups attributes under a namespace for the writing handler
func (s *SyslogHandler) WithGroup(name string) slog.Handler {
	return &SyslogHandler{handler: s.handler.WithGroup(name), buf: s.buf, writer: s.writer, mu: s.mu}
}

// Required by slog.Handler interface: Adds attributes to the writing handler
func (s *SyslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SyslogHandler{handler: s.handler.WithAttrs(attrs), buf: s.buf, writer: s.writer, mu: s.mu}
}
