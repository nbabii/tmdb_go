package main

import (
	"github.com/gin-gonic/gin"
	"github.com/nazarbabii/tmdb_go/internal/handlers"
)

func main() {
	router := gin.Default()

	healthHandler := handlers.NewHealthHandler()
	router.GET("/health", healthHandler.HealthCheck)

	router.Run(":8088")
}
