package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Logger(base zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		event := eventForStatus(base, c.Writer.Status()).
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Int("size", c.Writer.Size()).
			Str("latency", time.Since(startedAt).String()).
			Str("client_ip", c.ClientIP())

		if query != "" {
			event = event.Str("query", query)
		}

		if route := c.FullPath(); route != "" {
			event = event.Str("route", route)
		}

		if userAgent := c.Request.UserAgent(); userAgent != "" {
			event = event.Str("user_agent", userAgent)
		}

		if len(c.Errors) > 0 {
			event = event.Str("errors", c.Errors.String())
		}

		event.Msg("http request")
	}
}

func eventForStatus(base zerolog.Logger, status int) *zerolog.Event {
	switch {
	case status >= 500:
		return base.Error()
	case status >= 400:
		return base.Warn()
	default:
		return base.Info()
	}
}
