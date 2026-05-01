package config

import (
	"fmt"
	"os"
)

type Config struct {
	TMDBAPIKey  string
	TMDBBaseURL string
	Port        string
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	return &Config{
		TMDBAPIKey:  apiKey,
		TMDBBaseURL: baseURL,
		Port:        port,
	}, nil
}
