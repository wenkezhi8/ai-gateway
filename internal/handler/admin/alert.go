package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// generateID generates a random ID
func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// AlertRule represents an alert rule configuration
type AlertRule struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Enabled        bool           `json:"enabled"`
	Condition      AlertCondition `json:"condition"`
	NotifyChannels []string       `json:"notifyChannels"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt,omitempty"`
}

// AlertCondition represents the condition for triggering an alert
type AlertCondition struct {
	Type      string  `json:"type"`               // latency, error_rate, quota, availability
	Operator  string  `json:"operator"`           // >, <, >=, <=, ==
	Threshold float64 `json:"threshold"`          // threshold value
	Duration  int     `json:"duration,omitempty"` // duration in seconds
}

// AlertRecord represents an alert history record
type AlertRecord struct {
	ID         string `json:"id"`
	Time       string `json:"time"`
	Level      string `json:"level"` // critical, warning, info
	Source     string `json:"source"`
	Message    string `json:"message"`
	Status     string `json:"status"` // pending, resolved
	RuleID     string `json:"ruleId,omitempty"`
	ResolvedAt string `json:"resolvedAt,omitempty"`
}

// AlertStats represents alert statistics
type AlertStats struct {
	Critical   int `json:"critical"`
	Warning    int `json:"warning"`
	TodayTotal int `json:"todayTotal"`
	Resolved   int `json:"resolved"`
}

// AlertHandler handles alert-related requests
type AlertHandler struct {
	rules    []AlertRule
	alerts   []AlertRecord
	mu       sync.RWMutex
	dataPath string
}

// Global alert handler
var globalAlertHandler *AlertHandler

// NewAlertHandler creates a new alert handler
func NewAlertHandler() *AlertHandler {
	h := &AlertHandler{
		rules:    make([]AlertRule, 0),
		alerts:   make([]AlertRecord, 0),
		dataPath: "./data/alerts.json",
	}

	// Load persisted data
	h.loadData()

	// Add some default rules if empty
	if len(h.rules) == 0 {
		h.rules = []AlertRule{
			{
				ID:      "rule-latency",
				Name:    "高延迟告警",
				Enabled: true,
				Condition: AlertCondition{
					Type:      "latency",
					Operator:  ">",
					Threshold: 5000,
					Duration:  60,
				},
				NotifyChannels: []string{"email", "webhook"},
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
			{
				ID:      "rule-error-rate",
				Name:    "错误率告警",
				Enabled: true,
				Condition: AlertCondition{
					Type:      "error_rate",
					Operator:  ">",
					Threshold: 5,
					Duration:  300,
				},
				NotifyChannels: []string{"email"},
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
			{
				ID:      "rule-quota",
				Name:    "配额告警",
				Enabled: true,
				Condition: AlertCondition{
					Type:      "quota",
					Operator:  ">",
					Threshold: 80,
				},
				NotifyChannels: []string{"email", "webhook"},
				CreatedAt:      time.Now().Format(time.RFC3339),
			},
		}
		h.saveData()
	}

	globalAlertHandler = h
	return h
}

// GetAlertHandler returns the global alert handler
func GetAlertHandler() *AlertHandler {
	if globalAlertHandler == nil {
		return NewAlertHandler()
	}
	return globalAlertHandler
}

// loadData loads alert data from file
func (h *AlertHandler) loadData() {
	fileData, err := os.ReadFile(h.dataPath)
	if err != nil {
		return
	}

	var data struct {
		Rules  []AlertRule   `json:"rules"`
		Alerts []AlertRecord `json:"alerts"`
	}

	if err := json.Unmarshal(fileData, &data); err != nil {
		return
	}

	h.rules = data.Rules
	h.alerts = data.Alerts
}

// saveData saves alert data to file
func (h *AlertHandler) saveData() {
	dir := filepath.Dir(h.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	data := struct {
		Rules  []AlertRule   `json:"rules"`
		Alerts []AlertRecord `json:"alerts"`
	}{
		Rules:  h.rules,
		Alerts: h.alerts,
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(h.dataPath, jsonData, 0644)
}

// GetStats returns alert statistics
// GET /api/admin/alerts/stats
func (h *AlertHandler) GetStats(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	stats := AlertStats{}

	for _, alert := range h.alerts {
		if alert.Time[:10] == today {
			stats.TodayTotal++

			switch alert.Level {
			case "critical":
				stats.Critical++
			case "warning":
				stats.Warning++
			}

			if alert.Status == "resolved" {
				stats.Resolved++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetRules returns all alert rules
// GET /api/admin/alerts/rules
func (h *AlertHandler) GetRules(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.rules,
	})
}

// CreateRule creates a new alert rule
// POST /api/admin/alerts/rules
func (h *AlertHandler) CreateRule(c *gin.Context) {
	var req AlertRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Generate ID
	req.ID = "rule-" + generateID()
	req.CreatedAt = time.Now().Format(time.RFC3339)

	h.rules = append(h.rules, req)
	h.saveData()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    req,
	})
}

// UpdateRule updates an alert rule
// PUT /api/admin/alerts/rules/:id
func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")

	var req AlertRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for i, rule := range h.rules {
		if rule.ID == id {
			req.ID = id
			req.CreatedAt = rule.CreatedAt
			req.UpdatedAt = time.Now().Format(time.RFC3339)
			h.rules[i] = req
			h.saveData()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    req,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Rule not found",
		},
	})
}

// DeleteRule deletes an alert rule
// DELETE /api/admin/alerts/rules/:id
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	for i, rule := range h.rules {
		if rule.ID == id {
			h.rules = append(h.rules[:i], h.rules[i+1:]...)
			h.saveData()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Rule deleted",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Rule not found",
		},
	})
}

// GetHistory returns alert history
// GET /api/admin/alerts/history
func (h *AlertHandler) GetHistory(c *gin.Context) {
	level := c.Query("level")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	h.mu.RLock()
	defer h.mu.RUnlock()

	var filtered []AlertRecord
	for _, alert := range h.alerts {
		// Filter by level
		if level != "" && alert.Level != level {
			continue
		}

		// Filter by date range
		if startDate != "" && alert.Time < startDate {
			continue
		}
		if endDate != "" && alert.Time > endDate+"T23:59:59" {
			continue
		}

		filtered = append(filtered, alert)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"list":  filtered,
			"total": len(filtered),
		},
	})
}

// ResolveAlert resolves an alert
// PUT /api/admin/alerts/:id/resolve
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	for i, alert := range h.alerts {
		if alert.ID == id {
			h.alerts[i].Status = "resolved"
			h.alerts[i].ResolvedAt = time.Now().Format(time.RFC3339)
			h.saveData()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Alert resolved",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Alert not found",
		},
	})
}

// GetAlertDetail returns alert detail
// GET /api/admin/alerts/:id
func (h *AlertHandler) GetAlertDetail(c *gin.Context) {
	id := c.Param("id")

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, alert := range h.alerts {
		if alert.ID == id {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    alert,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Alert not found",
		},
	})
}

// AddAlert adds a new alert (internal use)
func (h *AlertHandler) AddAlert(level, source, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	alert := AlertRecord{
		ID:      "alert-" + generateID(),
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		Source:  source,
		Message: message,
		Status:  "pending",
	}

	h.alerts = append(h.alerts, alert)

	// Keep only last 1000 alerts
	if len(h.alerts) > 1000 {
		h.alerts = h.alerts[len(h.alerts)-1000:]
	}

	h.saveData()
}
