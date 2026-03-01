package vectordb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RegisterIndexConfigRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	configs := r.Group("/vector-db/index-config")
	configs.GET("/:name", handler.GetIndexConfig)
	configs.PUT("/:name", handler.UpdateIndexConfig)
}

func (h *CollectionHandler) GetIndexConfig(c *gin.Context) {
	config, err := h.service.GetIndexConfig(c.Request.Context(), c.Param("name"))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": config})
}

func (h *CollectionHandler) UpdateIndexConfig(c *gin.Context) {
	var req UpdateIndexConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid update index config request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	config, err := h.service.UpdateIndexConfig(c.Request.Context(), c.Param("name"), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": config})
}
