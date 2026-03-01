package vectordb

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type SearchHandler struct {
	service *Service
}

func NewSearchHandler(service *Service) *SearchHandler {
	if service == nil {
		service = NewService()
	}
	return &SearchHandler{service: service}
}

func RegisterVectorSearchRoutes(r *gin.RouterGroup, handler *SearchHandler) {
	vectors := r.Group("/vector/collections/:name")
	vectors.POST("/search", handler.SearchVectors)
	vectors.POST("/recommend", handler.RecommendVectors)
	vectors.GET("/vectors/:id", handler.GetVectorByID)
}

func RegisterVectorSearchRoutesWithRBAC(r *gin.RouterGroup, handler *SearchHandler, middlewares ...gin.HandlerFunc) {
	vectors := r.Group("/vector/collections/:name")
	for _, middleware := range middlewares {
		if middleware != nil {
			vectors.Use(middleware)
		}
	}
	vectors.POST("/search", handler.SearchVectors)
	vectors.POST("/recommend", handler.RecommendVectors)
	vectors.GET("/vectors/:id", handler.GetVectorByID)
}

//nolint:dupl // Request bind + service call pattern intentionally mirrors RecommendVectors handler.
func (h *SearchHandler) SearchVectors(c *gin.Context) {
	var req SearchVectorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid vector search request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	req.CollectionName = strings.TrimSpace(c.Param("name"))

	resp, err := h.service.SearchVectors(c.Request.Context(), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

//nolint:dupl // Request bind + service call pattern intentionally mirrors SearchVectors handler.
func (h *SearchHandler) RecommendVectors(c *gin.Context) {
	var req RecommendVectorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Warn("invalid vector recommend request")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	req.CollectionName = strings.TrimSpace(c.Param("name"))

	resp, err := h.service.RecommendVectors(c.Request.Context(), &req)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *SearchHandler) GetVectorByID(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	id := strings.TrimSpace(c.Param("id"))

	item, err := h.service.GetVectorByID(c.Request.Context(), name, id)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *SearchHandler) respondServiceError(c *gin.Context, err error) {
	tmp := &CollectionHandler{service: h.service}
	tmp.respondServiceError(c, err)
}
