package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nazarbabii/tmdb_go/internal/models"
)

type TMDBService struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewTMDBService(baseURL, apiKey string) *TMDBService {
	return &TMDBService{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// tmdbMovieResult matches the raw TMDB movie search result shape.
type tmdbMovieResult struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	OriginalTitle    string   `json:"original_title"`
	Overview         string   `json:"overview"`
	ReleaseDate      *string  `json:"release_date"`
	PosterPath       *string  `json:"poster_path"`
	BackdropPath     *string  `json:"backdrop_path"`
	Popularity       float64  `json:"popularity"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	GenreIDs         []int    `json:"genre_ids"`
	OriginalLanguage string   `json:"original_language"`
	Adult            bool     `json:"adult"`
	Video            bool     `json:"video"`
}

// tmdbTVResult matches the raw TMDB TV search result shape.
// TMDB uses "name"/"original_name" for TV instead of "title"/"original_title".
type tmdbTVResult struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	OriginalName     string   `json:"original_name"`
	Overview         string   `json:"overview"`
	FirstAirDate     *string  `json:"first_air_date"`
	PosterPath       *string  `json:"poster_path"`
	BackdropPath     *string  `json:"backdrop_path"`
	Popularity       float64  `json:"popularity"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	GenreIDs         []int    `json:"genre_ids"`
	OriginalLanguage string   `json:"original_language"`
	Adult            bool     `json:"adult"`
}

type tmdbMovieSearchResponse struct {
	Page         int               `json:"page"`
	Results      []tmdbMovieResult `json:"results"`
	TotalPages   int               `json:"total_pages"`
	TotalResults int               `json:"total_results"`
}

type tmdbTVSearchResponse struct {
	Page         int            `json:"page"`
	Results      []tmdbTVResult `json:"results"`
	TotalPages   int            `json:"total_pages"`
	TotalResults int            `json:"total_results"`
}

type tmdbMovieDetailsResponse struct {
	Overview    string   `json:"overview"`
	Runtime     *int     `json:"runtime"`
	PosterPath  *string  `json:"poster_path"`
	VoteAverage *float64 `json:"vote_average"`
	Genres      []tmdbGenre `json:"genres"`
}

type tmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (s *TMDBService) GetMovieDetails(ctx context.Context, movieID int) (*models.TMDBMovieDetails, error) {
	raw := &tmdbMovieDetailsResponse{}
	if err := s.get(ctx, fmt.Sprintf("/movie/%d", movieID), url.Values{}, raw); err != nil {
		return nil, err
	}

	fmt.Printf("!!!! Genres: %+v\n", raw.Genres)

	overview := raw.Overview

	var genres []models.Genre
	for _, genre := range raw.Genres {
		genres = append(genres, models.Genre{ID: genre.ID, Name: genre.Name})
	}

	return &models.TMDBMovieDetails{
		Overview:    &overview,
		Runtime:     raw.Runtime,
		PosterPath:  raw.PosterPath,
		VoteAverage: raw.VoteAverage,
		Genres:      genres,
	}, nil
}

func (s *TMDBService) SearchTitles(ctx context.Context, query string, titleType models.TitleType, page int, year *int) (*models.TitleSearchResponse, error) {
	if titleType == models.TitleTypeMovie {
		return s.searchMovies(ctx, query, page, year)
	}
	return s.searchTV(ctx, query, page, year)
}

func (s *TMDBService) searchMovies(ctx context.Context, query string, page int, year *int) (*models.TitleSearchResponse, error) {
	params := s.baseParams(query, page)
	if year != nil {
		params.Set("year", strconv.Itoa(*year))
	}

	raw := &tmdbMovieSearchResponse{}
	if err := s.get(ctx, "/search/movie", params, raw); err != nil {
		return nil, err
	}

	results := make([]models.TitleResult, len(raw.Results))
	for i, r := range raw.Results {
		results[i] = models.TitleResult{
			ID:               r.ID,
			Title:            r.Title,
			OriginalTitle:    r.OriginalTitle,
			Overview:         r.Overview,
			ReleaseDate:      r.ReleaseDate,
			PosterPath:       r.PosterPath,
			BackdropPath:     r.BackdropPath,
			Popularity:       r.Popularity,
			VoteAverage:      r.VoteAverage,
			VoteCount:        r.VoteCount,
			GenreIDs:         r.GenreIDs,
			OriginalLanguage: r.OriginalLanguage,
			Adult:            r.Adult,
			Video:            r.Video,
		}
	}

	return &models.TitleSearchResponse{
		Page:         raw.Page,
		Results:      results,
		TotalPages:   raw.TotalPages,
		TotalResults: raw.TotalResults,
	}, nil
}

func (s *TMDBService) searchTV(ctx context.Context, query string, page int, year *int) (*models.TitleSearchResponse, error) {
	params := s.baseParams(query, page)
	if year != nil {
		params.Set("first_air_date_year", strconv.Itoa(*year))
	}

	raw := &tmdbTVSearchResponse{}
	if err := s.get(ctx, "/search/tv", params, raw); err != nil {
		return nil, err
	}

	results := make([]models.TitleResult, len(raw.Results))
	for i, r := range raw.Results {
		results[i] = models.TitleResult{
			ID:               r.ID,
			Title:            r.Name,
			OriginalTitle:    r.OriginalName,
			Overview:         r.Overview,
			ReleaseDate:      r.FirstAirDate,
			PosterPath:       r.PosterPath,
			BackdropPath:     r.BackdropPath,
			Popularity:       r.Popularity,
			VoteAverage:      r.VoteAverage,
			VoteCount:        r.VoteCount,
			GenreIDs:         r.GenreIDs,
			OriginalLanguage: r.OriginalLanguage,
			Adult:            r.Adult,
		}
	}

	return &models.TitleSearchResponse{
		Page:         raw.Page,
		Results:      results,
		TotalPages:   raw.TotalPages,
		TotalResults: raw.TotalResults,
	}, nil
}

func (s *TMDBService) baseParams(query string, page int) url.Values {
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", strconv.Itoa(page))
	params.Set("language", "en-US")
	return params
}

func (s *TMDBService) get(ctx context.Context, path string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("calling TMDB API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding TMDB response: %w", err)
	}
	return nil
}
