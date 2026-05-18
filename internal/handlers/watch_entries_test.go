package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/models"
	"github.com/nazarbabii/tmdb_go/internal/services"
)

type mockWatchEntriesService struct {
	createResult services.CreateResult
	createErr    error
	listResult   models.WatchEntryListResponse
	listErr      error
	existsResult bool
	existsErr    error
}

func (m *mockWatchEntriesService) BulkCreate(_ context.Context, _ []services.CreateParams) (services.CreateResult, error) {
	return m.createResult, m.createErr
}

func (m *mockWatchEntriesService) List(_ context.Context, _ services.ListParams) (models.WatchEntryListResponse, error) {
	return m.listResult, m.listErr
}

func (m *mockWatchEntriesService) ExistsByTmdbID(_ context.Context, _ int) (bool, error) {
	return m.existsResult, m.existsErr
}

func newWatchEntriesRouter(svc *mockWatchEntriesService) *gin.Engine {
	r := gin.New()
	h := NewWatchEntriesHandler(svc)
	r.POST("/watch-entries", h.Create)
	r.GET("/watch-entries", h.List)
	r.GET("/watch-entries/exists", h.Exists)
	return r
}

func TestWatchEntriesCreate(t *testing.T) {
	someID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	cases := []struct {
		name       string
		body       string
		svc        *mockWatchEntriesService
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:       "empty array",
			body:       `[]`,
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing tmdb_id",
			body:       `[{"title":"Movie"}]`,
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing title",
			body:       `[{"tmdb_id":1}]`,
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "rating above max",
			body:       `[{"tmdb_id":1,"title":"Movie","my_rating":11}]`,
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "rating below min",
			body:       `[{"tmdb_id":1,"title":"Movie","my_rating":0}]`,
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "all created → 201",
			body: `[{"tmdb_id":1,"title":"Movie"}]`,
			svc: &mockWatchEntriesService{
				createResult: services.CreateResult{
					Created: []models.WatchEntryResponse{{ID: someID, TmdbID: 1, Title: "Movie"}},
					Skipped: []models.WatchEntrySkipped{},
				},
			},
			wantStatus: http.StatusCreated,
			check: func(t *testing.T, body []byte) {
				var result models.WatchEntryBulkResult
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("parsing response: %v", err)
				}
				if len(result.Created) != 1 {
					t.Errorf("created: got %d, want 1", len(result.Created))
				}
				if len(result.Skipped) != 0 {
					t.Errorf("skipped: got %d, want 0", len(result.Skipped))
				}
			},
		},
		{
			name: "some skipped → 207",
			body: `[{"tmdb_id":1,"title":"A"},{"tmdb_id":2,"title":"B"}]`,
			svc: &mockWatchEntriesService{
				createResult: services.CreateResult{
					Created: []models.WatchEntryResponse{{ID: someID, TmdbID: 2, Title: "B"}},
					Skipped: []models.WatchEntrySkipped{{TmdbID: 1, Title: "A", Reason: "already exists"}},
				},
			},
			wantStatus: http.StatusMultiStatus,
		},
		{
			name: "all skipped → 409 with detail shape",
			body: `[{"tmdb_id":1,"title":"A"}]`,
			svc: &mockWatchEntriesService{
				createResult: services.CreateResult{
					Created: []models.WatchEntryResponse{},
					Skipped: []models.WatchEntrySkipped{{TmdbID: 1, Title: "A", Reason: "already exists"}},
				},
			},
			wantStatus: http.StatusConflict,
			check: func(t *testing.T, body []byte) {
				var resp struct {
					Detail []models.WatchEntrySkipped `json:"detail"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("parsing response: %v", err)
				}
				if len(resp.Detail) != 1 {
					t.Errorf("detail: got %d, want 1", len(resp.Detail))
				}
			},
		},
		{
			name:       "service error → 500",
			body:       `[{"tmdb_id":1,"title":"A"}]`,
			svc:        &mockWatchEntriesService{createErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newWatchEntriesRouter(tc.svc)

			req := httptest.NewRequest(http.MethodPost, "/watch-entries", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d — body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.check != nil {
				tc.check(t, w.Body.Bytes())
			}
		})
	}
}

func TestWatchEntriesExists(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		svc        *mockWatchEntriesService
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:       "missing tmdb_id → 422",
			query:      "",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "tmdb_id=0 below min=1 → 422",
			query:      "?tmdb_id=0",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "tmdb_id=abc non-numeric → 422",
			query:      "?tmdb_id=abc",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "tmdb_id=! symbol → 422",
			query:      "?tmdb_id=!",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "found → 200 exists:true",
			query:        "?tmdb_id=1",
			svc:          &mockWatchEntriesService{existsResult: true},
			wantStatus:   http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var resp struct {
					Exists bool `json:"exists"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("parsing response: %v", err)
				}
				if !resp.Exists {
					t.Error("exists: got false, want true")
				}
			},
		},
		{
			name:         "not found → 200 exists:false",
			query:        "?tmdb_id=1",
			svc:          &mockWatchEntriesService{existsResult: false},
			wantStatus:   http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var resp struct {
					Exists bool `json:"exists"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("parsing response: %v", err)
				}
				if resp.Exists {
					t.Error("exists: got true, want false")
				}
			},
		},
		{
			name:       "service error → 500",
			query:      "?tmdb_id=1",
			svc:        &mockWatchEntriesService{existsErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
			check: func(t *testing.T, body []byte) {
				var resp struct {
					Detail string `json:"detail"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("parsing response: %v", err)
				}
				if resp.Detail != "internal server error" {
					t.Errorf("detail: got %q, want %q", resp.Detail, "internal server error")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newWatchEntriesRouter(tc.svc)

			req := httptest.NewRequest(http.MethodGet, "/watch-entries/exists"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d — body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.check != nil {
				tc.check(t, w.Body.Bytes())
			}
		})
	}
}

func TestWatchEntriesList(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		svc        *mockWatchEntriesService
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:  "defaults applied — response reflects limit=10",
			query: "",
			svc: &mockWatchEntriesService{
				listResult: models.WatchEntryListResponse{
					Items:  []models.WatchEntryListItem{},
					Total:  3,
					Limit:  10,
					Offset: 0,
				},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var resp models.WatchEntryListResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("parsing response: %v", err)
				}
				if resp.Limit != 10 {
					t.Errorf("limit: got %d, want 10", resp.Limit)
				}
				if resp.Total != 3 {
					t.Errorf("total: got %d, want 3", resp.Total)
				}
			},
		},
		{
			name:       "explicit limit=0",
			query:      "?limit=0",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "limit above max",
			query:      "?limit=101",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "negative offset",
			query:      "?offset=-1",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid sort_by",
			query:      "?sort_by=title",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid sort_order",
			query:      "?sort_order=random",
			svc:        &mockWatchEntriesService{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "service error → 500",
			query:      "",
			svc:        &mockWatchEntriesService{listErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newWatchEntriesRouter(tc.svc)

			req := httptest.NewRequest(http.MethodGet, "/watch-entries"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d — body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.check != nil {
				tc.check(t, w.Body.Bytes())
			}
		})
	}
}
