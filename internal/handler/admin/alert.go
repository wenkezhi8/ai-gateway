package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// generateID generates a random ID.
func generateID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

// AlertRule represents an alert rule configuration.
type AlertRule struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Enabled        bool           `json:"enabled"`
	Condition      AlertCondition `json:"condition"`
	NotifyChannels []string       `json:"notifyChannels"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt,omitempty"`
}

// AlertCondition represents the condition for triggering an alert.
type AlertCondition struct {
	Type      string  `json:"type"`               // latency, error_rate, quota, availability
	Operator  string  `json:"operator"`           // >, <, >=, <=, ==
	Threshold float64 `json:"threshold"`          // threshold value
	Duration  int     `json:"duration,omitempty"` // duration in seconds
}

// AlertRecord represents an alert history record.
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

// AlertStats represents alert statistics.
type AlertStats struct {
	Critical   int `json:"critical"`
	Warning    int `json:"warning"`
	TodayTotal int `json:"todayTotal"`
	Resolved   int `json:"resolved"`
}

type resolveSimilarAlertsRequest struct {
	Level   string `json:"level"`
	Source  string `json:"source"`
	Message string `json:"message"`
}

// AlertHandler handles alert-related requests.
type AlertHandler struct {
	rules    []AlertRule
	alerts   []AlertRecord
	mu       sync.RWMutex
	dataPath string

	alertCooldown time.Duration
	lastAlerts    map[string]time.Time
}

const (
	defaultAlertCooldown = 5 * time.Minute
	alertLevelWarning    = "warning"
	alertStatusResolved  = "resolved"
)

// Global alert handler.
var globalAlertHandler *AlertHandler

// NewAlertHandler creates a new alert handler.
func NewAlertHandler() *AlertHandler {
	h := &AlertHandler{
		rules:         make([]AlertRule, 0),
		alerts:        make([]AlertRecord, 0),
		dataPath:      "./data/alerts.json",
		alertCooldown: defaultAlertCooldown,
		lastAlerts:    make(map[string]time.Time),
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

// GetAlertHandler returns the global alert handler.
func GetAlertHandler() *AlertHandler {
	if globalAlertHandler == nil {
		return NewAlertHandler()
	}
	return globalAlertHandler
}

// loadData loads alert data from file.
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

// saveData saves alert data to file.
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

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(h.dataPath, jsonData, 0o640); err != nil {
		return
	}
}

// GET /api/admin/alerts/stats.
func (h *AlertHandler) GetStats(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	stats := AlertStats{}

	for i := range h.alerts {
		alert := &h.alerts[i]
		if strings.HasPrefix(alert.Time, today) {
			stats.TodayTotal++

			switch alert.Level {
			case "critical":
				stats.Critical++
			case alertLevelWarning:
				stats.Warning++
			}

			if alert.Status == alertStatusResolved {
				stats.Resolved++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GET /api/admin/alerts/rules.
func (h *AlertHandler) GetRules(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.rules,
	})
}

// POST /api/admin/alerts/rules.
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

// PUT /api/admin/alerts/rules/:id.
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

	for i := range h.rules {
		if h.rules[i].ID != id {
			continue
		}

		req.ID = id
		req.CreatedAt = h.rules[i].CreatedAt
		req.UpdatedAt = time.Now().Format(time.RFC3339)
		h.rules[i] = req
		h.saveData()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    req,
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Rule not found",
		},
	})
}

// DELETE /api/admin/alerts/rules/:id.
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.rules {
		if h.rules[i].ID != id {
			continue
		}

		h.rules = append(h.rules[:i], h.rules[i+1:]...)
		h.saveData()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Rule deleted",
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Rule not found",
		},
	})
}

// GET /api/admin/alerts/history.
func (h *AlertHandler) GetHistory(c *gin.Context) {
	level := c.Query("level")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	h.mu.RLock()
	defer h.mu.RUnlock()

	filtered := make([]AlertRecord, 0, len(h.alerts))
	for i := range h.alerts {
		alert := h.alerts[i]
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

// PUT /api/admin/alerts/:id/resolve.
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.alerts {
		if h.alerts[i].ID != id {
			continue
		}

		h.alerts[i].Status = alertStatusResolved
		h.alerts[i].ResolvedAt = time.Now().Format(time.RFC3339)
		h.saveData()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Alert resolved",
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_found",
			"message": "Alert not found",
		},
	})
}

// POST /api/admin/alerts/resolve-similar.
func (h *AlertHandler) ResolveSimilarAlerts(c *gin.Context) {
	var req resolveSimilarAlertsRequest
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

	level := strings.TrimSpace(req.Level)
	source := strings.TrimSpace(req.Source)
	message := strings.TrimSpace(req.Message)
	if level == "" || source == "" || message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": "level, source and message are required",
			},
		})
		return
	}

	targetKey := buildAlertDedupKey(level, source, message)
	resolvedAt := time.Now().Format(time.RFC3339)
	affected := 0

	h.mu.Lock()
	for i := range h.alerts {
		alert := &h.alerts[i]
		if alert.Status == alertStatusResolved {
			continue
		}
		if buildAlertDedupKey(alert.Level, alert.Source, alert.Message) != targetKey {
			continue
		}
		alert.Status = alertStatusResolved
		alert.ResolvedAt = resolvedAt
		affected++
	}
	if affected > 0 {
		h.saveData()
	}
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"affected": affected,
			"key":      targetKey,
		},
	})
}

// GET /api/admin/alerts/:id.
func (h *AlertHandler) GetAlertDetail(c *gin.Context) {
	id := c.Param("id")

	h.mu.RLock()
	defer h.mu.RUnlock()

	for i := range h.alerts {
		alert := &h.alerts[i]
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

// AddAlert adds a new alert (internal use).
func (h *AlertHandler) AddAlert(level, source, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := buildAlertDedupKey(level, source, message)
	now := time.Now()
	if h.alertCooldown > 0 {
		if h.lastAlerts == nil {
			h.lastAlerts = make(map[string]time.Time)
		}
		if last, ok := h.lastAlerts[key]; ok && now.Sub(last) < h.alertCooldown {
			return
		}
		h.lastAlerts[key] = now
	}

	alert := AlertRecord{
		ID:      "alert-" + generateID(),
		Time:    now.Format(time.RFC3339),
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

func buildAlertDedupKey(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.TrimSpace(part))
	}
	return strings.Join(normalized, "|")
}
