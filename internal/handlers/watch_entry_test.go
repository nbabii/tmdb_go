package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/models"
	"github.com/nazarbabii/tmdb_go/internal/services"
)

type mockWatchEntryService struct {
	result         *models.WatchEntryDetailResponse
	err            error
	capturedLookup services.Lookup
}

func (m *mockWatchEntryService) Get(_ context.Context, l services.Lookup) (*models.WatchEntryDetailResponse, error) {
	m.capturedLookup = l
	return m.result, m.err
}

func newWatchEntryRouter(svc *mockWatchEntryService) *gin.Engine {
	r := gin.New()
	h := NewWatchEntryHandler(svc)
	r.GET("/watch-entry", h.Get)
	return r
}

var validUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func sampleDetailResponse() *models.WatchEntryDetailResponse {
	rating := 8
	overview := "tmdb overview"
	runtime := 120
	poster := "/poster.jpg"
	avg := 7.5
	return &models.WatchEntryDetailResponse{
		ID:          validUUID,
		TmdbID:      42,
		Title:       "Test Movie",
		MyRating:    &rating,
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
		svc        *mockWatchEntryService
		wantStatus int
		check      func(t *testing.T, svc *mockWatchEntryService)
	}{
		{
			name:       "neither param → 400",
			query:      "",
			svc:        &mockWatchEntryService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid UUID → 400",
			query:      "?id=not-a-uuid",
			svc:        &mockWatchEntryService{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found → 404",
			query:      "?id=" + validUUID.String(),
			svc:        &mockWatchEntryService{err: services.ErrNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "TMDB error → 502",
			query:      "?id=" + validUUID.String(),
			svc:        &mockWatchEntryService{err: fmt.Errorf("%w: timeout", services.ErrTMDBUnavailable)},
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "db error → 500",
			query:      "?id=" + validUUID.String(),
			svc:        &mockWatchEntryService{err: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "success by id → 200",
			query:      "?id=" + validUUID.String(),
			svc:        &mockWatchEntryService{result: sampleDetailResponse()},
			wantStatus: http.StatusOK,
		},
		{
			name:       "success by tmdb_id → 200",
			query:      "?tmdb_id=42",
			svc:        &mockWatchEntryService{result: sampleDetailResponse()},
			wantStatus: http.StatusOK,
		},
		{
			name:       "id takes precedence — Lookup.ID set, Lookup.TmdbID nil",
			query:      "?id=" + validUUID.String() + "&tmdb_id=42",
			svc:        &mockWatchEntryService{result: sampleDetailResponse()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, svc *mockWatchEntryService) {
				if svc.capturedLookup.ID == nil {
					t.Error("expected Lookup.ID to be set")
				}
				if svc.capturedLookup.TmdbID != nil {
					t.Error("expected Lookup.TmdbID to be nil")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newWatchEntryRouter(tc.svc)

			req := httptest.NewRequest(http.MethodGet, "/watch-entry"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d — body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.check != nil {
				tc.check(t, tc.svc)
			}
		})
	}
}
