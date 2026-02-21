package alert

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles alert-related HTTP requests
type Handler struct {
	notifier *Notifier
}

// NewHandler creates a new alert handler
func NewHandler(notifier *Notifier) *Handler {
	return &Handler{
		notifier: notifier,
	}
}

// WebhookHandler handles Alertmanager webhook requests
func (h *Handler) WebhookHandler(c *gin.Context) {
	var payload map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&payload); err != nil {
		log.Printf("[Alert] Failed to decode webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	alerts, err := FormatAlertFromPrometheus(payload)
	if err != nil {
		log.Printf("[Alert] Failed to format alerts: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var errors []string
	for _, alert := range alerts {
		if err := h.notifier.Send(alert); err != nil {
			log.Printf("[Alert] Failed to send notification: %v", err)
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		c.JSON(http.StatusPartialContent, gin.H{
			"message": "some notifications failed",
			"errors":  errors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "notifications sent",
		"count":   len(alerts),
	})
}

// HealthHandler returns the health status of the alert service
func (h *Handler) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "alert",
	})
}
