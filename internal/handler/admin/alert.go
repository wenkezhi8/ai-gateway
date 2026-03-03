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
	ID               string `json:"id"`
	Time             string `json:"time"`
	Level            string `json:"level"` // critical, warning, info
	Source           string `json:"source"`
	Message          string `json:"message"`
	Status           string `json:"status"` // pending, resolved
	RuleID           string `json:"ruleId,omitempty"`
	ResolvedAt       string `json:"resolvedAt,omitempty"`
	DedupKey         string `json:"dedup_key,omitempty"`
	FirstTriggeredAt string `json:"first_triggered_at,omitempty"`
	LastTriggeredAt  string `json:"last_triggered_at,omitempty"`
	TriggerCount     int    `json:"trigger_count,omitempty"`
	AutoResolved     bool   `json:"auto_resolved,omitempty"`
}

// AlertStats represents alert statistics.
type AlertStats struct {
	Critical   int `json:"critical"`
	Warning    int `json:"warning"`
	TodayTotal int `json:"todayTotal"`
	Resolved   int `json:"resolved"`
}

type resolveSimilarAlertsRequest struct {
	Level    string `json:"level"`
	Source   string `json:"source"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key"`
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
	alertStatusPending   = "pending"
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
	h.compactLegacyAlertsOnce()
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
	dedupKey := strings.TrimSpace(req.DedupKey)
	if level == "" || source == "" || (dedupKey == "" && message == "") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": "level and source are required, and either dedup_key or message must be provided",
			},
		})
		return
	}

	targetDedup := dedupKey
	if targetDedup == "" {
		targetDedup = buildAlertDedupKey(level, source, message)
	}
	targetKey := buildAlertDedupKey(level, source, targetDedup)
	resolvedAt := time.Now().Format(time.RFC3339)
	affected := 0

	h.mu.Lock()
	for i := range h.alerts {
		alert := &h.alerts[i]
		if alert.Status == alertStatusResolved {
			continue
		}
		if strings.TrimSpace(alert.Level) != level || strings.TrimSpace(alert.Source) != source {
			continue
		}
		alertDedup := resolveAlertDedupKey(*alert)
		if alertDedup != targetDedup {
			continue
		}
		alert.Status = alertStatusResolved
		alert.ResolvedAt = resolvedAt
		alert.AutoResolved = false
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
	h.upsertAlertLocked(level, source, "", message, now)
}

func (h *AlertHandler) UpsertAlert(level, source, dedupKey, message string, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.upsertAlertLocked(level, source, dedupKey, message, now)
}

func (h *AlertHandler) upsertAlertLocked(level, source, dedupKey, message string, now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	level = strings.TrimSpace(level)
	source = strings.TrimSpace(source)
	message = strings.TrimSpace(message)
	dedupKey = strings.TrimSpace(dedupKey)
	if dedupKey == "" {
		dedupKey = buildAlertDedupKey(level, source, message)
	}
	nowStr := now.Format(time.RFC3339)

	for i := range h.alerts {
		alert := &h.alerts[i]
		if alert.Status != alertStatusPending {
			continue
		}
		if strings.TrimSpace(alert.Level) != level || strings.TrimSpace(alert.Source) != source {
			continue
		}
		if resolveAlertDedupKey(*alert) != dedupKey {
			continue
		}

		if alert.FirstTriggeredAt == "" {
			if alert.Time != "" {
				alert.FirstTriggeredAt = alert.Time
			} else {
				alert.FirstTriggeredAt = nowStr
			}
		}
		alert.LastTriggeredAt = nowStr
		if alert.TriggerCount <= 0 {
			alert.TriggerCount = 1
		}
		alert.TriggerCount++
		alert.Message = message
		alert.DedupKey = dedupKey
		h.saveData()
		return
	}

	h.alerts = append(h.alerts, AlertRecord{
		ID:               "alert-" + generateID(),
		Time:             nowStr,
		Level:            level,
		Source:           source,
		Message:          message,
		Status:           alertStatusPending,
		DedupKey:         dedupKey,
		FirstTriggeredAt: nowStr,
		LastTriggeredAt:  nowStr,
		TriggerCount:     1,
	})

	if len(h.alerts) > 1000 {
		h.alerts = h.alerts[len(h.alerts)-1000:]
	}

	h.saveData()
}

func (h *AlertHandler) ResolveByDedupKey(source, dedupKey string, auto bool) int {
	source = strings.TrimSpace(source)
	dedupKey = strings.TrimSpace(dedupKey)
	if source == "" || dedupKey == "" {
		return 0
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	resolvedAt := time.Now().Format(time.RFC3339)
	affected := 0
	for i := range h.alerts {
		alert := &h.alerts[i]
		if alert.Status == alertStatusResolved {
			continue
		}
		if strings.TrimSpace(alert.Source) != source {
			continue
		}
		if resolveAlertDedupKey(*alert) != dedupKey {
			continue
		}
		alert.Status = alertStatusResolved
		alert.ResolvedAt = resolvedAt
		alert.AutoResolved = auto
		affected++
	}

	if affected > 0 {
		h.saveData()
	}

	return affected
}

func (h *AlertHandler) compactLegacyAlertsOnce() {
	if len(h.alerts) == 0 {
		return
	}

	normalized := make([]AlertRecord, 0, len(h.alerts))
	pendingAggIndex := map[string]int{}

	for _, alert := range h.alerts {
		if alert.DedupKey == "" {
			if inferred := inferSystemHealthDedupKey(alert.Level, alert.Source, alert.Message); inferred != "" {
				alert.DedupKey = inferred
			} else {
				alert.DedupKey = buildAlertDedupKey(alert.Level, alert.Source, alert.Message)
			}
		}
		if alert.TriggerCount <= 0 {
			alert.TriggerCount = 1
		}
		if alert.FirstTriggeredAt == "" {
			alert.FirstTriggeredAt = alert.Time
		}
		if alert.LastTriggeredAt == "" {
			alert.LastTriggeredAt = alert.Time
		}

		if strings.TrimSpace(alert.Source) == "system" && alert.Status == alertStatusPending {
			aggKey := buildAlertDedupKey(alert.Level, alert.Source, alert.DedupKey)
			if idx, ok := pendingAggIndex[aggKey]; ok {
				target := &normalized[idx]
				target.TriggerCount += alert.TriggerCount
				target.FirstTriggeredAt = pickEarlierTime(target.FirstTriggeredAt, alert.FirstTriggeredAt)
				newLast := pickLaterTime(target.LastTriggeredAt, alert.LastTriggeredAt)
				if newLast != target.LastTriggeredAt {
					target.LastTriggeredAt = newLast
					target.Message = alert.Message
				}
				continue
			}
			pendingAggIndex[aggKey] = len(normalized)
		}

		normalized = append(normalized, alert)
	}

	h.alerts = normalized
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

func resolveAlertDedupKey(alert AlertRecord) string {
	dedupKey := strings.TrimSpace(alert.DedupKey)
	if dedupKey != "" {
		return dedupKey
	}
	if inferred := inferSystemHealthDedupKey(alert.Level, alert.Source, alert.Message); inferred != "" {
		return inferred
	}
	return buildAlertDedupKey(alert.Level, alert.Source, alert.Message)
}

func inferSystemHealthDedupKey(level, source, message string) string {
	if strings.TrimSpace(source) != "system" {
		return ""
	}
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return ""
	}

	switch {
	case strings.Contains(msg, "内存使用率过高"):
		return "memory_critical"
	case strings.Contains(msg, "内存使用率偏高"):
		return "memory_warning"
	case strings.Contains(msg, "goroutine 数过高"):
		return "goroutine_critical"
	case strings.Contains(msg, "goroutine 数偏高"):
		return "goroutine_warning"
	case strings.Contains(msg, "gc 暂停过长"):
		return "gc_pause_critical"
	case strings.Contains(msg, "gc 暂停偏长"):
		return "gc_pause_warning"
	case strings.Contains(msg, "memory") && strings.Contains(strings.ToLower(strings.TrimSpace(level)), "critical"):
		return "memory_critical"
	case strings.Contains(msg, "memory"):
		return "memory_warning"
	case strings.Contains(msg, "goroutine") && strings.Contains(strings.ToLower(strings.TrimSpace(level)), "critical"):
		return "goroutine_critical"
	case strings.Contains(msg, "goroutine"):
		return "goroutine_warning"
	case strings.Contains(msg, "gc") && strings.Contains(strings.ToLower(strings.TrimSpace(level)), "critical"):
		return "gc_pause_critical"
	case strings.Contains(msg, "gc"):
		return "gc_pause_warning"
	default:
		return ""
	}
}

func pickLaterTime(a, b string) string {
	ta := parseRFC3339OrZero(a)
	tb := parseRFC3339OrZero(b)
	if ta.IsZero() {
		return b
	}
	if tb.IsZero() {
		return a
	}
	if tb.After(ta) {
		return b
	}
	return a
}

func pickEarlierTime(a, b string) string {
	ta := parseRFC3339OrZero(a)
	tb := parseRFC3339OrZero(b)
	if ta.IsZero() {
		return b
	}
	if tb.IsZero() {
		return a
	}
	if tb.Before(ta) {
		return b
	}
	return a
}

func parseRFC3339OrZero(v string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
	if err != nil {
		return time.Time{}
	}
	return t
}
