package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func NewRecovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"error", r,
					"stack", string(debug.Stack()),
					"request_id", GetRequestID(c),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
			}
		}()
	}
}
