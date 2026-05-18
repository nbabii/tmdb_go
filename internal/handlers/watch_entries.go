package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nazarbabii/tmdb_go/internal/models"
	"github.com/nazarbabii/tmdb_go/internal/services"
)

type watchEntriesService interface {
	BulkCreate(ctx context.Context, items []services.CreateParams) (services.CreateResult, error)
	List(ctx context.Context, p services.ListParams) (models.WatchEntryListResponse, error)
	ExistsByTmdbID(ctx context.Context, tmdbID int) (bool, error)
}

type WatchEntriesHandler struct {
	svc watchEntriesService
}

func NewWatchEntriesHandler(svc watchEntriesService) *WatchEntriesHandler {
	return &WatchEntriesHandler{svc: svc}
}

type createItem struct {
	TmdbID        int          `json:"tmdb_id"         binding:"required"`
	Title         string       `json:"title"           binding:"required"`
	ReleaseDate   *models.Date `json:"release_date"`
	MyRating      *int         `json:"my_rating"       binding:"omitempty,min=1,max=10"`
	MyOverview    *string      `json:"my_overview"`
	MyDateWatched *models.Date `json:"my_date_watched"`
}

func (h *WatchEntriesHandler) Create(c *gin.Context) {
	var items []createItem
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "request body must not be empty"})
		return
	}

	params := make([]services.CreateParams, len(items))
	for i, it := range items {
		p := services.CreateParams{
			TmdbID:     it.TmdbID,
			Title:      it.Title,
			MyRating:   it.MyRating,
			MyOverview: it.MyOverview,
		}
		if it.ReleaseDate != nil {
			t := it.ReleaseDate.Time
			p.ReleaseDate = &t
		}
		if it.MyDateWatched != nil {
			t := it.MyDateWatched.Time
			p.MyDateWatched = &t
		}
		params[i] = p
	}

	result, err := h.svc.BulkCreate(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}

	if len(result.Created) == 0 {
		c.JSON(http.StatusConflict, gin.H{"detail": result.Skipped})
		return
	}

	status := http.StatusCreated
	if len(result.Skipped) > 0 {
		status = http.StatusMultiStatus
	}
	c.JSON(status, models.WatchEntryBulkResult{
		Created: result.Created,
		Skipped: result.Skipped,
	})
}

type listRequest struct {
	Limit     *int   `form:"limit"       binding:"omitempty,min=1,max=100"`
	Offset    int    `form:"offset"      binding:"min=0"`
	SortBy    string `form:"sort_by"     binding:"omitempty,oneof=my_rating my_date_watched"`
	SortOrder string `form:"sort_order"  binding:"omitempty,oneof=asc desc"`
}

func (h *WatchEntriesHandler) List(c *gin.Context) {
	var req listRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}

	p := services.ListParams{
		Offset:    req.Offset,
		Limit:     10,
		SortBy:    "my_rating",
		SortOrder: "desc",
	}
	if req.Limit != nil {
		p.Limit = *req.Limit
	}
	if req.SortBy != "" {
		p.SortBy = req.SortBy
	}
	if req.SortOrder != "" {
		p.SortOrder = req.SortOrder
	}

	resp, err := h.svc.List(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

type existsRequest struct {
	TmdbID *int `form:"tmdb_id" binding:"required,min=1"`
}

func (h *WatchEntriesHandler) Exists(c *gin.Context) {
	var req existsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": err.Error()})
		return
	}

	exists, err := h.svc.ExistsByTmdbID(c.Request.Context(), *req.TmdbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exists": exists})
}
