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
	"github.com/nazarbabii/tmdb_go/internal/repositories"
	"github.com/nazarbabii/tmdb_go/internal/server"
	"github.com/nazarbabii/tmdb_go/internal/services"
)

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

	router := server.NewRouter(logger, server.Handlers{
		Health:       handlers.NewHealthHandler(),
		Titles:       handlers.NewTitlesHandler(tmdbSvc),
		WatchEntries: handlers.NewWatchEntriesHandler(watchEntrySvc),
		WatchEntry:   handlers.NewWatchEntryHandler(watchEntrySvc),
	})

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
