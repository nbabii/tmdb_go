package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/nazarbabii/tmdb_go/internal/handlers"
	"github.com/nazarbabii/tmdb_go/internal/middleware"
)

type Handlers struct {
	Health       *handlers.HealthHandler
	Titles       *handlers.TitlesHandler
	WatchEntries *handlers.WatchEntriesHandler
	WatchEntry   *handlers.WatchEntryHandler
}

func NewRouter(logger *slog.Logger, h Handlers) *gin.Engine {
	router := gin.New()
	router.SetTrustedProxies(nil)
	router.Use(
		middleware.RequestID(),
		middleware.NewLogger(logger),
		middleware.NewRecovery(logger),
	)

	router.GET("/health", h.Health.HealthCheck)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/titles/search", h.Titles.Search)
		v1.POST("/watch-entries", h.WatchEntries.Create)
		v1.GET("/watch-entries", h.WatchEntries.List)
		v1.GET("/watch-entry", h.WatchEntry.Get)
	}

	return router
}
