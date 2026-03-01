package vectordb

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RegisterRBACRoutes(r *gin.RouterGroup, service *RBACService) {
	if service == nil {
		return
	}
	handler := &rbacHandler{service: service}
	group := r.Group("/vector-db/permissions")
	group.GET("", handler.ListPermissions)
	group.POST("", handler.CreatePermission)
	group.DELETE("/:id", handler.DeletePermission)
}

type rbacHandler struct {
	service *RBACService
}

func (h *rbacHandler) ListPermissions(c *gin.Context) {
	items, err := h.service.ListAPIKeys(c.Request.Context())
	if err != nil {
		logrus.WithError(err).Error("list vector permissions failed")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "total": len(items)}})
}

func (h *rbacHandler) CreatePermission(c *gin.Context) {
	var req CreateVectorPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if err := h.service.CreateAPIKey(c.Request.Context(), req.APIKey, req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true})
}

func (h *rbacHandler) DeletePermission(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}
	if err := h.service.DeleteAPIKey(c.Request.Context(), id); err != nil {
		if err == ErrVectorAPIKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
