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
	"time"

	"github.com/goph/logur"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

// ContextKey is the key the retrieved (or generated) correlation ID is stored under in the gin Context.
const ContextKey = "correlationid"

// Default correlation ID header
const defaultHeader = "Correlation-ID"

// MiddlewareOption configures the correlation ID middleware.
type MiddlewareOption interface {
	apply(*middleware)
}

// Header configures the header from where the correlation ID will be retrieved.
type Header string

// apply implements the MiddlewareOption interface.
func (h Header) apply(m *middleware) {
	m.header = string(h)
}

// MiddlewareCorrelationId returns a gin compatible handler.
func MiddlewareCorrelationId(opts ...MiddlewareOption) gin.HandlerFunc {
	m := new(middleware)

	for _, opt := range opts {
		opt.apply(m)
	}

	if m.header == "" {
		m.header = defaultHeader
	}

	return m.Handle
}

type middleware struct {
	header string
}

func (m *middleware) Handle(ctx *gin.Context) {
	if header := ctx.GetHeader(m.header); header != "" {
		ctx.Set(ContextKey, header)
	} else {
		ctx.Set(ContextKey, uuid.Must(uuid.NewV4()).String())
	}

	ctx.Next()
}

// GinMiddlewareLogger returns a gin compatible logur logger handler
func GinMiddlewareLogger(logger logur.Logger, notlogged ...string) gin.HandlerFunc {

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

			fields := logur.Fields{
				"latency":    latency,
				"statusCode": statusCode,
				"clientIP":   c.ClientIP(),
				"method":     c.Request.Method,
				"path":       path,
				"comment":    comment,
			}

			if cid := c.GetString(ContextKey); cid != "" {
				fields["correlation-id"] = cid
			}

			if pid := c.GetHeader("Banzai-Cloud-Pipeline-UUID"); pid != "" {
				fields["pipeline-instance"] = pid
			}

			entry := logur.WithFields(logger, fields)

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
