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
	"encoding/json"
)

type mockWatchEntryService struct {
	result         *models.WatchEntryDetailResponse
	err            error
	capturedLookup services.Lookup
	existsResult   bool
}

func (m *mockWatchEntryService) Get(_ context.Context, l services.Lookup) (*models.WatchEntryDetailResponse, error) {
	m.capturedLookup = l
	return m.result, m.err
}

func (m *mockWatchEntryService) ExistsByTmdbID(_ context.Context, _ int) (bool, error) {
    return m.existsResult, m.err
}

func newWatchEntryRouter(svc *mockWatchEntryService) *gin.Engine {
	r := gin.New()
	h := NewWatchEntryHandler(svc)
	r.GET("/watch-entry", h.Get)
	r.GET("/watch-entry/:tmdb_id", h.Exists)
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

func TestWatchEntryExists(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		svc        *mockWatchEntryService
		wantStatus int
		check      func(t *testing.T, w *httptest.ResponseRecorder, svc *mockWatchEntryService)
	}{
		{
			name:       "ID found → 200",
			query:      "/42",
			svc:        &mockWatchEntryService{existsResult: true},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, w *httptest.ResponseRecorder, svc *mockWatchEntryService) {
				var body struct {
					Exists bool `json:"exists"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to unmarshal body: %v", err)
				}
				if !body.Exists {
					t.Error("expected exists to be true")
				}
			},
		},
		{
			name:       "ID not found → 200",
			query:      "/42",
			svc:        &mockWatchEntryService{existsResult: false},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, w *httptest.ResponseRecorder, svc *mockWatchEntryService) {
				var body struct {
					Exists bool `json:"exists"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to unmarshal body: %v", err)
				}
				if body.Exists {
					t.Error("expected exists to be false")
				}
			},
		},
		{
			name:       "invalid ID → 400",
			query:      "/test_id",
			svc:        &mockWatchEntryService{},
			wantStatus: http.StatusBadRequest,
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
				tc.check(t, w, tc.svc)
			}
		})
	}
}