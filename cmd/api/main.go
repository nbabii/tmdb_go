package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nazarbabii/tmdb_go/internal/config"
	"github.com/nazarbabii/tmdb_go/internal/database"
	"github.com/nazarbabii/tmdb_go/internal/handlers"
	"github.com/nazarbabii/tmdb_go/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	tmdbService := services.NewTMDBService(cfg.TMDBBaseURL, cfg.TMDBAPIKey)

	pool, err := database.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	watchEntryRepo := database.NewWatchEntryRepository(pool)
	watchEntrySvc := services.NewWatchEntryService(watchEntryRepo, tmdbService)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.SetTrustedProxies(nil)

	healthHandler := handlers.NewHealthHandler()
	router.GET("/health", healthHandler.HealthCheck)

	titlesHandler := handlers.NewTitlesHandler(tmdbService)
	watchEntriesHandler := handlers.NewWatchEntriesHandler(watchEntrySvc)
	watchEntryHandler := handlers.NewWatchEntryHandler(watchEntrySvc)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/titles/search", titlesHandler.Search)
		v1.POST("/watch-entries", watchEntriesHandler.Create)
		v1.GET("/watch-entries", watchEntriesHandler.List)
		v1.GET("/watch-entry", watchEntryHandler.Get)
	}

	log.Printf("starting server on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
