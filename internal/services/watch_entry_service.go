package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

var ErrNotFound = errors.New("watch entry not found")
var ErrTMDBUnavailable = errors.New("TMDB unavailable")

type watchEntryStore interface {
	FindExistingTmdbIDs(ctx context.Context, tmdbIDs []int) (map[int]struct{}, error)
	BulkCreate(ctx context.Context, entries []models.WatchedMovie) ([]models.WatchedMovie, error)
	CountAll(ctx context.Context) (int, error)
	ListAll(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]models.WatchedMovie, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.WatchedMovie, error)
	FindByTmdbID(ctx context.Context, tmdbID int) (*models.WatchedMovie, error)
}

type tmdbDetailer interface {
	GetMovieDetails(ctx context.Context, movieID int) (*models.TMDBMovieDetails, error)
}

// CreateParams carries caller-supplied fields for one watch entry.
type CreateParams struct {
	TmdbID        int
	Title         string
	ReleaseDate   *time.Time
	MyRating      *int
	MyOverview    *string
	MyDateWatched *time.Time
}

// CreateResult is returned by BulkCreate.
type CreateResult struct {
	Created []models.WatchEntryResponse
	Skipped []models.WatchEntrySkipped
}

// ListParams carries validated, defaulted pagination and sort parameters.
type ListParams struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// Lookup identifies a watch entry by exactly one key.
type Lookup struct {
	ID     *uuid.UUID
	TmdbID *int
}

type WatchEntryService struct {
	store    watchEntryStore
	detailer tmdbDetailer
}

func NewWatchEntryService(store watchEntryStore, detailer tmdbDetailer) *WatchEntryService {
	return &WatchEntryService{store: store, detailer: detailer}
}

func (s *WatchEntryService) BulkCreate(ctx context.Context, items []CreateParams) (CreateResult, error) {
	tmdbIDs := make([]int, len(items))
	for i, it := range items {
		tmdbIDs[i] = it.TmdbID
	}

	existing, err := s.store.FindExistingTmdbIDs(ctx, tmdbIDs)
	if err != nil {
		return CreateResult{}, fmt.Errorf("finding existing entries: %w", err)
	}

	var toCreate []models.WatchedMovie
	skipped := []models.WatchEntrySkipped{}

	for _, it := range items {
		if _, dup := existing[it.TmdbID]; dup {
			skipped = append(skipped, models.WatchEntrySkipped{
				TmdbID: it.TmdbID,
				Title:  it.Title,
				Reason: "already exists",
			})
			continue
		}
		toCreate = append(toCreate, models.WatchedMovie{
			ID:            uuid.New(),
			TmdbID:        it.TmdbID,
			Title:         it.Title,
			MyRating:      it.MyRating,
			MyOverview:    it.MyOverview,
			ReleaseDate:   it.ReleaseDate,
			MyDateWatched: it.MyDateWatched,
		})
	}

	if len(toCreate) == 0 {
		return CreateResult{Created: []models.WatchEntryResponse{}, Skipped: skipped}, nil
	}

	created, err := s.store.BulkCreate(ctx, toCreate)
	if err != nil {
		return CreateResult{}, fmt.Errorf("bulk creating entries: %w", err)
	}

	responses := make([]models.WatchEntryResponse, len(created))
	for i, m := range created {
		responses[i] = m.ToResponse()
	}

	return CreateResult{Created: responses, Skipped: skipped}, nil
}

func (s *WatchEntryService) List(ctx context.Context, p ListParams) (models.WatchEntryListResponse, error) {
	total, err := s.store.CountAll(ctx)
	if err != nil {
		return models.WatchEntryListResponse{}, fmt.Errorf("counting entries: %w", err)
	}

	entries, err := s.store.ListAll(ctx, p.Limit, p.Offset, p.SortBy, p.SortOrder)
	if err != nil {
		return models.WatchEntryListResponse{}, fmt.Errorf("listing entries: %w", err)
	}

	items := make([]models.WatchEntryListItem, len(entries))
	for i, m := range entries {
		items[i] = m.ToListItem()
	}

	return models.WatchEntryListResponse{
		Items:  items,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}, nil
}

// Get returns ErrNotFound when no entry matches, ErrTMDBUnavailable when the
// TMDB call fails, or a plain error for DB failures.
func (s *WatchEntryService) Get(ctx context.Context, l Lookup) (*models.WatchEntryDetailResponse, error) {
	var movie *models.WatchedMovie
	var err error

	if l.ID != nil {
		movie, err = s.store.FindByID(ctx, *l.ID)
	} else {
		movie, err = s.store.FindByTmdbID(ctx, *l.TmdbID)
	}
	if err != nil {
		return nil, fmt.Errorf("finding watch entry: %w", err)
	}
	if movie == nil {
		return nil, ErrNotFound
	}

	details, err := s.detailer.GetMovieDetails(ctx, movie.TmdbID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBUnavailable, err)
	}

	resp := &models.WatchEntryDetailResponse{
		ID:         movie.ID,
		TmdbID:     movie.TmdbID,
		Title:      movie.Title,
		MyRating:   movie.MyRating,
		MyOverview: movie.MyOverview,
		CreatedAt:  movie.CreatedAt,
	}
	if movie.ReleaseDate != nil {
		d := models.Date{Time: *movie.ReleaseDate}
		resp.ReleaseDate = &d
	}
	if movie.MyDateWatched != nil {
		d := models.Date{Time: *movie.MyDateWatched}
		resp.MyDateWatched = &d
	}
	if details != nil {
		resp.Overview = details.Overview
		resp.Runtime = details.Runtime
		resp.PosterPath = details.PosterPath
		resp.VoteAverage = details.VoteAverage
	}

	return resp, nil
}

func (s *WatchEntryService) ExistsByTmdbID(ctx context.Context, tmdbID int) (bool, error) {
	movie, err := s.store.FindByTmdbID(ctx, tmdbID)
	if err != nil {
		return false, fmt.Errorf("finding watch entry: %w", err)
	}
	return movie != nil, nil
}