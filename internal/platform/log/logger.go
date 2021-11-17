// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/sirupsen/logrus"
)

// NewLogger creates a new logger.
func NewLogger(config Config) logur.Logger {
	logger := logrus.New()

	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:             config.NoColor,
		EnvironmentOverrideColors: true,
	})

	switch config.Format {
	case "logfmt":
		// Already the default

	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	if level, err := logrus.ParseLevel(config.Level); err == nil {
		logger.SetLevel(level)
	}

	return logrusadapter.New(logger)
}

// WithFields returns a new contextual logger instance with context added to it.
func WithFields(logger logur.Logger, fields map[string]interface{}) logur.Logger {
	return logur.WithFields(logger, fields)
}

const correlationIdField = "correlation-id"

// WithFieldsForHandlers returns a new logger instance with a correlation ID in it.
func WithFieldsForHandlers(ctx *gin.Context, logger logur.Logger, fields map[string]interface{}) logur.Logger {
	cid := ctx.GetString(ContextKey)

	if cid == "" {
		return logur.WithFields(logger, fields)
	}

	if fields == nil {
		fields = make(map[string]interface{}, 1)
	}

	fields[correlationIdField] = cid

	return logur.WithFields(logger, fields)
}

func GinLogger(logger logur.Logger, notlogged ...string) gin.HandlerFunc {

	var skip map[string]struct{}

	if length := len(notlogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notlogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			statusCode := c.Writer.Status()
			comment := c.Errors.ByType(gin.ErrorTypePrivate).String()

			if raw != "" {
				path = path + "?" + raw
			}

			timeFormat := "02/Jan/2006:15:04:05 -0700"

			entry := logur.WithFields(logger, map[string]interface{}{
				"statusCode": statusCode,
				"latency":    latency, // time to process
				"clientIP":   c.ClientIP(),
				"method":     c.Request.Method,
				"path":       path,
				"comment":    comment,
				"time":       time.Now().Format(timeFormat),
			})

			if len(c.Errors) > 0 {
				entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
			} else {
				msg := "ginLogger"

				if statusCode >= http.StatusInternalServerError {
					entry.Error(msg)
				} else if statusCode >= http.StatusBadRequest {
					entry.Warn(msg)
				} else {
					entry.Info(msg)
				}
			}
		}
	}
}
