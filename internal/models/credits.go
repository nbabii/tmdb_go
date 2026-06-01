package models

type TMDBMovieCredits struct {
	TmdbID   int `json:"id"`
	Cast []TMDBMovieCast `json:"cast"`
}

type TMDBMovieCast struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Character string `json:"character"`
}