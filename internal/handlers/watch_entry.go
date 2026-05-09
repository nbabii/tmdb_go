package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

type movieDetailer interface {
	GetMovieDetails(ctx context.Context, movieID int) (*models.TMDBMovieDetails, error)
}

type watchEntryLookup interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.WatchedMovie, error)
	FindByTmdbID(ctx context.Context, tmdbID int) (*models.WatchedMovie, error)
}

type WatchEntryHandler struct {
	repo   watchEntryLookup
	tmdb   movieDetailer
}

func NewWatchEntryHandler(repo watchEntryLookup, tmdb movieDetailer) *WatchEntryHandler {
	return &WatchEntryHandler{repo: repo, tmdb: tmdb}
}

type watchEntryRequest struct {
	ID     *string `form:"id"`
	TmdbID *int    `form:"tmdb_id"`
}

func (h *WatchEntryHandler) Get(c *gin.Context) {
	var req watchEntryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	if req.ID == nil && req.TmdbID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "either id or tmdb_id is required"})
		return
	}

	var movie *models.WatchedMovie
	var err error

	if req.ID != nil {
		parsed, parseErr := uuid.Parse(*req.ID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid UUID format"})
			return
		}
		movie, err = h.repo.FindByID(c.Request.Context(), parsed)
	} else {
		movie, err = h.repo.FindByTmdbID(c.Request.Context(), *req.TmdbID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}
	if movie == nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		return
	}

	details, err := h.tmdb.GetMovieDetails(c.Request.Context(), movie.TmdbID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"detail": "failed to fetch movie details from TMDB"})
		return
	}

	resp := models.WatchEntryDetailResponse{
		ID:        movie.ID,
		TmdbID:    movie.TmdbID,
		Title:     movie.Title,
		MyRating:  movie.MyRating,
		MyOverview: movie.MyOverview,
		CreatedAt: movie.CreatedAt,
	}
	if movie.ReleaseDate != nil {
		d := models.Date{Time: *movie.ReleaseDate}
		resp.ReleaseDate = &d
	}
	if movie.MyDateWatched != nil {
		d := models.Date{Time: *movie.MyDateWatched}
		resp.MyDateWatched = &d
	}
	if details != nil {
		resp.Overview = details.Overview
		resp.Runtime = details.Runtime
		resp.PosterPath = details.PosterPath
		resp.VoteAverage = details.VoteAverage
	}

	c.JSON(http.StatusOK, resp)
}
