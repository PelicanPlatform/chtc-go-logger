package handlers

import "log/slog"

// "Enum" of known log output streams
type HandlerType string

const (
	HandlerConsole HandlerType = "HandlerConsole"
	HandlerFile    HandlerType = "HandlerFile"
	HandlerSyslog  HandlerType = "HandlerSyslog"
)

type NamedHandler struct {
	slog.Handler
	HandlerType
}
