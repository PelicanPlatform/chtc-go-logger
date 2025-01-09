package main

import (
	"fmt"
	"log/slog"

	"github.com/chtc/chtc-go-logger/pkg/logger"
)

var log = logger.LogWith(slog.String("package", "main"))

var doneChan = make(chan bool)

func pollForLogErrors() {
	for {
		select {
		case err := <-logger.BaseErrChan():
			if err.Err != nil {
				// Can't log, just print it!
				fmt.Printf("Error: %v\n", err.Err)
			}
		case <-doneChan:
		}
	}
}

func main() {
	go pollForLogErrors()
	log.Info("Hello, world!")
	doneChan <- true
}
