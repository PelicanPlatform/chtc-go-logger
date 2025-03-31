package adapters

import (
	"log/slog"

	"github.com/sirupsen/logrus"
)

type logrusAdapter struct {
	slogger *slog.Logger
}

var levelMapper = map[logrus.Level]slog.Level{
	logrus.TraceLevel: slog.LevelDebug, // TODO
	logrus.DebugLevel: slog.LevelDebug,
	logrus.InfoLevel:  slog.LevelInfo,
	logrus.WarnLevel:  slog.LevelWarn,
	logrus.ErrorLevel: slog.LevelError,
	logrus.FatalLevel: slog.LevelError, // TODO
}

// Format implements logrus.Formatter.
func (l *logrusAdapter) Format(entry *logrus.Entry) (data []byte, err error) {
	level, exists := levelMapper[entry.Level]
	if !exists {
		level = slog.LevelInfo
	}

	fields := make([]any, 0)
	for field, val := range entry.Data {
		fields = append(fields, slog.Any(field, val))
	}

	l.slogger.Log(entry.Context, level, entry.Message, fields...)
	return data, err
}

// SlogLogrusAdapter returns a logrus formatter that short-circuits all logging
// info passed into the logrus logger to a backing slog logger
func SlogLogrusAdapter(slogger *slog.Logger) logrus.Formatter {
	return &logrusAdapter{slogger: slogger}
}
