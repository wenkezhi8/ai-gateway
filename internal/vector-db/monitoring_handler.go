package vectordb

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RegisterMonitoringRoutes(r *gin.RouterGroup, handler *CollectionHandler) {
	rules := r.Group("/vector-db/alerts/rules")
	rules.GET("", handler.ListAlertRules)
	rules.POST("", handler.CreateAlertRule)
	rules.POST("/notify-test", handler.NotifyAlertChannels)
	rules.PUT("/:id", handler.UpdateAlertRule)
	rules.DELETE("/:id", handler.DeleteAlertRule)

	metrics := r.Group("/vector-db/metrics")
	metrics.GET("/summary", handler.GetVectorMetricsSummary)
}

func (h *CollectionHandler) ListAlertRules(c *gin.Context) {
	rules, err := h.service.ListAlertRules(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"rules": rules, "total": len(rules)}})
}

//nolint:dupl // Request bind + create pattern mirrors collection creation handlers.
func (h *CollectionHandler) CreateAlertRule(c *gin.Context) {
	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid create alert rule request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	rule, err := h.service.CreateAlertRule(c.Request.Context(), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": rule})
}

func (h *CollectionHandler) UpdateAlertRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}
	var req UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid update alert rule request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if err := h.service.UpdateAlertRule(c.Request.Context(), id, &req); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CollectionHandler) DeleteAlertRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}
	if err := h.service.DeleteAlertRule(c.Request.Context(), id); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CollectionHandler) GetVectorMetricsSummary(c *gin.Context) {
	summary, err := h.service.GetVectorMetricsSummary(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

//nolint:dupl // Request bind + service call pattern mirrors other create-style handlers.
func (h *CollectionHandler) NotifyAlertChannels(c *gin.Context) {
	var req NotifyAlertChannelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid notify alert channels request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	resp, err := h.service.NotifyAlertChannels(c.Request.Context(), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}
