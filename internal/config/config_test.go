package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	cases := []struct {
		name        string
		env         map[string]string
		wantErr     string
		wantAPIKey  string
		wantBaseURL string
		wantPort    string
	}{
		{
			name:    "missing API key",
			env:     map[string]string{},
			wantErr: "TMDB_API_KEY environment variable is required",
		},
		{
			name:    "missing base URL",
			env:     map[string]string{"TMDB_API_KEY": "key"},
			wantErr: "TMDB_BASE_URL environment variable is required",
		},
		{
			name: "defaults port to 8088",
			env: map[string]string{
				"TMDB_API_KEY":  "key",
				"TMDB_BASE_URL": "https://api.example.com",
			},
			wantAPIKey:  "key",
			wantBaseURL: "https://api.example.com",
			wantPort:    "8088",
		},
		{
			name: "custom port",
			env: map[string]string{
				"TMDB_API_KEY":  "key",
				"TMDB_BASE_URL": "https://api.example.com",
				"PORT":          "9000",
			},
			wantAPIKey:  "key",
			wantBaseURL: "https://api.example.com",
			wantPort:    "9000",
		},
		{
			name: "all vars set",
			env: map[string]string{
				"TMDB_API_KEY":  "secret",
				"TMDB_BASE_URL": "https://api.themoviedb.org/3",
				"PORT":          "8080",
			},
			wantAPIKey:  "secret",
			wantBaseURL: "https://api.themoviedb.org/3",
			wantPort:    "8080",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TMDB_API_KEY", "")
			t.Setenv("TMDB_BASE_URL", "")
			t.Setenv("PORT", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			cfg, err := Load()

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %q", tc.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.TMDBAPIKey != tc.wantAPIKey {
				t.Errorf("TMDBAPIKey: got %q, want %q", cfg.TMDBAPIKey, tc.wantAPIKey)
			}
			if cfg.TMDBBaseURL != tc.wantBaseURL {
				t.Errorf("TMDBBaseURL: got %q, want %q", cfg.TMDBBaseURL, tc.wantBaseURL)
			}
			if cfg.Port != tc.wantPort {
				t.Errorf("Port: got %q, want %q", cfg.Port, tc.wantPort)
			}
		})
	}
}
