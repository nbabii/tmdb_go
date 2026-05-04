package models

type TitleType string

const (
	TitleTypeMovie TitleType = "movie"
	TitleTypeTV    TitleType = "tv"
)

type TitleResult struct {
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

type TitleSearchResponse struct {
	Page         int           `json:"page"`
	Results      []TitleResult `json:"results"`
	TotalPages   int           `json:"total_pages"`
	TotalResults int           `json:"total_results"`
}
