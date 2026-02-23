// Package admin provides feedback API handlers
// 改动点: 新增反馈 API 端点
package admin

import (
	"ai-gateway/internal/routing"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// FeedbackRequest represents a feedback submission request
type FeedbackRequest struct {
	RequestID  string `json:"request_id"`
	Model      string `json:"model"`
	Provider   string `json:"provider"`
	Rating     int    `json:"rating"` // 1-5
	Comment    string `json:"comment"`
	IsPositive bool   `json:"is_positive"`
}

// FeedbackHandler handles feedback-related requests
type FeedbackHandler struct {
	collector *routing.FeedbackCollector
}

var globalFeedbackHandler *FeedbackHandler

// InitFeedbackHandler initializes the global feedback handler
func InitFeedbackHandler(collector *routing.FeedbackCollector) {
	globalFeedbackHandler = &FeedbackHandler{
		collector: collector,
	}
}

// GetFeedbackHandler returns the global feedback handler
func GetFeedbackHandler() *FeedbackHandler {
	return globalFeedbackHandler
}

// SubmitFeedback handles feedback submission
func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	var req FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5"})
		return
	}

	feedbackType := routing.FeedbackNeutral
	if req.IsPositive {
		feedbackType = routing.FeedbackPositive
	} else if req.Rating <= 2 {
		feedbackType = routing.FeedbackNegative
	}

	feedback := routing.Feedback{
		RequestID:    req.RequestID,
		Model:        req.Model,
		Provider:     req.Provider,
		FeedbackType: feedbackType,
		Rating:       req.Rating,
		Comment:      req.Comment,
		Timestamp:    time.Now(),
	}

	h.collector.RecordFeedback(feedback)

	c.JSON(http.StatusOK, gin.H{
		"message": "Feedback recorded",
		"model":   req.Model,
		"rating":  req.Rating,
	})
}

// GetFeedbackStats returns feedback statistics
func (h *FeedbackHandler) GetFeedbackStats(c *gin.Context) {
	stats := h.collector.GetFeedbackStats()
	c.JSON(http.StatusOK, stats)
}

// GetModelPerformance returns performance metrics for a model
func (h *FeedbackHandler) GetModelPerformance(c *gin.Context) {
	model := c.Param("model")
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	perf := h.collector.GetPerformance(model)
	if perf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model performance not found"})
		return
	}

	c.JSON(http.StatusOK, perf)
}

// GetAllPerformance returns all model performance metrics
func (h *FeedbackHandler) GetAllPerformance(c *gin.Context) {
	perf := h.collector.GetAllPerformance()
	c.JSON(http.StatusOK, perf)
}

// GetTopModels returns top performing models
func (h *FeedbackHandler) GetTopModels(c *gin.Context) {
	taskType := c.DefaultQuery("task_type", "chat")
	limit := 10

	perf := h.collector.GetTopModels(routing.TaskType(taskType), limit)
	c.JSON(http.StatusOK, gin.H{
		"task_type": taskType,
		"models":    perf,
	})
}

// TriggerOptimization manually triggers score optimization
func (h *FeedbackHandler) TriggerOptimization(c *gin.Context) {
	h.collector.OptimizeScores()
	c.JSON(http.StatusOK, gin.H{
		"message": "Optimization completed",
	})
}

// GetRecentFeedback returns recent feedback entries
func (h *FeedbackHandler) GetRecentFeedback(c *gin.Context) {
	limit := 50
	feedback := h.collector.GetRecentFeedback(limit)
	c.JSON(http.StatusOK, gin.H{
		"feedback": feedback,
		"count":    len(feedback),
	})
}

// GetTaskTypeDistribution returns the distribution of task types
func (h *FeedbackHandler) GetTaskTypeDistribution(c *gin.Context) {
	distribution := h.collector.GetTaskTypeDistribution()
	c.JSON(http.StatusOK, gin.H{
		"distribution": distribution,
	})
}
