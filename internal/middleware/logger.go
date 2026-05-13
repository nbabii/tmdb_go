package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// NewLogger returns a middleware that logs each request after it completes.
func NewLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start).String(),
			"request_id", GetRequestID(c),
		)
	}
}
