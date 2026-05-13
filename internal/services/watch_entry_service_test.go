package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

// --- mocks ---

type mockStore struct {
	existingIDs    map[int]struct{}
	findErr        error
	bulkCreated    []models.WatchedMovie
	bulkErr        error
	countTotal     int
	countErr       error
	listEntries    []models.WatchedMovie
	listErr        error
	capturedCreate []models.WatchedMovie
	byIDResult     *models.WatchedMovie
	byIDErr        error
	byTmdbResult   *models.WatchedMovie
	byTmdbErr      error
	calledByID     bool
	calledByTmdb   bool
}

func (m *mockStore) FindExistingTmdbIDs(_ context.Context, _ []int) (map[int]struct{}, error) {
	return m.existingIDs, m.findErr
}

func (m *mockStore) BulkCreate(_ context.Context, entries []models.WatchedMovie) ([]models.WatchedMovie, error) {
	m.capturedCreate = entries
	return m.bulkCreated, m.bulkErr
}

func (m *mockStore) CountAll(_ context.Context) (int, error) {
	return m.countTotal, m.countErr
}

func (m *mockStore) ListAll(_ context.Context, _, _ int, _, _ string) ([]models.WatchedMovie, error) {
	return m.listEntries, m.listErr
}

func (m *mockStore) FindByID(_ context.Context, _ uuid.UUID) (*models.WatchedMovie, error) {
	m.calledByID = true
	return m.byIDResult, m.byIDErr
}

func (m *mockStore) FindByTmdbID(_ context.Context, _ int) (*models.WatchedMovie, error) {
	m.calledByTmdb = true
	return m.byTmdbResult, m.byTmdbErr
}

type mockDetailer struct {
	result *models.TMDBMovieDetails
	err    error
}

func (m *mockDetailer) GetMovieDetails(_ context.Context, _ int) (*models.TMDBMovieDetails, error) {
	return m.result, m.err
}

func newSvc(store *mockStore, detailer *mockDetailer) *WatchEntryService {
	return NewWatchEntryService(store, detailer)
}

// --- BulkCreate ---

func TestBulkCreate(t *testing.T) {
	someID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	cases := []struct {
		name    string
		items   []CreateParams
		store   *mockStore
		wantErr bool
		check   func(t *testing.T, result CreateResult, store *mockStore)
	}{
		{
			name:  "all new — UUIDs assigned, responses returned",
			items: []CreateParams{{TmdbID: 1, Title: "A"}},
			store: &mockStore{
				existingIDs: map[int]struct{}{},
				bulkCreated: []models.WatchedMovie{{ID: someID, TmdbID: 1, Title: "A"}},
			},
			check: func(t *testing.T, result CreateResult, store *mockStore) {
				if len(result.Created) != 1 {
					t.Errorf("created: got %d, want 1", len(result.Created))
				}
				if len(result.Skipped) != 0 {
					t.Errorf("skipped: got %d, want 0", len(result.Skipped))
				}
				if len(store.capturedCreate) != 1 {
					t.Fatalf("capturedCreate: got %d, want 1", len(store.capturedCreate))
				}
				if store.capturedCreate[0].ID == (uuid.UUID{}) {
					t.Error("expected non-zero UUID to be assigned")
				}
			},
		},
		{
			name:  "all duplicates — no BulkCreate call",
			items: []CreateParams{{TmdbID: 1, Title: "A"}},
			store: &mockStore{existingIDs: map[int]struct{}{1: {}}},
			check: func(t *testing.T, result CreateResult, store *mockStore) {
				if len(result.Created) != 0 {
					t.Errorf("created: got %d, want 0", len(result.Created))
				}
				if len(result.Skipped) != 1 {
					t.Errorf("skipped: got %d, want 1", len(result.Skipped))
				}
				if store.capturedCreate != nil {
					t.Error("BulkCreate should not have been called")
				}
				if result.Skipped[0].Reason != "already exists" {
					t.Errorf("reason: got %q, want %q", result.Skipped[0].Reason, "already exists")
				}
			},
		},
		{
			name: "some duplicates — only new ones sent to store",
			items: []CreateParams{
				{TmdbID: 1, Title: "A"},
				{TmdbID: 2, Title: "B"},
			},
			store: &mockStore{
				existingIDs: map[int]struct{}{1: {}},
				bulkCreated: []models.WatchedMovie{{ID: someID, TmdbID: 2, Title: "B"}},
			},
			check: func(t *testing.T, result CreateResult, store *mockStore) {
				if len(result.Created) != 1 {
					t.Errorf("created: got %d, want 1", len(result.Created))
				}
				if len(result.Skipped) != 1 {
					t.Errorf("skipped: got %d, want 1", len(result.Skipped))
				}
				if len(store.capturedCreate) != 1 || store.capturedCreate[0].TmdbID != 2 {
					t.Error("only non-duplicate should be sent to store")
				}
			},
		},
		{
			name:    "store find error propagated",
			items:   []CreateParams{{TmdbID: 1, Title: "A"}},
			store:   &mockStore{findErr: errors.New("db down")},
			wantErr: true,
		},
		{
			name:  "store bulk create error propagated",
			items: []CreateParams{{TmdbID: 1, Title: "A"}},
			store: &mockStore{
				existingIDs: map[int]struct{}{},
				bulkErr:     errors.New("db down"),
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc(tc.store, &mockDetailer{})
			result, err := svc.BulkCreate(context.Background(), tc.items)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, result, tc.store)
			}
		})
	}
}

// --- List ---

func TestList(t *testing.T) {
	entry := models.WatchedMovie{ID: uuid.New(), TmdbID: 1, Title: "A"}

	cases := []struct {
		name    string
		params  ListParams
		store   *mockStore
		wantErr bool
		check   func(t *testing.T, result models.WatchEntryListResponse)
	}{
		{
			name:   "assembles response correctly",
			params: ListParams{Limit: 10, Offset: 0, SortBy: "my_rating", SortOrder: "desc"},
			store:  &mockStore{countTotal: 5, listEntries: []models.WatchedMovie{entry}},
			check: func(t *testing.T, result models.WatchEntryListResponse) {
				if result.Total != 5 {
					t.Errorf("total: got %d, want 5", result.Total)
				}
				if len(result.Items) != 1 {
					t.Errorf("items: got %d, want 1", len(result.Items))
				}
				if result.Limit != 10 {
					t.Errorf("limit: got %d, want 10", result.Limit)
				}
			},
		},
		{
			name:    "count error propagated",
			params:  ListParams{Limit: 10},
			store:   &mockStore{countErr: errors.New("db down")},
			wantErr: true,
		},
		{
			name:    "list error propagated",
			params:  ListParams{Limit: 10},
			store:   &mockStore{countTotal: 1, listErr: errors.New("db down")},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc(tc.store, &mockDetailer{})
			result, err := svc.List(context.Background(), tc.params)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, result)
			}
		})
	}
}

// --- Get ---

func TestGet(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tmdbID := 42
	now := time.Now()

	movie := &models.WatchedMovie{
		ID:        id,
		TmdbID:    tmdbID,
		Title:     "Test Movie",
		CreatedAt: now,
	}

	tmdbOverview := "tmdb overview"
	runtime := 120
	poster := "/poster.jpg"
	avg := 7.5
	details := &models.TMDBMovieDetails{
		Overview:    &tmdbOverview,
		Runtime:     &runtime,
		PosterPath:  &poster,
		VoteAverage: &avg,
	}

	cases := []struct {
		name     string
		lookup   Lookup
		store    *mockStore
		detailer *mockDetailer
		wantErr  error
		check    func(t *testing.T, resp *models.WatchEntryDetailResponse, store *mockStore)
	}{
		{
			name:     "FindByID called when ID set",
			lookup:   Lookup{ID: &id},
			store:    &mockStore{byIDResult: movie},
			detailer: &mockDetailer{result: details},
			check: func(t *testing.T, _ *models.WatchEntryDetailResponse, store *mockStore) {
				if !store.calledByID {
					t.Error("expected FindByID to be called")
				}
				if store.calledByTmdb {
					t.Error("expected FindByTmdbID NOT to be called")
				}
			},
		},
		{
			name:     "FindByTmdbID called when TmdbID set",
			lookup:   Lookup{TmdbID: &tmdbID},
			store:    &mockStore{byTmdbResult: movie},
			detailer: &mockDetailer{result: details},
			check: func(t *testing.T, _ *models.WatchEntryDetailResponse, store *mockStore) {
				if !store.calledByTmdb {
					t.Error("expected FindByTmdbID to be called")
				}
				if store.calledByID {
					t.Error("expected FindByID NOT to be called")
				}
			},
		},
		{
			name:     "not found → ErrNotFound",
			lookup:   Lookup{ID: &id},
			store:    &mockStore{byIDResult: nil},
			detailer: &mockDetailer{},
			wantErr:  ErrNotFound,
		},
		{
			name:     "db error propagated as plain error — not ErrNotFound or ErrTMDBUnavailable",
			lookup:   Lookup{ID: &id},
			store:    &mockStore{byIDErr: errors.New("db down")},
			detailer: &mockDetailer{},
			check: func(t *testing.T, _ *models.WatchEntryDetailResponse, _ *mockStore) {},
		},
		{
			name:     "TMDB error → ErrTMDBUnavailable",
			lookup:   Lookup{ID: &id},
			store:    &mockStore{byIDResult: movie},
			detailer: &mockDetailer{err: errors.New("timeout")},
			wantErr:  ErrTMDBUnavailable,
		},
		{
			name:     "TMDB fields merged into response",
			lookup:   Lookup{ID: &id},
			store:    &mockStore{byIDResult: movie},
			detailer: &mockDetailer{result: details},
			check: func(t *testing.T, resp *models.WatchEntryDetailResponse, _ *mockStore) {
				if resp.Overview == nil || *resp.Overview != tmdbOverview {
					t.Errorf("overview: got %v, want %q", resp.Overview, tmdbOverview)
				}
				if resp.Runtime == nil || *resp.Runtime != runtime {
					t.Errorf("runtime: got %v, want %d", resp.Runtime, runtime)
				}
				if resp.TmdbID != tmdbID {
					t.Errorf("tmdb_id: got %d, want %d", resp.TmdbID, tmdbID)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc(tc.store, tc.detailer)
			resp, err := svc.Get(context.Background(), tc.lookup)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("error: got %v, want to wrap %v", err, tc.wantErr)
				}
				return
			}

			// For the db-error case (no wantErr sentinel): error must be non-nil
			// and must not be mistaken for a not-found or TMDB error.
			if tc.store.byIDErr != nil || tc.store.byTmdbErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(err, ErrNotFound) {
					t.Error("db error should not be wrapped as ErrNotFound")
				}
				if errors.Is(err, ErrTMDBUnavailable) {
					t.Error("db error should not be wrapped as ErrTMDBUnavailable")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, resp, tc.store)
			}
		})
	}
}
