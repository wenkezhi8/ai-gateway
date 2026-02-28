package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// NewHealthHandler creates a new health handler
func NewHealthHandler(checkers ...func() error) *HealthHandler {
	h := &HealthHandler{}
	if len(checkers) > 0 {
		h.readyCheck = checkers[0]
	}
	return h
}

type HealthHandler struct {
	readyCheck func() error
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
}

type ReadyResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Message   string    `json:"message,omitempty"`
}

// Check returns the health status of the service
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "ai-gateway",
	})
}

// CheckReady returns readiness status of the service dependencies
func (h *HealthHandler) CheckReady(c *gin.Context) {
	if h.readyCheck != nil {
		if err := h.readyCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, ReadyResponse{
				Status:    "not_ready",
				Timestamp: time.Now(),
				Service:   "ai-gateway",
				Message:   err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, ReadyResponse{
		Status:    "ready",
		Timestamp: time.Now(),
		Service:   "ai-gateway",
	})
}
