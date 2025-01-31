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
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/chtc/chtc-go-logger/logger"
	"github.com/gin-gonic/gin"
)

// GinLoggerMiddleware replaces Gin's default logger with chtc-go-logger
func GinLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Collect request details
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		logger := logger.GetLogger()
		logger.Info("Gin HTTP Request",
			slog.String("method", method),
			slog.String("path", path),
			slog.String("clientIP", clientIP),
			slog.String("status", fmt.Sprintf("%d", statusCode)),
			slog.String("latency", latency.String()),
		)
	}
}

// GinRecoveryMiddleware captures panic logs and sends them to chtc-go-logger
func GinRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic details
				logger := logger.GetLogger()
				logger.Error("Panic Recovered in Gin",
					slog.String("error", fmt.Sprintf("%v", err)),
					slog.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
