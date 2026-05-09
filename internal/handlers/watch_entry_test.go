package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

type mockWatchEntryLookup struct {
	byIDResult    *models.WatchedMovie
	byIDErr       error
	byTmdbResult  *models.WatchedMovie
	byTmdbErr     error
	calledByID    bool
	calledByTmdb  bool
}

func (m *mockWatchEntryLookup) FindByID(_ context.Context, _ uuid.UUID) (*models.WatchedMovie, error) {
	m.calledByID = true
	return m.byIDResult, m.byIDErr
}

func (m *mockWatchEntryLookup) FindByTmdbID(_ context.Context, _ int) (*models.WatchedMovie, error) {
	m.calledByTmdb = true
	return m.byTmdbResult, m.byTmdbErr
}

type mockMovieDetailer struct {
	result *models.TMDBMovieDetails
	err    error
}

func (m *mockMovieDetailer) GetMovieDetails(_ context.Context, _ int) (*models.TMDBMovieDetails, error) {
	return m.result, m.err
}

func newWatchEntryRouter(repo *mockWatchEntryLookup, tmdb *mockMovieDetailer) *gin.Engine {
	r := gin.New()
	h := NewWatchEntryHandler(repo, tmdb)
	r.GET("/watch-entry", h.Get)
	return r
}

var validUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func sampleMovie() *models.WatchedMovie {
	rating := 8
	overview := "great film"
	now := time.Now()
	return &models.WatchedMovie{
		ID:         validUUID,
		TmdbID:     42,
		Title:      "Test Movie",
		MyRating:   &rating,
		MyOverview: &overview,
		CreatedAt:  now,
	}
}

func sampleDetails() *models.TMDBMovieDetails {
	overview := "tmdb overview"
	runtime := 120
	poster := "/poster.jpg"
	avg := 7.5
	return &models.TMDBMovieDetails{
		Overview:    &overview,
		Runtime:     &runtime,
		PosterPath:  &poster,
		VoteAverage: &avg,
	}
}

func TestWatchEntryGet(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		repo       *mockWatchEntryLookup
		tmdb       *mockMovieDetailer
		wantStatus int
		check      func(t *testing.T, repo *mockWatchEntryLookup)
	}{
		{
			name:       "neither param",
			query:      "",
			repo:       &mockWatchEntryLookup{},
			tmdb:       &mockMovieDetailer{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid UUID",
			query:      "?id=not-a-uuid",
			repo:       &mockWatchEntryLookup{},
			tmdb:       &mockMovieDetailer{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found by id",
			query:      "?id=" + validUUID.String(),
			repo:       &mockWatchEntryLookup{byIDResult: nil},
			tmdb:       &mockMovieDetailer{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "not found by tmdb_id",
			query:      "?tmdb_id=42",
			repo:       &mockWatchEntryLookup{byTmdbResult: nil},
			tmdb:       &mockMovieDetailer{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "tmdb fetch fails",
			query:      "?id=" + validUUID.String(),
			repo:       &mockWatchEntryLookup{byIDResult: sampleMovie()},
			tmdb:       &mockMovieDetailer{err: errors.New("tmdb down")},
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "success by id",
			query:      "?id=" + validUUID.String(),
			repo:       &mockWatchEntryLookup{byIDResult: sampleMovie()},
			tmdb:       &mockMovieDetailer{result: sampleDetails()},
			wantStatus: http.StatusOK,
		},
		{
			name:       "success by tmdb_id",
			query:      "?tmdb_id=42",
			repo:       &mockWatchEntryLookup{byTmdbResult: sampleMovie()},
			tmdb:       &mockMovieDetailer{result: sampleDetails()},
			wantStatus: http.StatusOK,
		},
		{
			name:  "id takes precedence over tmdb_id",
			query: "?id=" + validUUID.String() + "&tmdb_id=42",
			repo:  &mockWatchEntryLookup{byIDResult: sampleMovie()},
			tmdb:  &mockMovieDetailer{result: sampleDetails()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, repo *mockWatchEntryLookup) {
				if !repo.calledByID {
					t.Error("expected FindByID to be called")
				}
				if repo.calledByTmdb {
					t.Error("expected FindByTmdbID NOT to be called")
				}
			},
		},
		{
			name:       "repo error",
			query:      "?id=" + validUUID.String(),
			repo:       &mockWatchEntryLookup{byIDErr: errors.New("db down")},
			tmdb:       &mockMovieDetailer{},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newWatchEntryRouter(tc.repo, tc.tmdb)

			req := httptest.NewRequest(http.MethodGet, "/watch-entry"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d — body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.check != nil {
				tc.check(t, tc.repo)
			}
		})
	}
}
