package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

type ActorRepository interface {
	FindByIDs(ctx context.Context, ids []int32) ([]models.Actor, error)
}

type pgxActorRepository struct {
	pool *pgxpool.Pool
}

func NewActorRepository(pool *pgxpool.Pool) ActorRepository {
	return &pgxActorRepository{pool: pool}
}

func (r *pgxActorRepository) FindByIDs(ctx context.Context, ids []int32) ([]models.Actor, error) {
	if len(ids) == 0 {
		return []models.Actor{}, nil
	}

	rows, err := r.pool.Query(ctx,
		"SELECT id, name, popularity, biography, birthday FROM actors WHERE id = ANY($1)",
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("finding actors by ids: %w", err)
	}
	defer rows.Close()

	var actors []models.Actor
	for rows.Next() {
		var a models.Actor
		if err := rows.Scan(&a.ID, &a.Name, &a.Popularity, &a.Biography, &a.Birthday); err != nil {
			return nil, fmt.Errorf("scanning actor: %w", err)
		}
		actors = append(actors, a)
	}
	return actors, rows.Err()
}
