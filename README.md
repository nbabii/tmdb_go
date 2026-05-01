# TMDB Go - Movie Tracking API

A REST API built with Go and Gin framework for managing and tracking watched movies using The Movie Database (TMDB) API and PostgreSQL.

## Features

- Fetch movie information from TMDB API
- Store and manage watched movies in PostgreSQL database
- RESTful API endpoints for movie operations
- Health check endpoint for monitoring

## Tech Stack

- **Language:** Go
- **Web Framework:** Gin
- **Database:** PostgreSQL
- **External API:** TMDB (The Movie Database)

## Project Structure

```
.
├── cmd/
│   └── api/              # Application entry point
├── configs/              # Configuration file templates
├── internal/
│   ├── config/           # Environment variable loading
│   ├── database/         # Database connection & repositories
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # Custom middleware
│   ├── models/           # Data models
│   └── services/         # Business logic & TMDB API client
├── pkg/                  # Public libraries
└── go.mod
```

## Prerequisites

- Go 1.21 or higher
- PostgreSQL (coming soon)
- TMDB API key (coming soon)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/nazarbabii/tmdb_go.git
cd tmdb_go
```

2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run cmd/api/main.go
```

The server will start on `http://localhost:8088`

## API Endpoints

### Health Check
```
GET /health
```

Returns the health status of the service.

**Response:**
```json
{
  "status": "ok",
  "message": "Service is running"
}
```

## Testing

Run all tests:
```bash
go test ./...
```

Run with verbose output to see each test case:
```bash
go test ./... -v
```

Run tests for a specific package:
```bash
go test ./internal/handlers/...
go test ./internal/services/...
go test ./internal/config/...
```

Run a single test by name:
```bash
go test ./internal/handlers/... -run TestTitlesSearch/missing_query_param
```

## Development

Run the server in development mode:
```bash
go run cmd/api/main.go
```

## License

MIT
