package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nazarbabii/tmdb_go/internal/config"
	"github.com/nazarbabii/tmdb_go/internal/database"
	"github.com/nazarbabii/tmdb_go/internal/handlers"
	"github.com/nazarbabii/tmdb_go/internal/middleware"
	"github.com/nazarbabii/tmdb_go/internal/repositories"
	"github.com/nazarbabii/tmdb_go/internal/services"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/nazarbabii/tmdb_go/docs"
)

// @title           TMDB Go API
// @version         1.0
// @description     Personal watch list backed by TMDB.
// @host            localhost:8088
// @BasePath        /api/v1
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := godotenv.Load(); err != nil {
		logger.Info("no .env file found, reading from environment")
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config", "error", err)
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	gin.SetMode(gin.ReleaseMode)

	tmdbSvc := services.NewTMDBService(cfg.TMDBBaseURL, cfg.TMDBAPIKey)
	watchEntrySvc := services.NewWatchEntryService(repositories.NewWatchEntryRepository(pool), tmdbSvc)

	router := gin.New()
	router.SetTrustedProxies(nil)
	router.Use(
		middleware.RequestID(),
		middleware.NewLogger(logger),
		middleware.NewRecovery(logger),
	)

	healthHandler := handlers.NewHealthHandler()
	titlesHandler := handlers.NewTitlesHandler(tmdbSvc)
	watchEntriesHandler := handlers.NewWatchEntriesHandler(watchEntrySvc)
	watchEntryHandler := handlers.NewWatchEntryHandler(watchEntrySvc)

	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	v1 := router.Group("/api/v1")
	{
		v1.GET("/titles/search", titlesHandler.Search)
		v1.GET("/titles/:tmdb_id/credits", titlesHandler.Credits)
		// v1.POST("/titles/:tmdb_id/credits", titlesHandler.AddCredits) //is this post will be related to movie 
		v1.POST("/watch-entries", watchEntriesHandler.Create)
		v1.GET("/watch-entries", watchEntriesHandler.List)
		v1.GET("/watch-entries/recommendations", watchEntriesHandler.GetRecommendations)
		v1.GET("/watch-entry", watchEntryHandler.Get)
		v1.GET("/watch-entry/:tmdb_id", watchEntryHandler.Exists)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to start", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}
