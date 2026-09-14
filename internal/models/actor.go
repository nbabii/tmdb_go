package models

import "time"

// Actor is the database representation of the actors table.
type Actor struct {
	ID         int
	Name       string
	Popularity *float64
	Biography  *string
	Birthday   *time.Time
}

// ActorResponse is the JSON shape for a single actor.
type ActorResponse struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Popularity *float64 `json:"popularity"`
	Biography  *string  `json:"biography"`
	Birthday   *Date    `json:"birthday"`
}

// ActorListResponse is the GET /watch-entry/{tmdb_id}/actors response body.
type ActorListResponse struct {
	Items []ActorResponse `json:"items"`
}

// ToResponse converts an Actor DB model to an ActorResponse JSON type.
func (a Actor) ToResponse() ActorResponse {
	r := ActorResponse{ID: a.ID, Name: a.Name, Popularity: a.Popularity, Biography: a.Biography}
	if a.Birthday != nil {
		d := Date{Time: *a.Birthday}
		r.Birthday = &d
	}
	return r
}
