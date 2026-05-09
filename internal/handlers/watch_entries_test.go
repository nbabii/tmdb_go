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
)

type mockWatchEntryRepo struct {
	existingIDs    map[int]struct{}
	findErr        error
	bulkCreated    []models.WatchedMovie
	bulkErr        error
	countTotal     int
	countErr       error
	listEntries    []models.WatchedMovie
	listErr        error
	capturedCreate []models.WatchedMovie
}

func (m *mockWatchEntryRepo) FindExistingTmdbIDs(_ context.Context, _ []int) (map[int]struct{}, error) {
	return m.existingIDs, m.findErr
}

func (m *mockWatchEntryRepo) BulkCreate(_ context.Context, entries []models.WatchedMovie) ([]models.WatchedMovie, error) {
	m.capturedCreate = entries
	return m.bulkCreated, m.bulkErr
}

func (m *mockWatchEntryRepo) CountAll(_ context.Context) (int, error) {
	return m.countTotal, m.countErr
}

func (m *mockWatchEntryRepo) ListAll(_ context.Context, _, _ int, _, _ string) ([]models.WatchedMovie, error) {
	return m.listEntries, m.listErr
}

func newWatchEntriesRouter(repo *mockWatchEntryRepo) *gin.Engine {
	r := gin.New()
	h := NewWatchEntriesHandler(repo)
	r.POST("/watch-entries", h.Create)
	r.GET("/watch-entries", h.List)
	return r
}

func TestWatchEntriesCreate(t *testing.T) {
	someID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	cases := []struct {
		name       string
		body       string
		repo       *mockWatchEntryRepo
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:       "empty array",
			body:       `[]`,
			repo:       &mockWatchEntryRepo{existingIDs: map[int]struct{}{}},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing tmdb_id",
			body:       `[{"title":"Movie"}]`,
			repo:       &mockWatchEntryRepo{existingIDs: map[int]struct{}{}},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing title",
			body:       `[{"tmdb_id":1}]`,
			repo:       &mockWatchEntryRepo{existingIDs: map[int]struct{}{}},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "rating above max",
			body:       `[{"tmdb_id":1,"title":"Movie","my_rating":11}]`,
			repo:       &mockWatchEntryRepo{existingIDs: map[int]struct{}{}},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "rating below min",
			body:       `[{"tmdb_id":1,"title":"Movie","my_rating":0}]`,
			repo:       &mockWatchEntryRepo{existingIDs: map[int]struct{}{}},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "single create no duplicates",
			body: `[{"tmdb_id":1,"title":"Movie"}]`,
			repo: &mockWatchEntryRepo{
				existingIDs: map[int]struct{}{},
				bulkCreated: []models.WatchedMovie{{ID: someID, TmdbID: 1, Title: "Movie"}},
			},
			wantStatus: http.StatusCreated,
			check: func(t *testing.T, body []byte) {
				var result models.WatchEntryBulkResult
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("unmarshal: %v", err)
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
			name: "multiple creates",
			body: `[{"tmdb_id":1,"title":"A"},{"tmdb_id":2,"title":"B"}]`,
			repo: &mockWatchEntryRepo{
				existingIDs: map[int]struct{}{},
				bulkCreated: []models.WatchedMovie{
					{ID: someID, TmdbID: 1, Title: "A"},
					{ID: uuid.New(), TmdbID: 2, Title: "B"},
				},
			},
			wantStatus: http.StatusCreated,
			check: func(t *testing.T, body []byte) {
				var result models.WatchEntryBulkResult
				json.Unmarshal(body, &result)
				if len(result.Created) != 2 {
					t.Errorf("created: got %d, want 2", len(result.Created))
				}
			},
		},
		{
			name: "some duplicates",
			body: `[{"tmdb_id":1,"title":"A"},{"tmdb_id":2,"title":"B"}]`,
			repo: &mockWatchEntryRepo{
				existingIDs: map[int]struct{}{1: {}},
				bulkCreated: []models.WatchedMovie{{ID: someID, TmdbID: 2, Title: "B"}},
			},
			wantStatus: http.StatusMultiStatus,
			check: func(t *testing.T, body []byte) {
				var result models.WatchEntryBulkResult
				json.Unmarshal(body, &result)
				if len(result.Created) != 1 {
					t.Errorf("created: got %d, want 1", len(result.Created))
				}
				if len(result.Skipped) != 1 {
					t.Errorf("skipped: got %d, want 1", len(result.Skipped))
				}
			},
		},
		{
			name: "all duplicates",
			body: `[{"tmdb_id":1,"title":"A"}]`,
			repo: &mockWatchEntryRepo{
				existingIDs: map[int]struct{}{1: {}},
			},
			wantStatus: http.StatusConflict,
			check: func(t *testing.T, body []byte) {
				var resp struct {
					Detail []models.WatchEntrySkipped `json:"detail"`
				}
				json.Unmarshal(body, &resp)
				if len(resp.Detail) != 1 {
					t.Errorf("detail: got %d, want 1", len(resp.Detail))
				}
			},
		},
		{
			name:       "repo find error",
			body:       `[{"tmdb_id":1,"title":"A"}]`,
			repo:       &mockWatchEntryRepo{findErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "repo bulk create error",
			body: `[{"tmdb_id":1,"title":"A"}]`,
			repo: &mockWatchEntryRepo{
				existingIDs: map[int]struct{}{},
				bulkErr:     errors.New("db down"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newWatchEntriesRouter(tc.repo)

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

func TestWatchEntriesList(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		repo       *mockWatchEntryRepo
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:  "defaults applied",
			query: "",
			repo: &mockWatchEntryRepo{
				countTotal:  3,
				listEntries: []models.WatchedMovie{{ID: uuid.New(), TmdbID: 1, Title: "A"}},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var resp models.WatchEntryListResponse
				json.Unmarshal(body, &resp)
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
			repo:       &mockWatchEntryRepo{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "limit above max",
			query:      "?limit=101",
			repo:       &mockWatchEntryRepo{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "negative offset",
			query:      "?offset=-1",
			repo:       &mockWatchEntryRepo{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid sort_by",
			query:      "?sort_by=title",
			repo:       &mockWatchEntryRepo{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid sort_order",
			query:      "?sort_order=random",
			repo:       &mockWatchEntryRepo{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:  "total from CountAll not len(items)",
			query: "?limit=1",
			repo: &mockWatchEntryRepo{
				countTotal:  99,
				listEntries: []models.WatchedMovie{{ID: uuid.New(), TmdbID: 1, Title: "A"}},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				var resp models.WatchEntryListResponse
				json.Unmarshal(body, &resp)
				if resp.Total != 99 {
					t.Errorf("total: got %d, want 99", resp.Total)
				}
			},
		},
		{
			name:       "count error",
			query:      "",
			repo:       &mockWatchEntryRepo{countErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "list error",
			query:      "",
			repo:       &mockWatchEntryRepo{countTotal: 1, listErr: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newWatchEntriesRouter(tc.repo)

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
