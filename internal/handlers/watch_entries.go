package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/models"
)

type watchEntryRepo interface {
	FindExistingTmdbIDs(ctx context.Context, tmdbIDs []int) (map[int]struct{}, error)
	BulkCreate(ctx context.Context, entries []models.WatchedMovie) ([]models.WatchedMovie, error)
	CountAll(ctx context.Context) (int, error)
	ListAll(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]models.WatchedMovie, error)
}

type WatchEntriesHandler struct {
	repo watchEntryRepo
}

func NewWatchEntriesHandler(repo watchEntryRepo) *WatchEntriesHandler {
	return &WatchEntriesHandler{repo: repo}
}

type createItem struct {
	TmdbID        int     `json:"tmdb_id"         binding:"required"`
	Title         string  `json:"title"           binding:"required"`
	ReleaseDate   *models.Date `json:"release_date"`
	MyRating      *int    `json:"my_rating"       binding:"omitempty,min=1,max=10"`
	MyOverview    *string `json:"my_overview"`
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

	tmdbIDs := make([]int, len(items))
	for i, it := range items {
		tmdbIDs[i] = it.TmdbID
	}

	existing, err := h.repo.FindExistingTmdbIDs(c.Request.Context(), tmdbIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}

	var toCreate []models.WatchedMovie
	var skipped []models.WatchEntrySkipped

	for _, it := range items {
		if _, dup := existing[it.TmdbID]; dup {
			skipped = append(skipped, models.WatchEntrySkipped{
				TmdbID: it.TmdbID,
				Title:  it.Title,
				Reason: "already exists",
			})
			continue
		}
		m := models.WatchedMovie{
			ID:         uuid.New(),
			TmdbID:     it.TmdbID,
			Title:      it.Title,
			MyRating:   it.MyRating,
			MyOverview: it.MyOverview,
		}
		if it.ReleaseDate != nil {
			t := it.ReleaseDate.Time
			m.ReleaseDate = &t
		}
		if it.MyDateWatched != nil {
			t := it.MyDateWatched.Time
			m.MyDateWatched = &t
		}
		toCreate = append(toCreate, m)
	}

	if len(toCreate) == 0 {
		c.JSON(http.StatusConflict, gin.H{"detail": skipped})
		return
	}

	created, err := h.repo.BulkCreate(c.Request.Context(), toCreate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}

	responses := make([]models.WatchEntryResponse, len(created))
	for i, m := range created {
		responses[i] = m.ToResponse()
	}

	result := models.WatchEntryBulkResult{
		Created: responses,
		Skipped: skipped,
	}
	if skipped == nil {
		result.Skipped = []models.WatchEntrySkipped{}
	}

	status := http.StatusCreated
	if len(skipped) > 0 {
		status = http.StatusMultiStatus
	}
	c.JSON(status, result)
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

	limit := 10
	if req.Limit != nil {
		limit = *req.Limit
	}
	sortBy := "my_rating"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}
	sortOrder := "desc"
	if req.SortOrder != "" {
		sortOrder = req.SortOrder
	}

	total, err := h.repo.CountAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}

	entries, err := h.repo.ListAll(c.Request.Context(), limit, req.Offset, sortBy, sortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		return
	}

	items := make([]models.WatchEntryListItem, len(entries))
	for i, m := range entries {
		items[i] = m.ToListItem()
	}

	c.JSON(http.StatusOK, models.WatchEntryListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: req.Offset,
	})
}
