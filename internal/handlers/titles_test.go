package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

type mockTMDB struct {
	result        *models.TitleSearchResponse
	err           error
	capturedQuery string
	capturedType  models.TitleType
	capturedPage  int
	capturedYear  *int
}

func (m *mockTMDB) SearchTitles(_ context.Context, query string, titleType models.TitleType, page int, year *int) (*models.TitleSearchResponse, error) {
	m.capturedQuery = query
	m.capturedType = titleType
	m.capturedPage = page
	m.capturedYear = year
	return m.result, m.err
}

func newTestRouter(mock *mockTMDB) *gin.Engine {
	router := gin.New()
	router.GET("/api/v1/titles/search", NewTitlesHandler(mock).Search)
	return router
}

func successResponse() *models.TitleSearchResponse {
	return &models.TitleSearchResponse{
		Page:         1,
		TotalPages:   1,
		TotalResults: 1,
		Results: []models.TitleResult{
			{ID: 1, Title: "Test", OriginalTitle: "Test", Overview: "desc", Popularity: 1.0},
		},
	}
}

func TestTitlesSearch(t *testing.T) {
	cases := []struct {
		name         string
		query        string
		mock         *mockTMDB
		wantStatus   int
		wantPage     int
		wantYear     *int
		wantYearNil  bool
		wantResults  int
	}{
		{
			name:       "missing query param",
			query:      "/api/v1/titles/search?type=movie",
			mock:       &mockTMDB{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing type param",
			query:      "/api/v1/titles/search?query=foo",
			mock:       &mockTMDB{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid type value",
			query:      "/api/v1/titles/search?query=foo&type=anime",
			mock:       &mockTMDB{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "page above max",
			query:      "/api/v1/titles/search?query=foo&type=movie&page=501",
			mock:       &mockTMDB{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "page below min",
			query:      "/api/v1/titles/search?query=foo&type=movie&page=0",
			mock:       &mockTMDB{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "defaults page to 1 when omitted",
			query:       "/api/v1/titles/search?query=foo&type=movie",
			mock:        &mockTMDB{result: successResponse()},
			wantStatus:  http.StatusOK,
			wantPage:    1,
			wantYearNil: true,
		},
		{
			name:       "passes year when provided",
			query:      "/api/v1/titles/search?query=foo&type=movie&year=2010",
			mock:       &mockTMDB{result: successResponse()},
			wantStatus: http.StatusOK,
			wantPage:   1,
			wantYear:   intPtr(2010),
		},
		{
			name:        "omits year when not provided",
			query:       "/api/v1/titles/search?query=foo&type=tv",
			mock:        &mockTMDB{result: successResponse()},
			wantStatus:  http.StatusOK,
			wantYearNil: true,
		},
		{
			name:       "TMDB error returns 502",
			query:      "/api/v1/titles/search?query=foo&type=movie",
			mock:       &mockTMDB{err: errors.New("upstream failure")},
			wantStatus: http.StatusBadGateway,
		},
		{
			name:        "success movie returns results",
			query:       "/api/v1/titles/search?query=inception&type=movie",
			mock:        &mockTMDB{result: successResponse()},
			wantStatus:  http.StatusOK,
			wantPage:    1,
			wantResults: 1,
		},
		{
			name:        "success tv returns results",
			query:       "/api/v1/titles/search?query=breaking&type=tv",
			mock:        &mockTMDB{result: successResponse()},
			wantStatus:  http.StatusOK,
			wantPage:    1,
			wantResults: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(tc.mock)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.query, nil)
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", w.Code, tc.wantStatus)
			}

			if tc.wantStatus != http.StatusOK {
				return
			}

			if tc.wantPage != 0 && tc.mock.capturedPage != tc.wantPage {
				t.Errorf("capturedPage: got %d, want %d", tc.mock.capturedPage, tc.wantPage)
			}
			if tc.wantYearNil && tc.mock.capturedYear != nil {
				t.Errorf("capturedYear: got %v, want nil", tc.mock.capturedYear)
			}
			if tc.wantYear != nil {
				if tc.mock.capturedYear == nil {
					t.Fatal("capturedYear: got nil, want non-nil")
				}
				if *tc.mock.capturedYear != *tc.wantYear {
					t.Errorf("capturedYear: got %d, want %d", *tc.mock.capturedYear, *tc.wantYear)
				}
			}

			var body models.TitleSearchResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal body: %v", err)
			}
			if tc.wantResults != 0 && len(body.Results) != tc.wantResults {
				t.Errorf("results count: got %d, want %d", len(body.Results), tc.wantResults)
			}
		})
	}
}

func intPtr(v int) *int { return &v }
