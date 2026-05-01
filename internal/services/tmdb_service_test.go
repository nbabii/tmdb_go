package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nazarbabii/tmdb_go/internal/models"
)

type requestCapture struct {
	path   string
	params map[string]string
	auth   string
}

func newTestServer(t *testing.T, statusCode int, body any) (*httptest.Server, *requestCapture) {
	t.Helper()
	capture := &requestCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.path = r.URL.Path
		capture.auth = r.Header.Get("Authorization")
		capture.params = make(map[string]string)
		for k, v := range r.URL.Query() {
			capture.params[k] = v[0]
		}
		w.WriteHeader(statusCode)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, capture
}

func movieResponse(results []tmdbMovieResult) tmdbMovieSearchResponse {
	return tmdbMovieSearchResponse{Page: 1, TotalPages: 1, TotalResults: len(results), Results: results}
}

func tvResponse(results []tmdbTVResult) tmdbTVSearchResponse {
	return tmdbTVSearchResponse{Page: 1, TotalPages: 1, TotalResults: len(results), Results: results}
}

func TestSearchTitles(t *testing.T) {
	ctx := context.Background()

	t.Run("movie uses /search/movie path", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cap.path != "/search/movie" {
			t.Errorf("path: got %q, want %q", cap.path, "/search/movie")
		}
	})

	t.Run("tv uses /search/tv path", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, tvResponse(nil))
		svc := NewTMDBService(srv.URL, "key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeTV, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cap.path != "/search/tv" {
			t.Errorf("path: got %q, want %q", cap.path, "/search/tv")
		}
	})

	t.Run("sends query param", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "key")

		_, err := svc.SearchTitles(ctx, "inception", models.TitleTypeMovie, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cap.params["query"] != "inception" {
			t.Errorf("query param: got %q, want %q", cap.params["query"], "inception")
		}
	})

	t.Run("sends language en-US", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cap.params["language"] != "en-US" {
			t.Errorf("language param: got %q, want %q", cap.params["language"], "en-US")
		}
	})

	t.Run("sends page param", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 3, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cap.params["page"] != "3" {
			t.Errorf("page param: got %q, want %q", cap.params["page"], "3")
		}
	})

	t.Run("sends Authorization header", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "test-key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cap.auth != "Bearer test-key" {
			t.Errorf("auth header: got %q, want %q", cap.auth, "Bearer test-key")
		}
	})

	t.Run("movie sends year param", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "key")
		year := 2010

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, &year)
		if err != nil {
			t.Fatal(err)
		}
		if cap.params["year"] != "2010" {
			t.Errorf("year param: got %q, want %q", cap.params["year"], "2010")
		}
	})

	t.Run("tv sends first_air_date_year param", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, tvResponse(nil))
		svc := NewTMDBService(srv.URL, "key")
		year := 2010

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeTV, 1, &year)
		if err != nil {
			t.Fatal(err)
		}
		if cap.params["first_air_date_year"] != "2010" {
			t.Errorf("first_air_date_year param: got %q, want %q", cap.params["first_air_date_year"], "2010")
		}
		if _, ok := cap.params["year"]; ok {
			t.Error("tv search should not send 'year' param")
		}
	})

	t.Run("omits year params when nil", func(t *testing.T) {
		srv, cap := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := cap.params["year"]; ok {
			t.Error("year param should not be sent when nil")
		}
		if _, ok := cap.params["first_air_date_year"]; ok {
			t.Error("first_air_date_year param should not be sent when nil")
		}
	})

	t.Run("TV normalizes name fields to title fields", func(t *testing.T) {
		name := "Breaking Bad"
		origName := "Breaking Bad"
		date := "2008-01-20"
		srv, _ := newTestServer(t, 200, tvResponse([]tmdbTVResult{
			{ID: 1, Name: name, OriginalName: origName, FirstAirDate: &date, Overview: "desc", OriginalLanguage: "en"},
		}))
		svc := NewTMDBService(srv.URL, "key")

		resp, err := svc.SearchTitles(ctx, "breaking", models.TitleTypeTV, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Results) != 1 {
			t.Fatalf("results: got %d, want 1", len(resp.Results))
		}
		r := resp.Results[0]
		if r.Title != name {
			t.Errorf("Title: got %q, want %q", r.Title, name)
		}
		if r.OriginalTitle != origName {
			t.Errorf("OriginalTitle: got %q, want %q", r.OriginalTitle, origName)
		}
		if r.ReleaseDate == nil || *r.ReleaseDate != date {
			t.Errorf("ReleaseDate: got %v, want %q", r.ReleaseDate, date)
		}
	})

	t.Run("empty results returns empty slice not nil", func(t *testing.T) {
		srv, _ := newTestServer(t, 200, movieResponse([]tmdbMovieResult{}))
		svc := NewTMDBService(srv.URL, "key")

		resp, err := svc.SearchTitles(ctx, "xyzzy", models.TitleTypeMovie, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Results == nil {
			t.Error("Results: got nil, want empty slice")
		}
		if len(resp.Results) != 0 {
			t.Errorf("Results: got %d items, want 0", len(resp.Results))
		}
	})

	t.Run("non-2xx returns error", func(t *testing.T) {
		srv, _ := newTestServer(t, 401, nil)
		svc := NewTMDBService(srv.URL, "bad-key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, nil)
		if err == nil {
			t.Fatal("expected error for 401, got nil")
		}
	})

	t.Run("network failure returns error", func(t *testing.T) {
		srv, _ := newTestServer(t, 200, movieResponse(nil))
		svc := NewTMDBService(srv.URL, "key")
		srv.Close()

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, nil)
		if err == nil {
			t.Fatal("expected error after server close, got nil")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{bad json`))
		}))
		t.Cleanup(srv.Close)
		svc := NewTMDBService(srv.URL, "key")

		_, err := svc.SearchTitles(ctx, "foo", models.TitleTypeMovie, 1, nil)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}
