package main

import (
	"log/slog"

	"github.com/chtc/chtc-go-logger/logger"
)

var log = logger.LogWith(nil, slog.String("package", "main"))

func main() {
	log.Info("Hello, world!")
}
