package vectordb

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterAuditRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	group := r.Group("/vector-db/audit")
	group.GET("/logs", handler.ListAuditLogs)
}

//nolint:dupl // Query bind + list response pattern intentionally mirrors backup list handler.
func (h *CollectionHandler) ListAuditLogs(c *gin.Context) {
	query := &ListAuditLogsQuery{
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		ResourceID:   strings.TrimSpace(c.Query("resource_id")),
		Action:       strings.TrimSpace(c.Query("action")),
		Limit:        parseIntDefault(c.Query("limit"), 50),
		Offset:       parseIntDefault(c.Query("offset"), 0),
	}
	items, err := h.service.ListAuditLogs(c.Request.Context(), query)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "total": len(items)}})
}
