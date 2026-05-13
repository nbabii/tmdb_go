# TMDB Go — Movie Tracking API

A REST API built with Go and Gin for managing a personal watched-movies list, backed by PostgreSQL and the [TMDB API](https://developer.themoviedb.org/).

## Tech Stack

- **Language:** Go 1.26
- **Framework:** Gin
- **Database:** PostgreSQL via `pgx/v5` (no ORM)
- **External API:** TMDB (The Movie Database)

## Project Structure

```
cmd/api/          entry point — wiring and graceful shutdown
internal/
  config/         environment variable loading, fails fast on missing required vars
  server/         route registration (NewRouter)
  middleware/     request_id, logger, recovery
  handlers/       HTTP layer — bind, validate, call service, return JSON
  services/       business logic — TMDBService, WatchEntryService
  repositories/   database access — WatchEntryRepository (pgx)
  database/       connection pool init (NewPool)
  models/         shared types: DB models, JSON request/response shapes
migrations/       SQL migration files (golang-migrate)
```

## Prerequisites

- Go 1.26+
- PostgreSQL
- TMDB API key — get one at [themoviedb.org](https://www.themoviedb.org/settings/api)
- [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI for running migrations

## Setup

1. Clone and install dependencies:
```bash
git clone https://github.com/nazarbabii/tmdb_go.git
cd tmdb_go
go mod download
```

2. Copy and fill in environment variables:
```bash
cp .env.example .env
```

| Variable | Required | Notes |
|---|---|---|
| `TMDB_API_KEY` | yes | Bearer token from TMDB |
| `TMDB_BASE_URL` | yes | `https://api.themoviedb.org/3` |
| `GO_DATABASE_URL` | yes | `postgres://user:pass@host:5432/db` |
| `PORT` | no | defaults to `8088` |

3. Run database migrations:
```bash
migrate -path ./migrations -database $GO_DATABASE_URL up
```

4. Start the server:
```bash
go run ./cmd/api/main.go
```

Server starts on `http://localhost:8088`.

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/api/v1/titles/search` | Search movies or TV shows via TMDB |
| POST | `/api/v1/watch-entries` | Bulk-add movies to your watch list |
| GET | `/api/v1/watch-entries` | List watched movies (paginated, sortable) |
| GET | `/api/v1/watch-entry` | Get a single entry by `?id=<uuid>` or `?tmdb_id=<int>` |

### Search titles

```
GET /api/v1/titles/search?query=inception&type=movie&page=1&year=2010
```

Query params: `query` (required), `type` (`movie` or `tv`, required), `page` (1–500), `year`.

### Add to watch list

```
POST /api/v1/watch-entries
Content-Type: application/json

[
  {
    "tmdb_id": 27205,
    "title": "Inception",
    "release_date": "2010-07-16",
    "my_rating": 9,
    "my_overview": "Mind-bending.",
    "my_date_watched": "2024-01-15"
  }
]
```

Duplicate `tmdb_id` entries are skipped and reported in the `skipped` array.

### List watch entries

```
GET /api/v1/watch-entries?limit=20&offset=0&sort_by=my_rating&sort_order=desc
```

Sort options: `sort_by` = `my_rating` or `my_date_watched`; `sort_order` = `asc` or `desc`.

## Testing

```bash
go test ./...                              # all tests
go test ./internal/handlers/... -v        # handlers with verbose output
go test ./internal/handlers/... -run TestTitlesSearch/missing_query_param
```

## License

MIT
