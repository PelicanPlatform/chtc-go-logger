package main

import (
	"log/slog"

	"github.com/chtc/chtc-go-logger/pkg/logger"
)

var log = logger.LogWith(slog.String("package", "main"))

func main() {
	log.Info("Hello, world!")
}
