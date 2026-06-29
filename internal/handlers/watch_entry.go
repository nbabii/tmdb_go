package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nazarbabii/tmdb_go/internal/services"
)

type WatchEntryHandler struct {
	svc services.WatchEntryServiceUser
}

func NewWatchEntryHandler(svc services.WatchEntryServiceUser) *WatchEntryHandler {
	return &WatchEntryHandler{svc: svc}
}

type watchEntryRequest struct {
	ID     *string `form:"id"`
	TmdbID *int    `form:"tmdb_id"`
}

// Get retrieves a single watch entry by its UUID or TMDB ID.
//
// @Summary      Get a watch entry
// @Tags         watch-entry
// @Produce      json
// @Param        id       query     string  false  "Watch entry UUID"
// @Param        tmdb_id  query     int     false  "TMDB movie ID"
// @Success      200      {object}  models.WatchEntryDetailResponse
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      502      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /watch-entry [get]
func (h *WatchEntryHandler) Get(c *gin.Context) {
	var req watchEntryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	if req.ID == nil && req.TmdbID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "at least one of id or tmdb_id is required"})
		return
	}

	var lookup services.Lookup
	if req.ID != nil {
		parsed, err := uuid.Parse(*req.ID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid UUID format"})
			return
		}
		lookup = services.Lookup{ID: &parsed}
	} else {
		lookup = services.Lookup{TmdbID: req.TmdbID}
	}

	resp, err := h.svc.Get(c.Request.Context(), lookup)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		case errors.Is(err, services.ErrTMDBUnavailable):
			c.JSON(http.StatusBadGateway, gin.H{"detail": "failed to fetch movie details from TMDB"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Exists checks whether a movie is in the watch list by its TMDB ID.
//
// @Summary      Check if a watch entry exists
// @Tags         watch-entry
// @Produce      json
// @Param        tmdb_id  path      int  true  "TMDB movie ID"
// @Success      200      {object}  models.WatchEntryExistsResponse
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /watch-entry/{tmdb_id} [get]
func (h *WatchEntryHandler) Exists(c *gin.Context) {
	rawId := c.Param("tmdb_id")
	id, err := strconv.Atoi(rawId)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid TMDB ID format"})
		return
	}

	exists, err := h.svc.ExistsByTmdbID(c.Request.Context(), id)
	if errors.Is(err, services.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		return
	}


	c.JSON(http.StatusOK, exists)
}