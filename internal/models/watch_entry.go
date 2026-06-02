package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WatchedMovie is the database representation of the watched_movies table.
type WatchedMovie struct {
	ID            uuid.UUID
	TmdbID        int
	Title         string
	ReleaseDate   *time.Time
	MyRating      *int
	MyOverview    *string
	MyDateWatched *time.Time
	CreatedAt     time.Time
	Genres        []Genre
}

// Date wraps time.Time and marshals to/from "YYYY-MM-DD" JSON strings.
// Go's default time.Time marshals as RFC3339 which breaks date-only fields.
type Date struct {
	time.Time
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format("2006-01-02"))
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

// TMDBMovieDetails holds the fields fetched from TMDB's /movie/{id} endpoint.
// Defined here (not in services/) so handlers don't need to import the services package.
type TMDBMovieDetails struct {
	Overview    *string  `json:"overview"`
	Runtime     *int     `json:"runtime"`
	PosterPath  *string  `json:"poster_path"`
	VoteAverage *float64 `json:"vote_average"`
	Genres      []Genre  `json:"genres"`
}

// WatchEntryResponse is the JSON shape for a single created entry in the POST response.
type WatchEntryResponse struct {
	ID            uuid.UUID `json:"id"`
	TmdbID        int       `json:"tmdb_id"`
	Title         string    `json:"title"`
	ReleaseDate   *Date     `json:"release_date"`
	MyRating      *int      `json:"my_rating"`
	MyOverview    *string   `json:"my_overview"`
	MyDateWatched *Date     `json:"my_date_watched"`
	CreatedAt     time.Time `json:"created_at"`
}

// WatchEntrySkipped is the JSON shape for a duplicate entry that was not created.
type WatchEntrySkipped struct {
	TmdbID int    `json:"tmdb_id"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

// WatchEntryBulkResult is the top-level POST /watch-entries response body.
type WatchEntryBulkResult struct {
	Created []WatchEntryResponse `json:"created"`
	Skipped []WatchEntrySkipped  `json:"skipped"`
}

// WatchEntryListItem is the JSON shape for a single item in the GET /watch-entries list.
type WatchEntryListItem struct {
	ID            uuid.UUID `json:"id"`
	TmdbID        int       `json:"tmdb_id"`
	Title         string    `json:"title"`
	ReleaseDate   *Date     `json:"release_date"`
	MyRating      *int      `json:"my_rating"`
	MyOverview    *string   `json:"my_overview"`
	MyDateWatched *Date     `json:"my_date_watched"`
}

// WatchEntryListResponse is the top-level GET /watch-entries response body.
type WatchEntryListResponse struct {
	Items  []WatchEntryListItem `json:"items"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

// WatchEntryDetailResponse is the GET /watch-entry response — DB fields merged with TMDB data.
type WatchEntryDetailResponse struct {
	ID            uuid.UUID `json:"id"`
	TmdbID        int       `json:"tmdb_id"`
	Title         string    `json:"title"`
	Overview      *string   `json:"overview"`
	ReleaseDate   *Date     `json:"release_date"`
	Runtime       *int      `json:"runtime"`
	PosterPath    *string   `json:"poster_path"`
	VoteAverage   *float64  `json:"vote_average"`
	MyRating      *int      `json:"my_rating"`
	MyOverview    *string   `json:"my_overview"`
	MyDateWatched *Date     `json:"my_date_watched"`
	CreatedAt     time.Time `json:"created_at"`
	Genres        []Genre   `json:"genres"`
}

// ToResponse converts a WatchedMovie DB model to a WatchEntryResponse JSON type.
func (m WatchedMovie) ToResponse() WatchEntryResponse {
	r := WatchEntryResponse{
		ID:         m.ID,
		TmdbID:     m.TmdbID,
		Title:      m.Title,
		MyRating:   m.MyRating,
		MyOverview: m.MyOverview,
		CreatedAt:  m.CreatedAt,
	}
	if m.ReleaseDate != nil {
		d := Date{Time: *m.ReleaseDate}
		r.ReleaseDate = &d
	}
	if m.MyDateWatched != nil {
		d := Date{Time: *m.MyDateWatched}
		r.MyDateWatched = &d
	}
	return r
}

// ToListItem converts a WatchedMovie DB model to a WatchEntryListItem JSON type.
func (m WatchedMovie) ToListItem() WatchEntryListItem {
	item := WatchEntryListItem{
		ID:         m.ID,
		TmdbID:     m.TmdbID,
		Title:      m.Title,
		MyRating:   m.MyRating,
		MyOverview: m.MyOverview,
	}
	if m.ReleaseDate != nil {
		d := Date{Time: *m.ReleaseDate}
		item.ReleaseDate = &d
	}
	if m.MyDateWatched != nil {
		d := Date{Time: *m.MyDateWatched}
		item.MyDateWatched = &d
	}
	return item
}
