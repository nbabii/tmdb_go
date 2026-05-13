package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// NewRecovery returns a middleware that catches panics, logs them with the
// request ID and stack trace, and responds with a generic 500 body so
// internal details never leak to clients.
// Must be registered after RequestID so the request ID is available.
func NewRecovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"error", r,
					"stack", string(debug.Stack()),
					"request_id", GetRequestID(c),
				)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}
