package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nazarbabii/tmdb_go/internal/config"
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

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.SetTrustedProxies(nil)

	healthHandler := handlers.NewHealthHandler()
	router.GET("/health", healthHandler.HealthCheck)

	titlesHandler := handlers.NewTitlesHandler(tmdbService)
	v1 := router.Group("/api/v1")
	{
		v1.GET("/titles/search", titlesHandler.Search)
	}

	log.Printf("starting server on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
