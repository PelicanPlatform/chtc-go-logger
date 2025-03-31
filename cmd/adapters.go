package main

import (
	"github.com/chtc/chtc-go-logger/adapters"
	"github.com/chtc/chtc-go-logger/logger"
	"github.com/sirupsen/logrus"
)

// ** ADAPT MODE: Runs a few quick examples using the CHTC logger as a backend to other logging libraries **
func runAdaptMode() {
	// CHTC backing logger
	log := logger.GetLogger()

	// logrusLogger
	logrus.SetFormatter(adapters.SlogLogrusAdapter(log))
	logrusLogger := logrus.WithFields(logrus.Fields{
		"sample field": map[string]int{
			"sample field 2": 5,
		},
	})

	logrusLogger.Info("This is an info message!")

}
