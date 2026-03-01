package vectordb

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterVisualizationRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	group := r.Group("/vector-db/visualization")
	group.GET("/scatter", handler.GetScatterData)
}

func (h *CollectionHandler) GetScatterData(c *gin.Context) {
	query := &GetScatterDataRequest{
		CollectionName: strings.TrimSpace(c.Query("collection_name")),
		SampleSize:     parseIntDefault(c.Query("sample_size"), 200),
	}
	resp, err := h.service.GetScatterData(c.Request.Context(), query)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}
