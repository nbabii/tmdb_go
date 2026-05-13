package config

import (
	"fmt"
	"os"
)

type Config struct {
	TMDBAPIKey  string
	TMDBBaseURL string
	Port        string
	DatabaseURL string
}

func Load() (*Config, error) {
	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TMDB_API_KEY environment variable is required")
	}

	baseURL := os.Getenv("TMDB_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("TMDB_BASE_URL environment variable is required")
	}

	databaseURL := os.Getenv("GO_DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("GO_DATABASE_URL environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	return &Config{
		TMDBAPIKey:  apiKey,
		TMDBBaseURL: baseURL,
		Port:        port,
		DatabaseURL: databaseURL,
	}, nil
}
