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

func (h *CollectionHandler) RBACService() *RBACService {
	if h == nil || h.service == nil {
		return nil
	}
	return NewRBACService(h.service.GetRepository())
}

// RegisterCollectionRoutes registers collection routes.
func RegisterCollectionRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	collections := r.Group("/vector-db/collections")
	collections.POST("", handler.CreateCollection)
	collections.GET("", handler.ListCollections)
	collections.GET("/:name", handler.GetCollection)
	collections.PUT("/:name", handler.UpdateCollection)
	collections.POST("/:name/empty", handler.EmptyCollection)
	collections.DELETE("/:name", handler.DeleteCollection)
}

// RegisterImportJobRoutes registers import job routes.
func RegisterImportJobRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	jobs := r.Group("/vector-db/import-jobs")
	jobs.POST("", handler.CreateImportJob)
	jobs.GET("", handler.ListImportJobs)
	jobs.GET("/summary", handler.GetImportJobSummary)
	jobs.GET("/:id", handler.GetImportJob)
	jobs.PUT("/:id/status", handler.UpdateImportJobStatus)
	jobs.POST("/:id/run", handler.RunImportJob)
	jobs.POST("/:id/retry", handler.RetryImportJob)
	jobs.POST("/retry-failed", handler.RetryFailedImportJobs)
	jobs.GET("/:id/errors", handler.GetImportJobErrors)
}

// GetImportJobSummary handles GET /api/admin/vector-db/import-jobs/summary.
func (h *CollectionHandler) GetImportJobSummary(c *gin.Context) {
	query := &ListImportJobsQuery{
		CollectionName: strings.TrimSpace(c.Query("collection_name")),
	}
	summary, err := h.service.GetImportJobSummary(c.Request.Context(), query)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

// CreateCollection handles POST /api/admin/vector-db/collections.
//
//nolint:dupl // Request bind + create pattern mirrors import job create handler intentionally.
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

// EmptyCollection handles POST /api/admin/vector-db/collections/:name/empty.
func (h *CollectionHandler) EmptyCollection(c *gin.Context) {
	if err := h.service.EmptyCollection(c.Request.Context(), c.Param("name")); err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CreateImportJob handles POST /api/admin/vector-db/import-jobs.
//
//nolint:dupl // Request bind + create pattern mirrors collection create handler intentionally.
func (h *CollectionHandler) CreateImportJob(c *gin.Context) {
	var req CreateImportJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid create import job request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	job, err := h.service.CreateImportJob(c.Request.Context(), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": job})
}

// ListImportJobs handles GET /api/admin/vector-db/import-jobs.
func (h *CollectionHandler) ListImportJobs(c *gin.Context) {
	query := &ListImportJobsQuery{
		CollectionName: strings.TrimSpace(c.Query("collection_name")),
		Status:         strings.TrimSpace(c.Query("status")),
		Offset:         parseIntDefault(c.Query("offset"), 0),
		Limit:          parseIntDefault(c.Query("limit"), 0),
	}

	jobs, err := h.service.ListImportJobs(c.Request.Context(), query)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"jobs": jobs, "total": len(jobs)}})
}

// GetImportJob handles GET /api/admin/vector-db/import-jobs/:id.
func (h *CollectionHandler) GetImportJob(c *gin.Context) {
	job, err := h.service.GetImportJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": job})
}

// UpdateImportJobStatus handles PUT /api/admin/vector-db/import-jobs/:id/status.
func (h *CollectionHandler) UpdateImportJobStatus(c *gin.Context) {
	var req UpdateImportJobStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid update import job status request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.service.UpdateImportJobStatus(c.Request.Context(), c.Param("id"), &req); err != nil {
		h.respondServiceError(c, err)
		return
	}

	job, err := h.service.GetImportJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": job})
}

// RunImportJob handles POST /api/admin/vector-db/import-jobs/:id/run.
func (h *CollectionHandler) RunImportJob(c *gin.Context) {
	job, err := h.service.RunImportJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": job})
}

// RetryImportJob handles POST /api/admin/vector-db/import-jobs/:id/retry.
func (h *CollectionHandler) RetryImportJob(c *gin.Context) {
	job, err := h.service.RetryImportJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": job})
}

// RetryFailedImportJobs handles POST /api/admin/vector-db/import-jobs/retry-failed.
func (h *CollectionHandler) RetryFailedImportJobs(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 20)
	if limit < 1 {
		limit = 1
	}
	jobs, err := h.service.RetryFailedImportJobs(c.Request.Context(), limit)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"jobs": jobs, "total": len(jobs)}})
}

// GetImportJobErrors handles GET /api/admin/vector-db/import-jobs/:id/errors.
func (h *CollectionHandler) GetImportJobErrors(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 20)
	if limit < 1 {
		limit = 1
	}
	offset := parseIntDefault(c.Query("offset"), 0)
	if offset < 0 {
		offset = 0
	}
	action := strings.TrimSpace(c.Query("action"))
	logs, err := h.service.GetImportJobErrors(c.Request.Context(), c.Param("id"), action, limit, offset)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"logs": logs, "total": len(logs)}})
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
	case errors.Is(err, ErrImportJobNotFound):
		logEntry.Warn("vector db import job not found")
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, ErrImportJobRetryExceeded):
		logEntry.Warn("vector db import job retry exceeded")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, ErrAlertRuleNotFound):
		logEntry.Warn("vector db alert rule not found")
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, ErrVectorAPIKeyNotFound):
		logEntry.Warn("vector db api key not found")
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
	case errors.Is(err, ErrBackupTaskNotFound):
		logEntry.Warn("vector db backup task not found")
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, ErrBackendUnavailable):
		logEntry.Warn("vector db backend unavailable")
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, ErrTextSearchNotSupported):
		logEntry.Warn("vector db text search is not supported")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
	case strings.Contains(err.Error(), "repository is required"):
		logEntry.Error("vector db internal dependency missing")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
	default:
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "positive") || strings.Contains(err.Error(), "only allowed") {
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
