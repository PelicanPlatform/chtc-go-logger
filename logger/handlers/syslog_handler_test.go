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
package handlers_test

import (
	"log/slog"
	"log/syslog"
	"strings"
	"testing"

	"github.com/chtc/chtc-go-logger/config"
	"github.com/chtc/chtc-go-logger/logger"
	syslogServer "gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

var (
	localSyslogServer = "0.0.0.0:10514"
	testMsg           = "Hello, world!"
	testMsg2          = "Warning, world!"
	syslogTag         = "chtc-syslog"
)

// Create a local syslog server to log against, to avoid having to filter
// out actual linux syslog messages
func mkSyslogServer(outChan syslogServer.LogPartsChannel) *syslogServer.Server {
	channel := make(syslogServer.LogPartsChannel)
	handler := syslogServer.NewChannelHandler(channel)

	server := syslogServer.NewServer()
	server.SetFormat(syslogServer.Automatic)
	server.SetHandler(handler)
	server.ListenTCP(localSyslogServer)
	server.Boot()

	go (func() {
		for logParts := range channel {
			outChan <- logParts
		}
	})()

	return server
}

func verifyLogMsg(t *testing.T, logParts format.LogParts, expectedMsg string, expectedLevel syslog.Priority) {
	priority := logParts["priority"].(int)
	content := logParts["content"].(string)
	if priority != int(expectedLevel) {
		t.Fatalf("Expected priority %v from INFO message, got %v", syslog.LOG_INFO, priority)
	}
	if !strings.Contains(content, expectedMsg) {
		t.Fatalf("Expected syslog message %v to contain string %v", content, expectedMsg)
	}
}

// Ensure that messages sent to a local syslog message
func TestSyslogServer(t *testing.T) {
	outChan := make(syslogServer.LogPartsChannel)
	srv := mkSyslogServer(outChan)
	defer srv.Kill()

	config := config.Config{
		SyslogOutput: config.SyslogOutputConfig{
			Enabled:    true,
			JSONOutput: true,
			Network:    "tcp",
			Addr:       localSyslogServer,
		},
	}

	logger, err := logger.NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to construct syslog handler: %v", err)
	}

	// Test that log levels work
	logger.Info(testMsg)
	logParts := <-outChan
	verifyLogMsg(t, logParts, testMsg, syslog.LOG_INFO)

	logger.Warn(testMsg2)
	logParts = <-outChan
	verifyLogMsg(t, logParts, testMsg2, syslog.LOG_WARNING)

	// Test that child loggers work
	childLogger := logger.With(slog.String("child", "key"))

	childLogger.Error(testMsg)
	logParts = <-outChan
	verifyLogMsg(t, logParts, testMsg, syslog.LOG_ERR)
	verifyLogMsg(t, logParts, "\"child\":\"key\"", syslog.LOG_ERR)

	// Test that child loggers don't interfere with parent logger
	// This is important since child loggers write to the same io.Writer
	// as their parent
	logger.Error(testMsg)
	logParts = <-outChan
	verifyLogMsg(t, logParts, testMsg, syslog.LOG_ERR)

}
