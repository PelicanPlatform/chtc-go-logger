/***************************************************************
 *
 * Copyright (C) 2025, Pelican Project, Morgridge Institute for Research
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
	"log/slog"
	"sync"
)

// Wrapper for ContextAwareErrorLogger that returns the latest log info with each logger call
type ContextAwareErrorLogger struct {
	mu sync.Mutex
	ContextAwareLogger
}

func (l *ContextAwareErrorLogger) Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) LogStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.ContextAwareLogger.Log(ctx, level, msg, attrs...)
	return l.statHandler.GetLatestStats()
}

// Convenience methods for log levels
func (l *ContextAwareErrorLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) LogStats {
	return l.Log(ctx, slog.LevelInfo, msg, attrs...)
}

func (l *ContextAwareErrorLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) LogStats {
	return l.Log(ctx, slog.LevelDebug, msg, attrs...)
}

func (l *ContextAwareErrorLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) LogStats {
	return l.Log(ctx, slog.LevelWarn, msg, attrs...)
}

func (l *ContextAwareErrorLogger) Error(ctx context.Context, msg string, attrs ...slog.Attr) LogStats {
	return l.Log(ctx, slog.LevelError, msg, attrs...)
}
