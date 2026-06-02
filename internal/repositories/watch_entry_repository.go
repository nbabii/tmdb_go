package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

type WatchEntryRepository interface {
	FindExistingTmdbIDs(ctx context.Context, tmdbIDs []int) (map[int]struct{}, error)
	BulkCreate(ctx context.Context, entries []models.WatchedMovie) ([]models.WatchedMovie, error)
	CountAll(ctx context.Context) (int, error)
	ListAll(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]models.WatchedMovie, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.WatchedMovie, error)
	FindByTmdbID(ctx context.Context, tmdbID int) (*models.WatchedMovie, error)
}

type pgxWatchEntryRepository struct {
	pool *pgxpool.Pool
}

func NewWatchEntryRepository(pool *pgxpool.Pool) WatchEntryRepository {
	return &pgxWatchEntryRepository{pool: pool}
}

func (r *pgxWatchEntryRepository) FindExistingTmdbIDs(ctx context.Context, tmdbIDs []int) (map[int]struct{}, error) {
	ids := make([]int32, len(tmdbIDs))
	for i, id := range tmdbIDs {
		ids[i] = int32(id)
	}

	rows, err := r.pool.Query(ctx,
		"SELECT tmdb_id FROM watched_movies WHERE tmdb_id = ANY($1)",
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("finding existing tmdb ids: %w", err)
	}
	defer rows.Close()

	existing := make(map[int]struct{})
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning tmdb id: %w", err)
		}
		existing[id] = struct{}{}
	}
	return existing, rows.Err()
}

func (r *pgxWatchEntryRepository) BulkCreate(ctx context.Context, entries []models.WatchedMovie) ([]models.WatchedMovie, error) {
	if len(entries) == 0 {
		return []models.WatchedMovie{}, nil
	}

	// Build dynamic multi-row INSERT.
	// 7 columns × N rows: ($1,$2,$3,$4,$5,$6,$7), ($8,$9,$10,$11,$12,$13,$14), ...
	var sb strings.Builder
	sb.WriteString(`INSERT INTO watched_movies (id, tmdb_id, title, release_date, my_rating, my_overview, my_date_watched) VALUES `)

	args := make([]any, 0, len(entries)*7)
	for i, e := range entries {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 7
		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		args = append(args, e.ID, e.TmdbID, e.Title, e.ReleaseDate, e.MyRating, e.MyOverview, e.MyDateWatched)
	}
	sb.WriteString(` RETURNING id, tmdb_id, title, release_date, my_rating, my_overview, my_date_watched, created_at`)

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("bulk create: %w", err)
	}
	defer rows.Close()

	var result []models.WatchedMovie
	for rows.Next() {
		m, err := scanWatchedMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning created entry: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *pgxWatchEntryRepository) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM watched_movies").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting watch entries: %w", err)
	}
	return count, nil
}

func (r *pgxWatchEntryRepository) ListAll(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]models.WatchedMovie, error) {
	// sortBy and sortOrder are validated by the handler before reaching here.
	// Using fmt.Sprintf is safe because values come from a bounded whitelist, not raw user input.
	allowedColumns := map[string]string{
		"my_rating":       "my_rating",
		"my_date_watched": "my_date_watched",
	}
	allowedOrders := map[string]string{
		"asc":  "ASC",
		"desc": "DESC",
	}
	col := allowedColumns[sortBy]
	ord := allowedOrders[sortOrder]

	q := fmt.Sprintf(`
		SELECT id, tmdb_id, title, release_date, my_rating, my_overview, my_date_watched, created_at
		FROM watched_movies
		ORDER BY %s %s NULLS LAST
		LIMIT $1 OFFSET $2`, col, ord)

	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing watch entries: %w", err)
	}
	defer rows.Close()

	var entries []models.WatchedMovie
	for rows.Next() {
		m, err := scanWatchedMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning watch entry: %w", err)
		}
		entries = append(entries, m)
	}
	return entries, rows.Err()
}

func (r *pgxWatchEntryRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.WatchedMovie, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tmdb_id, title, release_date, my_rating, my_overview, my_date_watched, created_at
		FROM watched_movies WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("finding watch entry by id: %w", err)
	}
	defer rows.Close()
	return scanOptionalWatchedMovie(rows)
}

func (r *pgxWatchEntryRepository) FindByTmdbID(ctx context.Context, tmdbID int) (*models.WatchedMovie, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tmdb_id, title, release_date, my_rating, my_overview, my_date_watched, created_at
		FROM watched_movies WHERE tmdb_id = $1`, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("finding watch entry by tmdb id: %w", err)
	}
	defer rows.Close()
	return scanOptionalWatchedMovie(rows)
}

func scanWatchedMovie(rows pgx.Rows) (models.WatchedMovie, error) {
	var m models.WatchedMovie
	err := rows.Scan(
		&m.ID,
		&m.TmdbID,
		&m.Title,
		&m.ReleaseDate,
		&m.MyRating,
		&m.MyOverview,
		&m.MyDateWatched,
		&m.CreatedAt,
	)
	return m, err
}

func scanOptionalWatchedMovie(rows pgx.Rows) (*models.WatchedMovie, error) {
	if !rows.Next() {
		return nil, nil
	}	
	m, err := scanWatchedMovie(rows)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
