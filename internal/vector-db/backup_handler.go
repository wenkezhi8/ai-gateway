package vectordb

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RegisterBackupRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	group := r.Group("/vector-db/backups")
	group.POST("", handler.CreateBackup)
	group.GET("", handler.ListBackups)
	group.POST("/:id/restore", handler.TriggerRestore)
	group.POST("/:id/retry", handler.RetryBackupTask)
}

//nolint:dupl // Request bind + create pattern intentionally mirrors other create handlers.
func (h *CollectionHandler) CreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid create backup request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	item, err := h.service.CreateBackup(c.Request.Context(), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": item})
}

func (h *CollectionHandler) ListBackups(c *gin.Context) {
	query := &ListBackupsQuery{
		CollectionName: strings.TrimSpace(c.Query("collection_name")),
		Action:         strings.TrimSpace(c.Query("action")),
		Status:         strings.TrimSpace(c.Query("status")),
		Offset:         parseIntDefault(c.Query("offset"), 0),
		Limit:          parseIntDefault(c.Query("limit"), 0),
	}
	items, err := h.service.ListBackups(c.Request.Context(), query)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "total": len(items)}})
}

func (h *CollectionHandler) TriggerRestore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}
	item, err := h.service.TriggerRestore(c.Request.Context(), id, "system")
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *CollectionHandler) RetryBackupTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}
	item, err := h.service.RetryBackupTask(c.Request.Context(), id)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}
