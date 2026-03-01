package vectordb

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CollectionHandler provides HTTP handlers for collection management.
type CollectionHandler struct {
	service *Service
}

// NewCollectionHandler creates collection handler.
func NewCollectionHandler(service *Service) *CollectionHandler {
	if service == nil {
		service = NewService()
	}
	return &CollectionHandler{service: service}
}

// RegisterCollectionRoutes registers collection routes.
func RegisterCollectionRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	collections := r.Group("/vector-db/collections")
	collections.POST("", handler.CreateCollection)
	collections.GET("", handler.ListCollections)
	collections.GET("/:name", handler.GetCollection)
	collections.PUT("/:name", handler.UpdateCollection)
	collections.DELETE("/:name", handler.DeleteCollection)
}

// CreateCollection handles POST /api/admin/vector-db/collections.
func (h *CollectionHandler) CreateCollection(c *gin.Context) {
	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid create collection request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	collection, err := h.service.CreateCollection(c.Request.Context(), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": collection})
}

// ListCollections handles GET /api/admin/vector-db/collections.
func (h *CollectionHandler) ListCollections(c *gin.Context) {
	query := ListCollectionsQuery{
		Name:        strings.TrimSpace(c.Query("name")),
		Search:      strings.TrimSpace(c.Query("search")),
		Environment: strings.TrimSpace(c.Query("environment")),
		Status:      strings.TrimSpace(c.Query("status")),
		Tag:         strings.TrimSpace(c.Query("tag")),
		Offset:      parseIntDefault(c.Query("offset"), 0),
		Limit:       parseIntDefault(c.Query("limit"), 0),
	}

	if publicRaw := strings.TrimSpace(c.Query("is_public")); publicRaw != "" {
		parsed, err := strconv.ParseBool(publicRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid is_public"})
			return
		}
		query.IsPublic = &parsed
	}

	collections, err := h.service.ListCollections(c.Request.Context(), &query)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	resp := CollectionListResponse{Collections: collections, Total: len(collections)}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// GetCollection handles GET /api/admin/vector-db/collections/:name.
func (h *CollectionHandler) GetCollection(c *gin.Context) {
	name := c.Param("name")
	collection, err := h.service.GetCollection(c.Request.Context(), name)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	resp := CollectionDetailResponse{Collection: collection}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// UpdateCollection handles PUT /api/admin/vector-db/collections/:name.
func (h *CollectionHandler) UpdateCollection(c *gin.Context) {
	name := c.Param("name")
	var req UpdateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid update collection request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.service.UpdateCollection(c.Request.Context(), name, &req); err != nil {
		h.respondServiceError(c, err)
		return
	}

	updated, err := h.service.GetCollection(c.Request.Context(), name)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": updated})
}

// DeleteCollection handles DELETE /api/admin/vector-db/collections/:name.
func (h *CollectionHandler) DeleteCollection(c *gin.Context) {
	name := c.Param("name")

	var req DeleteCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		logrus.WithError(err).Warn("invalid delete collection request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.service.DeleteCollection(c.Request.Context(), name); err != nil {
		h.respondServiceError(c, err)
		return
	}

	logrus.WithFields(logrus.Fields{"name": name, "force": req.Force}).Info("delete collection request completed")
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CollectionHandler) respondServiceError(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "unknown error"})
		return
	}

	logEntry := logrus.WithError(err)
	switch {
	case errors.Is(err, ErrCollectionNotFound):
		logEntry.Warn("vector db collection not found")
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, ErrCollectionExists):
		logEntry.Warn("vector db collection conflict")
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
	default:
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "positive") {
			logEntry.Warn("vector db collection bad request")
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		logEntry.Error("vector db collection internal error")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
	}
}

func parseIntDefault(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}
