package main

import (
	"fmt"
	"log/slog"

	"github.com/chtc/chtc-go-logger/pkg/logger"
)

var log = logger.LogWith(slog.String("package", "main"))

func main() {
	defer logger.StopErrorHandlers()
	logger.AddErrHandler(func(le logger.LogError) {
		// Can't log the error, just print it!
		fmt.Printf("Error: %v\n", le.Err)
	})

	log.Info("Hello, world!")
}
