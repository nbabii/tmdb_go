package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

type titleSearcher interface {
	SearchTitles(ctx context.Context, query string, titleType models.TitleType, page int, year *int) (*models.TitleSearchResponse, error)
	GetCredits(ctx context.Context, movieID int) (*models.TMDBMovieCredits, error)
}

type TitlesHandler struct {
	tmdb titleSearcher
}

func NewTitlesHandler(tmdb titleSearcher) *TitlesHandler {
	return &TitlesHandler{tmdb: tmdb}
}

type searchRequest struct {
	Query string           `form:"query" binding:"required"`
	Type  models.TitleType `form:"type"  binding:"required,oneof=movie tv"`
	Page  *int             `form:"page"  binding:"omitempty,min=1,max=500"`
	Year  *int             `form:"year"`
}

func (h *TitlesHandler) Search(c *gin.Context) {
	var req searchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid request parameters"})
		return
	}

	page := 1
	if req.Page != nil {
		page = *req.Page
	}

	result, err := h.tmdb.SearchTitles(c.Request.Context(), req.Query, req.Type, page, req.Year)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"detail": "failed to fetch results from TMDB"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *TitlesHandler) Credits(c *gin.Context) {
	tmdbID := c.Param("tmdb_id")
	movieID, err := strconv.Atoi(tmdbID)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid TMDB ID format"})
		return
	}

	credits, err := h.tmdb.GetCredits(c.Request.Context(), movieID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"detail": "failed to fetch credits from TMDB"})
		return
	}

	c.JSON(http.StatusOK, credits)
}